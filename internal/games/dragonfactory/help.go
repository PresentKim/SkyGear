package dragonfactory

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// helpTopics are the in-game help pages (menu → 도움말), in display order.
var helpTopics = []struct{ title, icon, text string }{
	{"시작하기", iconMission, "§f맨손으로 섬 가운데 나무를 캐 §e원목§f을 모으세요. 원목 → 참나무 판자 → §e조합대§f → §d연구대§f(조합대 2개)를 만들고, 연구대에서 §d막대기§f부터 연구하세요. 처음에는 이 셋만 만들 수 있고, 나머지 조합법은 연구로 열립니다.\n\n§f묘목을 모두 잃었다면 §e/묘목§f(10분에 한 번). 묘목 옆에서 §e웅크리기를 반복§f하면 빨리 자랍니다.\n\n§7메뉴는 핫바 마지막 칸의 메뉴 아이템, /menu 또는 /df 로 엽니다. 오른쪽의 퀘스트 창이 다음 연구를 알려 줍니다."},
	{"손작업 자원", iconCores, "§f막대기를 들고 §e잔디·흙을 우클릭§f하면 가끔 §e돌멩이§f가 나옵니다. 돌멩이 4개 → 조약돌(연구 필요).\n§f돌 이상 도끼로 §e원목을 우클릭§f해 벗기면 가끔 §e나무껍질 섬유§f가 나옵니다. 섬유 4개 → 실.\n§f망치로 조약돌 → 자갈 → 모래 → 먼지로 부수고, 체로 거르면 광물 조각 등이 나옵니다."},
	{"기계", iconAll, "§f분쇄기: 돌→조약돌→자갈→모래→먼지.\n§f자동 자원 채굴기: 바로 아래 블록을 캐서 옆·위 상자로.\n§f자동 농작물 수확기: 같은 높이 9×9의 다 자란 작물을 거둠.\n\n§7기계를 우클릭하면 창이 열리고, 위·옆 호퍼로 넣고 아래 호퍼로 뺍니다. 기계를 부수면 기계와 내용물이 함께 나옵니다. 창의 정보 아이콘(닫기 버튼 왼쪽)에 설명이 있고, 기계 위 글자는 진행도입니다."},
	{"체와 망치", iconSell, "§f체: 자갈·모래·흙·먼지를 들고 우클릭하면 하나가 얹히고, 우클릭할 때마다 걸러져 8번째에 부산물(부싯돌·철/금 조각·점토·씨앗 등)이 위로 튀어나옵니다. 2칸 안의 체도 함께 걸러집니다.\n§f자동 체: 호퍼·손으로 넣으면 저절로 걸러 옆·아래 상자로 보냅니다.\n\n§f망치: 돌·조약돌·자갈·모래를 캐면 분쇄된 결과(조약돌·자갈·모래·먼지)가 나옵니다."},
	{"서랍", iconLost, "§f앞면 우클릭: 손의 아이템 넣기. 0.5초 안에 두 번: 인벤토리의 같은 아이템 모두 넣기.\n§f앞면 좌클릭: 1개, 웅크리고 좌클릭: 1스택 꺼내기.\n§f잠금 키로 우클릭: 잠금(비어도 아이템 종류 유지).\n§f업그레이드: 용량(×2·×4·×8), 공허(넘치는 것은 사라짐).\n\n§7웅크리고 빈손 우클릭: 서랍 메뉴(회수 등). 옆·위를 캐면 내용물째 회수됩니다."},
	{"섬 확장", iconIsland, "§f메뉴 → 섬 확장: 위에서 내려다보는 모드가 됩니다. 이동 키로 청크를 고르고, 웅크리면 구매(조약돌)·끝내기.\n\n§a초록§f: 소유, §e노랑§f: 구매 가능(소유 청크와 맞닿은 칸), §6주황§f: 고른 칸.\n§7소유한 청크 안에서만 건축·채굴할 수 있습니다. 공허로 떨어진 아이템은 분실물 보관함으로 갑니다."},
	{"연구", iconMission, "§f조합법은 처음에 참나무 판자·조합대·연구대만 열려 있습니다. §d연구대§f(조합대 2개)를 놓고 사용하거나 메뉴 → 연구에서 연구를 고르세요.\n\n§e노란§f 연구는 지금 할 수 있고, §8회색§f은 먼저 해야 할 연구가 있습니다. 납품하기를 누르면 인벤토리에서 필요한 아이템을 냅니다(나눠 내도 됩니다). 다 내면 그 조합법이 섬의 모두에게 바로 열립니다.\n\n§f무엇을 할지 모르겠으면 오른쪽 퀘스트 창이나 메뉴 → §e발전과제§f의 힌트를 보세요.\n\n§7화로 굽기와 기계 가공은 잠기지 않습니다."},
}

// showHelp lists the help topics.
func (s *session) showHelp(p *player.Player) {
	var opts []ui.Option
	for _, t := range helpTopics {
		opts = append(opts, s.opt("§f"+t.title, t.icon, func(p *player.Player, _ *world.Tx) {
			ui.Menu(p, "도움말 · "+t.title, t.text, s.opt("§7← 도움말", iconBack, func(p *player.Player, _ *world.Tx) { s.showHelp(p) }))
		}))
	}
	ui.CardMenu(p, "도움말", "§7궁금한 항목을 고르세요.", append(opts, s.backOption())...)
}
