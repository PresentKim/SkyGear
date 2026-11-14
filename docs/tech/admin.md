# 운영자 도구 (`/sgadmin`, `/sgdebug`)

`platform.toml` 의 `[Admin] Operators` 에 있는 플레이어(XUID, 숫자)만 보고 쓸 수 있다. 이름을 적어도 되지만 Xbox 닉네임은 다른 사람에게 넘어갈 수 있어, 이름으로만 운영자가 된 플레이어가 접속하면 서버 로그에 XUID 와 함께 경고가 남는다. 인증을 끈 시험 서버에서는 XUID 가 비어 있어 이름으로만 판별한다. 세이브 안에서 쓴다(`tps` 는 로비에서도).

운영자는 들어가 있는 어느 세이브에서든 호스트처럼 참여자를 관리한다: `/members`, 또는 플레이어를 웅크린 채 우클릭(인벤토리 보기·추방, [01](../design/01-platform.md) 참여자 관리).

## 관리 (`/sgadmin`)

서버를 운영하며 쓰는 명령. 코드: `content/skygear/admin.go`.

| 명령 | 하는 일 |
|---|---|
| `/sgadmin give <ID> [개수]` | 게임 아이템·기계·도구(`research_table`, `drawer`, `drawer_key`, `barrel`, `hammer`, `sieve`, `auto_sieve`, `crusher`, `auto_miner`, `harvester`, `drawer_controller`, `tinker_table`, `relic_table`, `essence_collector`, `net_conduit`, `dimension_stone`, `fairy_emerald`, `fairy_dust`, `ingot_<금속>`, `essence_<몹>`, `seed_<씨앗>`, `petal_<색>`, 보구 `relic_<종류>`(1등급)·`relic_<종류>_<등급 1~5>` 등. 따로 적지 않은 게임 아이템도 `skygear:` 를 뗀 ID로 모두 들어 있고, 화면용 아이콘만 빠진다. 바닐라 아이템은 `minecraft:diamond` 처럼 전체 이름으로, 변형이 있는 아이템은 `minecraft:<이름>:<메타>` 로 들어 있다. 새로 등록한 아이템은 따로 적지 않아도 자동으로 들어간다. 자동 완성에는 네임스페이스를 붙인 전체 목록(`skygear:hammer`, `minecraft:diamond`)이 나오고, 한쪽에만 있는 이름은 네임스페이스를 빼고 써도 된다(`diamond`, `hammer`). 두 쪽에 다 있는 이름(`barrel`)은 네임스페이스를 붙이라고 알린다) |
| `/sgadmin research <연구 ID\|all\|reset\|free>` | 연구 하나·전부 완료, 연구 모두 지우기, `free`는 모든 조합법을 인스턴스가 닫힐 때까지 열기(다시 하면 잠금) |
| `/sgadmin advance <발전과제 ID\|all\|reset>` | 발전과제 하나(앞 단계 포함)·전부 달성, 모두 지우기 |
| `/sgadmin tps` | 내 HUD 왼쪽 위 서버 부하 줄 켜기·끄기(아래, 로비에서도) |

## 디버그 (`/sgdebug`)

시험에 쓰는 명령: 일을 지금 일으키거나 상태를 출력한다. 코드: `content/skygear/debug.go`, `debug_spline.go`, `debug_ui.go`, `debug_factory.go`, `debug_oreui.go`.

