package skygear

import (
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/sghook"
)

// debugChunkCache tests the client blob cache (/sgdebug chunkcache):
//
//	stats   whether the client has the cache on, and the counters
//	reset   sets the counters to zero
//	resend  sends every loaded chunk again (all hits expected)
//	log     switches INFO logs of each blob status and miss response
type debugChunkCache struct {
	adminBase
	Sub    cmd.SubCommand `cmd:"chunkcache"`
	Action chunkCacheEnum `cmd:"action"`
}

type chunkCacheEnum string

func (chunkCacheEnum) Type() string { return "sg_chunkcache" }
func (chunkCacheEnum) Options(cmd.Source) []string {
	return []string{"stats", "reset", "resend", "log"}
}

func (c debugChunkCache) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p := src.(*player.Player)
	switch c.Action {
	case "reset":
		sghook.ResetChunkCache(p)
		o.Printf("chunk cache counters reset")
	case "resend":
		o.Printf("resent %d chunks", sghook.ResendChunks(p, tx))
	case "log":
		on := !sghook.ChunkCacheLog.Load()
		sghook.ChunkCacheLog.Store(on)
		o.Printf("chunk cache log: %v", on)
	default:
		enabled, st, ok := sghook.ChunkCache(p)
		if !ok {
			o.Errorf("no hooked connection")
			return
		}
		ratio := 0.0
		if n := st.Hits + st.Misses; n > 0 {
			ratio = float64(st.Hits) / float64(n) * 100
		}
		o.Printf("client cache: %v", enabled)
		o.Printf("LevelChunk %d (cache %d), sub chunk entries %d (blob %d), inline %d B",
			st.LevelChunks, st.CachedLevelChunks, st.SubChunkEntries, st.BlobEntries, st.RawBytes)
		o.Printf("hits %d, misses %d (%.1f%% hit), miss response %d blobs %d B",
			st.Hits, st.Misses, ratio, st.MissBlobs, st.MissBytes)
	}
}
