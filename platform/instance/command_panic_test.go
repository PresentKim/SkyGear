package instance

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook/hooktest"
	_ "github.com/presentkim/skygear/platform/locales"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Commands of the pin test. They record what they were given in observed.
var (
	observedMu sync.Mutex
	observed   []string
)

func observe(format string, a ...any) {
	observedMu.Lock()
	defer observedMu.Unlock()
	observed = append(observed, fmt.Sprintf(format, a...))
}

func takeObserved() []string {
	observedMu.Lock()
	defer observedMu.Unlock()
	o := observed
	observed = nil
	return o
}

type pinMode string

func (pinMode) Type() string                { return "sgpinmode" }
func (pinMode) Options(cmd.Source) []string { return []string{"fast", "slow"} }

type pinOK struct{}

func (pinOK) Run(_ cmd.Source, o *cmd.Output, tx *world.Tx) {
	observe("pinOK tx=%v", tx != nil)
	o.Print("line one")
	o.Printf("line %d", 2)
	o.Print("§aline three")
}

type pinErr struct{}

func (pinErr) Run(_ cmd.Source, o *cmd.Output, _ *world.Tx) {
	observe("pinErr")
	o.Print("before")
	o.Error("first error")
	o.Errorf("second %s", "error")
}

type pinArgs struct {
	Who   string                    `cmd:"who"`
	Count int                       `cmd:"count"`
	Mode  cmd.Optional[pinMode]     `cmd:"mode"`
	Rest  cmd.Optional[cmd.Varargs] `cmd:"rest"`
}

func (a pinArgs) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	mode, set := a.Mode.Load()
	rest, rset := a.Rest.Load()
	observe("pinArgs who=%q count=%d mode=%q/%v rest=%q/%v src=%T", a.Who, a.Count, mode, set, rest, rset, src)
	o.Printf("who=%s count=%d", a.Who, a.Count)
}

type pinDeny struct{}

func (pinDeny) Allow(cmd.Source) bool { return false }
func (pinDeny) Run(cmd.Source, *cmd.Output, *world.Tx) {
	observe("pinDeny ran")
}

type pinCancelled struct{}

func (pinCancelled) Run(_ cmd.Source, o *cmd.Output, _ *world.Tx) {
	observe("pinCancelled ran")
	o.Print("must not run")
}

type pinEnd struct{}

func (pinEnd) Run(_ cmd.Source, o *cmd.Output, _ *world.Tx) { o.Print("end") }

type pinPanic struct{}

func (pinPanic) Run(_ cmd.Source, o *cmd.Output, _ *world.Tx) {
	o.Print("printed before the panic")
	panic("boom")
}

func init() {
	cmd.Register(cmd.New("sgpinok", "", []string{"sgpinokalias"}, pinOK{}))
	cmd.Register(cmd.New("sgpinerr", "", nil, pinErr{}))
	cmd.Register(cmd.New("sgpinargs", "", nil, pinArgs{}))
	cmd.Register(cmd.New("sgpindeny", "", nil, pinDeny{}))
	cmd.Register(cmd.New("sgpincancel", "", nil, pinCancelled{}))
	cmd.Register(cmd.New("sgpinend", "", nil, pinEnd{}))
	cmd.Register(cmd.New("sgpinpanic", "", nil, pinPanic{}))
}

// inner is the handler of the player's location in these tests.
type inner struct {
	player.NopHandler
	cancel string // command name to cancel
}

func (h inner) HandleCommandExecution(ctx *player.Context, c cmd.Command, args []string) {
	observe("handler %s args=%q", c.Name(), args)
	if c.Name() == h.cancel {
		ctx.Cancel()
	}
}

// pinClient joins a client whose player has the platform Handler around inner.
type pinClient struct {
	*hooktest.Client
	t    *testing.T
	done int // packets already attributed to a run
}

func join(t *testing.T) *pinClient {
	cl := hooktest.Join(t, server.Config{}, nil)
	pc := &pinClient{Client: cl, t: t}
	pc.do(func(p *player.Player) {
		h := &Handler{}
		h.set(nil, inner{cancel: "sgpincancel"})
		p.Handle(h)
	})
	// The Handler has no Manager: give the player a plain handler back before
	// the server closes, so quitting does not reach it.
	t.Cleanup(func() { pc.do(func(p *player.Player) { p.Handle(player.NopHandler{}) }) })
	return pc
}