| 명령 | 하는 일 |
|---|---|
| `/sgdebug locator <규칙 이름> [off]` | 위치 표시 막대 게임 규칙을 그 이름으로 켜서(`off` 면 꺼서) 나에게만 보내고, 앞 20칸에 빨간 웨이포인트를 단다. 실제 클라이언트에서 `locatorBar` 로는 막대가 뜨지 않아 이름(`locatorbar` 등)을 가리는 시험이다([new-packets.md](new-packets.md)) |
| `/sgdebug spline [rot=py\|yp] [flip] [noset] [rel] [wait] [tfrac\|tticks] [noprog] [endup] [lin]` | 첫 입장 카메라 이동(`intro.Keys`, 내 위치 기준: 위에서 내려다보는 자리에서 눈 위치까지 1.5초)을 서로 다른 점 다섯 개로 나눠(`sghook.Subdivide`, Catmull-Rom 은 점이 넷 이상 필요하고 연속된 같은 점에서 깨진다) 클라이언트 스플라인으로 재생하고 끝나면 카메라를 푼다. 진행도는 지나온 거리 ÷ 전체 길이(0~1)다. `rot=yp` 는 회전 값을 요, 피치 순서로 보내고(기본 `py` 는 피치, 요), `flip` 은 피치 부호를 뒤집고, `noset` 은 스플라인 앞의 시작 위치 `Set` 을 보내지 않으며, `rel` 은 제어점을 눈 위치 기준 상대 좌표로 보내고, `wait` 는 시작 위치 `Set` 을 먼저 보내고 1초 뒤 다른 틱에 스플라인을 보낸다. `tfrac`·`tticks` 는 진행도 키프레임의 시간을 전체 길이의 비율·틱으로 보내고, `noprog` 는 진행도 키프레임을 빼고 보낸다(클라이언트가 강제 종료된다). `endup` 은 끝점을 눈보다 6칸 위에 두고, `lin` 은 직선 스플라인(종류 1)으로 보낸다. 함께 쓸 수 있다 |
| `/sgdebug oreinput` | Ore UI 페이지가 받는 입력 이벤트(마우스, 휠, 키, 터치, 포인터, 포커스, 글 입력, 게임패드)를 화면에 줄로 보이고 서버로 보낸다. 서버가 받은 줄과 그동안 클라이언트가 보낸 패킷(`ScriptMessage` 는 내용까지)이 창 위에 보인다([oreui-probes.md](oreui-probes.md)) |
| `/sgdebug oreres` | 서버가 Ore UI 페이지에 화면 리소스를 줄 수 있는 길을 차례로 시험한다([oreui-probes.md](oreui-probes.md)) |
| `/sgdebug research` | 연구 화면을 연다(연구대를 쓴 것과 같다. 게임 메뉴에는 연구 카드가 없다) |
| `/sgdebug ui [기계]` | 기계를 놓지 않고 그 기계 창을 연다(입력·출력 칸에 예시 아이템, 진행도·에너지 절반). 기계 없이 쓰면 창이 있는 기계 목록 폼이 열리고 버튼으로 차례로 본다. 창 모양 확인용 |
| `/sgdebug place <아이템> <dx> <dy> <dz>` | 내 발 위치에서 그만큼 떨어진 칸에 블록을 놓는다. 그 아이템을 든 채 아래 블록 윗면을 누른 것처럼 게임의 놓기 처리를 그대로 거친다(기계 상태·방향). 아이템은 `give` ID 나 바닐라 이름. 조준 없이 짓는 용도 |
| `/sgdebug floor <반지름>` | 발밑 높이에 반지름만큼 돌 바닥을 깐다(있는 블록은 그대로). 빈 공간에 시험할 자리를 만든다 |
| `/sgdebug look <yaw> <pitch>` | 시선을 돌리고(yaw 0 남쪽, pitch 90 아래) WAILA 패널을 바로 갱신한 뒤 보는 블록을 출력 |
| `/sgdebug terminal` | 보고 있는 망 단말·망 컨트롤러의 단말 창을 연다(단말 우클릭과 같다) |
| `/sgdebug net <개수> [위]` | 컨트롤러·단말·가장 큰 디스크가 든 드라이브로 이루어진 전원이 켜진 망을 내 북쪽 두 칸, 발보다 `위`칸 위에 짓고 `개수`가지 아이템을 채운 뒤 그 단말 창을 연다. 단말이 이미 있으면 창만 연다 |
| `/sgdebug status` | `researched= chunks= advanced=` 출력(확인용) |
| `/sgdebug skip <분>` | 인스턴스 시간을 n분 앞당김(날짜 변경·작물·분실물 만료 확인용) |
| `/sgdebug timed <초>` | 손에 든 아이템을 지금부터 n초(현실 시간) 기간제로 만든다. 0 이면 기간을 푼다 ([기간제 아이템](timed-items.md)) |
| `/sgdebug holderpose <x> <y> <z> <크기%> [dx dy dz]` | 서랍 앞면의 손에 쥐는 모양 아이템(막대기·뼈다귀)의 보정 회전·크기·위치를 팩을 다시 받지 않고 바꿔 본다([drawer.md](drawer.md) '손에 쥐는 모양 아이템') |
| `/sgdebug fairy` | 선물 요정을 지금 나에게 부름 |
| `/sgdebug weather <clear\|rain\|snow>` | 요정의 에메랄드 없이 날씨 마법을 씀(같은 날씨면 3일 더함) |
| `/sgdebug sun <flow\|day\|dusk\|night>` | 요정의 에메랄드 없이 해를 고정하거나 흐르게 함 |
| `/sgdebug hunt [몬스터]` | 해금 없이 그 몬스터(기본 슬라임)의 이차원 사냥을 바로 시작 ([17](../design/17-dimension.md)) |
| `/sgdebug fx <spin_blade\|crossbow\|flame_ring\|lightning_staff\|frost_boomerang\|poof\|blast\|blink\|mend>` | 이차원 무기 보구의 이펙트(또는 몬스터가 죽는 연기, 몬스터의 폭발·순간이동·회복)를 내 주위에서 5초 동안 재생. 어디서든 되고 아무것도 때리지 않는다. 그림은 `tools/pixelfx/export.mjs` 가 pixelfx 에서 만든다 |
| `/sgdebug research random [퍼센트]` | 연구를 무작위로 그만큼 완료(연구 화면의 두 탭 채우기, [research-screen.md](research-screen.md)) |
| `/sgdebug research load <개수>` | 연구 복사본을 개수만큼 만든다(긴 목록 시험, 저장하지 않음, 0 이면 없앰) |
| `/sgdebug factory <개수>`, `/sgdebug dense <개수>` | 부하 시험용 공장을 짓는다(아래) |

