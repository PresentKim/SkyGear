package game

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/presentkim/dragonfactory/internal/farm"
	"github.com/presentkim/dragonfactory/internal/oneblock"
	"github.com/presentkim/dragonfactory/internal/store"
)

func registerCommands() {
	cmd.Register(cmd.New("is", "원블럭 섬 명령어", []string{"island"},
		isMenu{}, isCreate{}, isHome{}, isSetHome{}, isInfo{}, isInvite{}, isAccept{},
		isLeave{}, isKick{}, isVisit{}, isTop{}))
	cmd.Register(cmd.New("sell", "작물을 판매합니다", nil, sellHand{}, sellAll{}))
	cmd.Register(cmd.New("shop", "농사 상점을 엽니다", nil, shopCmd{}))
	cmd.Register(cmd.New("money", "소지금을 확인합니다", []string{"bal"}, moneyCmd{}))
	cmd.Register(cmd.New("farm", "농사 레벨 정보", nil, farmInfo{}, farmReplant{}))
	cmd.Register(cmd.New("spawn", "로비로 이동합니다", []string{"lobby"}, spawnCmd{}))
}

// playerOnly restricts a command to players.
type playerOnly struct{}

func (playerOnly) Allow(src cmd.Source) bool {
	_, ok := src.(*player.Player)
	return ok
}

type isMenu struct{ playerOnly }

func (isMenu) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	src.(*player.Player).SendForm(mainMenu())
}

type isCreate struct {
	playerOnly
	Create cmd.SubCommand `cmd:"create"`
}

func (isCreate) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	idx, created := -1, false
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		if prof.Island >= 0 {
			return
		}
		idx, created = g.createIsland(tx, p), true
	})
	if !created {
		o.Error("이미 섬이 있습니다. /is home 으로 이동하세요.")
		return
	}
	g.Home(p, idx)
	g.UpdateBoard(p)
	o.Print("§a새 원블럭 섬이 생성되었습니다! 발밑의 블록을 캐보세요.")
}

type isHome struct {
	playerOnly
	Home cmd.SubCommand `cmd:"home"`
}

func (isHome) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	idx, ok := g.IslandOf(p.UUID())
	if !ok {
		o.Error("섬이 없습니다. /is create 로 섬을 만드세요.")
		return
	}
	g.Home(p, idx)
}

type isSetHome struct {
	playerOnly
	SetHome cmd.SubCommand `cmd:"sethome"`
}

func (isSetHome) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	idx, ok := g.IslandOf(p.UUID())
	if !ok || !g.CanBuild(p, cubePos(p)) {
		o.Error("자신의 섬 위에서만 설정할 수 있습니다.")
		return
	}
	if at, _ := g.grid.IndexAt(cubePos(p)); at != idx {
		o.Error("자신의 섬 위에서만 설정할 수 있습니다.")
		return
	}
	off := p.Position().Sub(g.grid.Centre(idx).Vec3())
	g.store.Do(func(tx store.Tx) {
		if isl, ok := tx.Island(idx); ok {
			isl.Home = &[3]float64{off[0], off[1], off[2]}
		}
	})
	o.Print("§a섬 홈 위치가 설정되었습니다.")
}

type isInfo struct {
	playerOnly
	Info cmd.SubCommand `cmd:"info"`
}

func (isInfo) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	idx, ok := g.IslandOf(p.UUID())
	if !ok {
		o.Error("섬이 없습니다.")
		return
	}
	g.store.Do(func(tx store.Tx) {
		isl, ok := tx.Island(idx)
		if !ok {
			return
		}
		cur, ph := oneblock.PhaseAt(isl.Broken)
		o.Printf("§l§a[ %s 님의 섬 ]", isl.OwnerName)
		o.Printf("§f단계: §e%s §7(%d/%d)", ph.Name, cur+1, len(oneblock.Phases))
		o.Printf("§f캔 블록 수: §b%d", isl.Broken)
		if cur+1 < len(oneblock.Phases) {
			next := oneblock.Phases[cur+1]
			o.Printf("§f다음 단계: §e%s §7(%d블록 남음)", next.Name, next.Start-isl.Broken)
		}
		var names []string
		for _, m := range isl.Members {
			if prof, ok := tx.LookupProfile(m); ok {
				names = append(names, prof.Name)
			}
		}
		if len(names) > 0 {
			o.Printf("§f멤버: §7%s", strings.Join(names, ", "))
		}
	})
}

type isInvite struct {
	playerOnly
	Invite  cmd.SubCommand `cmd:"invite"`
	Targets []cmd.Target   `cmd:"player"`
}

