package game

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/title"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/farm"
	"github.com/presentkim/dragonfactory/internal/oneblock"
	"github.com/presentkim/dragonfactory/internal/store"
)

// Handler handles the events of a single player.
type Handler struct {
	player.NopHandler
	g *Game
}

func (h *Handler) deny(ctx *player.Context, pos cube.Pos) bool {
	if h.g.CanBuild(ctx.Player(), pos) {
		return false
	}
	ctx.Cancel()
	ctx.Player().SendTip("§c이곳은 당신의 섬이 아닙니다.")
	return true
}

// HandleStartBreak ...
func (h *Handler) HandleStartBreak(ctx *player.Context, pos cube.Pos) {
	h.deny(ctx, pos)
}

// HandleBlockPlace ...
func (h *Handler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, _ world.Block) {
	h.deny(ctx, pos)
}

// HandleItemUseOnBlock ...
func (h *Handler) HandleItemUseOnBlock(ctx *player.Context, pos cube.Pos, face cube.Face, _ mgl64.Vec3) {
	if !h.deny(ctx, pos) {
		h.deny(ctx, pos.Side(face))
	}
}

// HandleBlockBreak ...
func (h *Handler) HandleBlockBreak(ctx *player.Context, pos cube.Pos, drops *[]item.Stack, _ *int) {
	if h.deny(ctx, pos) {
		return
	}
	p := ctx.Player()
	idx, _ := h.g.grid.IndexAt(pos)
	if pos == h.g.grid.Centre(idx) {
		h.breakOneBlock(p, idx, pos)
		return
	}
	if hv, ok := farm.HarvestOf(p.Tx().Block(pos)); ok {
		h.harvest(p, pos, hv, drops)
	}
}

// breakOneBlock advances the island's OneBlock and regenerates it.
func (h *Handler) breakOneBlock(p *player.Player, idx int, pos cube.Pos) {
	var broken int
	h.g.store.Do(func(tx store.Tx) {
		if isl, ok := tx.Island(idx); ok {
			isl.Broken++
			broken = isl.Broken
		}
	})
	tx := p.Tx()
	tx.Defer(func(tx *world.Tx) {
		tx.SetBlock(pos, oneblock.Next(broken), nil)
	})

	prev, _ := oneblock.PhaseAt(broken - 1)
	cur, ph := oneblock.PhaseAt(broken)
	gifts := ph.Rolls()
	if cur != prev {
		p.SendTitle(title.New("§a새로운 단계!").WithSubtitle("§e" + ph.Name))
		p.PlaySound(sound.LevelUp{})
		h.g.broadcastIsland(idx, fmt.Sprintf("§a섬이 §e%s §a단계에 도달했습니다!", ph.Name))
		gifts = append(gifts, ph.Reward...)
	}
	spawnItems(tx, pos.Vec3Centre().Add(mgl64.Vec3{0, 1, 0}), gifts)

	if cur+1 < len(oneblock.Phases) {
		p.SendTip(fmt.Sprintf("§e%s §7| 다음 단계까지 §b%d§7블록", ph.Name, oneblock.Phases[cur+1].Start-broken))
	} else {
		p.SendTip(fmt.Sprintf("§e%s §7| 총 §b%d§7블록", ph.Name, broken))
	}
	h.g.UpdateBoard(p)
}

// harvest grants farming XP, bonus crops and replants if enabled.
func (h *Handler) harvest(p *player.Player, pos cube.Pos, hv farm.Harvest, drops *[]item.Stack) {
	var before, after int
	var replant bool
	h.g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		before = farm.Level(prof.FarmXP)
		prof.FarmXP += hv.XP
		after = farm.Level(prof.FarmXP)
		if prof.Harvested == nil {
			prof.Harvested = map[string]int{}
		}
		prof.Harvested[hv.Name]++
		replant = prof.AutoReplant && before >= farm.ReplantLevel
	})
	if rand.Float64() < farm.BonusChance(before) {
		*drops = append(*drops, item.NewStack(hv.Crop, 1))
	}
	if replant && takeOne(drops, hv.Seed) {
		p.Tx().Defer(func(tx *world.Tx) {
			tx.SetBlock(pos, hv.Replant, nil)
		})
	}
	if after > before {
		p.SendTitle(title.New("§a농사 레벨 업!").WithSubtitle(fmt.Sprintf("§eLv.%d", after)))
		p.PlaySound(sound.LevelUp{})
		if after == farm.ReplantLevel {
			p.Message("§a자동 재파종이 해금되었습니다! §7(/farm replant 로 켜고 끌 수 있습니다)")
		}
	}
	h.g.UpdateBoard(p)
}

