# 기능 확장

게임의 기능은 `content/skygear` 아래 패키지 하나씩이다. 기능은 서버 이벤트를 훅으로 받고, 자기 데이터를 `core` 의 파트에 두고, 루트 패키지의 기능 목록에 한 줄로 오른다.
블록의 패널 줄, 아이템의 칸 장식, 창작 목록의 항목, 명령어, 설정도 기능이 자기 패키지에서 선언한다.
이 문서는 그 바탕이 되는 층, 이벤트가 기능에 닿는 길, 새 기능을 더하는 순서, 구조를 지키는 테스트를 적는다.
Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있다.

## 층

아래에서 위로 쌓인다. 각 층은 자기보다 아래 층만 import 한다.

| 층 | 위치 | 하는 일 |
|---|---|---|
| Dragonfly 패치 | `third_party/dragonfly` | Dragonfly 의 동작 자체. 패치 규칙은 [dragonfly.md](dragonfly.md) |
| `sghook` | `common/sghook` | 패킷 훅, 세션 내부 접근, 클라이언트에 보내는 아이템·블록 항목 고치기(`EditCustomItems`, `EditCustomBlocks`, `ApplyEntryEdits`). Dragonfly 내부는 여기서만 건드린다 |
| `clientdef` | `common/clientdef` | 커스텀 아이템·블록이 클라이언트에 보내는 컴포넌트. 공개 레지스트리만 쓰고 게임을 모른다 |
| `core` | `content/skygear/core` | 기능의 커널. `Session`과 그 파트(`Part`, `Players`, `Saved`), 훅(`Hook`, `CancelHook`)과 이벤트, 핸들러(`PlayerHandler`, `WorldHandler`), 수명 호출(`Lifecycle`), 기능 서술자(`Feature`)와 `Install`, 창작 목록 순서(`CreativeItem`, `CreativeItems`), 기능의 명령어(`Commands`, `PlayerSession`), 플레이어를 붙잡는 모드(`RegisterTakeover`, `Takeover`) |
| 기능 패키지 | `content/skygear/<기능>` | 한 기능의 블록·아이템·화면·규칙과, 그 기능이 훅에 거는 구독자 |
| 루트 | `content/skygear` | 기능 목록(`features.go`)과 플랫폼 접착제(`Game`, `session`), 운영자 도구 |

`content/skygear` 안의 패키지는 `content/skygear/layers_test.go` 의 `layers` 에 아래에서 위로 적혀 있다.

- 데이터: `locales`, `naming`, `textfmt`, `formicon`, `blockmodel`, `netshape`, `sgitem`, `sievetable`, `wintext`, `machinerecipe`, `advancement`, `hopper`, `migrate`, `wire`
- 핵심: `core`, `network`, `achievement`, `preview`, `look`, `autofill`, `decor`
- 진행: `progress`, `island`, `climate`
- 기능: `machine`, `tinker`, `conduit`, `storage`, `farm`, `fairy`, `dimension`, `handwork`, `expand`, `copier`, `wand`, `netstore`, `elevator`
- 내용 목록: `catalog`
- 화면: `codex`, `screens`

`look`(WAILA 패널), `autofill`(손에 든 묶음 다시 채우기), `decor`(아이템 칸 장식)는 핵심 칸에 있어서 기능 칸의 패키지가 셋을 부를 수 있다.

루트 패키지에 남은 파일은 다음과 같다.

| 파일 | 하는 일 |
|---|---|
| `features.go` | 기능 목록 `Features` 와 `core.Install`. 루트 자신의 기능 `itemsFeature` 는 아이템 항목의 이름을 번역 키로 바꾸고(`naming.LocalItemName`) 클라이언트가 빈 나무 양동이로 액체를 겨누게 한다(`sgitem.ClipWoodenBucket`) |
| `game.go` | `Game` 과 `session`. 플랫폼의 `instance.Game`·`instance.Session` 을 구현한다. `New` 가 기능의 명령어(`core.Commands()`)와 창작 목록(`catalog.RegisterCreativeItems`)과 운영자 명령을 등록한다 |
| `state.go` | 세이브 상태 문서의 버전(`State`, `core.NewSaved`) |
| `blockcounts.go` | 커스텀 블록 수와 상태 수(`CustomBlockCounts`, `LogCustomBlockCounts`). 서버가 시작할 때 로그에 남긴다 |
| `admin.go` | 운영자 명령 `/sgadmin` |
| `debug.go`, `debug_factory.go`, `debug_oreui.go`, `debug_ui.go` | 운영자 디버그 명령 `/sgdebug` 와 그 도구 |

무엇을 import 할 수 있는지는 다음과 같다.

- `sghook` 과 `clientdef` 는 `content/` 와 `platform/` 을 import 하지 않는다. 다른 `common/*` 패키지도 마찬가지다(`TestFeaturesLiftable`).
- `core` 는 `common`, `platform`, 그리고 데이터 층의 패키지를 import 한다. 기능 패키지는 import 하지 않는다(`TestFeaturesLiftable`).
- 기능 패키지는 `core`, `common`, `platform`, 자기보다 아래 칸의 `content/skygear/*` 패키지를 import 한다. 같은 칸의 기능끼리는 `content/skygear/layers_test.go` 의 `siblings` 표에 적힌 import 만 허용된다(`TestLayers`, `TestFeatureSiblings`). 루트 패키지는 직접도 간접도 import 하지 않는다(`TestFeaturesLiftable`).
- 루트 패키지는 모든 기능을 import 한다.
- `look` 은 게임 패키지 가운데 `core` 와 `naming` 만, `decor` 는 `naming` 만 import 한다(`layers_test.go` 의 `onlyImports`, `TestCapabilityImports`). 블록과 아이템이 인터페이스(`waila.Provider`, `slotdeco.Decorated`)로 자기를 설명하므로 두 패키지는 블록·아이템을 둔 패키지를 모른다.

위 칸이 아래 칸을 불러야 하면 아래 칸이 함수 변수를 두고 위 칸이 `init` 에서 채운다(예: `core.ShowMenu` 는 `screens` 가 채운다).

## 이벤트의 길

플레이어 이벤트는 Dragonfly 가 `core.PlayerHandler` 의 `Handle...` 메서드를 불러 시작한다. 월드 이벤트는 `core.WorldHandler` 가 받는다(`HandleEntityDespawn`, `HandleColumnLoad`, `HandleColumnUnload`).
세션 수명 이벤트는 플랫폼이 세션의 메서드를 불러 시작한다.

| 호출하는 쪽 | 호출 | 내보내는 훅 |
|---|---|---|
| `Game.Start` | `core.Load.Emit` (월드 트랜잭션 밖), `core.Start.Emit` (월드 트랜잭션 안) | `Load`, `Start` |
| 인스턴스 | `Join`, `Leave`, `NewDay`, `Second`, `Fast`, `Autosave`, `Close` | 이름이 같은 훅 |

세션 타입은 `core.Lifecycle` 을 품고 있어 이 호출들이 `Lifecycle` 의 메서드로 간다. 메서드마다 이벤트를 만들어 훅의 `Emit` 을 한 번 부르고, 아래 꼬리 일을 한다.

- `Leave`: 구독자가 끝나면 플레이어 클라이언트의 도형을 지우고(`RemoveAllDebugShapes`) 플레이어별 파트 항목을 지운다(`Session.Leave`).
- `Second`: 구독자가 끝나면 바뀐 상태를 저장 주기에 맞을 때만 쓴다(`FlushIfDue`).
- `Autosave`: 구독자가 끝나면 바뀐 상태를 쓴다(`Flush`).
- `Close`: 구독자가 끝나면 저장 파트를 쓰고 쓰기 고루틴을 멈추고(`Session.Close`), 세션 등록을 지운다(`Unregister`). 루트의 `session.Close` 는 그 뒤에 `Game` 의 세션 표에서도 지운다.
- `NewDay`: 구독자가 끝나면 모든 플레이어에게 `game.newDay` 를 알린다.

`Emit` 은 구독자를 티어 순서대로 부른다.

### 티어

구독자는 티어(`core.Priority`)에 선다. 티어는 여섯이고 표의 순서대로 돈다. 티어가 정하는 것은 역할뿐이다. 구독자는 다른 구독자의 앞이나 뒤가 아니라 자기가 하는 일에 맞는 티어를 고른다.

| 티어 | 상수 | 역할 |
|---|---|---|
| LOWEST | `core.Lowest` | 전처리 - 먼저 이벤트 자체를 가공하거나 차단해야 할 때 사용합니다. |
| LOW | `core.Low` | 전처리 이후 처리되어도 될 이벤트 가공이나 차단에 사용합니다. |
| NORMAL | `core.Normal` | 전처리 이후에 처리되어도 될 일반 우선순위에 사용합니다. |
| HIGH | `core.High` | 후처리에 해당하진 않지만, 우선순위가 낮은 작업에 사용합니다. |
| HIGHEST | `core.Highest` | 이벤트를 가공할 수 있는 마지막 후처리 작업에 사용합니다. |
| MONITOR | `core.Monitor` | 이벤트를 가공하거나 취소하지 않는 결과 처리(기록, 업적 처리 등)에 사용합니다. |

같은 훅의 같은 티어에 선 구독자 사이에는 순서가 없다. 실제로는 등록한 순서(기능 목록 `features.go` 의 순서, 한 기능 안에서는 `subscribe()` 가 단 순서)로 돌지만, 그 순서는 실행이 매번 같게 할 뿐이고 거기에 기대는 코드는 쓰지 않는다. 같은 티어의 두 구독자는 어느 쪽이 먼저 돌아도 결과가 같아야 한다. 앞뒤가 필요하면 역할에 맞게 티어를 나누거나, 구독자의 조건에 그 사실을 적거나, 한 구독자로 합친다. 순서를 지켜 주는 예외 목록은 없다. `TestTierOrder` 가 이 규칙을 지킨다(아래 "핀").

