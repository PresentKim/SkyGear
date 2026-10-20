package playerstate

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestCaptureMarshalApply(t *testing.T) {
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	defer w.Close()
	w.Do(func(tx *world.Tx) {
		h := world.EntitySpawnOpts{Position: mgl64.Vec3{1, 70, 2}}.New(player.Type, player.Config{Name: "a", Position: mgl64.Vec3{1, 70, 2}})
		p := tx.AddEntity(h).(*player.Player)
		p.SetGameMode(world.GameModeSurvival)
		_ = p.Inventory().SetItem(5, item.NewStack(item.Diamond{}, 7).WithValue("k", "v"))
		_ = p.EnderChestInventory().SetItem(0, item.NewStack(item.Apple{}, 3))
		p.Armour().SetHelmet(item.NewStack(item.Helmet{Tier: item.ArmourTierIron{}}, 1))
		p.SetHeldItems(item.Stack{}, item.NewStack(item.Arrow{}, 2))
		p.SetFood(11)
		p.SetExperienceLevel(4)
		p.AddEffect(effect.New(effect.Speed, 2, time.Minute))
		p.SetMaxHealth(20)
		p.SetMaxHealth(20)
		setHealth(p, 9)

		b, err := Marshal(Capture(p))
		if err != nil {
			t.Fatal(err)
		}
		s, err := Unmarshal(b)
		if err != nil {
			t.Fatal(err)
		}
		if s.Position() != (mgl64.Vec3{1, 70, 2}) {
			t.Errorf("position = %v", s.Position())
		}

		Reset(p)
		if !p.Inventory().Empty() || p.Health() != p.MaxHealth() || p.Food() != 20 {
			t.Fatalf("reset failed: inv empty=%v health=%v food=%v", p.Inventory().Empty(), p.Health(), p.Food())
		}

		Apply(p, s)
		if it, _ := p.Inventory().Item(5); it.Count() != 7 {
			t.Errorf("slot 5 = %v", it)
		} else if v, _ := it.Value("k"); v != "v" {
			t.Errorf("custom value lost: %v", v)
		}
		if it, _ := p.EnderChestInventory().Item(0); it.Count() != 3 {
			t.Errorf("ender chest = %v", it)
		}
		if p.Armour().Helmet().Empty() {
			t.Error("helmet lost")
		}
		if _, off := p.HeldItems(); off.Count() != 2 {
			t.Errorf("offhand = %v", off)
		}
		if p.Health() != 9 || p.Food() != 11 || p.ExperienceLevel() != 4 {
			t.Errorf("health %v food %v level %v", p.Health(), p.Food(), p.ExperienceLevel())
		}
		if p.MaxHealth() != 20 {
			t.Errorf("max health changed to %v", p.MaxHealth())
		}
		if len(p.Effects()) != 1 {
			t.Errorf("effects = %v", p.Effects())
		}
	})
}
