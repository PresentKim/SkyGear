// Command cachebot connects to a Sky Gear server as a client with the blob
// cache (chunk cache) enabled, to test the server side of the cache without
// a game client. It keeps the blobs it is sent in a file, as the game keeps
// its cache between sessions, so a second run shows what a reconnecting
// client still has.
//
// It answers every LevelChunk with a SubChunkRequest for the whole column
// and every blob hash with ClientCacheBlobStatus (hits it has, misses it
// lacks), stores the blobs of ClientCacheMissResponse and prints the counts.
// With -resend it runs "/sgdebug chunkcache resend" after the first wait
// (the bot's name must be in Admin.Operators) and counts again.
//
// The server must have AuthEnabled = false in config.toml.
//
//	go run ./tools/cachebot -addr 127.0.0.1:19132 -cache cachebot.blobs -resend
package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type stats struct {
	levelChunks, subEntries, blobEntries int
	hits, misses, missBlobs, missBytes   int
	inlineBytes, badBlobs, unanswered    int
}

func (s stats) String() string {
	return fmt.Sprintf("LevelChunk %d, sub chunk entries %d (blob %d), inline %d B | hits %d, misses %d | miss response %d blobs %d B, bad hash %d, unanswered/queued %d",
		s.levelChunks, s.subEntries, s.blobEntries, s.inlineBytes, s.hits, s.misses, s.missBlobs, s.missBytes, s.badBlobs, s.unanswered)
}

type bot struct {
	conn  *minecraft.Conn
	mu    sync.Mutex
	cache map[uint64][]byte
	want  map[uint64]struct{} // misses not answered yet
	st    stats
	// pending is the blob status not sent yet: like the game, the bot
	// sends one ClientCacheBlobStatus per tick for all hashes since.
	pending packet.ClientCacheBlobStatus
	// requests are the sub chunk requests not sent yet. The server closes
	// a connection whose batch holds too many packets, so at most
	// requestsPerTick go out per tick.
	requests []*packet.SubChunkRequest
}

const requestsPerTick = 64

func main() {
	addr := flag.String("addr", "127.0.0.1:19132", "server address")
	name := flag.String("name", "cachebot", "player name")
	file := flag.String("cache", "cachebot.blobs", "blob cache file (kept between runs)")
	wait := flag.Duration("wait", 15*time.Second, "how long to count after spawning (and after the resend)")
	resend := flag.Bool("resend", false, "run /sgdebug chunkcache resend after the first wait")
	flag.Parse()

	b := &bot{cache: load(*file), want: map[uint64]struct{}{}}
	log.Printf("blob cache: %d blobs from %s", len(b.cache), *file)

	conn, err := minecraft.Dialer{
		EnableClientCache: true,
		IdentityData:      login.IdentityData{DisplayName: *name},
	}.Dial("raknet", *addr)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	b.conn = conn
	if err := conn.DoSpawn(); err != nil {
		log.Fatalf("spawn: %v", err)
	}
	log.Printf("spawned (client cache %v)", conn.ClientCacheEnabled())
	go b.read()
	go b.flush()

	time.Sleep(*wait)
	log.Printf("join:   %v", b.take())
	if *resend {
		_ = conn.WritePacket(&packet.CommandRequest{
			CommandLine:   "/sgdebug chunkcache resend",
			CommandOrigin: protocol.CommandOrigin{Origin: protocol.CommandOriginPlayer, UUID: uuid.New()},
			Version:       "latest",
		})
		time.Sleep(*wait)
		log.Printf("resend: %v", b.take())
	}
	_ = conn.Close()
	b.mu.Lock()
	save(*file, b.cache)
	log.Printf("blob cache: %d blobs saved to %s", len(b.cache), *file)
	b.mu.Unlock()
}

// take returns the counts so far and starts new ones.
func (b *bot) take() stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.st
	s.unanswered = len(b.want) + len(b.requests)
	b.st = stats{}
	return s
}