## 서버 부하 표시

운영자가 `/sgadmin tps` 로 켜고 끈다(플레이어마다, 서버를 다시 켜면 꺼짐; 로비에서도 됨). 켜면 HUD 왼쪽 위(일차 줄 아래)에 1초마다 한 줄로 보인다(로비에서는 그 줄만). 코드: `common/perf`(`Line`, `Toggle`), `platform/load.go`, `content/skygear/screens/sidebar.go`.

`TPS 20.0 · 부하 12% · 틱 6.1ms/9.7ms · 메모리 76MB`

| 항목 | 뜻 |
|---|---|
| TPS | 지금 있는 월드의 최근 5초 초당 틱 수(정상 20). 19 아래 노랑, 15 아래 빨강 |
| 부하 | 평균 틱 시간 ÷ 50ms. 50% 아래 초록, 100% 아래 노랑, 넘으면 빨강(틱이 밀려 TPS 가 떨어짐) |
| 틱 | 최근 5초 평균/최대 틱 시간(그 월드의 다른 작업에 밀려 기다린 시간 포함). 읽는 법은 [performance.md](performance.md) |
| 메모리 | Go 힙 메모리 |

## 부하 시험

`/sgdebug factory <개수>`(1~100): 운영자를 가운데 두고 공장 모듈(채굴기 줄·체 줄·농장과 수확기)을 지어 부하를 본다(`/sgadmin tps` 로 켜고). 결과와 Dragonfly 스레딩은 [performance.md](performance.md).

`/sgdebug dense <개수>`(1~4000): 운영자를 가운데 두고 2×1 칸짜리 밀집 셀(자동 자원 채굴기, 옆 분쇄기 → 호퍼 → 서랍)을 빽빽하게 짓는다. 기계를 많이 둘 때의 부하를 게임 안에서 잰다(결과는 [performance.md](performance.md)).
