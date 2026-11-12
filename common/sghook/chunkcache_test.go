package sghook

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestChunkCacheStats(t *testing.T) {
	c := &Conn{}
	run := func(hooks []func(*Conn, packet.Packet) bool, pk packet.Packet) {
		for _, h := range hooks {
			h(c, pk)
		}
	}
	run(serverHooks, &packet.LevelChunk{CacheEnabled: true, BlobHashes: []uint64{1}, RawPayload: []byte{0}})
	run(serverHooks, &packet.LevelChunk{RawPayload: []byte{1, 2, 3}})
	run(serverHooks, &packet.SubChunk{CacheEnabled: true, SubChunkEntries: []protocol.SubChunkEntry{
		{BlobHash: protocol.Option[uint64](2), RawPayload: protocol.Option([]byte{})},
		{Result: protocol.SubChunkResultSuccessAllAir},
		{RawPayload: protocol.Option([]byte{1, 2})},
	}})
	run(clientHooks, &packet.ClientCacheBlobStatus{HitHashes: []uint64{1, 2}, MissHashes: []uint64{3}})
	run(serverHooks, &packet.ClientCacheMissResponse{Blobs: []protocol.CacheBlob{{Hash: 3, Payload: make([]byte, 10)}}})

	want := ChunkCacheStats{LevelChunks: 2, CachedLevelChunks: 1, SubChunkEntries: 3, BlobEntries: 1, RawBytes: 6,
		Hits: 2, Misses: 1, MissBlobs: 1, MissBytes: 10}
	if got := c.cache.load(); got != want {
		t.Fatalf("stats = %+v, want %+v", got, want)
	}
	c.cache.reset()
	if got := c.cache.load(); got != (ChunkCacheStats{}) {
		t.Fatalf("after reset = %+v", got)
	}
}
