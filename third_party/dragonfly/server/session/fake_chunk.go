package session

// This file is part of the dragonfactory patch to Dragonfly (third_party/README.md):
// client-only chunk views.

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ViewFakeChunk shows c in place of the world's chunk at pos to this player
// only, until ClearFakeChunk. Meanwhile block updates of the world in that
// chunk are not sent (ViewFakeBlock still is), and sub-chunk requests and
// chunk resends are answered from c.
func (s *Session) ViewFakeChunk(pos world.ChunkPos, dim world.Dimension, c *chunk.Chunk) {
	if s == Nop {
		return
	}
	s.fakeChunkMu.Lock()
	if s.fakeChunks == nil {
		s.fakeChunks = map[world.ChunkPos]*world.Column{}
	}
	s.fakeChunks[pos] = &world.Column{Chunk: c}
	s.fakeChunkMu.Unlock()
	s.ViewChunk(pos, dim, nil, c)
}

// ClearFakeChunk shows the world's chunk at pos again after ViewFakeChunk.
func (s *Session) ClearFakeChunk(pos world.ChunkPos, dim world.Dimension) {
	if s == Nop {
		return
	}
	s.fakeChunkMu.Lock()
	_, ok := s.fakeChunks[pos]
	delete(s.fakeChunks, pos)
	s.fakeChunkMu.Unlock()
	if !ok {
		return
	}
	if col, ok := s.chunkLoader.Chunk(pos); ok {
		s.ViewChunk(pos, dim, col.BlockEntities, col.Chunk)
	}
}

// fakeChunk returns the chunk shown at pos with ViewFakeChunk, if any.
func (s *Session) fakeChunk(pos world.ChunkPos) (*world.Column, bool) {
	s.fakeChunkMu.Lock()
	defer s.fakeChunkMu.Unlock()
	col, ok := s.fakeChunks[pos]
	return col, ok
}

// inFakeChunk reports if the block at pos is in a chunk shown with
// ViewFakeChunk.
func (s *Session) inFakeChunk(pos cube.Pos) bool {
	_, ok := s.fakeChunk(world.ChunkPos{int32(pos[0] >> 4), int32(pos[2] >> 4)})
	return ok
}

// ViewChunkCentre makes the client keep and draw the chunks around pos
// instead of around the player until the player next moves to another
// chunk, for chunks shown with ViewFakeChunk right after a teleport (they
// are dropped if outside the client's chunk centre).
func (s *Session) ViewChunkCentre(pos mgl64.Vec3) {
	if s == Nop {
		return
	}
	s.writePacket(&packet.NetworkChunkPublisherUpdate{
		Position: protocol.BlockPos{int32(pos[0]), int32(pos[1]), int32(pos[2])},
		Radius:   uint32(s.chunkRadius) << 4,
	})
}