MONITOR 구독자는 이벤트를 가공하지도 취소하지도 않는다. MONITOR 구독자의 함수가 `Cancel()` 을 부르거나 이벤트의 포인터에 쓰면 `TestMonitorSubscribersDoNotShape` 가 실패한다. 함수가 부른 도우미 안에서 취소하면 장면을 도는 테스트에서 `Emit` 이 패닉한다(`core.Trace` 가 켜진 동안만 검사한다). MONITOR 구독자도 기본으로는 취소된 이벤트를 받지 않는다. 막힌 일은 일어나지 않은 일이라 세지 않는다. 취소된 이벤트도 봐야 하면 `core.HandleCancelled` 를 준다(다음 절).

수명 훅(`Join`, `Leave`, `Second`, `Fast`, `Autosave`, `Close` 따위)에는 가공하거나 취소할 것이 없다. 거기서는 티어를 일의 차례로 읽는다.

- LOWEST, LOW: 뒤의 일이 바꾸기 전에 상태를 읽어 두거나 먼저 갖춰 두는 일(`Second` 의 `progress.checkHeld`, `Join` 의 `screens.menuItem` 이 처음 들어온 플레이어의 고정 칸에 메뉴 아이템을 두는 일).
- NORMAL: 기능이 그때 하는 일.
- HIGH: 다른 기능의 일이 끝난 뒤에 해도 되는 일.
- HIGHEST: 마지막 정리.
- MONITOR: 결과를 보여 주거나 세는 일(사이드바, 패널, 업적).

틱 훅(`Second`, `Fast`)에서 같은 티어의 구독자는 서로의 이번 틱 결과를 읽지 않는다. 읽어야 하면 티어를 나눈다.

### 구독자와 취소

어느 구독자도 다른 구독자를 멈추지 못한다. 훅의 구독자는 티어 순서대로 모두 불린다. 구독자가 불리지 않는 경우는 하나뿐이다. 이벤트가 취소됐고, 그 구독자가 취소된 이벤트도 받겠다고 하지 않았을 때다.

- `On(티어, name, f)` 로 단 구독자는 취소된 이벤트를 받지 않는다. 이것이 기본이다.
- `On(티어, name, f, core.HandleCancelled)` 로 단 구독자는 취소된 이벤트도 받는다(전통적인 handleCancelled 옵션). 함수 모양은 같고, 취소됐는지는 `e.Cancelled()` 로 읽는다. 지금은 `Move` 의 `look.update` 가 쓴다. 섬 확장 모드가 이동을 취소해도, 섬이 공허에서 플레이어를 되돌려도 패널을 갱신한다.
- 취소는 이벤트로 한다. `e.Cancel()` 을 부르면 플레이어가 한 행동이 Dragonfly 에서 일어나지 않고, 뒤에 선 기본 구독자가 불리지 않는다. 취소된 이벤트는 Dragonfly 가 준 컨텍스트가 취소된 이벤트다. 그 밖의 뜻은 없고, 취소를 되돌릴 수도 없다.
- `Emit` 은 값을 돌려주지 않는다. Dragonfly 는 자기가 준 컨텍스트를 읽는다.

취소할 수 있는 이벤트는 `Actor`(`S`, `P`, `Tx`, `Ctx`)를 품은 이벤트이고, 그 훅은 `core.CancelHook` 이다. `Jump`, `Death`, `Respawn` 과 세션·월드 훅의 이벤트는 취소할 수 없다. 이 이벤트에는 `Cancel()` 과 `Cancelled()` 가 없고(`Jump`·`Death` 의 `NoticeEvent` 와 `RespawnEvent` 는 `Who` 만 품는다), 훅이 `core.Hook` 이라 `On` 이 옵션을 받지 않는다. 취소할 수 없는 이벤트를 취소하는 코드와 그런 훅에 handleCancelled 를 다는 코드는 컴파일되지 않는다. 이런 훅에서는 모든 구독자가 언제나 불린다.

구독자는 다음 둘 중 하나로 쓴다.

- 어떤 상태에서만 처리하고 취소한다. 자기 조건(기능의 상태, 든 아이템, 블록, 플레이어가 든 모드)이 맞을 때만 일하고 `e.Cancel()` 을 부른다. 바닐라 동작이나 뒤의 기본 구독자가 일어나면 안 될 때 쓴다. 든 복사기가 클릭을 쓰면 블록이 열리면 안 되고(`copier.use`), 섬 밖에는 짓지 못한다(`island.guard`).
- 기능은 처리하되 불필요하면 취소하지 않는다. 막을 것이 없으면 일만 하고 이벤트는 그대로 둔다. 다른 구독자도 계속 일한다. 엘리베이터는 웅크리면 내려가지만 웅크리기를 취소하지 않는다(`elevator.down`). 그래서 그 웅크리기도 세어지고 묘목도 자랄 기회를 얻는다.

"앞에서 아무도 가져가지 않았으니 내가 한다"는 조건은 쓰지 않는다. 구독자는 자기 조건으로 정한다. 다른 모드가 플레이어를 붙잡고 있을 때 하지 않는 일은 `core.Takeover(s, p) == ""` 로 적는다(`elevator.down`, `farm.crouchGrow`).

같은 티어에서 한 구독자가 취소하고 다른 기본 구독자가 같은 이벤트에 일할 수 있으면, 도는 순서에 따라 뒤 구독자가 불리기도 하고 불리지 않기도 한다. 그런 쌍은 조건이 겹치지 않게 적거나(든 아이템, 블록 종류, 웅크림, `island.CanBuild`) 역할에 맞게 다른 티어에 둔다.

### 훅과 블록·아이템의 메서드

클릭과 캐기는 훅으로만 오지 않는다. Dragonfly 는 핸들러(훅)를 먼저 부르고, 아무도 취소하지 않으면 눌린 블록에 묻고, 그다음 든 아이템에 묻는다.

| 플레이어가 한 일 | 훅 다음에 Dragonfly 가 부르는 것 |
|---|---|
| 블록을 누른다 | 눌린 블록의 `Activate`(`block.Activatable`). 웅크리고 아이템을 들었으면 건너뛴다. 블록이 false 를 돌려주면 든 아이템의 `UseOnBlock`(`item.UsableOnBlock`), 그것도 없으면 든 블록을 놓는다 |
| 공중을 누른다 | 든 아이템의 `Use`(`item.Usable`) |
| 블록을 친다 | 블록의 `Punch`(`block.Punchable`). 캐기를 막지는 못한다 |
| 블록을 다 캔다 | 블록을 지운 뒤 `BreakInfo().BreakHandler`. 폭발과 액체가 블록을 없앨 때도 부른다 |
| 블록이 바뀐다 | 블록의 `BlockChanged`(`world.BlockChangeHook`). 누가 바꿨든 `SetBlock` 마다 부른다 |

이 메서드로 할 수 있는 일은 구독자로 만들지 않는다. 블록이나 아이템이 이 저장소의 타입이고, 눌린 블록보다 먼저 클릭을 써야 할 까닭이 없으면 메서드에 둔다.

- 기계는 자기를 누른 클릭을 `Machine.Activate` 에서 받는다. 창을 열고, 든 등급 키트(`sgitem.TierKit`)와 요정의 모래시계를 쓴다. 웅크린 플레이어의 모래시계는 `sgitem.FairyHourglass` 의 `UseOnBlock` 이 받는다.
- 기계 아이템은 `Machine.UseOnBlock` 이 놓는다. 누른 면을 놓이는 블록에 실어 `machine.place` 에 넘기므로 자동 자원 채굴기와 저장망의 입력기·출력기가 누른 블록을 향한다. 도관 아이템과 `hopper.FlowHopper` 도 자기 `UseOnBlock` 으로 놓인다.
- 서랍을 친 일은 `Drawer.Punch` 가, 도관 묶음이 없어질 때의 정리는 `Bundle.BlockChanged` 가 한다.
- 공중에서 쓰는 아이템은 자기 `Use` 에서 일한다. 요정 판(`fairy.Board`)은 화면을 열고, 복사기는 비워지고, 건축 지팡이는 웅크리면 설정을 보이고, 철 망치는 웅크리면 대각선 광맥 캐기를 바꾸고, 이차원 텔레포트석은 이차원 창을 연다. 뒤의 넷은 타입이 `sgitem` 에 있어서 기능을 부를 수 없다. `sgitem/airuse.go` 가 아이템마다 함수 변수(`CopierUsed`, `WandUsed`, `HammerUsed`, `StoneUsed`)를 두고, 기능이 `Setup` 에서 채운다. 이 `Use` 는 언제나 false 를 돌려준다. true 를 돌려주면 Dragonfly 가 `Use` 전에 읽어 둔 묶음을 손에 다시 써서 `Use` 가 바꾼 손이 되돌아간다.
- `item.Usable` 인 커스텀 아이템은 클라이언트 항목에 `minecraft:use_modifiers` 를 받는다(`clientdef.ItemEntry`). 클라이언트는 쓸 수 있다고 아는 아이템만 공중 클릭을 보낸다.

구독자로 두는 경우는 넷이다.

- 가드: 어느 블록을 무엇으로 누르든 플레이어가 있는 곳이나 든 모드를 보고 막는 일(`island.guard`, `expand.guard`, `dimension.hunting`).
- 눌린 블록보다 먼저 클릭을 써야 하는 든 도구: 복사기와 건축 지팡이(LOW), 메뉴 아이템과 이차원 텔레포트석(LOWEST). Dragonfly 는 블록에 먼저 물으므로 아이템의 메서드로는 상자나 기계 위에서 블록이 먼저 열린다. 공중에는 먼저 물을 블록이 없어서 같은 아이템도 공중 클릭은 메서드로 받는다.
- 메서드를 달 수 없는 Dragonfly 의 블록과 아이템: 막대기, 도끼, 뼛가루, 작물, 퇴비통. 눌린 블록이 그 기능의 블록이 아닌 경우도 같다. 도관 판은 도관 옆 블록의 면에 그려져 있어서, 판을 누르면 기계나 상자가 눌린다(`conduit.plate`).
- 기록: 종류를 가리지 않고 세는 일(업적, 자동 채움).