func (c isInvite) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	idx, owner := -1, false
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		if isl, ok := tx.Island(prof.Island); ok && isl.Owner == p.UUID() {
			idx, owner = isl.Index, true
		}
	})
	if !owner {
		o.Error("섬 주인만 초대할 수 있습니다.")
		return
	}
	for _, t := range c.Targets {
		target, ok := t.(*player.Player)
		if !ok || target == p {
			continue
		}
		g.mu.Lock()
		g.invites[target.UUID()] = idx
		g.mu.Unlock()
		target.Message(fmt.Sprintf("§e%s§f 님이 섬에 초대했습니다. §a/is accept §f로 수락하세요.", p.Name()))
		target.Message("§c주의: 혼자 쓰던 섬이 있다면 그 섬은 삭제됩니다.")
		o.Printf("§a%s 님을 초대했습니다.", target.Name())
	}
}

type isAccept struct {
	playerOnly
	Accept cmd.SubCommand `cmd:"accept"`
}

func (isAccept) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	g.mu.Lock()
	idx, ok := g.invites[p.UUID()]
	delete(g.invites, p.UUID())
	g.mu.Unlock()
	if !ok {
		o.Error("받은 초대가 없습니다.")
		return
	}
	var errMsg string
	g.store.Do(func(tx store.Tx) {
		target, ok := tx.Island(idx)
		if !ok {
			errMsg = "초대한 섬이 더 이상 존재하지 않습니다."
			return
		}
		prof := tx.Profile(p.UUID(), p.Name())
		if cur, ok := tx.Island(prof.Island); ok {
			if cur.Owner == p.UUID() {
				if len(cur.Members) > 0 {
					errMsg = "멤버가 있는 섬의 주인은 다른 섬에 합류할 수 없습니다."
					return
				}
				tx.DeleteIsland(cur.Index)
			} else {
				cur.Members = slices.DeleteFunc(cur.Members, func(id uuid.UUID) bool { return id == p.UUID() })
			}
		}
		target.Members = append(target.Members, p.UUID())
		prof.Island = idx
	})
	if errMsg != "" {
		o.Error(errMsg)
		return
	}
	g.broadcastIsland(idx, fmt.Sprintf("§a%s 님이 섬에 합류했습니다!", p.Name()))
	g.Home(p, idx)
	g.UpdateBoard(p)
}

type isLeave struct {
	playerOnly
	Leave cmd.SubCommand `cmd:"leave"`
}

func (isLeave) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	var errMsg string
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		isl, ok := tx.Island(prof.Island)
		if !ok {
			errMsg = "섬이 없습니다."
			return
		}
		if isl.Owner == p.UUID() {
			errMsg = "섬 주인은 섬을 떠날 수 없습니다."
			return
		}
		isl.Members = slices.DeleteFunc(isl.Members, func(id uuid.UUID) bool { return id == p.UUID() })
		prof.Island = -1
	})
	if errMsg != "" {
		o.Error(errMsg)
		return
	}
	g.sendHome(p)
	g.UpdateBoard(p)
	o.Print("§e섬을 떠났습니다. /is create 로 새 섬을 만들 수 있습니다.")
}

type isKick struct {
	playerOnly
	Kick cmd.SubCommand `cmd:"kick"`
	Name string         `cmd:"player"`
}

func (c isKick) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	var errMsg string
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		isl, ok := tx.Island(prof.Island)
		if !ok || isl.Owner != p.UUID() {
			errMsg = "섬 주인만 추방할 수 있습니다."
			return
		}
		id, target, ok := tx.ProfileByName(c.Name)
		if !ok || !slices.Contains(isl.Members, id) {
			errMsg = "해당 멤버를 찾을 수 없습니다."
			return
		}
		isl.Members = slices.DeleteFunc(isl.Members, func(m uuid.UUID) bool { return m == id })
		target.Island = -1
	})
	if errMsg != "" {
		o.Error(errMsg)
		return
	}
	o.Printf("§e%s 님을 섬에서 추방했습니다.", c.Name)
}

type isVisit struct {
	playerOnly
	Visit cmd.SubCommand `cmd:"visit"`
	Name  string         `cmd:"player"`
}

func (c isVisit) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	idx := -1
	g.store.Do(func(tx store.Tx) {
		if _, prof, ok := tx.ProfileByName(c.Name); ok {
			idx = prof.Island
		}
	})
	if idx < 0 {
		o.Error("해당 플레이어의 섬을 찾을 수 없습니다.")
		return
	}
	p.Teleport(g.grid.Spawn(idx))
	o.Printf("§a%s 님의 섬을 방문합니다.", c.Name)
}

type isTop struct {
	playerOnly
	Top cmd.SubCommand `cmd:"top"`
}

