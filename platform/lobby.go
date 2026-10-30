package platform

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// lobbyHandler protects the lobby: no building, damage, dropping or PvP.
type lobbyHandler struct {
	player.NopHandler
}

func (lobbyHandler) HandleStartBreak(ctx *player.Context, _ cube.Pos) { ctx.Cancel() }
func (lobbyHandler) HandleBlockBreak(ctx *player.Context, _ cube.Pos, _ *[]item.Stack, _ *int) {
	ctx.Cancel()
}
func (lobbyHandler) HandleBlockPlace(ctx *player.Context, _ cube.Pos, _ world.Block) { ctx.Cancel() }
func (lobbyHandler) HandleItemDrop(ctx *player.Context, _ item.Stack)                { ctx.Cancel() }
func (lobbyHandler) HandleHurt(ctx *player.Context, _ *float64, _ bool, _ *time.Duration, src world.DamageSource) {
	// Void damage is left to the platform handler, which returns the player.
	ctx.Cancel()
}
func (lobbyHandler) HandleFoodLoss(ctx *player.Context, _ int, _ *int) { ctx.Cancel() }
func (lobbyHandler) HandleAttackEntity(ctx *player.Context, _ world.Entity, _, _ *float64, _ *bool) {
	ctx.Cancel()
}
func (lobbyHandler) HandleItemUse(ctx *player.Context) { ctx.Cancel() }
func (lobbyHandler) HandleItemUseOnBlock(ctx *player.Context, _ cube.Pos, _ cube.Face, _ mgl64.Vec3) {
	ctx.Cancel()
}

// HandleMove keeps the body in place: the lobby is seen through a fixed
// camera (lobbyCamera), so it is a menu, not a place to walk. Jumping (up and
// down on the spot) is left.
func (lobbyHandler) HandleMove(ctx *player.Context, pos mgl64.Vec3, _ cube.Rotation) {
	if old := ctx.Player().Position(); pos.X() != old.X() || pos.Z() != old.Z() {
		ctx.Cancel()
	}
}

// The lobby looks like a game's lobby: a camera on a small sky island with
// the player's character, under the main screen (ui.HomeScreen).
var (
	// lobbyCamera is where the camera looks from: south of the island,
	// slightly above, towards north (yaw 180) and a little down.
	lobbyCamera              = LobbySpawn.Vec3Middle().Add(mgl64.Vec3{0, 3.5, 12})
	lobbyPitch, lobbyYaw     = 14.0, 180.0
	lobbyIslandR             = 4
	lobbyTree                = LobbySpawn.Add(cube.Pos{-2, 0, -2})
	lobbyLantern, lobbyTorch = LobbySpawn.Add(cube.Pos{3, 0, -1}), LobbySpawn.Add(cube.Pos{-3, 0, 2})
)

// lobbyLocks are the client inputs locked in the lobby: all but jumping and
// sneaking.
const lobbyLocks = packet.ClientInputLockCamera | packet.ClientInputLockMovement |
	packet.ClientInputLockLateralMovement | packet.ClientInputLockMount | packet.ClientInputLockDismount |
	packet.ClientInputLockMoveForward | packet.ClientInputLockMoveBackward |
	packet.ClientInputLockMoveLeft | packet.ClientInputLockMoveRight

// showLobbyScene sets p's lobby view: the camera on the lobby island, the
// vanilla HUD hidden, the inputs locked but jumping and sneaking, and the
// other players hidden both ways. It runs in the lobby transaction.
func showLobbyScene(p *player.Player, tx *world.Tx) {
	sghook.SetCamera(p, lobbyCamera, lobbyPitch, lobbyYaw, 0)
	sghook.HideHUD(p, true)
	sghook.LockInputs(p, lobbyLocks)
	for e := range tx.Players() {
		if o := e.(*player.Player); o.H() != p.H() {
			p.HideEntity(o)
			o.HideEntity(p)
		}
	}
}

// leaveLobbyScene undoes showLobbyScene for p going into a save: the save
// shows its players as its chunks arrive.
func leaveLobbyScene(p *player.Player) {
	stopLobbyMotion(p)
	sghook.ClearCamera(p)
	sghook.HideHUD(p, false)
	sghook.LockInputs(p, 0)
	sghook.ForgetHiddenEntities(p)
}

// lobbyClear is how far around the spawn the lobby is cleared before the
// island is built, as the lobby world is saved and may hold an older layout.
const lobbyClear = 12

// buildLobbyIsland builds the lobby's sky island: a round grass top that
// narrows into dirt and stone below, an oak tree, flowers and two lights.
func buildLobbyIsland(tx *world.Tx) {
	for x := -lobbyClear; x <= lobbyClear; x++ {
		for z := -lobbyClear; z <= lobbyClear; z++ {
			for y := -lobbyClear; y <= lobbyClear; y++ {
				pos := LobbySpawn.Add(cube.Pos{x, y, z})
				if _, air := tx.Block(pos).(block.Air); !air {
					tx.SetBlock(pos, nil, nil)
				}
			}
		}
	}
	top := LobbySpawn.Add(cube.Pos{0, -1, 0})
	for depth := 0; depth <= lobbyIslandR; depth++ {
		r := lobbyIslandR - depth
		for x := -r; x <= r; x++ {
			for z := -r; z <= r; z++ {
				if x*x+z*z > r*r+r {
					continue
				}
				var b world.Block = block.Stone{}
				switch {
				case depth == 0:
					b = block.Grass{}
				case depth <= 2:
					b = block.Dirt{}
				}
				tx.SetBlock(top.Add(cube.Pos{x, -depth, z}), b, nil)
			}
		}
	}
	for y := 0; y < 4; y++ {
		tx.SetBlock(lobbyTree.Add(cube.Pos{0, y, 0}), block.Log{Wood: block.OakWood(), Axis: cube.Y}, nil)
	}
	leaves := block.Leaves{Type: block.OakLeaves(), Persistent: true}
	for x := -2; x <= 2; x++ {
		for z := -2; z <= 2; z++ {
			for y := 2; y <= 4; y++ {
				if (x == 0 && z == 0 && y < 4) || x*x+z*z+(y-3)*(y-3) > 5 {
					continue
				}
				tx.SetBlock(lobbyTree.Add(cube.Pos{x, y, z}), leaves, nil)
			}
		}
	}
	for _, f := range []struct {
		pos cube.Pos
		b   world.Block
	}{
		{cube.Pos{2, 0, 2}, block.Flower{Type: block.Poppy()}},
		{cube.Pos{-1, 0, 3}, block.Flower{Type: block.Dandelion()}},
		{cube.Pos{3, 0, 1}, block.Flower{Type: block.Cornflower()}},
		{cube.Pos{-3, 0, -1}, block.ShortGrass{}},
		{cube.Pos{1, 0, -3}, block.ShortGrass{}},
	} {
		tx.SetBlock(LobbySpawn.Add(f.pos), f.b, nil)
	}
	tx.SetBlock(lobbyLantern, block.WoodFence{Wood: block.OakWood()}, nil)
	tx.SetBlock(lobbyLantern.Add(cube.Pos{0, 1, 0}), block.Lantern{Type: block.NormalFire()}, nil)
	tx.SetBlock(lobbyTorch, block.Torch{Facing: cube.FaceDown, Type: block.NormalFire()}, nil)
}
