package sghook

// Per-chunk blob cache reports: which chunks a client got new data for and
// which it already had in its blob cache.

import (
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ChunkReport is what happened to the blobs of one chunk sent to a client.
type ChunkReport struct {
	Pos world.ChunkPos
	// Blobs is the number of blob hashes sent for the chunk (biomes and sub
	// chunks with blocks).
	Blobs int
	// Fresh is the number of those the client lacked (misses): their data
	// was sent now. Cached is the number it already had (hits).
	Fresh, Cached int
	// CachedHere, CachedElsewhere and CachedBefore split Cached by where the
	// server had sent that hash before on this connection: to this chunk, only
	// to other chunks, or never (the client kept it from an earlier session).
	CachedHere, CachedElsewhere, CachedBefore int
	// Unanswered is the number of hashes the client did not report within
	// chunkReportTimeout.
	Unanswered int
	// AllAir is the number of sub chunks sent as all air (no data at all),
	// Inline the number sent with their block data in the packet (no cache).
	AllAir, Inline int
	// InlineBytes is the payload sent in the packets themselves: block
	// entities, and the block data of inline sub chunks.
	InlineBytes int
}

const (
	// chunkReportQuiet is how long a chunk must have no open hashes and no
	// new packets before it is reported.
	chunkReportQuiet = 1500 * time.Millisecond
	// chunkReportTimeout reports a chunk with hashes still open.
	chunkReportTimeout = 10 * time.Second
)

// hitClass is where a hash was sent before, decided when it is sent.
type hitClass uint8

const (
	hitBefore hitClass = iota
	hitElsewhere
	hitHere
)

type pendingHash struct {
	pos   world.ChunkPos
	class hitClass
}

type chunkState struct {
	r     ChunkReport
	open  int
	first time.Time
	last  time.Time
}

type chunkWatch struct {
	mu      sync.Mutex
	fn      func(ChunkReport)
	chunks  map[world.ChunkPos]*chunkState
	queue   map[uint64][]pendingHash
	seen    map[uint64]map[world.ChunkPos]struct{}
	stop    chan struct{}
	stopped sync.Once
}

var watches sync.Map // *Conn -> *chunkWatch

// WatchChunks calls fn with a ChunkReport for every chunk sent to p from now
// on, once the client answered its blob hashes (or did not in time). fn runs
// on a goroutine of its own, not in a world transaction. Watching again
// replaces fn and forgets what was sent before. It returns false if p has no
// hooked connection.
func WatchChunks(p *player.Player, fn func(ChunkReport)) bool {
	c := Of(p)
	if c == nil {
		return false
	}
	w := &chunkWatch{
		fn:     fn,
		chunks: map[world.ChunkPos]*chunkState{},
		queue:  map[uint64][]pendingHash{},
		seen:   map[uint64]map[world.ChunkPos]struct{}{},
		stop:   make(chan struct{}),
	}
	if old, ok := watches.Swap(c, w); ok {
		old.(*chunkWatch).close()
	}
	go w.run()
	return true
}

// UnwatchChunks stops the reports of p's connection.
func UnwatchChunks(p *player.Player) {
	if c := Of(p); c != nil {
		unwatch(c)
	}
}

func unwatch(c *Conn) {
	if w, ok := watches.LoadAndDelete(c); ok {
		w.(*chunkWatch).close()
	}
}

func (w *chunkWatch) close() { w.stopped.Do(func() { close(w.stop) }) }

func (w *chunkWatch) run() {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-w.stop:
			return
		case now := <-t.C:
			for _, r := range w.due(now) {
				w.fn(r)
			}
		}
	}
}

// due takes the chunks to report at now.
func (w *chunkWatch) due(now time.Time) []ChunkReport {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []ChunkReport
	for pos, st := range w.chunks {
		quiet := st.open == 0 && now.Sub(st.last) >= chunkReportQuiet
		if !quiet && now.Sub(st.first) < chunkReportTimeout {
			continue
		}
		st.r.Unanswered = st.open
		out = append(out, st.r)
		delete(w.chunks, pos)
		if st.open > 0 {
			w.dropPending(pos)
		}
	}
	return out
}

