package sghook

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func newWatch() *chunkWatch {
	return &chunkWatch{chunks: map[world.ChunkPos]*chunkState{}, queue: map[uint64][]pendingHash{},
		seen: map[uint64]map[world.ChunkPos]struct{}{}, stop: make(chan struct{})}
}

func reports(w *chunkWatch) map[world.ChunkPos]ChunkReport {
	out := map[world.ChunkPos]ChunkReport{}
	for _, r := range w.due(time.Now().Add(time.Minute)) {
		out[r.Pos] = r
	}
	return out
}

func TestChunkWatchClasses(t *testing.T) {
	w := newWatch()
	// Chunk (0,0): biome 1, one sub chunk 10, one all air; client lacks both.
	w.serverPacket(&packet.LevelChunk{Position: protocol.ChunkPos{0, 0}, CacheEnabled: true, BlobHashes: []uint64{1}, RawPayload: []byte{0}})
	w.serverPacket(&packet.SubChunk{CacheEnabled: true, Position: protocol.SubChunkPos{0, 4, 0}, SubChunkEntries: []protocol.SubChunkEntry{
		{Result: protocol.SubChunkResultSuccess, BlobHash: protocol.Option[uint64](10), RawPayload: protocol.Option([]byte{1, 2})},
		{Result: protocol.SubChunkResultSuccessAllAir, Offset: protocol.SubChunkOffset{0, 1, 0}},
	}})
	w.blobStatus(&packet.ClientCacheBlobStatus{MissHashes: []uint64{1, 10}})
	// Chunk (1,0) has the same blobs (another position); (0,0) is sent again;
	// (2,0) has a hash this connection never sent.
	w.serverPacket(&packet.LevelChunk{Position: protocol.ChunkPos{1, 0}, CacheEnabled: true, BlobHashes: []uint64{1}})
	w.serverPacket(&packet.SubChunk{CacheEnabled: true, Position: protocol.SubChunkPos{1, 4, 0}, SubChunkEntries: []protocol.SubChunkEntry{
		{Result: protocol.SubChunkResultSuccess, BlobHash: protocol.Option[uint64](10)},
	}})
	w.blobStatus(&packet.ClientCacheBlobStatus{HitHashes: []uint64{1, 10}})
	first := reports(w)
	if r := first[world.ChunkPos{0, 0}]; r.Blobs != 2 || r.Fresh != 2 || r.Cached != 0 || r.AllAir != 1 || r.InlineBytes != 3 {
		t.Fatalf("(0,0) = %+v", r)
	}
	if r := first[world.ChunkPos{1, 0}]; r.Blobs != 2 || r.Cached != 2 || r.CachedElsewhere != 2 {
		t.Fatalf("(1,0) = %+v", r)
	}

	w.serverPacket(&packet.LevelChunk{Position: protocol.ChunkPos{0, 0}, CacheEnabled: true, BlobHashes: []uint64{1}})
	w.serverPacket(&packet.LevelChunk{Position: protocol.ChunkPos{2, 0}, CacheEnabled: true, BlobHashes: []uint64{99, 98}})
	w.blobStatus(&packet.ClientCacheBlobStatus{HitHashes: []uint64{1, 99}})
	second := reports(w)
	if r := second[world.ChunkPos{0, 0}]; r.Cached != 1 || r.CachedHere != 1 {
		t.Fatalf("(0,0) again = %+v", r)
	}
	if r := second[world.ChunkPos{2, 0}]; r.Cached != 1 || r.CachedBefore != 1 || r.Unanswered != 1 {
		t.Fatalf("(2,0) = %+v", r)
	}
	if len(w.queue) != 0 {
		t.Fatalf("queue left: %v", w.queue)
	}
}

func TestChunkWatchWaitsForAnswers(t *testing.T) {
	w := newWatch()
	w.serverPacket(&packet.LevelChunk{Position: protocol.ChunkPos{0, 0}, CacheEnabled: true, BlobHashes: []uint64{1}})
	if got := w.due(time.Now().Add(chunkReportQuiet * 2)); len(got) != 0 {
		t.Fatalf("reported with an open hash: %+v", got)
	}
	w.blobStatus(&packet.ClientCacheBlobStatus{MissHashes: []uint64{1}})
	if got := w.due(time.Now()); len(got) != 0 {
		t.Fatalf("reported before quiet: %+v", got)
	}
	if got := w.due(time.Now().Add(chunkReportQuiet * 2)); len(got) != 1 || got[0].Fresh != 1 {
		t.Fatalf("got %+v", got)
	}
}