메서드는 구독자가 모두 돈 뒤에 불린다. 그래서 가드가 막은 클릭은 메서드에 닿지 않는다. 섬 끝줄에 놓인 기계의 바깥쪽 면은 옆 칸 가드(`island.guardSide`)가 막으므로 거기서는 기계 창이 열리지 않고 모래시계도 쓰이지 않는다. 사냥꾼의 공중 클릭은 `dimension.hunting` 이 취소하므로 사냥꾼이 든 복사기, 지팡이, 망치, 텔레포트석은 아무 일도 하지 않는다. 섬 확장 모드에는 공중 클릭을 막는 가드가 없다.

장면 골든은 핸들러만 부르므로 메서드가 한 일을 보지 못한다. 메서드로 하는 일은 `Player.UseItemOnBlock`, `Player.BreakBlock` 처럼 Dragonfly 를 거쳐 누르고 캐는 테스트로 확인한다(`hourglass_test.go`, `machineface_test.go`, `airuse_test.go`).

### 이벤트의 값

이벤트는 값으로 건네므로 `Emit` 이 할당하지 않는다. 필드는 두 가지다.

- 이벤트를 만들 때 찍은 사본: `Held`, `Off`, `Block`, `Item`. 앞 구독자가 손이나 월드를 바꿔도 뒤 구독자는 옛 값을 본다. 새 값은 플레이어나 `Tx` 에 다시 물어야 한다.
- Dragonfly 가 핸들러에 준 포인터: `Drops`, `XP`, `Damage`, `Pos`(`RespawnEvent`), `Pick`(`ItemPickup` 에서 더해질 묶음). 포인터에 쓰면 Dragonfly 가 하는 일이 바뀐다.

`Sneak` 과 `Sprint` 는 시작할 때만 내보낸다.

## 훅 목록

훅은 35개이고 `content/skygear/core/events.go` 에 선언되어 있다. 플레이어 훅 23개와 세션·월드 훅 12개다.
"취소" 칸은 그 훅의 이벤트를 취소할 수 있는지를 나타낸다. "됨" 인 훅은 `core.CancelHook` 이고 나머지는 `core.Hook` 이다.

| 훅 | 내보내는 때 | 이벤트 | 취소 |
|---|---|---|---|
| `StartBreak` | 블록 캐기를 시작할 때 | `StartBreakEvent` | 됨 |
| `BlockBreak` | 블록을 캘 때 | `BlockBreakEvent` | 됨 |
| `BlockPlace` | 블록을 놓을 때 | `BlockPlaceEvent` | 됨 |
| `UseOnBlock` | 블록에 아이템을 쓸 때 | `UseOnBlockEvent` | 됨 |
| `UseOnEntity` | 엔티티에 아이템을 쓸 때 | `EntityEvent` | 됨 |
| `AttackEntity` | 엔티티를 때릴 때 | `EntityEvent` | 됨 |
| `Use` | 공중에서 아이템을 쓸 때 | `UseEvent` | 됨 |
| `Move` | 움직일 때 | `MoveEvent` | 됨 |
| `Sneak` | 웅크리기를 시작할 때 | `PlayerEvent` | 됨 |
| `Sprint` | 달리기를 시작할 때 | `PlayerEvent` | 됨 |
| `Jump` | 점프할 때 | `NoticeEvent` | 안 됨 |
| `ItemDamage` | 아이템이 내구도를 잃을 때 | `ItemDamageEvent` | 됨 |
| `ItemConsume` | 아이템을 먹거나 마실 때 | `ItemEvent` | 됨 |
| `ItemPickup` | 아이템을 주울 때 | `ItemEvent` | 됨 |
| `ItemDrop` | 아이템을 버릴 때 | `ItemEvent` | 됨 |
| `Death` | 죽을 때 | `NoticeEvent` | 안 됨 |
| `Respawn` | 되살아날 때 | `RespawnEvent` | 안 됨 |
| `Hurt` | 피해를 입기 전 | `HurtEvent` | 됨 |
| `PunchAir` | 허공을 칠 때 | `PlayerEvent` | 됨 |
| `Chat` | 채팅을 보낼 때 | `PlayerEvent` | 됨 |
| `BlockPick` | 블록을 집어 들 때 | `PlayerEvent` | 됨 |
| `HeldSlot` | 들고 있는 칸을 바꿀 때 | `PlayerEvent` | 됨 |
| `Command` | 명령어를 실행할 때 | `PlayerEvent` | 됨 |
| `Load` | 세이브의 첫 트랜잭션 앞(`Tx` 는 nil) | `Tick` | 안 됨 |
| `Start` | 세이브의 첫 트랜잭션 안 | `Tick` | 안 됨 |
| `Join` | 플레이어가 입장한 뒤 | `Presence` | 안 됨 |
| `Leave` | 플레이어가 나가기 전 | `Presence` | 안 됨 |
| `NewDay` | 인스턴스 날짜가 바뀔 때 | `DayEvent` | 안 됨 |
| `Second` | 약 1초마다 | `Tick` | 안 됨 |
| `Fast` | 0.2초마다 | `Tick` | 안 됨 |
| `Autosave` | 자동 저장 주기마다 | `Tick` | 안 됨 |
| `Close` | 세이브를 닫을 때 | `Tick` | 안 됨 |
| `ColumnLoad` | 청크 칼럼이 월드에 더해질 때 | `ColumnEvent` | 안 됨 |
| `ColumnUnload` | 청크 칼럼이 월드에서 빠질 때 | `ColumnEvent` | 안 됨 |
| `EntityDespawn` | 엔티티가 월드를 떠날 때 | `DespawnEvent` | 안 됨 |

`PlayerEvent` 는 인자가 없는 취소할 수 있는 이벤트이고 `Actor`(`S`, `P`, `Tx`, `Ctx`)만 든다. `NoticeEvent` 는 Dragonfly 가 알리기만 하는 이벤트(`Jump`, `Death`)이고 `Who`(`S`, `P`, `Tx`)만 든다. 위치와 방향, 대상 같은 인자는 이벤트 종류마다 `core/events.go` 에서 본다.
이벤트에 없는 Dragonfly 인자(채팅 글, 사망의 `keepInv`, 공격의 힘과 치명타 따위)는 구독자가 받지 못한다.

## 새 기능 더하기

엘리베이터(`content/skygear/elevator`)가 가장 작은 예다. `feature.go`, `subscribe.go`, `creative.go` 세 파일로 서술자, 구독자, 창작 목록 항목을 둔다.

1. `content/skygear/<기능>/` 패키지를 만든다. 기능의 규칙과 블록·아이템은 이 패키지 안에 둔다. 블록과 아이템은 패키지의 `init` 에서 `world.RegisterBlock`, `world.RegisterItem` 으로 등록하고(`elevator/elevator.go`), 이름은 리소스팩의 `assets/resourcepack/skygear/texts/ko_KR.lang`, `en_US.lang` 에 둔다(`pack_test.go` 가 검사한다).
2. `feature.go` 에 서술자를 둔다. 필드는 `Name`, `Entities`(기능이 더하는 엔티티 종류), `Settings`(플레이어 설정), `ItemEntry`(커스텀 아이템 항목 고치기), `Creative`(창작 목록 항목), `Commands`(명령어), `Setup`(구독자 달기), `Check`(다른 기능이 내놓은 것 검사)이다. `Check` 는 `Install` 의 끝에서 모든 기능의 `Setup` 이 돈 뒤에 기능 목록 순서대로 돌고, 오류를 돌려주면 `Install` 이 그 오류로 패닉한다. `Name` 은 패키지 이름과 같아야 한다. 쓰지 않는 필드는 비워 둔다.

   ```go
   package elevator

   import "github.com/presentkim/skygear/content/skygear/core"

   // Feature is the elevator feature, listed in the game's feature list.
   var Feature = &core.Feature{Name: "elevator", Creative: creativeItems, Setup: subscribe}
   ```

3. `subscribe.go` 에서 `Setup` 이 부르는 `subscribe()` 가 구독자를 단다. 이름은 `<기능>.<단계>` 이고 접두어가 기능 이름과 같아야 한다.

   ```go
   func subscribe() {
       subscribeJump()
       subscribeSneak()
   }

   func subscribeJump() {
       core.Jump.On(core.Normal, "elevator.up", func(e core.NoticeEvent) {
           Up(e.S, e.P, e.Tx)
       })
   }

   func subscribeSneak() {
       core.Sneak.On(core.Normal, "elevator.down", func(e core.PlayerEvent) {
           if core.Takeover(e.S, e.P) == "" {
               Down(e.S, e.P, e.Tx)
           }
       })
   }
   ```

4. 기능의 블록이 WAILA 패널에 줄을 더하면 `look.go` 에 그 블록의 `Waila(v, info)` 메서드를 둔다(`waila.Provider`). `info` 에는 `look` 이 채운 이름, 아이콘, 도구, 바닐라 보관 블록의 내용이 들어 있고, 메서드는 고친 `info` 를 돌려준다. 예는 `hopper/look.go` 다.

   ```go
   func (h FlowHopper) Waila(v waila.View, info waila.Info) waila.Info {
       info.Lines = HopperLines(v.Lang, h)
       if h.Inv != nil {
           info.Container = &waila.Container{Layout: waila.LayoutChest, Items: h.Inv.Items()}
       }
       return info
   }
   ```

   세션의 데이터가 필요하면 `core.At(v.Tx)` 로 세션을 찾는다(nil 이면 `info` 를 그대로 돌려준다). 기능이 소유하지 않은 블록(바닐라 퇴비통, 작물, 화로)의 패널은 `Setup` 에서 `waila.RegisterAdapter` 로 단 어댑터가 채운다. 예는 `farm/look.go` 의 `registerLook` 이고 `subscribe()` 가 부른다.

   ```go
   func registerLook() {
       waila.RegisterAdapter(func(v waila.View, b world.Block, info waila.Info) waila.Info {
           if c, ok := b.(block.Composter); ok {
               if s := core.At(v.Tx); s != nil {
                   info.Lines = ComposterLines(s, v.Lang, v.Pos, c)
               }
           }
           // ...
           return info
       })
   }
   ```

   어댑터는 다른 블록에는 `info` 를 그대로 돌려줘야 한다. `look` 에 블록 종류를 더하지 않는다. `look` 이 기능 패키지를 import 하면 `TestCapabilityImports` 가 실패한다. 패널이 달라지면 `TestPinLook`(`look.golden`)이 어긋난 장면을 보여 준다.