func (pc *pinClient) do(f func(p *player.Player)) {
	pc.t.Helper()
	<-pc.Server().World().Do(func(tx *world.Tx) {
		e, ok := pc.Handle().Entity(tx)
		if !ok {
			pc.t.Error("player not in the world")
			return
		}
		f(e.(*player.Player))
	}).Done()
}

func isOutput(p packet.Packet) bool { _, ok := p.(*packet.CommandOutput); return ok }

// run executes line as the player and returns what the client was sent for it
// (every CommandOutput, canonical) and what the commands and the handler observed.
func (pc *pinClient) run(line string) string {
	pc.t.Helper()
	before := len(pc.Packets())
	takeObserved()
	pc.do(func(p *player.Player) { p.ExecuteCommand(line) })
	obs := takeObserved()
	pc.do(func(p *player.Player) { p.ExecuteCommand("/sgpinend") })
	takeObserved()
	end := pc.Wait(func(p packet.Packet) bool {
		o, ok := p.(*packet.CommandOutput)
		return ok && len(o.OutputMessages) == 1 && o.OutputMessages[0].Message == "end"
	})
	var b strings.Builder
	fmt.Fprintf(&b, "line %q\nobserved %q\n", line, obs)
	for _, p := range pc.Packets()[before:] {
		if p == end {
			break
		}
		if !isOutput(p) {
			continue // join and tick traffic arrives at any time
		}
		fmt.Fprintf(&b, "%s\n", hooktest.Canon(p))
	}
	return b.String()
}

// TestCommandPin: what a player gets from running a command is the same
// whoever executes it. The expectations were recorded before the panic
// recovery moved from Dragonfly into Handler and did not change after.
func TestCommandPin(t *testing.T) {
	pc := join(t)
	cases := []struct{ name, line string }{
		{"ok", "/sgpinok"},
		{"ok-alias", "/sgpinokalias"},
		{"errors", "/sgpinerr"},
		{"args-all", `/sgpinargs "Kim Min" 3 fast tail words "quoted tail"`},
		{"args-required-only", "/sgpinargs a 1"},
		{"args-double-space", "/sgpinargs  a  1"},
		{"args-bad-enum", "/sgpinargs a 1 medium"},
		{"usage-bad-int", "/sgpinargs a x"},
		{"usage-missing", "/sgpinargs a"},
		{"usage-none", "/sgpinargs"},
		{"unknown", "/sgpinnone arg"},
		{"cancelled", "/sgpincancel"},
		{"denied", "/sgpindeny"},
		{"no-slash", "sgpinok"},
		{"trailing-space", "/sgpinok "},
	}
	entries := map[string]string{}
	for _, c := range cases {
		entries[c.name] = pc.run(c.line)
	}
	hooktest.Golden(t, "testdata/command_panic.golden", entries)
}

// TestCommandPanicRecovered: a command that panics reports one error to the
// player in the client's language, writes one ERROR record with the stack,
// and neither the server nor the player's transaction is harmed.
func TestCommandPanicRecovered(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(old)

	pc := join(t)
	got := pc.run("/sgpinpanic")
	if n := strings.Count(logs.String(), "level=ERROR"); n != 1 {
		t.Fatalf("%d ERROR records, want 1:\n%s", n, logs.String())
	}
	for _, want := range []string{"command panicked", "command=sgpinpanic", "panic=boom", "stack="} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, logs.String())
		}
	}
	const text = "명령어를 처리하다 오류가 났습니다."
	if n := strings.Count(got, text); n != 1 {
		t.Fatalf("error text %d times, want 1:\n%s", n, got)
	}
	// The transaction is usable: a normal command still runs.
	if after := pc.run("/sgpinok"); !strings.Contains(after, "line one") {
		t.Fatalf("command after the panic:\n%s", after)
	}
	t.Logf("panic run:\n%s", got)
}
