package platform

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/instance"
	"github.com/presentkim/dragonfactory/internal/ui"
)

// GuideVersion is increased when the guide changes; players see it again once.
const GuideVersion = 2

func (pf *Platform) guideText() string {
	return fmt.Sprintf(`§l%[1]s§r 에 오신 것을 환영합니다!

§e■ 어떤 서버인가요?§r
공허 위의 작은 섬에서 시작해 자원을 만들고, 가공하고, 자동화해 나가는 공장 만들기 서버입니다.

§e■ 시작하기§r
로비에서 §a나침반§r을 사용하면 메뉴가 열립니다. 세이브는 %[2]d개까지 만들 수 있어요.
세이브마다 섬, 인벤토리, 진행 상황이 따로 저장됩니다.

§e■ 함께 하기§r
친구를 추가하고, 세이브 안에서 §a/invite§r로 친구를 초대하거나 초대 코드를 알려 주세요.
초대 코드는 §a/join 코드§r 또는 메뉴의 '초대 코드 입력'으로 씁니다. 세이브당 최대 %[3]d명.
초대받은 세이브는 호스트가 플레이 중일 때 들어갈 수 있습니다.

§e■ 알아두세요§r
· 게임 시간은 세이브 주인이 플레이하는 동안에만 흐릅니다.
· 낙하 피해가 없고, 죽어도 아이템을 잃지 않습니다.
· §a/lobby§r 로비로 이동 · §a/saves§r 세이브 선택 · §a/friends§r 친구 · §a/guide§r 이 안내 다시 보기`, pf.game.Name(), Slots, MaxMembers)
}

// showGuide shows the server guide. With first set, closing it records that
// the account has seen the current version.
func (pf *Platform) showGuide(p *player.Player, first bool) {
	ui.Menu(p, "서버 안내", pf.guideText(), ui.Option{Label: "확인", Run: func(p *player.Player, _ *world.Tx) {
		if first {
			if err := pf.db.SetGuideVersion(instance.AccountID(p), GuideVersion); err != nil {
				pf.log.Error("set guide version", "err", err)
			}
			pf.showSaves(p)
		}
	}})
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%d시간 %d분", h, m)
	}
	return fmt.Sprintf("%d분", m)
}
