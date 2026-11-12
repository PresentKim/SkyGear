package platform

import (
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/platform/instance"
)

// Blob cache test (/cachetest): an operator picks a save, the save's world is
// copied into a test world, and the operator is moved into it. Every chunk
// sent there is reported in chat: how many of its blobs the client got new
// (misses) and how many it already had in its blob cache (hits), split by
// where the server had sent that hash before on this connection.

type cacheTest struct {
	target int64
	tw     *instance.TestWorld
	sum    cacheSum
}

// cacheSum adds up the reports of one test run.
type cacheSum struct {
	mu                                  sync.Mutex
	chunks, empty                       int
	blobs, fresh, cached                int
	here, elsewhere, before, unanswered int
	last                                time.Time
	waiting                             bool
}

var cacheTests sync.Map // account ID -> *cacheTest

func cacheTestOf(p *player.Player) *cacheTest {
	v, _ := cacheTests.LoadOrStore(instance.AccountID(p), &cacheTest{})
	return v.(*cacheTest)
}

type cacheTestCmd struct {
	pf     *Platform
	Action cacheTestAction     `cmd:"action"`
	Save   cmd.Optional[int64] `cmd:"save"`
}

type cacheTestAction string

func (cacheTestAction) Type() string { return "sg_cachetest" }
func (cacheTestAction) Options(cmd.Source) []string {
	return []string{"target", "start", "stop", "resend", "status"}
}

func (c cacheTestCmd) Allow(src cmd.Source) bool {
	p, ok := src.(*player.Player)
	return ok && c.pf.isOperator != nil && c.pf.isOperator(p)
}

func (c cacheTestCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	lang := i18n.Of(p)
	t := cacheTestOf(p)
	switch c.Action {
	case "target":
		id, ok := c.Save.Load()
		if !ok {
			o.Error(i18n.T(lang, "platform.cachetest.needSave"))
			return
		}
		s, err := c.pf.db.Save(id)
		if err != nil {
			o.Errorf(i18n.T(lang, "platform.cachetest.noSave"), id, err)
			return
		}
		t.target = id
		o.Printf(i18n.T(lang, "platform.cachetest.target"), s.ID, s.Name, s.Day)
	case "start":
		c.start(p, o, t)
	case "stop":
		if t.tw == nil || tx.World() != t.tw.World {
			o.Error(i18n.T(lang, "platform.cachetest.notIn"))
			return
		}
		sghook.UnwatchChunks(p)
		c.pf.m.LeaveTestWorld(p)
		tw := t.tw
		t.tw = nil
		go func() {
			time.Sleep(time.Second)
			_ = tw.Close()
		}()
		o.Print(i18n.T(lang, "platform.cachetest.stopped"))
	case "resend":
		if t.tw == nil || tx.World() != t.tw.World {
			o.Error(i18n.T(lang, "platform.cachetest.notIn"))
			return
		}
		t.sum.reset()
		o.Printf(i18n.T(lang, "platform.cachetest.resent"), sghook.ResendChunks(p, tx))
	default:
		enabled, st, _ := sghook.ChunkCache(p)
		o.Printf(i18n.T(lang, "platform.cachetest.status"), t.target, t.tw != nil, enabled)
		o.Printf(i18n.T(lang, "platform.cachetest.counters"), st.Hits, st.Misses, st.MissBlobs, st.MissBytes)
	}
}

func (c cacheTestCmd) start(p *player.Player, o *cmd.Output, t *cacheTest) {
	lang := i18n.Of(p)
	if t.target == 0 {
		o.Error(i18n.T(lang, "platform.cachetest.noTarget"))
		return
	}
	if t.tw != nil || c.pf.m.Location(instance.AccountID(p)) != nil || !c.pf.m.InLobby(instance.AccountID(p)) {
		o.Error(i18n.T(lang, "platform.cachetest.lobbyOnly"))
		return
	}
	enabled, _, _ := sghook.ChunkCache(p)
	o.Printf(i18n.T(lang, "platform.cachetest.copying"), t.target, enabled)
	target, acc := t.target, instance.AccountID(p)
	go func() {
		tw, err := c.pf.m.OpenTestWorld(target, acc)
		p.H().Do(func(tx *world.Tx, e world.Entity) {
			p := e.(*player.Player)
			if err != nil {
				p.Message(i18n.T(i18n.Of(p), "platform.cachetest.openFailed", err))
				return
			}
			t.sum.reset()
			sghook.WatchChunks(p, func(r sghook.ChunkReport) { c.report(p, t, r) })
			if err := c.pf.m.EnterTestWorld(p, tw); err != nil {
				sghook.UnwatchChunks(p)
				_ = tw.Close()
				p.Message(i18n.T(i18n.Of(p), "platform.cachetest.enterFailed", err))
				return
			}
			t.tw = tw
		})
	}()
}