// takeOne removes a single item of the same type as it from drops.
func takeOne(drops *[]item.Stack, it world.Item) bool {
	name, meta := it.EncodeItem()
	for i, s := range *drops {
		n, m := s.Item().EncodeItem()
		if n != name || m != meta {
			continue
		}
		if s.Count() == 1 {
			*drops = append((*drops)[:i], (*drops)[i+1:]...)
		} else {
			(*drops)[i] = s.Grow(-1)
		}
		return true
	}
	return false
}

func spawnItems(tx *world.Tx, pos mgl64.Vec3, items []item.Stack) {
	for _, it := range items {
		tx.AddEntity(entity.NewItemPickupDelay(world.EntitySpawnOpts{Position: pos}, it, time.Second/2))
	}
}

// HandleMove teleports players falling into the void back home.
func (h *Handler) HandleMove(ctx *player.Context, newPos mgl64.Vec3, _ cube.Rotation) {
	if newPos.Y() < float64(h.g.conf.Island.Height-64) {
		ctx.Cancel()
		h.g.sendHome(ctx.Player())
	}
}

// HandleHurt cancels void damage and all damage in the lobby.
func (h *Handler) HandleHurt(ctx *player.Context, _ *float64, _ bool, _ *time.Duration, src world.DamageSource) {
	if _, ok := src.(entity.VoidDamageSource); ok {
		ctx.Cancel()
		h.g.sendHome(ctx.Player())
		return
	}
	if _, ok := h.g.grid.IndexAt(cube.PosFromVec3(ctx.Player().Position())); !ok {
		ctx.Cancel()
	}
}

// HandleAttackEntity disables PvP.
func (h *Handler) HandleAttackEntity(ctx *player.Context, e world.Entity, _, _ *float64, _ *bool) {
	if _, ok := e.(*player.Player); ok {
		ctx.Cancel()
	}
}

// HandleRespawn respawns players on their island.
func (h *Handler) HandleRespawn(p *player.Player, pos *mgl64.Vec3, w **world.World) {
	if idx, ok := h.g.IslandOf(p.UUID()); ok {
		*pos, *w = h.g.grid.Spawn(idx), h.g.w
	}
}

// HandleQuit ...
func (h *Handler) HandleQuit(p *player.Player) {
	h.g.mu.Lock()
	delete(h.g.invites, p.UUID())
	h.g.mu.Unlock()
}

// sendHome teleports p to its island or the lobby.
func (g *Game) sendHome(p *player.Player) {
	if idx, ok := g.IslandOf(p.UUID()); ok {
		g.Home(p, idx)
		return
	}
	p.Teleport(LobbySpawn.Vec3Middle())
}

// broadcastIsland messages all online members of island idx in the current world.
func (g *Game) broadcastIsland(idx int, msg string) {
	var members []uuid.UUID
	g.store.Do(func(tx store.Tx) {
		if isl, ok := tx.Island(idx); ok {
			members = append([]uuid.UUID{isl.Owner}, isl.Members...)
		}
	})
	g.messageAll(members, msg)
}

// messageAll sends msg to every online player in ids.
func (g *Game) messageAll(ids []uuid.UUID, msg string) {
	for _, id := range ids {
		if h, ok := g.srv.Player(id); ok {
			h.Do(func(_ *world.Tx, e world.Entity) {
				e.(*player.Player).Message(msg)
			})
		}
	}
}
