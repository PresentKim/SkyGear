package sghook

// Client blob cache (chunk cache) test tools.
//
// A client that sent ClientCacheStatus{Enabled: true} at login keeps the
// sub chunk and biome blobs it is sent, keyed by their xxhash64, in a cache
// that outlives the connection. Dragonfly then sends only the hashes
// (LevelChunk.BlobHashes for biomes, SubChunkEntry.BlobHash for blocks);
// the client answers with ClientCacheBlobStatus (hits it already has,
// misses it lacks) and the server sends the misses in
// ClientCacheMissResponse. These counters show how much of that happens.

import (
	"log/slog"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ChunkCacheLog logs every ClientCacheBlobStatus and ClientCacheMissResponse
// at INFO level. It may be switched at any time.
var ChunkCacheLog atomic.Bool

// ChunkCacheStats are the blob cache counters of one connection.
type ChunkCacheStats struct {
	// LevelChunks is the number of LevelChunk packets sent, CachedLevelChunks
	// those with CacheEnabled (biomes sent as a blob hash).
	LevelChunks, CachedLevelChunks uint64
	// SubChunkEntries is the number of sub chunk entries sent, BlobEntries
	// those sent as a blob hash instead of the block data. RawBytes is the
	// payload sent inline (block data without the cache, block entities).
	SubChunkEntries, BlobEntries, RawBytes uint64
	// Hits and Misses are the hashes the client reported in
	// ClientCacheBlobStatus.
	Hits, Misses uint64
	// MissBlobs and MissBytes are the blobs sent in ClientCacheMissResponse.
	MissBlobs, MissBytes uint64
}

type chunkCacheStats struct {
	levelChunks, cachedLevelChunks         atomic.Uint64
	subChunkEntries, blobEntries, rawBytes atomic.Uint64
	hits, misses                           atomic.Uint64
	missBlobs, missBytes                   atomic.Uint64
}

func (s *chunkCacheStats) load() ChunkCacheStats {
	return ChunkCacheStats{
		LevelChunks: s.levelChunks.Load(), CachedLevelChunks: s.cachedLevelChunks.Load(),
		SubChunkEntries: s.subChunkEntries.Load(), BlobEntries: s.blobEntries.Load(), RawBytes: s.rawBytes.Load(),
		Hits: s.hits.Load(), Misses: s.misses.Load(),
		MissBlobs: s.missBlobs.Load(), MissBytes: s.missBytes.Load(),
	}
}

func (s *chunkCacheStats) reset() {
	for _, v := range []*atomic.Uint64{&s.levelChunks, &s.cachedLevelChunks, &s.subChunkEntries, &s.blobEntries,
		&s.rawBytes, &s.hits, &s.misses, &s.missBlobs, &s.missBytes} {
		v.Store(0)
	}
}

func init() {
	OnClientPacket(func(c *Conn, pk packet.Packet) bool {
		if pk, ok := pk.(*packet.ClientCacheBlobStatus); ok {
			c.cache.hits.Add(uint64(len(pk.HitHashes)))
			c.cache.misses.Add(uint64(len(pk.MissHashes)))
			if ChunkCacheLog.Load() {
				slog.Info("chunk cache: blob status", "player", c.IdentityData().DisplayName, "hits", len(pk.HitHashes), "misses", len(pk.MissHashes))
			}
		}
		return false
	})
	OnServerPacket(func(c *Conn, pk packet.Packet) bool {
		switch pk := pk.(type) {
		case *packet.LevelChunk:
			c.cache.levelChunks.Add(1)
			if pk.CacheEnabled {
				c.cache.cachedLevelChunks.Add(1)
			}
			c.cache.rawBytes.Add(uint64(len(pk.RawPayload)))
		case *packet.SubChunk:
			c.cache.subChunkEntries.Add(uint64(len(pk.SubChunkEntries)))
			for _, e := range pk.SubChunkEntries {
				if _, ok := e.BlobHash.Value(); ok {
					c.cache.blobEntries.Add(1)
				}
				if raw, ok := e.RawPayload.Value(); ok {
					c.cache.rawBytes.Add(uint64(len(raw)))
				}
			}
		case *packet.ClientCacheMissResponse:
			var n int
			for _, b := range pk.Blobs {
				n += len(b.Payload)
			}
			c.cache.missBlobs.Add(uint64(len(pk.Blobs)))
			c.cache.missBytes.Add(uint64(n))
			if ChunkCacheLog.Load() {
				slog.Info("chunk cache: miss response", "player", c.IdentityData().DisplayName, "blobs", len(pk.Blobs), "bytes", n)
			}
		}
		return false
	})
}

// ChunkCache returns whether p's client has the blob cache enabled and the
// counters of its connection (ok is false for players without one).
func ChunkCache(p *player.Player) (enabled bool, stats ChunkCacheStats, ok bool) {
	c := Of(p)
	if c == nil {
		return false, ChunkCacheStats{}, false
	}
	return c.ClientCacheEnabled(), c.cache.load(), true
}

// ResetChunkCache sets the counters of p's connection to zero.
func ResetChunkCache(p *player.Player) {
	if c := Of(p); c != nil {
		c.cache.reset()
	}
}

// ResendChunks sends p every chunk it has loaded again, as if it had just
// come into view: with the blob cache the client gets the hashes again and
// should answer them all as hits. It returns the number of chunks sent.
func ResendChunks(p *player.Player, tx *world.Tx) int {
	s := Session(p)
	if s == session.Nop {
		return 0
	}
	loader := *field[*world.Loader](s, "chunkLoader")
	r := int32(*field[int32](s, "chunkRadius"))
	pos := p.Position()
	centre := world.ChunkPos{int32(pos[0]) >> 4, int32(pos[2]) >> 4}
	dim := tx.World().Dimension()
	n := 0
	for x := centre[0] - r; x <= centre[0]+r; x++ {
		for z := centre[1] - r; z <= centre[1]+r; z++ {
			cp := world.ChunkPos{x, z}
			col, ok := loader.Chunk(cp)
			if !ok {
				continue
			}
			s.ViewChunk(cp, dim, col.BlockEntities, col.Chunk)
			n++
		}
	}
	return n
}
