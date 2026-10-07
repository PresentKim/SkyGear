package game

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/oneblock"
)

// menuAction is the command run by each main menu button.
var menuActions = []struct{ label, command string }{
	{"섬으로 이동", "/is home"},
	{"섬 정보", "/is info"},
	{"농사 상점", "/shop"},
	{"작물 전체 판매", "/sell all"},
	{"농사 정보", "/farm"},
	{"섬 순위", "/is top"},
	{"로비로 이동", "/spawn"},
}

type mainMenuForm struct{}

func mainMenu() form.Menu {
	m := form.NewMenu(mainMenuForm{}, "§l원블럭 팜").WithBody("원하는 기능을 선택하세요.")
	for _, a := range menuActions {
		m = m.AddButton(form.NewButton(a.label, ""))
	}
	return m
}

func (mainMenuForm) Submit(submitter form.Submitter, pressed form.Button, _ *world.Tx) {
	p := submitter.(*player.Player)
	for _, a := range menuActions {
		if a.label == pressed.Text {
			p.ExecuteCommand(a.command)
			return
		}
	}
}

// ShopEntry is an item that can be bought in the farming shop.
type ShopEntry struct {
	Name  string
	Item  item.Stack
	Price int64
	// Phase is the OneBlock phase index required to buy the entry.
	Phase int
}

var waterBucket = item.Bucket{Content: item.LiquidBucketContent(block.Water{Depth: 8, Still: true})}

// Shop lists all items sold in the shop.
var Shop = []ShopEntry{
	{"흙 x16", item.NewStack(block.Dirt{}, 16), 40, 0},
	{"밀 씨앗 x8", item.NewStack(block.WheatSeeds{}, 8), 10, 0},
	{"뼛가루 x16", item.NewStack(item.BoneMeal{}, 16), 40, 0},
	{"참나무 묘목 x2", item.NewStack(block.Sapling{Type: block.OakSapling()}, 2), 20, 0},
	{"당근 x4", item.NewStack(block.Carrot{}, 4), 20, 1},
	{"감자 x4", item.NewStack(block.Potato{}, 4), 20, 1},
	{"비트 씨앗 x8", item.NewStack(block.BeetrootSeeds{}, 8), 15, 1},
	{"호박 씨앗 x2", item.NewStack(block.PumpkinSeeds{}, 2), 30, 1},
	{"수박 씨앗 x2", item.NewStack(block.MelonSeeds{}, 2), 30, 1},
	{"사탕수수 x2", item.NewStack(block.SugarCane{}, 2), 25, 1},
	{"물 양동이", item.NewStack(waterBucket, 1), 100, 1},
	{"철 괭이", item.NewStack(item.Hoe{Tier: item.ToolTierIron}, 1), 150, 1},
	{"코코아 콩 x2", item.NewStack(block.CocoaBean{}, 2), 40, 2},
	{"영혼 모래 x4", item.NewStack(block.SoulSand{}, 4), 80, 5},
	{"네더 사마귀 x2", item.NewStack(block.NetherWart{}, 2), 120, 5},
	{"다이아몬드 괭이", item.NewStack(item.Hoe{Tier: item.ToolTierDiamond}, 1), 1500, 6},
}

type shopForm struct{ phase int }

func shopMenu(phase int, money int64) form.Menu {
	m := form.NewMenu(shopForm{phase: phase}, "§l농사 상점").
		WithBody(fmt.Sprintf("소지금: §6%dG\n§f섬 단계가 오를수록 더 많은 상품이 열립니다.", money))
	for _, e := range Shop {
		m = m.AddButton(form.NewButton(shopLabel(e, phase), ""))
	}
	return m
}

func shopLabel(e ShopEntry, phase int) string {
	if e.Phase > phase {
		return fmt.Sprintf("§8%s\n§8[%s 단계 필요]", e.Name, oneblock.Phases[e.Phase].Name)
	}
	return fmt.Sprintf("%s\n§6%dG", e.Name, e.Price)
}

func (f shopForm) Submit(submitter form.Submitter, pressed form.Button, _ *world.Tx) {
	p := submitter.(*player.Player)
	for _, e := range Shop {
		if shopLabel(e, f.phase) != pressed.Text {
			continue
		}
		switch {
		case e.Phase > f.phase:
			p.Message("§c아직 구매할 수 없는 상품입니다.")
		case !g.takeMoney(p, e.Price):
			p.Message("§c돈이 부족합니다.")
		default:
			if n, _ := p.Inventory().AddItem(e.Item); n < e.Item.Count() {
				spawnItems(p.Tx(), p.Position(), []item.Stack{e.Item.Grow(-n)})
			}
			p.Messagef("§a%s 을(를) 구매했습니다.", e.Name)
		}
		return
	}
}