// dropPending forgets the open hashes of pos (reported unanswered).
func (w *chunkWatch) dropPending(pos world.ChunkPos) {
	for h, q := range w.queue {
		kept := q[:0]
		for _, e := range q {
			if e.pos != pos {
				kept = append(kept, e)
			}
		}
		if len(kept) == 0 {
			delete(w.queue, h)
		} else {
			w.queue[h] = kept
		}
	}
}

func (w *chunkWatch) chunk(pos world.ChunkPos) *chunkState {
	st, ok := w.chunks[pos]
	now := time.Now()
	if !ok {
		st = &chunkState{r: ChunkReport{Pos: pos}, first: now}
		w.chunks[pos] = st
	}
	st.last = now
	return st
}

// sent records that hash h was sent for the chunk at pos.
func (w *chunkWatch) sent(st *chunkState, h uint64) {
	pos := st.r.Pos
	class := hitBefore
	if at := w.seen[h]; len(at) > 0 {
		class = hitElsewhere
		if _, ok := at[pos]; ok {
			class = hitHere
		}
	} else if at == nil {
		w.seen[h] = map[world.ChunkPos]struct{}{}
	}
	w.seen[h][pos] = struct{}{}
	w.queue[h] = append(w.queue[h], pendingHash{pos: pos, class: class})
	st.r.Blobs++
	st.open++
}

func (w *chunkWatch) serverPacket(pk packet.Packet) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch pk := pk.(type) {
	case *packet.LevelChunk:
		st := w.chunk(world.ChunkPos{pk.Position[0], pk.Position[1]})
		if pk.CacheEnabled {
			for _, h := range pk.BlobHashes {
				w.sent(st, h)
			}
		} else {
			st.r.Inline += int(pk.SubChunkCount)
		}
		st.r.InlineBytes += len(pk.RawPayload)
	case *packet.SubChunk:
		for _, e := range pk.SubChunkEntries {
			st := w.chunk(world.ChunkPos{pk.Position[0] + int32(e.Offset[0]), pk.Position[2] + int32(e.Offset[2])})
			raw, _ := e.RawPayload.Value()
			st.r.InlineBytes += len(raw)
			switch e.Result {
			case protocol.SubChunkResultSuccessAllAir:
				st.r.AllAir++
			case protocol.SubChunkResultSuccess:
				if h, ok := e.BlobHash.Value(); ok && pk.CacheEnabled {
					w.sent(st, h)
				} else {
					st.r.Inline++
				}
			}
		}
	}
}

func (w *chunkWatch) blobStatus(pk *packet.ClientCacheBlobStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	resolve := func(h uint64, hit bool) {
		q := w.queue[h]
		if len(q) == 0 {
			return
		}
		e := q[0]
		if len(q) == 1 {
			delete(w.queue, h)
		} else {
			w.queue[h] = q[1:]
		}
		st, ok := w.chunks[e.pos]
		if !ok {
			return
		}
		st.open--
		st.last = time.Now()
		if !hit {
			st.r.Fresh++
			return
		}
		st.r.Cached++
		switch e.class {
		case hitHere:
			st.r.CachedHere++
		case hitElsewhere:
			st.r.CachedElsewhere++
		default:
			st.r.CachedBefore++
		}
	}
	for _, h := range pk.HitHashes {
		resolve(h, true)
	}
	for _, h := range pk.MissHashes {
		resolve(h, false)
	}
}

func init() {
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		if w, ok := watches.Load(c); ok {
			w.(*chunkWatch).serverPacket(pk)
		}
		return false
	})
	OnClientPacket(func(c *Conn, pk packet.Packet) bool {
		if pk, ok := pk.(*packet.ClientCacheBlobStatus); ok {
			if w, ok := watches.Load(c); ok {
				w.(*chunkWatch).blobStatus(pk)
			}
		}
		return false
	})
	OnClose(unwatch)
}
