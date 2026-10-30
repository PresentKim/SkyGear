package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/slotdeco"
)

// fixtureScreens are the DDUI screens the visual baseline draws, by name:
// each shows exactly one screen with rich, fixed data (no clock, no random
// ids) and needs no player.
func fixtureScreens() map[string]func() {
	var p *player.Player
	nop := func(*player.Player, *world.Tx) {}
	page := func(*player.Player, *world.Tx, int) {}
	slot := func(icon, count, tip string) ItemSlot {
		return ItemSlot{Icon: icon, Count: count, Tip: tip, Run: nop}
	}
	rare := func(r slotdeco.Rarity, name string) string { return r.Mark() + name }
	slots := func(n int) []ItemSlot {
		marks := []string{"§f흙", rare(slotdeco.Uncommon, "§a구리 곡괭이"), rare(slotdeco.Rare, "§9철 검"), rare(slotdeco.Epic, "§5다이아 갑옷"), rare(slotdeco.Legendary, "§6전설의 활")}
		out := make([]ItemSlot, 0, n)
		for i := range n {
			sl := slot("textures/ui/confirm", CountText(i*437+1), marks[i%len(marks)]+"\n§7설명 "+CountText(i))
			sl.Selected = i == 2
			out = append(out, sl)
		}
		return out
	}
	tabs := func() []Option {
		return []Option{
			Tab("§l첫째", "textures/ui/confirm", true, nop),
			Tab("둘째", "textures/ui/cancel", false, nop),
			Tab("셋째", "textures/ui/confirm", false, nop),
		}
	}
	chrome := func(extra ...Option) []Option {
		opts := append(tabs(), BarPager(p, 1, 3, page)...)
		opts = append(opts, Help("§e도움말\n§7무엇을 하는지", nop), Back("뒤로", nop))
		return append(opts, extra...)
	}
	opts := []Option{
		{Label: "§a첫 번째 §7(추천)", Icon: "textures/ui/confirm", Run: nop},
		Toggle("소리 켜기", true, nop),
		Toggle("진동 켜기", false, nop),
		{Label: "§c위험한 선택", Run: nop},
		{Label: "§l굵게 §r보통", Icon: "textures/ui/cancel", Run: nop},
	}
	boardRows := func() [][]BoardCell {
		return [][]BoardCell{
			{MiniCell(slot("textures/ui/confirm", "", "")), TextCell("§e재료"), Blank(), Blank(), Blank(), Blank(), Blank(), Blank()},
			{SlotCell(slot("textures/ui/confirm", "64", rare(slotdeco.Rare, "§9돌"))), SlotCell(slot("textures/ui/cancel", "3", "")), Arrow(), SlotCell(slot("textures/ui/confirm", "1", rare(slotdeco.Epic, "§5결과"))), Frame()},
			{TextCell("가운데 칸을 비운 긴 글 줄"), Blank(), Blank(), Blank(), SlotCell(slot("textures/ui/confirm", "12k", ""))},
		}
	}
	sieveGroups := func() []SieveGroup {
		return []SieveGroup{
			{Head: slot("textures/ui/confirm", "", "§f체"), Name: "참나무 체", Recipes: []SieveRecipe{
				{Input: slot("textures/ui/confirm", "", "흙"), Mesh: slot("textures/ui/cancel", "", "끈 그물"), Drops: slots(11)},
				{Input: slot("textures/ui/cancel", "", "자갈"), Mesh: slot("textures/ui/confirm", "", "철 그물"), Drops: slots(4)},
			}},
			{Head: slot("textures/ui/cancel", "", "§f철 체"), Name: "철 체", Recipes: []SieveRecipe{
				{Input: slot("textures/ui/confirm", "", "모래"), Mesh: slot("textures/ui/cancel", "", "끈 그물"), Drops: slots(2)},
			}},
		}
	}
	sections := func() []GridSection {
		return []GridSection{{Heading: "§l무기", Slots: slots(12)}, {Heading: "도구", Slots: slots(5)}, {Slots: slots(3)}}
	}
	spec := func() SideBoard {
		cs := []PortChoice{
			{Colour: "#ff3030", Tip: "빨강", Run: nop}, {Colour: "#30c030", Tip: "초록", Run: nop},
			{Colour: "#3060ff", Tip: "파랑", Run: nop}, {Icon: "textures/ui/confirm", Tip: "아이콘", Run: nop},
		}
		var b SideBoard
		b.Center = "textures/ui/confirm"
		b.Focus = 1
		names := []string{"아래", "위", "북", "남", "서", "동"}
		for i := range b.Sides {
			b.Sides[i] = SideView{Name: names[i]}
			switch i {
			case 0, 2:
				b.Sides[i].Icon, b.Sides[i].Tip = "textures/ui/cancel", "이웃 블록"
				b.Sides[i].Ports = []SidePort{{Label: "입력", Chosen: 1, Choices: cs}, {Label: "출력", Chosen: 0, Choices: cs[:2]}}
			case 1, 4:
				b.Sides[i].Ports = []SidePort{{Label: "입력", Chosen: 3, Choices: cs}}
			}
		}
		return b
	}
	lobby := func(show func()) func() {
		return func() {
			inLobby, header := InLobby, LobbyHeader
			InLobby = func(*player.Player) bool { return true }
			LobbyHeader = func(*player.Player) (string, string) { return "SkyGear", "textures/ui/confirm" }
			defer func() { InLobby, LobbyHeader = inLobby, header }()
			show()
		}
	}
	home := func(overlay *HomeOverlay) Home {
		face := make([]string, 64)
		for i := range face {
			face[i] = []string{"c68642", "8d5524", "3b2f2f", "1e90ff"}[(i/8+i)%4]
		}
		return Home{
			Profile: HomeProfile{Face: face, Name: "§e플레이어", Title: "초보 모험가", Fill: 0.4, Line: "레벨 3 · 120/300", Run: nop},
			Corner:  []HomeButton{{Icon: "textures/ui/confirm", Label: "알림", Badge: 3, Run: nop}, {Icon: "textures/ui/cancel", Label: "설정", Run: nop}},
			Side:    []HomeButton{{Icon: "textures/ui/confirm", Label: "상점", Run: nop}, {Icon: "textures/ui/cancel", Label: "친구", Badge: 12, Run: nop}},
			Feed:    []string{"§a철수 님이 접속했습니다", "영희 님이 퀘스트를 완료했습니다"},
			Save: HomeSave{Head: "최근 저장", Name: "내 섬", Icon: "textures/ui/confirm", Lines: []string{"§7참나무 섬", "§7마지막 접속 어제"},
				Prev: nop, Next: nop},
			Play: HomeButton{Label: "§l플레이", Run: nop}, PlaySub: "이어서 하기", Manage: HomeButton{Label: "관리", Run: nop},
			Overlay: overlay, Close: nop,
		}
	}
	quest := func(panel *QuestPanel) QuestBook {
		return QuestBook{
			Chapters: []QuestChapter{
				{Name: "§l시작하기", Icon: "textures/ui/confirm", Count: "3/5", On: true, Run: nop},
				{Name: "광산", Icon: "textures/ui/cancel", Count: "0/8", Run: nop},
				{Name: "농사", Icon: "textures/ui/confirm", Count: "1/4", Run: nop},
			},
			Heading: "시작하기", Progress: "3/5", Focus: "c", Key: "start", Panel: panel,
			Nodes: []QuestNode{
				{ID: "a", X: 0, Y: 1, Icon: "textures/ui/confirm", Shape: ShapeSquare, State: QuestDone, Tip: "§a완료\n첫 퀘스트", Run: nop},
				{ID: "b", X: 1, Y: 0, Icon: "textures/ui/cancel", Shape: ShapeDiamond, State: QuestDone, Tip: "둘", Deps: []string{"a"}, Run: nop},
				{ID: "c", X: 1, Y: 2, Icon: "textures/ui/confirm", Shape: ShapeCircle, State: QuestOpen, Tracked: true, Tip: "§e진행 중", Deps: []string{"a"}, Run: nop},
				{ID: "d", X: 2, Y: 1, Icon: "textures/ui/confirm", Shape: ShapeSquare, State: QuestOpen, Tip: "넷", Deps: []string{"b", "c"}, Run: nop},
				{ID: "e", X: 3, Y: 0, Icon: "textures/ui/cancel", Shape: ShapeSquare, State: QuestLocked, Tip: "잠김", Deps: []string{"d"}, Run: nop},
				{ID: "f", X: 3, Y: 2, Icon: "textures/ui/confirm", Shape: ShapeGate, State: QuestLocked, Tip: "다른 장으로", Deps: []string{"d"}, Run: nop},
			},
		}
	}
	return map[string]func(){
		"menu": func() { Menu(p, "§l메뉴", "본문 §e강조§r 글자\n둘째 줄", append(chrome(), opts...)...) },
		"confirm": func() {
			Confirm(p, "확인", "정말로 삭제하시겠습니까?", "§c삭제", nop)
		},
		"cards": func() {
			cards := []Option{}
			for _, label := range []string{"집", "§a농장", "광산", "상점", "§6보물", "우편함", "지도", "설정", "정보"} {
				cards = append(cards, Option{Label: label, Icon: "textures/ui/confirm", Run: nop})
			}
			CardMenu(p, "카드", "골라 보세요", append(chrome(), cards...)...)
		},
		"grid": func() {
			ItemGrid(p, "목록", "아이템 §7(30)", slots(30), chrome(Option{Label: "닫기", Run: nop})...)
		},
		"sections": func() {
			ItemGridSections(p, "구역", "본문", sections(), chrome(Option{Label: "더 보기", Run: nop})...)
		},
		"board": func() {
			ItemBoard(p, "제작", "§7재료를 놓으세요", boardRows(), chrome(Option{Label: "제작", Icon: "textures/ui/confirm", Run: nop})...)
		},
		"sieve": func() {
			SieveBoard(p, "체", [][]BoardCell{{SlotCell(slot("textures/ui/confirm", "", "흙")), SlotCell(slot("textures/ui/cancel", "", "자갈")), Blank()}}, sieveGroups(), chrome()...)
		},
		"split": func() {
			SplitMenu(p, "분할", Split{
				Tabs: tabs(), Slots: slots(40), Page: 0, Pages: 2, Turn: page,
				Detail:  Detail{Icon: "textures/ui/confirm", Name: "§9철 검", Status: "§a보유 중", Body: "날카로운 검입니다.\n§7공격력 6"},
				Rows:    []SlotRow{{Heading: "재료", Slots: slots(3)}, {Heading: "결과", Slots: slots(1)}},
				Actions: []Option{{Label: "만들기", Icon: "textures/ui/confirm", Run: nop}, {Label: "분해", Run: nop}, Help("도움말", nop), Back("뒤로", nop)},
			})
		},
		"search": func() {
			SearchGrid(p, "검색", "이름 입력", "철", func(*player.Player, string) SearchResult {
				return SearchResult{Info: "§7결과 17개", Sections: sections()}
			}, chrome()...)
		},
		"listsplit": func() {
			ListSplitMenu(p, "친구", ListSplit{
				Rows: []ListRow{
					{Label: ListRowText("§f친구 하나", "§a접속 중"), Icon: "textures/ui/confirm", Selected: true, Run: nop},
					{Label: ListRowText("친구 둘", "§7오프라인"), Icon: "textures/ui/cancel", Run: nop},
					{Label: "정보 없는 친구", Run: nop},
				},
				Body:    "친구 하나\n§7마지막 접속 방금",
				Actions: append(tabs(), Option{Label: "귓속말", Run: nop}, Option{Label: "초대", Run: nop}, Option{Label: "삭제", Run: nop}, Option{Label: "차단", Run: nop}, Back("뒤로", nop)),
			})
		},
		"input": func() {
			Input(p, CreateToken+"새 섬", "섬 이름을 입력하세요", "내 섬", func(*player.Player, *world.Tx, string) {}, func(*player.Player, *world.Tx) {})
		},
		"slotmenu": func() {
			SlotMenu(p, "창고", Slot{Icon: "textures/ui/confirm", Text: "1.2k", Run: nop}, "§7보관 중 1,234개\n최대 4,096개",
				[]Slot{{Icon: "textures/ui/confirm", Text: "x2", Run: nop}, {Icon: "textures/ui/cancel", Run: nop}},
				[]Slot{{Icon: "textures/ui/confirm", Run: nop}, {Icon: "textures/ui/cancel", Text: "9", Run: nop}},
				chrome(Option{Label: "모두 꺼내기", Run: nop})...)
		},
		"choice": func() {
			ChoiceMenu(p, "선택", ChoiceBoard{
				Stats: []Stat{{Icon: "textures/ui/confirm", Label: "속도", Value: "12"}, {Icon: "textures/ui/cancel", Label: "힘", Value: "§c7"}, {Label: "운", Value: "3"}},
				Note:  "§7하나씩 고르세요",
				Groups: []ChoiceGroup{
					{Title: "§l모드", Choices: []Choice{
						{Label: "일반", Info: "기본 설정", Icon: "textures/ui/confirm", On: true, Badge: "사용 중", Run: nop},
						{Label: "빠름", Info: "더 빠르게", Icon: "textures/ui/cancel", Run: nop},
						{Label: "잠김", Info: "아직 못 씀", Icon: "textures/ui/cancel", Dim: true, Run: nop},
					}},
					{Title: "색", Choices: []Choice{{Label: "빨강", Run: nop}, {Label: "초록", On: true, Run: nop}}},
				},
				Actions: chrome(Option{Label: "초기화", Run: nop}),
			})
		},
		"sides": func() {
			SidesMenu(p, "면 설정", spec(), chrome()...)
		},
		"sides-open": func() {
			updateSides(p, "면 설정", spec(), &sidePicker{side: 0, port: 0}, chrome())
		},
		"plane": func() {
			PlaneMenu(p, Plane{
				FOV: 70, Distance: 12,
				Spots: []PlaneSpot{
					{X: -3, Z: -2, W: 2, H: 2, On: true, Tip: "선택됨", Run: nop},
					{X: 0, Z: -2, W: 2, H: 2, Tip: "빈 땅", Run: nop},
					{X: 3, Z: 1, W: 3, H: 2, Run: nop},
				},
				Info:    "§e땅을 고르세요\n§7왼쪽 위 칸이 선택되어 있습니다",
				Actions: []Option{{Label: "확정", Icon: "textures/ui/confirm", Run: nop}, {Label: "취소", Run: nop}},
				Corner:  "§a12/20 칸",
				Close:   nop,
				Keys:    map[Key]func(*player.Player, *world.Tx){KeyUp: nop, KeyDown: nop, KeyLeft: nop, KeyRight: nop, KeyOK: nop},
			})
		},
		"quest": func() {
			QuestBookMenu(p, "퀘스트", quest(nil), Help("도움말", nop), Back("뒤로", nop))
		},
		"quest-panel": func() {
			QuestBookMenu(p, "퀘스트", quest(&QuestPanel{
				Icon: "textures/ui/confirm", Title: "§e진행 중인 퀘스트", Status: "§7진행 1/3",
				Body:    "철 곡괭이를 만들어 오세요.\n§7광산에서 쓸 수 있습니다.",
				Rows:    []SlotRow{{Heading: "가져올 것", Slots: slots(3)}, {Heading: "보상", Slots: slots(2)}},
				Actions: []Option{{Label: "제출", Icon: "textures/ui/confirm", Run: nop}, {Label: "추적", Run: nop}},
				Close:   nop,
			}), Help("도움말", nop), Back("뒤로", nop))
		},
		"home": lobby(func() { HomeScreen(p, home(nil)) }),
		"home-overlay": lobby(func() {
			HomeScreen(p, home(&HomeOverlay{Title: "시스템", Body: "무엇을 할까요?", Buttons: []HomeButton{{Label: "계속하기", Run: nop}, {Label: "나가기", Run: nop}}}))
		}),
		"lobby-menu": lobby(func() { Menu(p, "§l로비", "본문", append(chrome(), opts...)...) }),
		"loading": lobby(func() {
			LoadingScreen(p, Loading{Icon: "textures/ui/confirm", Title: "§l내 섬", Sub: "참나무 섬", Line: "불러오는 중", Tip: "§7팁: 나무를 먼저 심어 보세요"})
		}),
		"console": func() {
			ConsoleScreen(p, "콘솔", Console{
				Lines: []string{"> list", "§a플레이어 2명 접속 중", "> tp 0 64 0", "§c알 수 없는 명령입니다"},
				Hint:  "명령어", Submit: "실행", Run: func(*player.Player, *world.Tx, string) {}, Back: nop, BackLabel: "뒤로",
			})
		},
		"probe": func() {
			ProbeScreen(p, "시험", []any{ProbeNode("probe", "id", "p1"), ProbeNode("box", "class", "mc-gap-3")}, nil, nil)
		},
	}
}