5. 기능이 기계 종류를 소유하면 종류의 정체는 `machine` 에, 동작은 기능에 둔다.
   - 종류의 이름, id, `Order` 의 자리, `Def` 는 다른 종류처럼 `machine/machine.go` 에 적고 `Def` 에 `Owner: "<기능>"` 을 쓴다. 블록과 아이템은 `machine` 의 `init` 이 등록하고, 클라이언트가 받는 아이템 id 가 그 순서를 따른다.
   - 동작은 기능의 `kinds.go` 에 둔다. `registerKinds()` 가 종류마다 `machine.Register(kind, machine.Behaviour{…})` 를 한 번 부르고, `subscribe()` 가 맨 끝에서 `registerKinds()` 를 부른다. 예는 `dimension/kinds.go` 다.

   ```go
   func registerKinds() {
       machine.Register(machine.KindRelicTable, machine.Behaviour{
           Click: func(s *core.Session, p *player.Player, tx *world.Tx, pos cube.Pos) bool {
               if !s.Debounced(p) {
                   ShowRelicTable(s, p, "")
               }
               return true
           },
           Lines: machine.NoLines,
       })
   }
   ```

   `machine.Behaviour` 의 필드는 모두 비워 둘 수 있다. 비운 필드는 `machine` 이 자기 종류에 하는 대로 한다.

   | 필드 | 하는 일 |
   |---|---|
   | `Click` | 놓인 기계를 클릭하면 기계 창 대신 이 일을 한다. 클릭을 맡았으면 true 를 돌려준다 |
   | `Run` | 기계 틱 한 번의 일 |
   | `Insert` | 호퍼와 아이템 도관이 블록에 넣는 일 |
   | `Peek` | 호퍼와 아이템 도관이 꺼낼 아이템을 미리 보는 일 |
   | `Extract` | 호퍼와 아이템 도관이 블록에서 꺼내는 일 |
   | `Accepts` | 입력 칸이 받는 아이템(`slot` 이 -1 이면 아무 입력 칸). 기계의 규칙 대신 쓴다 |
   | `Lines` | 패널 줄. 기계의 줄 대신 쓰고, 줄이 없는 종류는 `machine.NoLines` 를 쓴다 |
   | `Info` | 기계 창 정보 아이콘에 설명 뒤로 더하는 줄 |

   종류가 어떤 모습인지는 동작이 아니라 `Def` 의 플래그로 적고, 기계 틀이 읽는다.

   | 플래그 | 뜻 |
   |---|---|
   | `NoDisplay` | 계속 갱신할 기계 창이 없다. 틱이 창의 진행, 에너지 게이지, 정보 아이콘을 갱신하기 전에 끝난다 |
   | `NoWindow` | 클릭해도 아무 일이 없고, 클릭은 든 아이템으로 넘어간다(저장망 케이블) |
   | `Cycle` | 일이 있든 없든 주기가 돈다. 창과 패널이 입력 대신 주기를 보인다(수확기) |

   - `content/skygear/kinds_test.go` 의 `kindOwners` 에 종류와 기능을 한 줄 더한다. `TestKindOwners` 가 이 표와 `Def.Owner` 를 맞춰 본다.
   - 소유한 기능이 등록하지 않으면 게임이 시작하지 않는다. 기계 기능의 `Check` 인 `machine.CheckKinds` 가 `machine kinds without behaviour: …` 오류를 돌려주고 `Install` 이 그 오류로 패닉한다. 소유 기능이 기능 목록에 없을 때도 같다.
   - 기능 패키지의 테스트는 기능 목록 없이 돈다. 종류의 동작이 필요한 테스트는 테스트 파일의 `init` 에서 `coretest.Setup(<기능>.Feature)` 를 부른다(`netstore/keys_test.go`, 아래 "기능의 테스트").

   `machine` 은 기능 칸의 패키지를 하나도 import 하지 않는다. 종류를 소유한 기능이 `machine` 을 import 하므로 그 기능의 패키지는 `machine` 보다 나중에 초기화된다. 그래서 그 기능이 `init` 에서 등록하는 아이템은 기계 아이템보다 뒤의 id 를 받는다. import 를 더하거나 방향을 바꿔 초기화 순서가 달라지면 커스텀 아이템의 id 가 바뀐다(아래 "핀"의 "id 만 바뀌는 변경").
6. 기능의 아이템에 등급이나 단계 표시가 있으면 그 아이템에 `Decor()` 메서드를 둔다(`slotdeco.Decorated`). 예는 `fairy/fairymagic.go` 의 요정 판이다.

   ```go
   // Decor is the board's slot decoration (slotdeco.Decorated).
   func (Board) Decor() (slotdeco.Rarity, int) { return slotdeco.Rare, 0 }
   ```

   여러 기능이 같이 쓰는 아이템의 `Decor()` 는 `sgitem/decor.go` 에 있다. `decor.ItemDecor` 가 이 메서드를 읽는다. `TestItemDecor` 에 아이템의 등급과 단계를 한 줄 더한다. 언어 파일의 이름 표식은 `go run ./tools/slotmarks assets/resourcepack/skygear` 로 다시 쓰고, 빠지면 `TestPackHasItemAssets` 가 실패한다.
7. 기능의 아이템과 블록을 창작 목록에 올리려면 `creative.go` 에 `creativeItems` 를 두고 서술자에 `Creative: creativeItems` 를 쓴다. 클라이언트 레시피 책은 창작 목록에 있는 결과만 보이므로 만들 수 있는 아이템은 모두 올린다(`TestEveryGameRecipeResultIsCreative`). 예는 `elevator/creative.go` 다.

   ```go
   func creativeItems() []core.CreativeItem {
       var out []core.CreativeItem
       for _, c := range item.Colours() {
           out = append(out, core.CreativeItem{Slot: 70, Category: creative.ConstructionCategory(), Item: Elevator{Colour: c}})
       }
       return out
   }
   ```

   `Slot` 이 목록의 자리다. 항목은 `Slot` 순서로 놓이고, 같은 `Slot` 안에서는 기능이 돌려준 순서를 지킨다(`core.CreativeItems`). 지금 자리는 10 `progress`, 20 `storage`, 30 `farm`, 40 `conduit`, 50 `machine`, 60 `catalog`, 70 `elevator`, 80 `catalog`, 90 `farm`, 100 `catalog`, 110 `netstore`, 120 `catalog`, 130 `fairy`, 140 `catalog`, 150 `farm`, 160 `catalog` 이다. 새 기능은 앞뒤에 올 이웃 사이의 빈 수를 쓴다(75 등). 한 `Slot` 을 두 주인이 쓰면 게임이 시작할 때 창작 목록을 만들며 패닉한다. `catalog` 자리는 기능이 아닌 패키지(`sgitem`, `hopper`)의 항목이고 `catalog/creative.go` 의 `ownCreative` 에 있다. 기능의 자리는 `TestCreativeListNamesNoFeature` 의 `want` 에도 더한다. 창작 목록이 바뀌므로 `creative.golden`(`TestPinCreative`)과 `content/skygear/wire/testdata/creative.golden`(`TestWire`)을 의도한 변경으로 다시 기록한다(아래 "핀").
8. 기능의 명령어는 `commands.go` 의 `commands()` 가 돌려주고 서술자에 `Commands: commands` 를 쓴다. 게임은 `skygear.New` 에서 `core.Commands()` 로 모든 기능의 명령어를 기능 목록 순서대로 받아 등록한다. 예는 `farm/commands.go` 다.

   ```go
   func commands() []cmd.Command {
       return []cmd.Command{
           cmd.New("묘목", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{}),
           cmd.New("sapling", i18n.T(i18n.Default, "cmd.sapling"), nil, saplingCmd{}),
       }
   }
   ```

   플레이어와 세션이 필요한 명령은 `Run` 의 처음에 `core.PlayerSession(src, o, tx)` 를 부른다. 명령을 보낸 쪽이 플레이어가 아니면 조용히, 플레이어가 세이브 밖에 있으면 세이브에 들어간 뒤 쓸 수 있다고 알리고 false 를 돌려준다(`machine/commands.go`, `screens/commands.go`). 한국어 명령 이름이 든 파일은 `content/skygear/locales/locales_test.go` 의 `TestNoKoreanInCode` 허용 목록에 더한다. 명령 이름은 `TestFeatureCommands` 의 `want` 에 기능 목록 순서대로 더한다.
9. 기능이 패널을 숨기거나 패널 갱신을 따라가야 하면 `look` 의 두 자리를 쓴다. `look.Hidden` 은 패널을 숨길지 정하는 함수 하나이고 `expand` 가 `subscribe()` 에서 채운다(섬 확장 모드). `look.Follow(name, f)` 는 패널을 갱신할 때마다 `f` 를 부르고, 단 순서대로 돈다. 이름은 `<기능>.<단계>` 이고 `subscribe()` 의 끝에서 단다. 예는 `wand/subscribe.go` 다.

   ```go
   func followLook() {
       look.Follow("wand.preview", func(t look.Target) {
           ShowPreview(t.S, t.P, t.Tx)
       })
   }
   ```

   따라가는 단계를 더하면 `TestLookFollowers` 의 목록에 이름을 더한다.
10. 플레이어 설정은 서술자의 `Settings` 에 둔다. `Install` 이 기능 목록 순서대로 등록한다. 예는 `autofill/feature.go` 다.

   ```go
   Settings: []account.Setting{
       {Key: Setting, NameKey: "setting.autofill", Default: true, Group: "setting.group.play"},
   },
   ```

   `NameKey` 의 글은 `locales` 에 둔다. 설정을 더하면 `settings.golden`(`TestPinSettings`)이 달라지므로 의도한 변경으로 다시 기록한다.