// report shows one chunk report to p (chunks with block data; chunks with
// only air go into the total) and the total once the reports stop.
func (c cacheTestCmd) report(p *player.Player, t *cacheTest, r sghook.ChunkReport) {
	c.pf.log.Info("cache test chunk", "player", p.Name(), "x", r.Pos[0], "z", r.Pos[1], "blobs", r.Blobs,
		"fresh", r.Fresh, "cached", r.Cached, "here", r.CachedHere, "elsewhere", r.CachedElsewhere,
		"before", r.CachedBefore, "unanswered", r.Unanswered, "air", r.AllAir, "inline", r.Inline, "inline_bytes", r.InlineBytes)
	empty := r.Blobs <= 1 && r.Inline == 0
	if t.sum.add(r, empty) {
		// The first report since the last total: print the next total once
		// the reports stop.
		go func() {
			for {
				time.Sleep(time.Second)
				if line, ok := t.sum.total(i18n.Of(p), 3*time.Second); ok {
					p.H().Do(func(_ *world.Tx, e world.Entity) { e.(*player.Player).Message(line) })
					return
				}
			}
		}()
	}
	if empty {
		return
	}
	msg := chunkLine(i18n.Of(p), r)
	p.H().Do(func(_ *world.Tx, e world.Entity) { e.(*player.Player).Message(msg) })
}

func chunkLine(lang string, r sghook.ChunkReport) string {
	var b strings.Builder
	b.WriteString(i18n.T(lang, "platform.cachetest.chunk", r.Pos[0], r.Pos[1], r.Blobs))
	switch {
	case r.Blobs > 0 && r.Cached == r.Blobs:
		b.WriteString(i18n.T(lang, "platform.cachetest.allCached"))
	case r.Cached == 0 && r.Blobs > 0:
		b.WriteString(i18n.T(lang, "platform.cachetest.allFresh"))
	default:
		b.WriteString(i18n.T(lang, "platform.cachetest.mixed", r.Fresh, r.Cached))
	}
	if r.Cached > 0 {
		b.WriteString(i18n.T(lang, "platform.cachetest.where", r.CachedHere, r.CachedElsewhere, r.CachedBefore))
	}
	if r.Unanswered > 0 {
		b.WriteString(i18n.T(lang, "platform.cachetest.unanswered", r.Unanswered))
	}
	if r.Inline > 0 {
		b.WriteString(i18n.T(lang, "platform.cachetest.inline", r.Inline))
	}
	b.WriteString(i18n.T(lang, "platform.cachetest.tail", r.AllAir, r.InlineBytes))
	return b.String()
}

func (s *cacheSum) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zero()
}

func (s *cacheSum) zero() {
	s.chunks, s.empty, s.blobs, s.fresh, s.cached = 0, 0, 0, 0, 0
	s.here, s.elsewhere, s.before, s.unanswered = 0, 0, 0, 0
	s.last = time.Time{}
}

// add adds r to the total. It returns true for the first report since the
// last total was taken.
func (s *cacheSum) add(r sghook.ChunkReport, empty bool) (first bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	first = !s.waiting
	s.waiting = true
	s.chunks++
	if empty {
		s.empty++
	}
	s.blobs += r.Blobs
	s.fresh += r.Fresh
	s.cached += r.Cached
	s.here += r.CachedHere
	s.elsewhere += r.CachedElsewhere
	s.before += r.CachedBefore
	s.unanswered += r.Unanswered
	s.last = time.Now()
	return first
}

// total returns the total line once, if no report came for quiet.
func (s *cacheSum) total(lang string, quiet time.Duration) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.last) < quiet {
		return "", false
	}
	s.waiting = false
	if s.chunks == 0 {
		return "", false
	}
	defer s.zero()
	return i18n.T(lang, "platform.cachetest.total",
		s.chunks, s.empty, s.blobs, s.fresh, s.cached, s.here, s.elsewhere, s.before, s.unanswered), true
}

// quitCacheTest closes the test world of a player leaving the server.
func quitCacheTest(p *player.Player) {
	v, ok := cacheTests.LoadAndDelete(instance.AccountID(p))
	if !ok {
		return
	}
	if tw := v.(*cacheTest).tw; tw != nil {
		go func() {
			time.Sleep(2 * time.Second)
			_ = tw.Close()
		}()
	}
}