// runFixture shows fixture name and returns the roots it showed.
func runFixture(t *testing.T, name string) (roots []any) {
	t.Helper()
	show, ok := fixtureScreens()[name]
	if !ok {
		t.Fatalf("no fixture %q", name)
	}
	oreShown = func(_ *oreLayout, root any) { roots = append(roots, root) }
	defer func() { oreShown = nil }()
	show()
	return roots
}

// fixtureNames are the screens the visual baseline must cover.
var fixtureNames = []string{
	"menu", "confirm", "cards", "grid", "sections", "board", "sieve", "split", "search", "listsplit", "input",
	"slotmenu", "choice", "sides", "sides-open", "plane", "quest", "quest-panel", "home", "home-overlay",
	"lobby-menu", "loading", "console", "probe",
}

// TestFixturesCoverAll checks the fixtures are exactly the screens the
// baseline lists.
func TestFixturesCoverAll(t *testing.T) {
	var got []string
	for name := range fixtureScreens() {
		got = append(got, name)
	}
	sort.Strings(got)
	want := slices.Clone(fixtureNames)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("fixtures = %v, want %v", got, want)
	}
}

// TestFixturesShowOnce checks every fixture shows exactly one screen, with
// no player.
func TestFixturesShowOnce(t *testing.T) {
	for _, name := range fixtureNames {
		roots := runFixture(t, name)
		if len(roots) != 1 {
			t.Errorf("%s showed %d roots, want 1", name, len(roots))
		}
	}
}

// TestFixturesKeepLobby checks no fixture leaves InLobby or LobbyHeader set.
func TestFixturesKeepLobby(t *testing.T) {
	for _, name := range fixtureNames {
		runFixture(t, name)
	}
	if InLobby != nil || LobbyHeader != nil {
		t.Error("a lobby fixture left InLobby or LobbyHeader set")
	}
}

// TestDumpFixtures writes every fixture as <name>.json in $OREUI_FIXTURES:
// {"root": the root showScreen was given}.
func TestDumpFixtures(t *testing.T) {
	dir := os.Getenv("OREUI_FIXTURES")
	if dir == "" {
		t.Skip("OREUI_FIXTURES not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range fixtureNames {
		roots := runFixture(t, name)
		if len(roots) != 1 {
			t.Fatalf("%s showed %d roots, want 1", name, len(roots))
		}
		data, err := json.MarshalIndent(map[string]any{"root": roots[0]}, "", " ")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