11. 커스텀 아이템 항목을 고치는 함수는 서술자의 `ItemEntry` 에 둔다(아래 "클라이언트 항목 고치기"). 지금은 루트의 `itemsFeature`(`features.go`)만 쓴다. 클라이언트가 받는 항목이 바뀌므로 `content/skygear/wire/testdata/items.golden`(`TestWire`)이 달라진다.
12. 루트의 `content/skygear/features.go` 의 `Features` 에 `<기능>.Feature` 를 한 줄 더하고 import 를 더한다. 설정, 아이템 항목 고치기, 창작 항목, 명령어는 목록 순서대로 모이고 `Setup` 도 그 순서대로 돈다.
13. `content/skygear/layers_test.go` 에 기능을 알린다. `layers` 에서 칸을 정하고(자기보다 아래 칸에 든 패키지만 import 할 수 있다), 기능 칸이면 `featureLayer` 에도 이름을 더한다(`featureLayer` 가 `layers` 의 기능 칸과 다르면 `TestFeatureSiblings` 가 실패한다). 같은 칸의 기능을 import 하면 `siblings` 에 행을 더한다. 기능의 순서는 `TestFeatureNames` 의 `want` 목록에도 적혀 있으니 `Features` 에 더한 자리에 같이 더한다.
14. 구독자를 달았으면 골든이 달라진다. `order.golden` 에는 새 줄(`<훅> <티어> <이름>`)이 생기고, 그 훅을 거치는 장면이 있으면 `handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden`, `lifecycle.golden` 의 그 장면 줄에도 새 이름이 들어간다. 의도한 변경이므로 `SKYGEAR_GOLDEN=update go test ./content/skygear -run TestPin -count=1` 로 다시 기록하고 `git diff` 로 새 단계의 이름만 늘었는지 확인한다. 커밋 메시지에 어느 단계를 더했는지 적는다.
15. `go test ./content/skygear -run 'TestLayers|TestFeature|TestCapability|TestLook|TestCreative|TestItemDecor|TestEvery|TestRoot|TestPin|TestHandlerMethodsOneWay|TestKind|TestMachine' -count=1` 로 확인한다. 위 순서를 빠뜨리면 이 테스트들이 `TestLayers`(`package <기능> is not in layers`), `TestFeatureNames`, `TestPinOrder`, `TestEverySubscriberHasItsFeature`, `TestPinSimple`, `TestCreativeListNamesNoFeature`, `TestFeatureCommands` 따위로 어긋난 곳을 알려 준다.

### 티어 고르기

이름은 한 훅 안에서 겹치면 안 된다(훅이 다르면 같아도 된다). 겹치면 `Install` 이 패닉한다. 티어는 겹쳐도 된다.
먼저 구독자로 할 일인지 본다. 블록이나 아이템의 메서드로 할 수 있으면 구독자를 달지 않는다(위 "훅과 블록·아이템의 메서드").
다른 구독자의 자리를 보고 고르지 않는다. 내 구독자가 하는 일을 보고 고른다.

1. 이벤트를 가공하지도 취소하지도 않고 결과를 세거나 보여 주기만 한다: MONITOR. 취소된 이벤트도 봐야 하면 `core.HandleCancelled` 를 준다.
2. 플레이어를 붙잡은 모드처럼 무엇보다 먼저 막아야 한다: LOWEST. 다른 구독자가 읽을 값을 고치거나 그 뒤에 막아도 된다: LOW. 든 도구가 블록보다 먼저 클릭을 쓰는 일도 LOW 다.
3. 다른 구독자가 다 고친 뒤의 값을 보고 마지막으로 가공하거나 취소한다: HIGHEST.
4. 다른 구독자의 일이 끝난 뒤의 상태를 읽어서 하는 일: HIGH.
5. 그 밖의 일: NORMAL.

그다음 취소할지 정한다. 내 일 때문에 바닐라 동작이나 뒤의 구독자가 일어나면 안 되면, 조건이 맞을 때만 일하고 취소한다. 그렇지 않으면 취소하지 않는다.

마지막으로 같은 티어에 이미 있는 구독자와 겹치는지 본다. 한쪽이 취소하고 다른 쪽이 같은 이벤트에 일할 수 있으면 조건이 겹치지 않게 적는다(든 아이템, 블록 종류, 웅크림, `island.CanBuild`, `core.Takeover`). 조건으로 가를 수 없으면 티어를 나눈다. 구독자를 달고 `go test ./content/skygear -run TestTierOrder -count=1` 을 돌린다.

### 순서를 지키는 것

지금 순서가 필요한 곳과 그것을 지키는 방법이다. 같은 티어 안의 자리로 지키는 곳은 없고, 역할에 맞지 않는 티어에 둔 구독자도 없다.

티어가 지키는 것:

- `StartBreak`, `BlockBreak`, `BlockPlace`, `UseOnBlock`: `expand.guard`(LOWEST)가 `island.guard`(LOW)보다 먼저다. 확장 모드에서 섬 밖을 누르면 확장 모드가 막고 지도를 다시 보여 준다. 월드를 바꾸는 단계는 모두 NORMAL 이후다. 가드는 조건이 맞으면 취소하고, 취소된 이벤트는 뒤의 기본 구독자에게 가지 않는다.
- `BlockBreak`: `storage.front`(LOW)가 서랍 앞면을 친 캐기를 취소한다. 취소된 캐기는 세지 않고 도구 경험치도 광맥 캐기도 없다.
- `UseOnBlock`: 든 도구가 클릭을 쓰는 단계(`copier.use`, `wand.use`, LOW)가 블록에 하는 일(NORMAL, HIGH)보다 먼저다. 든 아이템을 쓰는 단계(`handwork.stick`, `fairy.time`, NORMAL)가 블록 자체의 동작(`farm.composter`, `farm.crop`, HIGH)보다 먼저다. 옆 칸 가드(`expand.guardSide`, `island.guardSide`, HIGHEST)는 아무도 취소하지 않았을 때 마지막에 막는다. `handwork.stickOther`(MONITOR)는 취소되지 않고 끝까지 간 클릭만 받는다.
- `UseOnBlock` 의 구독자가 모두 돈 뒤에 Dragonfly 가 눌린 블록의 `Activate` 를, 그다음 든 아이템의 `UseOnBlock` 을 부른다. 서서 도관 판을 누르면 `conduit.plate` 가 취소하므로 판 뒤의 기계는 열리지 않고, 든 등급 키트와 요정의 모래시계도 쓰이지 않는다. 웅크리면 판이 클릭을 쓰지 않아 모래시계가 기계를 감는다.
- `Sneak`: `expand.show`, `elevator.down`(NORMAL), `farm.crouchGrow`(HIGH), `achievement.sneak`(MONITOR) 순서다. 웅크리기는 아무도 취소하지 않으므로 모두 불린다. 엘리베이터가 플레이어를 옮긴 뒤에 `farm.crouchGrow` 가 플레이어 둘레의 묘목을 본다.
- `Hurt`: `dimension.frozen`(LOWEST)이 강화를 고르는 사냥꾼의 피해를 취소한다. 사냥꾼을 죽이는 타격은 막지 않는다. 사냥꾼은 실제로 죽고 `achievement.hurt` 가 그 타격을 센다. `Move` 의 `island.void`(HIGHEST)는 마지막에 취소한다. 취소된 피해와 이동은 `achievement.hurt`, `achievement.walk`(MONITOR)가 세지 않는다.
- `Death`, `Respawn`: 사냥꾼이 죽으면 `dimension.died`(NORMAL)가 판을 실패로 끝낸다. 맡겨 둔 소지품과 보상이 그때 돌아오고 `achievement.death`(MONITOR)가 죽음을 센다. 부활할 때 `dimension.respawned`(LOW)가 부활할 자리를 들어왔던 자리로 고친 뒤에 `tinker.restoreFood`(NORMAL)가 허기를 되돌린다. 인벤토리는 플랫폼의 핸들러가 모든 죽음에서 지킨다(`platform/instance` 의 `HandleDeath`).
- `ItemDamage`: `tinker.ironHammer`(LOW)가 피해를 0 으로 고친 뒤에 `autofill.broken`(HIGH)이 읽는다.
- `BlockPlace`: `netstore.place`(HIGH)는 `machine.place`(NORMAL)가 블록 설치를 미뤄 둔 뒤에 망을 버린다.
- `Leave`: 기능이 퇴장 때 하는 일(`dimension.leave`, `expand.stop`, NORMAL)이 클라이언트에 남은 것을 지우는 정리(`look.clear`, `screens.board`, `progress.recipes`, HIGHEST)보다 먼저다. 확장 모드가 끝나면 사이드바를 다시 그리고, 사냥이 끝나면 레시피를 다시 보낼 수 있다.
- `Join`: `screens.menuItem`(LOWEST)이 처음 들어온 플레이어의 8번 칸에 메뉴 아이템을 둔 뒤에 아이템을 주는 입장 단계(`fairy.backfillDust`, NORMAL)가 돈다. 쌓아 둔 요정 가루가 여덟 묶음을 넘어도 메뉴 아이템이 가루를 덮어쓰지 않는다. `dimension.joined`(LOW)도 그 단계보다 먼저다. 서버가 멈출 때 이차원에 있던 플레이어는 맡겨 둔 소지품을 먼저 돌려받고, 가루는 돌려받은 인벤토리에 들어간다.
- `Second`: `progress.checkHeld`(LOW)가 플레이어가 든 아이템을 읽어 파트에 둔 뒤에 `dimension.second`(NORMAL)가 `RetireItems` 로 아이템을 정리한다. `progress.collection`(MONITOR)이 그 파트를 읽는다.

구독자의 조건이 지키는 것:

- 사냥 중인 플레이어가 블록을 누르면 무엇을 들었든 `dimension.hunting` 이 취소한다. `screens.menuItem`(`UseOnBlock`)과 `dimension.stone` 은 사냥 중이면 클릭을 쓰지 않는다. 공중에서는 메뉴 아이템이 사냥에서 나가는 길이라 `dimension.hunting`(`Use`)이 메뉴 아이템을 든 클릭을 취소하지 않는다.
- 메뉴 아이템과 이차원석은 어디서나 자기 화면을 연다. `expand.guard`(`UseOnBlock`)는 그런 아이템(`sgitem.OpensScreen`)을 든 클릭을 취소하지 않는다.
- `copier.use`, `wand.use` 는 섬이 짓기를 허락하는 곳(`island.CanBuild`)에서만 클릭을 쓴다. 섬 밖에서는 `island.guard` 가 취소한다. `storage.front` 도 같은 조건을 단다.
- `UseOnBlock` 의 NORMAL 구독자는 블록과 든 아이템과 웅크림으로 갈린다. `conduit.plate` 는 서서 누른 도관 판, `storage.sneakUse` 는 웅크리고 누른 서랍, `handwork.stick` 은 막대기로 누른 흙·잔디와 웅크리고 누른 상자, `handwork.gather` 는 도끼로 누른 원목과 괭이로 누른 흙, `fairy.time` 은 요정의 뼛가루로 누른 작물·묘목·마법 식물과 웅크리고 뼛가루로 누른 작물이다. 도관 판은 아이템이나 에너지, 액체를 주고받는 블록에만 붙으므로 `conduit.plate` 와 `fairy.time` 이 같은 클릭에 일하는 때는 없다.
- `Sneak` 의 `expand.show` 는 섬 확장 모드에서만 화면을 다시 보여 준다. `elevator.down` 과 `farm.crouchGrow` 는 다른 모드가 플레이어를 붙잡고 있으면(`core.Takeover`) 일하지 않는다.
- `handwork.stickOther` 는 막대기로 흙도 잔디도 아닌 블록을 눌렀고 아무도 그 클릭을 취소하지 않았을 때 `stick_other` 를 센다. 서랍 메뉴, 도관 판, 작물, 가득 찬 퇴비통이 쓴 클릭은 세지 않는다.

커널이 지키는 것:

- `Leave`: 플레이어 클라이언트의 도형 지우기와 `Players` 항목 지우기는 구독자가 아니다. `Lifecycle.Leave` 가 모든 구독자 뒤에 한다. 퇴장 구독자는 어느 티어에 서든 플레이어별 데이터를 읽을 수 있다.
- `Second`, `Autosave`, `Close`, `NewDay` 의 꼬리 일도 구독자 뒤다. `Close` 의 구독자는 세션이 아직 등록된 상태에서 돈다.
- `Move`: 이벤트의 `From` 은 이벤트를 만들 때 플레이어가 서 있던 곳이다. 앞의 구독자가 플레이어를 옮겨도 `achievement.walk` 는 같은 거리를 센다.
- 커널의 구독자는 `Join` 의 `core.shapes` 하나다(NORMAL). 다른 구독자와 같은 규칙을 따른다.

## 기능의 데이터

기능은 `Session` 에 필드를 더하지 않고 `core` 의 파트를 패키지 변수로 둔다.

- `core.NewPart[T](mk)`: 세션마다 하나, 처음 `Of(s)` 를 부를 때 만든다. 저장하지 않는다. 예: `progress` 의 `held`.
- `core.NewPlayers[T]()`: 세션 안에서 플레이어마다 하나. `Of(s)` 는 `map[*world.EntityHandle]T` 를 돌려준다. 플레이어가 나갈 때 `Lifecycle.Leave` 가 구독자가 모두 돈 뒤에 항목을 지운다(`Session.Leave`). 예: `tinker` 의 `keptFood`.
- `core.NewSaved[T](mk)`: 저장하는 데이터. 모든 `Saved` 파트는 세이브 상태 문서(`skygear.state`) 하나의 JSON 필드이고, 필드 이름은 파트끼리 겹치지 않는다.

```go
var keptFood = core.NewPlayers[int]()

core.Death.On(core.Normal, "tinker.keepFood", func(e core.PlayerEvent) {
    keptFood.Of(e.S)[e.P.H()] = e.P.Food()
})
```

### 플레이어를 붙잡는 모드

한 기능이 플레이어의 게임 모드, 카메라, 몸을 맡아 두었다가 모드가 끝날 때 돌려주는 모드를 붙잡는 모드(takeover)라 한다. 섬 확장 모드(`expand`), 이차원 사냥(`dimension.hunt`), 이차원 사냥 관전(`dimension.watch`)이 그렇다. 둘은 겹치지 않는다. 저마다 시작할 때 돌려줄 것을 저장해 두므로, 겹치면 끝날 때 엉뚱한 모드로 돌아간다.

- 기능은 `Setup` 에서 `core.RegisterTakeover(name, holds)` 로 자기 모드를 등록한다. 이름은 `"<기능>"` 또는 `"<기능>.<모드>"` 이고, `holds` 는 플레이어가 그 모드에 있는지 알려 준다. 빈 이름, 겹치는 이름, nil `holds` 는 패닉한다.
- 모드를 시작하기 전에 `core.Takeover(s, p)` 를 묻는다. 다른 기능의 모드 이름이 돌아오면 시작하지 않고 플레이어에게 알리기만 한다. 여럿이 붙잡고 있으면 먼저 등록된 이름이 돌아온다.
- 등록은 `Setup` 에서만 하고 묻기는 그 뒤에 하므로, `Takeover` 는 잠금 없이 읽는다.

## 기능의 테스트

기능의 테스트는 기능 패키지 안에 둔다. 파일의 패키지 이름은 `<기능>_test` 다(바깥 테스트 패키지). 바깥 테스트 패키지는 그 기능을 import 하는 패키지도 import 할 수 있어서 순환이 생기지 않는다. 한 패키지만 돌려 볼 수 있다: `go test ./content/skygear/farm`.

기능 패키지의 테스트 바이너리에는 루트 패키지가 없으므로 기능 목록이 설치되지 않는다(`core.Install` 은 루트의 `init` 에서만 돈다). 구독자도, 설정도, 다른 기능이 등록하는 기계 종류의 동작도 없는 상태다. 테스트에 필요한 것은 지원 패키지 둘이 준다.

- `content/skygear/core/coretest`
  - `coretest.Setup(fs ...*core.Feature)`: 블록 등록을 닫고, 이 바이너리에서 아직 준비하지 않은 기능마다 설정을 등록하고 `Setup` 을 부른다. `Check` 는 부르지 않는다. 이미 준비한 기능과 `Install` 로 설치된 기능은 건너뛰므로 같은 기능을 두 번 적어도 된다. 훅은 첫 이벤트 뒤에 구독자를 받지 않으므로 테스트 함수가 아니라 테스트 파일의 `init` 에서 부른다.
  - `coretest.Session(t)`: 동기 월드 위의 세션을 `core.Lifecycle` 로 돌려준다. `s.Session` 이 코어 세션이고, `s.Join`·`s.Second`·`s.Fast` 같은 메서드가 인스턴스가 하는 호출이며, `s.Handler(p)` 가 플레이어 핸들러다. `core.Of` 가 이 세션을 찾고, 월드의 핸들러는 `core.WorldHandler` 다. 데이터베이스는 없다.
  - `coretest.LoadAround(s)`, `coretest.SpawnPlayer(s)`, `coretest.WithPlayer(t, s, h, f)`, `coretest.Tick(s, n)`, `coretest.ShapeAs(s, key)`: 청크 읽기, 플레이어 넣기, 플레이어와 함께 트랜잭션 돌리기, 틱 진행, 세션이 둔 디버그 도형 읽기.
- `content/skygear/machine/machinetest`: `machinetest.Place(s, pos, kind)` 가 기계를 놓고 상태를 돌려준다(에너지가 필요 없고, 체에는 실 메쉬가 끼워져 있다). `machinetest.Pos` 는 테스트가 기계를 놓는 자리다. `machine` 을 import 하므로 `machine` 패키지에서는 `machine_test` 패키지의 파일만 쓸 수 있다.

테스트가 세션을 만드는 패키지에는 `setup_test.go` 가 있고, 그 패키지의 테스트에 필요한 기능을 준비한다. 기본은 자기 기능 하나다.

```go
package farm_test

import (
	"github.com/presentkim/skygear/content/skygear/core/coretest"
	"github.com/presentkim/skygear/content/skygear/farm"
)

func init() { coretest.Setup(farm.Feature) }
```

자기 기능 말고 더 준비하는 곳은 셋이다. `conduit` 는 연결망 단계가 돌도록 `network` 를 같이 준비한다. `netstore` 는 `setup_test.go` 없이 `keys_test.go` 의 `init` 이 자기 기능과 `storage` 를 준비한다(외부 저장소가 서랍 컨트롤러를 읽는다). `machine` 은 `netkinds_test.go` 의 `init` 이 저장망 종류의 동작 때문에 `netstore` 를 더 준비한다.

루트 패키지(`content/skygear`)에는 조립된 게임을 보는 테스트를 둔다.

- 핀과 골든(`pin_*_test.go`), 구조 규칙(`layers_test.go`, `kinds_test.go`, `testsupport_test.go` 등), 벤치마크.
- `catalog` 의 레시피와 연구, 도감, 리소스팩, 세이브(저장, 다시 열기, 마이그레이션)를 보는 테스트.
- 기능 셋 이상이 설치되어야 통과하는 테스트, 자기가 import 하지 않는 기능이 등록한 아이템이 필요한 테스트.
- 루트의 내보내지 않은 이름(`Game`, `session`, `giveItems` 등)을 쓰는 테스트와 런타임 id 를 보는 테스트.

루트의 `helpers_test.go` 는 같은 도우미를 루트의 `session` 위에서 준다(`testSession`, `loadAround`, `spawnPlayer`, `withPlayer`, `tick`, `placeTestMachine`, `machinePos`). `testSession` 은 데이터베이스와 `Game` 도 만든다.

지킬 것이 셋 있다.

- 패키지의 테스트는 루트 패키지를 import 하지 않고, 내용 목록 아래 칸의 패키지 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다(`TestPackageTestsStayBelow`).
- 테스트 지원 패키지는 테스트 파일만 import 한다(`TestSupportPackagesAreTestOnly`).
- `dimension` 의 `popup_wire_test.go` 는 서버에 접속해 데미지 숫자의 패킷을 `testdata/popup.golden` 에 고정한다. 그 패킷에는 공기 블록의 런타임 id 가 들어 있다. 블록의 런타임 id 는 테스트 바이너리가 등록한 블록 전체로 정해진다. `dimension` 이나 그 테스트의 import 가 달라져 등록되는 블록이 늘거나 줄면 이 값이 바뀐다. 같은 패킷을 블록 이름으로 적은 `testdata/popup.byname.golden` 은 그때도 그대로다.