func (b *bot) read() {
	for {
		pk, err := b.conn.ReadPacket()
		if err != nil {
			log.Printf("connection closed: %v", err)
			return
		}
		switch pk := pk.(type) {
		case *packet.LevelChunk:
			b.levelChunk(pk)
		case *packet.SubChunk:
			b.subChunk(pk)
		case *packet.ClientCacheMissResponse:
			b.mu.Lock()
			for _, blob := range pk.Blobs {
				b.st.missBlobs++
				b.st.missBytes += len(blob.Payload)
				if xxhash.Sum64(blob.Payload) != blob.Hash {
					b.st.badBlobs++
				}
				b.cache[blob.Hash] = blob.Payload
				delete(b.want, blob.Hash)
			}
			b.mu.Unlock()
		case *packet.Text:
			log.Printf("text: %s", pk.Message)
		case *packet.CommandOutput:
			for _, m := range pk.OutputMessages {
				log.Printf("command: %s %v", m.Message, m.Parameters)
			}
		}
	}
}

func (b *bot) levelChunk(pk *packet.LevelChunk) {
	b.mu.Lock()
	b.st.levelChunks++
	b.st.inlineBytes += len(pk.RawPayload)
	b.mu.Unlock()
	if pk.CacheEnabled {
		b.status(pk.BlobHashes)
	}
	if os.Getenv("CACHEBOT_DEBUG") != "" {
		lim, ok := pk.SubChunkLimit.Value()
		log.Printf("LevelChunk %v count=%d limit=%d/%v cache=%v hashes=%d", pk.Position, pk.SubChunkCount, lim, ok, pk.CacheEnabled, len(pk.BlobHashes))
	}
	if _, request := pk.SubChunkLimit.Value(); !request {
		return // sub chunks sent in the LevelChunk itself
	}
	// Whole column, -64..319 at most; out of range answers are fine.
	offsets := make([]protocol.SubChunkOffset, 0, 24)
	for y := int8(-4); y < 20; y++ {
		offsets = append(offsets, protocol.SubChunkOffset{0, y, 0})
	}
	b.mu.Lock()
	b.requests = append(b.requests, &packet.SubChunkRequest{
		Dimension: pk.Dimension,
		Position:  protocol.SubChunkPos{pk.Position[0], 0, pk.Position[1]},
		Offsets:   offsets,
	})
	b.mu.Unlock()
}

func (b *bot) subChunk(pk *packet.SubChunk) {
	if os.Getenv("CACHEBOT_DEBUG") != "" {
		log.Printf("SubChunk %v entries=%d cache=%v", pk.Position, len(pk.SubChunkEntries), pk.CacheEnabled)
	}
	var hashes []uint64
	b.mu.Lock()
	for _, e := range pk.SubChunkEntries {
		b.st.subEntries++
		if raw, ok := e.RawPayload.Value(); ok {
			b.st.inlineBytes += len(raw)
		}
		if h, ok := e.BlobHash.Value(); ok && pk.CacheEnabled {
			b.st.blobEntries++
			hashes = append(hashes, h)
		}
	}
	b.mu.Unlock()
	if len(hashes) > 0 {
		b.status(hashes)
	}
}

// flush sends the pending blob status every 50 ms.
func (b *bot) flush() {
	for range time.Tick(50 * time.Millisecond) {
		b.mu.Lock()
		st := b.pending
		b.pending = packet.ClientCacheBlobStatus{}
		n := min(len(b.requests), requestsPerTick)
		reqs := b.requests[:n]
		b.requests = b.requests[n:]
		b.mu.Unlock()
		for _, r := range reqs {
			if err := b.conn.WritePacket(r); err != nil {
				return
			}
		}
		if len(st.HitHashes)+len(st.MissHashes) == 0 {
			continue
		}
		if err := b.conn.WritePacket(&st); err != nil {
			return
		}
	}
}

// status queues which hashes the bot has and which it lacks.
func (b *bot) status(hashes []uint64) {
	st := &b.pending
	b.mu.Lock()
	for _, h := range hashes {
		if _, ok := b.cache[h]; ok {
			st.HitHashes = append(st.HitHashes, h)
			b.st.hits++
		} else {
			st.MissHashes = append(st.MissHashes, h)
			b.st.misses++
			b.want[h] = struct{}{}
		}
	}
	b.mu.Unlock()
}

func load(file string) map[uint64][]byte {
	m := map[uint64][]byte{}
	f, err := os.Open(file)
	if errors.Is(err, os.ErrNotExist) {
		return m
	} else if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := gob.NewDecoder(f).Decode(&m); err != nil {
		log.Fatalf("read %s: %v", file, err)
	}
	return m
}

func save(file string, m map[uint64][]byte) {
	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(m); err != nil {
		log.Fatal(err)
	}
}