func (isTop) Run(_ cmd.Source, o *cmd.Output, _ *world.Tx) {
	var islands []store.Island
	g.store.Do(func(tx store.Tx) {
		for _, isl := range tx.Islands() {
			islands = append(islands, *isl)
		}
	})
	slices.SortFunc(islands, func(a, b store.Island) int { return b.Broken - a.Broken })
	o.Print("§l§a[ 섬 순위 ]")
	for i, isl := range islands[:min(10, len(islands))] {
		_, ph := oneblock.PhaseAt(isl.Broken)
		o.Printf("§e%d. §f%s §7- %s, %d블록", i+1, isl.OwnerName, ph.Name, isl.Broken)
	}
}

type sellHand struct {
	playerOnly
	Hand cmd.SubCommand `cmd:"hand"`
}

func (sellHand) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	held, off := p.HeldItems()
	price, ok := farm.PriceOf(held.Item())
	if held.Empty() || !ok {
		o.Error("손에 든 아이템은 판매할 수 없습니다.")
		return
	}
	total := price * int64(held.Count())
	p.SetHeldItems(item.Stack{}, off)
	g.addMoney(p, total)
	o.Printf("§a%s x%d 을(를) §6%dG§a 에 판매했습니다.", itemName(held), held.Count(), total)
}

type sellAll struct {
	playerOnly
	All cmd.SubCommand `cmd:"all"`
}

func (sellAll) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	if n, total := sellInventory(p); n > 0 {
		o.Printf("§a작물 %d개를 §6%dG§a 에 판매했습니다.", n, total)
		return
	}
	o.Error("판매할 수 있는 작물이 없습니다.")
}

// sellInventory sells every sellable item in p's inventory.
func sellInventory(p *player.Player) (count int, total int64) {
	inv := p.Inventory()
	for slot, s := range inv.Slots() {
		if s.Empty() {
			continue
		}
		if price, ok := farm.PriceOf(s.Item()); ok {
			count += s.Count()
			total += price * int64(s.Count())
			_ = inv.SetItem(slot, item.Stack{})
		}
	}
	if total > 0 {
		g.addMoney(p, total)
	}
	return count, total
}

type moneyCmd struct{ playerOnly }

func (moneyCmd) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	g.store.Do(func(tx store.Tx) {
		o.Printf("§f소지금: §6%dG", tx.Profile(p.UUID(), p.Name()).Money)
	})
}

type farmInfo struct{ playerOnly }

func (farmInfo) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		lvl := farm.Level(prof.FarmXP)
		o.Print("§l§a[ 농사 정보 ]")
		if lvl < farm.MaxLevel {
			o.Printf("§f레벨: §a%d §7(%d/%d XP)", lvl, prof.FarmXP, farm.XPForLevel(lvl+1))
		} else {
			o.Printf("§f레벨: §a%d §7(MAX)", lvl)
		}
		o.Printf("§f추가 수확 확률: §e%.0f%%", farm.BonusChance(lvl)*100)
		switch {
		case lvl < farm.ReplantLevel:
			o.Printf("§f자동 재파종: §7Lv.%d 에 해금", farm.ReplantLevel)
		case prof.AutoReplant:
			o.Print("§f자동 재파종: §a켜짐")
		default:
			o.Print("§f자동 재파종: §c꺼짐")
		}
		for _, k := range slices.Sorted(maps.Keys(prof.Harvested)) {
			o.Printf("§7- %s: %d", k, prof.Harvested[k])
		}
	})
}

type farmReplant struct {
	playerOnly
	Replant cmd.SubCommand `cmd:"replant"`
}

func (farmReplant) Run(src cmd.Source, o *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		if farm.Level(prof.FarmXP) < farm.ReplantLevel {
			o.Errorf("자동 재파종은 농사 Lv.%d 에 해금됩니다.", farm.ReplantLevel)
			return
		}
		prof.AutoReplant = !prof.AutoReplant
		if prof.AutoReplant {
			o.Print("§a자동 재파종을 켰습니다.")
		} else {
			o.Print("§c자동 재파종을 껐습니다.")
		}
	})
}

type spawnCmd struct{ playerOnly }

func (spawnCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	src.(*player.Player).Teleport(LobbySpawn.Vec3Middle())
}

type shopCmd struct{ playerOnly }

func (shopCmd) Run(src cmd.Source, _ *cmd.Output, _ *world.Tx) {
	p := src.(*player.Player)
	var money int64
	broken := 0
	g.store.Do(func(tx store.Tx) {
		prof := tx.Profile(p.UUID(), p.Name())
		money = prof.Money
		if isl, ok := tx.Island(prof.Island); ok {
			broken = isl.Broken
		}
	})
	phase, _ := oneblock.PhaseAt(broken)
	p.SendForm(shopMenu(phase, money))
}