## 클라이언트 항목 고치기

클라이언트가 받는 커스텀 아이템·블록 항목은 `core.EditEntries(srv)` 가 한 번에 정한다. 서버를 만든 뒤 플레이어가 들어오기 전에 한 번 부른다.

- 아이템 항목은 `clientdef.ItemEntry` 를 거친 뒤 기능 목록 순서대로 각 기능의 `Feature.ItemEntry` 를 거친다. 블록 항목은 `clientdef.BlockEntry` 를 거친다.
- `core.CheckEntries()` 는 `srv.Listen()` 직후에 불러 모든 항목에 고치기를 미리 적용한다. 항목 선언이 잘못되면 패닉 대신 블록이나 아이템 이름이 든 오류를 돌려주고, 서버는 그 오류를 남기고 멈춘다.
- 새 아이템이 `minecraft:digger` 같은 컴포넌트가 필요하면 `clientdef` 의 인터페이스를 구현한다. `ItemEntry` 필드는 인터페이스로 되지 않는 항목 고치기에만 쓴다.

## 기능이 아직 손대는 중앙 목록

구독, 데이터, 패널 줄, 칸 장식, 창작 항목, 명령어, 설정은 기능 패키지 안에서 끝나지만, 다음은 여러 기능의 이름이 한곳에 오르는 목록이다. 해당하는 기능만 고친다.

- 게임 아이템 레시피와 연구 트리: `content/skygear/catalog` 의 `recipes.go`, `research.go`.
- 글: `content/skygear/locales` 의 `ko.json`, `en.json` 과 나뉜 파일들.
- 기계 종류의 이름, id, `Order`, `Def`: `content/skygear/machine/machine.go`. 동작은 소유한 기능이 등록한다(위 5단계).

## 구조를 지키는 규칙

규칙마다 어긴 때 알려 주는 테스트가 있다.

| 규칙 | 테스트 |
|---|---|
| 루트는 아무것도 구독하지 않는다. 루트 패키지의 비테스트 파일에는 훅의 `On` 호출, `core.Mark` 호출, `Handle...` 메서드가 없다 | `TestRootSubscribesToNothing` |
| 모든 구독자는 목록에 오른 기능이 `Setup` 에서 단다. 이름의 접두어가 그 기능 이름이다 | `TestEverySubscriberHasItsFeature`, `core.TestInstallingPrefix` |
| `feature.go` 가 있는 패키지는 `Features` 에 정확히 한 번 오르고 `Name` 이 패키지 이름이다 | `TestEveryFeatureListed`, `TestFeatureNames` |
| 기능 패키지는 루트를 import 하지 않는다(간접도). `core` 는 기능을 import 하지 않는다. `common` 은 `content/`, `platform/` 을 import 하지 않는다 | `TestFeaturesLiftable` |
| 패키지는 `layers` 의 아래 칸만 import 하고, 같은 칸의 기능 import 는 `siblings` 가 허용한 것뿐이다. `featureLayer` 는 `layers` 의 기능 칸과 같다 | `TestLayers`, `TestFeatureSiblings` |
| `look` 은 `core`, `naming` 만, `decor` 는 `naming` 만 import 한다. 블록과 아이템은 인터페이스로 자기를 설명한다 | `TestCapabilityImports` |
| `machine` 은 기능 칸의 패키지를 import 하지 않고 `siblings` 에 자기 줄이 없다 | `TestMachineImportsNoFeature` |
| 다른 기능의 기계 종류는 `kindOwners` 의 기능이 소유하고, 그 기능은 목록에 있으며 `Setup` 에서 종류를 등록한다 | `TestKindOwners` |
| 등록되지 않은 종류가 있으면 `machine.CheckKinds` 가 종류와 기능 이름을 든 오류를 돌려준다. `machine.Register` 는 없는 종류, 소유 기능이 없는 종류, 두 번째 등록, 검사가 끝난 뒤의 등록에 패닉한다 | `machine.TestCheckKindsNamesTheMissingOwner`, `machine.TestRegisterPanics` |
| 기능의 `Check` 는 모든 기능의 `Setup` 이 돈 뒤에 목록 순서대로 돈다 | `core.TestInstallRunsChecksAfterEverySetup` |
| 패널 갱신을 따라가는 단계는 `tinker.vein`, `wand.preview` 순서다 | `TestLookFollowers` |
| `catalog/creative.go` 는 `core`, `hopper`, `machine`, `sgitem` 만 import 하고, 기능의 창작 자리는 표에 적힌 대로다 | `TestCreativeListNamesNoFeature` |
| 한 창작 자리는 주인이 하나다. 두 주인이 쓰면 패닉이다 | `core.TestCreativeSlotOfTwoOwnersPanics`, `core.TestCreativeItemsSortBySlot` |
| 기능의 명령어가 기능 목록 순서대로 모인다 | `TestFeatureCommands`, `core.TestCommandsInListOrder` |
| 아이템이 자기 칸 장식을 말하고, 말하지 않는 아이템은 일반 등급이다 | `TestItemDecor` |
| 기능 하나가 목록과 자기 패키지만으로 이벤트를 받는다 | `core.TestNewFeatureNeedsOnlyTheList` |
| 테스트 지원 패키지(`core/coretest`, `machine/machinetest`)는 테스트 파일만 import 한다 | `TestSupportPackagesAreTestOnly` |
| 패키지의 테스트는 루트를 import 하지 않는다(`wire` 는 예외). 내용 목록 아래 칸의 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다. `core` 밖의 테스트는 `core.Install` 이나 기능의 `Setup` 을 직접 부르지 않고 `coretest.Setup` 을 쓴다 | `TestPackageTestsStayBelow` |
| 플레이어 핸들러의 모든 메서드가 `core.PlayerHandler` 에 있고 훅마다 구독자가 있으며, 루트가 같은 이름의 메서드를 두지 않는다 | `TestHandlerMethodsOneWay` |
| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다. 취소된 이벤트를 도는 길도 같다 | `core.TestEmitAllocatesNothing`, `core.TestEmitAllocatesNothingAfterCancel`, `core.TestEventsEmitAllocateNothing` |
| 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름 겹침, 없는 티어는 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSameNamePanics`, `core.TestNotATierPanics` |
| 티어는 상수의 순서대로 돈다. 같은 티어 안에서는 등록한 순서로 돌지만 구독자는 그 순서에 기대지 않는다 | `core.TestRunsInTierOrder`, `core.TestSameTierRunsInRegistrationOrder` |
| 같은 티어의 구독자 사이에는 순서가 없다. 티어 안의 순서를 뒤집고 섞어도 모든 테스트가 통과하고 장면의 결과, 업적 카운터, 플레이어의 인벤토리가 같다. 순서를 지켜 주는 쌍은 없다 | `TestTierOrder` |
| 어느 구독자도 다른 구독자를 멈추지 못한다. 취소할 수 없는 이벤트의 훅에서는 모두 불린다 | `core.TestNoSubscriberStopsAnother`, `core.TestOnlyActorEventsCancel` |
| 취소된 이벤트는 기본 구독자에게 가지 않고 `core.HandleCancelled` 구독자에게만 간다. 취소는 Dragonfly 의 컨텍스트 하나다 | `core.TestCancelSkipsDefaultSubscribers`, `core.TestAfterCancelOnlyHandleCancelledRuns`, `core.TestCancelIsDragonflysCancel`, `core.TestCancelledTellsTheContext` |
| MONITOR 구독자는 취소하지도 가공하지도 않는다 | `TestMonitorSubscribersDoNotShape`, `core.TestMonitorCancelPanicsWhileTraced` |
| `Leave` 의 구독자가 모두 돈 뒤에 플레이어별 파트 항목을 지운다 | `core.TestLeaveDropsThePlayersEntriesLast` |
| 구독자의 조건이 지키는 것: 사냥꾼의 이차원석, 섬 밖의 복사기와 지팡이, 모드에 붙잡힌 플레이어의 웅크리기 | `TestHunterStoneOpensNothing`, `TestToolsLeaveTheClickOutsideTheIsland`, `elevator.TestHeldPlayerDoesNotRideDown`, `farm.TestHeldPlayerGrowsNoSapling` |
| 취소된 일은 일어나지 않은 일이다: 공허로 떨어진 걸음과 서랍 앞면을 친 캐기는 세지 않는다. 웅크리기는 언제나 센다 | `TestMoveIntoTheVoidIsNotWalked`, `TestDrawerFrontHitIsNoBreak`, `TestEverySneakCounts`, `TestSneakRideGrowsSaplingsWhereItArrives` |
| 사냥꾼을 죽이는 타격은 취소하지 않는다. 사냥꾼은 죽고, 판은 한 번 실패로 끝나고, 소지품과 경험치는 그대로이고, 부활하면 들어왔던 자리로 간다. 죽음과 부활 사이에 나가도 같다. 섬에서 죽는 일은 달라지지 않는다 | `TestHunterDiesAndComesBack`, `TestDeadHunterWhoLeavesIsBackOnTheIsland`, `TestHitThatKillsLeavesNoEffect`, `TestBlastThatKillsDoesNotThrowTheHunter`, `TestDeathOnTheIslandIsUntouched`, `instance.TestDeathKeepsTheInventoryEverywhere` |
| 공중에서 쓰는 아이템은 `item.Usable` 이고 자기 `Use` 에서 일한다. 사냥꾼의 공중 클릭은 아무 일도 하지 않는다 | `TestAirItemsAreUsable`, `TestItemsUsedOnTheAir`, `TestHunterUsesNothingOnTheAir` |
| 막대기로 다른 블록을 두드린 기록은 아무도 취소하지 않은 클릭만 센다 | `TestStickOtherCountsTheTapNothingTook` |
| 블록과 아이템의 메서드로 하는 일은 Dragonfly 를 거친 클릭으로 확인한다. 요정의 모래시계는 기계가 쓰고, 도관 판과 옆 칸 가드가 그보다 먼저다. 기계 아이템은 누른 면을 자기 설치에 넘기고, 클릭 없이 놓인 기계는 기본 방향을 본다 | `TestHourglassThroughTheMachine`, `TestHourglassOnTheOutwardFaceIsRefused`, `machine.TestHourglassIsUsedByTheMachine`, `TestMachineItemPlacesAndFaces`, `TestMachinePlacedWithoutAClickFacesItsDefault` |
| 서버가 멈출 때 이차원에 있던 플레이어는 입장 때 소지품을 먼저 돌려받고, 그 뒤에 받은 요정 가루를 지닌다 | `TestStrandedPlayerKeepsBackfilledDust` |

핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse`, `BenchmarkBlockBreak`, `BenchmarkBlockPlace` 가 잰다. 구독자는 취소되지 않은 이벤트마다 모두 불리므로, 조건은 싼 것부터 본다(든 아이템의 종류, 그다음 맵 조회나 블록 읽기). 패널 하나의 비용은 루트의 `BenchmarkLookMachine`(시선 추적과 패널 보내기), `BenchmarkLookInfoMachine`, `BenchmarkLookInfoStone`(패널 만들기)이 잰다. 다른 기능이 동작을 등록한 기계의 틱 한 번은 `BenchmarkTickHarvester`, `BenchmarkTickDrawerController` 가 잰다.

```sh
go test -run xxx -bench 'HandleMove|Second|UseOnBlock|Use$|BlockBreak|BlockPlace|LookMachine|LookInfo|TickHarvester|TickDrawerController' -benchmem -count 5 ./content/skygear
go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
```

구독자는 하나에 대략 3~10ns 가 든다. 할당 수와 바이트 수는 늘지 않아야 한다.

## 핀

구독 순서, 패널, 기계 종류의 동작, 창작 목록, 클라이언트가 받는 데이터는 골든 파일로 고정한다.

| 파일 | 담는 것 | 테스트 |
|---|---|---|
| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 훅과 티어 순서대로 `<훅> <티어> <이름>`. 한 티어 안은 이름순이다(순서가 없으므로). `core.HandleCancelled` 로 단 구독자는 끝에 ` handleCancelled` 가 붙는다 | `TestPinOrder` |
| `content/skygear/testdata/pin/handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden` | 장면마다 이벤트가 거친 구독자 이름과 결과(취소 여부, 효과) | `TestPinSimple`, `TestPinBlocks`, `TestPinUse` |
| `content/skygear/testdata/pin/lifecycle.golden` | 입장·퇴장·매초·빠른 틱·새 날·자동 저장·닫기·시작의 구독자 이름과 결과 | `TestPinLifecycle` |
| `content/skygear/testdata/pin/settings.golden` | 설정이 등록된 순서 | `TestPinSettings` |
| `content/skygear/testdata/pin/look.golden` | 장면마다 바라본 블록의 패널 한 줄(이름, 번역 키, 아이콘, 줄, 아이템 줄, 보관 칸, 도구, 표시, 출처), 143개 | `TestPinLook` |
| `content/skygear/testdata/pin/kinds.golden` | 기계 종류마다 패널 밖에서 하는 일: 서서·웅크려 클릭한 결과와 열린 화면, 창의 정보 아이콘, 창 27칸의 역할과 칸이 받는 견본 아이템, 145줄 | `TestPinKinds` |
| `content/skygear/testdata/pin/creative.golden` | 게임이 더한 창작 목록 항목을 클라이언트가 보이는 순서대로 그룹과 함께, 243개 | `TestPinCreative` |
| `content/skygear/wire/testdata/*.golden` | 접속한 플레이어가 받는 블록 항목, 아이템 등록, 제작 데이터, 창작 목록의 항목별 해시 | `TestWire` |
| `content/skygear/wire/testdata/items.byname.golden`, `recipes.byname.golden`, `creative.byname.golden` | 위의 아이템 등록, 제작 데이터, 창작 목록과 같은 항목인데 아이템 id 자리에 아이템 이름을 적은 것. 커스텀 아이템 id 의 집합(`meta/custom_ids`)도 담는다 | `TestWire` |
| `content/skygear/dimension/testdata/popup.golden`, `popup.byname.golden` | 이차원 데미지 숫자와 경험치 보석의 패킷. `byname` 은 블록 런타임 id 자리에 블록 이름을 적은 것 | `dimension.TestPopupAddActor` |
| `content/skygear/testdata/save/oldsave.zip`, `oldsave.golden` | 2026-11-11 의 빌드가 쓴 세이브(모든 커스텀 블록 상태, 상자에 든 모든 커스텀 아이템, 아이템을 든 분쇄기·서랍·나무 호퍼, 플레이어 상태)와 그 세이브를 읽은 결과를 이름으로 적은 2,822줄 | `TestOldSaveLoads` |

```sh
go test ./content/skygear -run 'TestPin' -count=1     # 구독 순서, 장면, 패널, 기계 종류, 창작 목록
go test ./content/skygear/wire/                        # 클라이언트 데이터
go test ./content/skygear -run 'TestOldSaveLoads' -count=1   # 옛 세이브 읽기
```

골든이 달라지면 테스트가 어느 줄(장면)이 어떻게 달라졌는지 `want` 와 `got` 로 보여 준다. 디스패치 순서나 클라이언트 데이터가 바뀌면 골든 차이로 나타난다.

장면 골든의 한 줄은 `<장면>: <거친 구독자 이름> | <결과>` 다. `|` 앞은 등록한 순서로 돌았을 때의 자취이고 뒤가 결과다.
장면은 핸들러만 부른다. 블록의 `Activate` 나 아이템의 `UseOnBlock` 처럼 Dragonfly 가 핸들러 뒤에 부르는 메서드가 한 일은 장면에 나오지 않는다(위 "훅과 블록·아이템의 메서드").
`TestTierOrder` 는 루트의 테스트 바이너리를 여섯 번 다시 돌린다. 한 번은 등록한 순서 그대로, 다섯 번은 모든 훅의 구독자를 티어 안에서 뒤집거나 섞어서다(`SKYGEAR_HOOK_ORDER=reverse`, `shuffle:1` 부터 `shuffle:4`). 섞은 실행에서 확인하는 것은 셋이다.

- 루트의 모든 테스트가 통과한다. 장면 골든은 `|` 뒤의 결과만 견준다.
- 장면 234개의 결과가 골든과 글자 하나까지 같다.
- 장면마다 플레이어의 업적 카운터가 늘어난 양과, 장면의 플레이어가 든 아이템(칸, 종류, 개수)이 등록한 순서로 돈 실행과 같다. 둘 다 골든에 없어서 따로 견준다.

플레이어가 받는 팁·글·소리와 어느 장면도 읽지 않는 상태는 이 테스트가 보지 못한다. 순서를 지켜 주는 쌍은 없다. 모든 구독자를 티어 안에서 섞는다.
기능 패키지의 테스트도 같은 변수로 섞어서 돌릴 수 있다. `coretest.Session` 이 처음 불릴 때 섞는다.

```sh
go test ./content/skygear -run TestTierOrder -count=1 -v
SKYGEAR_HOOK_ORDER=reverse go test ./content/skygear/... -count=1
```
골든은 동작을 바꾸려는 변경에서만 다시 기록한다. 같은 티어에 구독자를 더하면 그 훅을 거치는 장면의 자취(`|` 앞)가 달라지므로 그때도 다시 기록하고, `|` 뒤가 그대로인지 `git diff` 로 본다.

```sh
SKYGEAR_GOLDEN=update go test ./content/skygear -run 'TestPin' -count=1
SKYGEAR_GOLDEN=update go test ./content/skygear/wire/
```

다시 기록한 골든은 `git diff` 로 의도한 줄만 바뀌었는지 확인하고, 커밋 메시지에 왜 바꿨는지 적는다. 테스트를 통과시키려고 다시 기록하지 않는다.

### id 만 바뀌는 변경

커스텀 아이템의 숫자 id 는 아이템이 등록된 순서다. 등록 순서는 패키지가 초기화되는 순서를 따른다. 패키지 사이의 import 를 더하거나 방향을 바꾸거나 등록하는 곳을 옮기면 플레이어가 보는 것은 그대로인데 `items.golden`, `recipes.golden`, `creative.golden` 의 수백 줄이 바뀐다. 이런 변경은 다음과 같이 확인한다.

1. 코드를 바꾼 뒤 다시 기록하지 않고 테스트를 돌린다. `TestWire` 의 `items`, `recipes`, `creative` 만 실패하고 `by name` 은 통과해야 한다. `TestPin…` 과 `TestOldSaveLoads` 도 통과해야 한다.
2. `SKYGEAR_GOLDEN=update go test ./content/skygear/wire/ -run 'TestWire$' -count=1` 로 다시 기록한다. `popup.golden` 이 달라졌으면 `SKYGEAR_GOLDEN=update go test ./content/skygear/dimension/ -run 'TestPopupAddActor$' -count=1` 도 돌린다.
3. `git status --short` 에 `*.byname.golden`, `testdata/save/`, `testdata/pin/` 의 파일이 없어야 한다. 이름으로 적은 골든이 그대로면 바뀐 것은 id 뿐이다. 하나라도 있으면 id 말고 다른 것이 바뀐 것이므로 다시 기록한 것을 되돌리고 원인을 찾는다.
4. `git diff --numstat` 의 줄 수를 커밋 메시지에 적는다.

세이브는 블록과 아이템을 이름으로 저장하므로 id 가 바뀌어도 그대로 읽힌다. `TestOldSaveLoads` 가 옛 빌드가 쓴 세이브를 지금 빌드로 읽어 이를 확인한다. `oldsave.zip` 과 `oldsave.golden` 은 세이브에 담을 것을 바꿀 때만 다시 기록한다. 게임을 바꾼 뒤 이 테스트가 실패하면 옛 세이브가 전과 다르게 읽힌다는 뜻이다.
