# 기능 확장

게임의 기능은 `content/skygear` 아래 패키지 하나씩이다. 기능은 서버 이벤트를 훅으로 받고, 자기 데이터를 `core` 의 파트에 두고, 루트 패키지의 기능 목록에 한 줄로 오른다.
이 문서는 그 바탕이 되는 층, 이벤트가 기능에 닿는 길, 새 기능을 더하는 순서, 구조를 지키는 테스트를 적는다.
Dragonfly 자체를 확장하는 방법은 [dragonfly.md](dragonfly.md) 에 있다.

## 층

아래에서 위로 쌓인다. 각 층은 자기보다 아래 층만 import 한다.

| 층 | 위치 | 하는 일 |
|---|---|---|
| Dragonfly 패치 | `third_party/dragonfly` | Dragonfly 의 동작 자체. 패치 규칙은 [dragonfly.md](dragonfly.md) |
| `sghook` | `common/sghook` | 패킷 훅, 세션 내부 접근, 클라이언트에 보내는 아이템·블록 항목 고치기(`EditCustomItems`, `EditCustomBlocks`, `ApplyEntryEdits`). Dragonfly 내부는 여기서만 건드린다 |
| `clientdef` | `common/clientdef` | 커스텀 아이템·블록이 클라이언트에 보내는 컴포넌트. 공개 레지스트리만 쓰고 게임을 모른다 |
| `core` | `content/skygear/core` | 기능의 커널. `Session`과 그 파트(`Part`, `Players`, `Saved`), 훅(`Hook`)과 이벤트, 핸들러(`PlayerHandler`, `WorldHandler`), 수명 호출(`Lifecycle`), 기능 서술자(`Feature`)와 `Install` |
| 기능 패키지 | `content/skygear/<기능>` | 한 기능의 블록·아이템·화면·규칙과, 그 기능이 훅에 거는 구독자 |
| 루트 | `content/skygear` | 기능 목록(`features.go`)과 플랫폼 접착제(`Game`, `session`), 명령어, 운영자 도구 |

무엇을 import 할 수 있는지는 다음과 같다.

- `sghook` 과 `clientdef` 는 `content/` 와 `platform/` 을 import 하지 않는다. 다른 `common/*` 패키지도 마찬가지다(`TestFeaturesLiftable`).
- `core` 는 `common`, `platform`, 그리고 데이터 층의 패키지를 import 한다. 기능 패키지는 import 하지 않는다(`TestFeaturesLiftable`).
- 기능 패키지는 `core`, `common`, `platform`, 자기보다 아래 칸의 `content/skygear/*` 패키지를 import 한다. 같은 칸의 기능끼리는 `content/skygear/layers_test.go` 의 `siblings` 표에 적힌 import 만 허용된다(`TestLayers`, `TestFeatureSiblings`). 루트 패키지는 직접도 간접도 import 하지 않는다(`TestFeaturesLiftable`).
- 루트 패키지는 모든 기능을 import 한다.

위 칸이 아래 칸을 불러야 하면 아래 칸이 함수 변수를 두고 위 칸이 `init` 에서 채운다(예: `core.ShowMenu` 는 `screens` 가 채운다).

## 이벤트의 길

플레이어 이벤트는 Dragonfly 가 `core.PlayerHandler` 의 `Handle...` 메서드를 불러 시작한다. 월드 이벤트는 `core.WorldHandler` 가 받는다(`HandleEntityDespawn`, `HandleColumnLoad`, `HandleColumnUnload`).
세션 수명 이벤트는 플랫폼이 세션의 메서드를 불러 시작한다.

| 호출하는 쪽 | 호출 | 내보내는 훅 |
|---|---|---|
| `Game.Start` | `core.Load.Emit` (월드 트랜잭션 밖), `core.Start.Emit` (월드 트랜잭션 안) | `Load`, `Start` |
| 인스턴스 | `Join`, `Leave`, `NewDay`, `Second`, `Fast`, `Autosave`, `Close` | 이름이 같은 훅 |

세션 타입은 `core.Lifecycle` 을 품고 있어 이 호출들이 `Lifecycle` 의 메서드로 간다. 메서드마다 이벤트를 만들어 훅의 `Emit` 을 한 번 부르고, 아래 꼬리 일을 한다.

- `Second`: 구독자가 끝나면 바뀐 상태를 저장 주기에 맞을 때만 쓴다(`FlushIfDue`).
- `Autosave`: 구독자가 끝나면 바뀐 상태를 쓴다(`Flush`).
- `Close`: 구독자가 끝나면 저장 파트를 쓰고 쓰기 고루틴을 멈추고(`Session.Close`), 세션 등록을 지운다(`Unregister`). 루트의 `session.Close` 는 그 뒤에 `Game` 의 세션 표에서도 지운다.
- `NewDay`: 구독자가 끝나면 모든 플레이어에게 `game.newDay` 를 알린다.

`Emit` 은 구독자를 우선순위가 작은 것부터 차례로 부른다.

### On 과 Handle

- `On(prio, name, f)` 로 단 구독자는 이벤트를 멈추지 못한다.
- `Handle(prio, name, f)` 로 단 구독자가 true 를 돌려주면 뒤의 구독자는 돌지 않는다. `Emit` 도 true 를 돌려주지만 핸들러와 수명 호출은 그 값을 쓰지 않는다.

멈춤과 취소는 다르다. 멈춤은 훅의 일이고 뒤의 구독자만 건너뛴다. 취소는 Dragonfly 의 일이고 플레이어가 한 행동이 일어나지 않게 한다. `Actor.Cancel()`(`e.Cancel()`)이 취소하며, 취소해도 뒤의 구독자는 계속 돈다.
구독자는 둘 중 하나만 하거나 둘 다 하거나 아무것도 안 할 수 있다. `Jump`, `Death`, `Respawn` 은 Dragonfly 가 취소할 수 없는 이벤트라 `Ctx` 가 nil 이고 `Cancel()` 은 아무 일도 하지 않는다.

### 이벤트의 값

이벤트는 값으로 건네므로 `Emit` 이 할당하지 않는다. 필드는 두 가지다.

- 이벤트를 만들 때 찍은 사본: `Held`, `Off`, `Block`, `Item`. 앞 구독자가 손이나 월드를 바꿔도 뒤 구독자는 옛 값을 본다. 새 값은 플레이어나 `Tx` 에 다시 물어야 한다.
- Dragonfly 가 핸들러에 준 포인터: `Drops`, `XP`, `Damage`, `Pos`(`RespawnEvent`), `Pick`(`ItemPickup` 에서 더해질 묶음). 포인터에 쓰면 Dragonfly 가 하는 일이 바뀐다.

`Sneak` 과 `Sprint` 는 시작할 때만 내보낸다.

## 훅 목록

훅은 35개이고 `content/skygear/core/events.go` 에 선언되어 있다. 플레이어 훅 23개와 세션·월드 훅 12개다.
"멈춤" 칸은 지금 그 훅에 `Handle` 구독자가 있는지를 나타낸다. 없는 훅에도 `Handle` 을 달 수 있다.

| 훅 | 내보내는 때 | 이벤트 | 멈춤 |
|---|---|---|---|
| `StartBreak` | 블록 캐기를 시작할 때 | `StartBreakEvent` | 있음 |
| `BlockBreak` | 블록을 캘 때 | `BlockBreakEvent` | 있음 |
| `BlockPlace` | 블록을 놓을 때 | `BlockPlaceEvent` | 있음 |
| `UseOnBlock` | 블록에 아이템을 쓸 때 | `UseOnBlockEvent` | 있음 |
| `UseOnEntity` | 엔티티에 아이템을 쓸 때 | `EntityEvent` | 있음 |
| `AttackEntity` | 엔티티를 때릴 때 | `EntityEvent` | 있음 |
| `Use` | 공중에서 아이템을 쓸 때 | `UseEvent` | 있음 |
| `Move` | 움직일 때 | `MoveEvent` | 있음 |
| `Sneak` | 웅크리기를 시작할 때 | `PlayerEvent` | 있음 |
| `Sprint` | 달리기를 시작할 때 | `PlayerEvent` | 없음 |
| `Jump` | 점프할 때 | `PlayerEvent` | 없음 |
| `ItemDamage` | 아이템이 내구도를 잃을 때 | `ItemDamageEvent` | 없음 |
| `ItemConsume` | 아이템을 먹거나 마실 때 | `ItemEvent` | 없음 |
| `ItemPickup` | 아이템을 주울 때 | `ItemEvent` | 없음 |
| `ItemDrop` | 아이템을 버릴 때 | `ItemEvent` | 있음 |
| `Death` | 죽을 때 | `PlayerEvent` | 없음 |
| `Respawn` | 되살아날 때 | `RespawnEvent` | 없음 |
| `Hurt` | 피해를 입기 전 | `HurtEvent` | 있음 |
| `PunchAir` | 허공을 칠 때 | `PlayerEvent` | 없음 |
| `Chat` | 채팅을 보낼 때 | `PlayerEvent` | 없음 |
| `BlockPick` | 블록을 집어 들 때 | `PlayerEvent` | 없음 |
| `HeldSlot` | 들고 있는 칸을 바꿀 때 | `PlayerEvent` | 없음 |
| `Command` | 명령어를 실행할 때 | `PlayerEvent` | 없음 |
| `Load` | 세이브의 첫 트랜잭션 앞(`Tx` 는 nil) | `Tick` | 없음 |
| `Start` | 세이브의 첫 트랜잭션 안 | `Tick` | 없음 |
| `Join` | 플레이어가 입장한 뒤 | `Presence` | 없음 |
| `Leave` | 플레이어가 나가기 전 | `Presence` | 없음 |
| `NewDay` | 인스턴스 날짜가 바뀔 때 | `DayEvent` | 없음 |
| `Second` | 약 1초마다 | `Tick` | 없음 |
| `Fast` | 0.2초마다 | `Tick` | 없음 |
| `Autosave` | 자동 저장 주기마다 | `Tick` | 없음 |
| `Close` | 세이브를 닫을 때 | `Tick` | 없음 |
| `ColumnLoad` | 청크 칼럼이 월드에 더해질 때 | `ColumnEvent` | 없음 |
| `ColumnUnload` | 청크 칼럼이 월드에서 빠질 때 | `ColumnEvent` | 없음 |
| `EntityDespawn` | 엔티티가 월드를 떠날 때 | `DespawnEvent` | 없음 |

`PlayerEvent` 는 인자가 없는 이벤트이고, `Actor`(`S`, `P`, `Tx`, `Ctx`)만 든다. 위치와 방향, 대상 같은 인자는 이벤트 종류마다 `core/events.go` 에서 본다.
이벤트에 없는 Dragonfly 인자(채팅 글, 사망의 `keepInv`, 공격의 힘과 치명타 따위)는 구독자가 받지 못한다.

## 새 기능 더하기

엘리베이터(`content/skygear/elevator`)가 가장 작은 예다.

1. `content/skygear/<기능>/` 패키지를 만든다. 기능의 규칙과 블록·아이템은 이 패키지 안에 둔다.
2. `feature.go` 에 서술자를 둔다. 필드는 `Name`, `Entities`(기능이 더하는 엔티티 종류), `Settings`(플레이어 설정), `ItemEntry`(커스텀 아이템 항목 고치기), `Setup`(구독자 달기)이다. `Name` 은 패키지 이름과 같아야 한다.

   ```go
   package elevator

   import "github.com/presentkim/skygear/content/skygear/core"

   // Feature is the elevator feature, listed in the game's feature list.
   var Feature = &core.Feature{Name: "elevator", Setup: subscribe}
   ```

3. `subscribe.go` 에서 `Setup` 이 부르는 `subscribe()` 가 구독자를 단다. 이름은 `<기능>.<단계>` 이고 접두어가 기능 이름과 같아야 한다.

   ```go
   func subscribe() {
       subscribeJump()
       subscribeSneak()
   }

   func subscribeJump() {
       core.Jump.On(20, "elevator.up", func(e core.PlayerEvent) {
           Up(e.S, e.P, e.Tx)
       })
   }

   func subscribeSneak() {
       core.Sneak.Handle(30, "elevator.down", func(e core.PlayerEvent) bool {
           return Down(e.S, e.P, e.Tx)
       })
   }
   ```

4. 루트의 `content/skygear/features.go` 의 `Features` 에 `<기능>.Feature` 를 한 줄 더하고 import 를 더한다. 설정과 아이템 항목 고치기는 목록 순서대로 적용되고 `Setup` 도 그 순서대로 돈다.
5. `content/skygear/layers_test.go` 에 기능을 알린다. `layers` 에서 칸을 정하고(자기보다 아래 칸에 든 패키지만 import 할 수 있다), 기능 칸이면 `featureLayer` 에도 이름을 더한다. 같은 칸의 기능을 import 하면 `siblings` 에 행을 더한다. 기능의 순서는 `TestFeatureNames` 의 `want` 목록에도 적혀 있으니 `Features` 에 더한 자리에 같이 더한다.
6. 구독자를 달았으면 골든이 달라진다. `order.golden` 에는 새 줄(`<훅> <우선순위> <이름>`)이 생기고, 그 훅을 거치는 장면이 있으면 `handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden`, `lifecycle.golden` 의 그 장면 줄에도 새 이름이 들어간다. 의도한 변경이므로 `SKYGEAR_GOLDEN=update go test ./content/skygear -run TestPin -count=1` 로 다시 기록하고 `git diff` 로 새 단계의 이름만 늘었는지 확인한다. 커밋 메시지에 어느 단계를 더했는지 적는다.
7. `go test ./content/skygear -run 'TestLayers|TestFeature|TestEvery|TestRoot|TestPin|TestHandlerMethodsOneWay' -count=1` 로 확인한다. 위 순서를 빠뜨리면 이 테스트들이 `TestLayers`(`package <기능> is not in layers`), `TestFeatureNames`, `TestPinOrder`, `TestEverySubscriberHasItsFeature`, `TestPinSimple` 따위로 어긋난 곳을 알려 준다.

### 우선순위 고르기

우선순위는 한 훅 안에서 겹치면 안 되고, 이름도 한 훅 안에서 겹치면 안 된다(이름은 훅이 다르면 같아도 된다). 겹치면 `Install` 이 패닉한다.
기존 단계는 10, 20, 30처럼 10씩 띄워 두었다. 둘 사이에 끼우려면 그 사이의 수를 쓴다(15 등). 현재 순서는 `order.golden` 에 훅별로 모여 있으니 먼저 거기서 이웃 단계를 확인한다.
무엇보다 앞에서 도는지, 어느 단계가 멈추면 내 단계가 건너뛰어도 되는지를 본다(다음 절).

### 순서가 중요한 곳

지금 순서에서 알아 둘 사실이다.

- `StartBreak`, `BlockBreak`, `BlockPlace` 에서는 `expand.guard`(10)와 `island.guard`(20)가 맨 앞이다. 월드를 바꾸는 단계는 모두 그 뒤에 선다. 가드는 `Handle` 로 멈추면서 `Cancel()` 도 한다.
- `UseOnBlock` 에서는 `dimension.hunting`(10), `screens.menuItem`(20), `dimension.stone`(30)이 두 가드(`expand.guard` 40, `island.guard` 50)보다 앞이고, 블록을 쓰는 단계(`copier.use` 70부터)는 가드 뒤다. `fairy.time`(140) 다음에 `farm.crop`(150)이 돌고, 옆 칸 가드(`expand.guardSide` 160, `island.guardSide` 170)는 그 뒤다.
- `UseOnBlock` 에서는 사냥 검사(`dimension.hunting`)가 메뉴 아이템(`screens.menuItem`)보다 앞이고, `Use` 에서는 메뉴 아이템(20)이 사냥 검사(30)보다 앞이다. `Use` 의 `autofill.use`(10)는 메뉴 아이템보다 앞이다.
- `Second` 에서 `progress.checkHeld`(20)는 `dimension.second`(60)보다 앞이다. 앞의 것이 플레이어가 든 아이템을 읽어 파트에 두고, 뒤의 것이 `RetireItems` 로 아이템을 정리한다. `progress.collection`(120)이 그 파트를 읽는다.
- `Leave` 에서 `dimension.leave`(10)는 `core.parts`(20)보다 앞이다. `core.parts` 는 `Players` 의 플레이어 항목을 지우므로, 그 뒤(`expand.stop` 30 등)에 도는 구독자는 `Players` 에서 그 플레이어의 항목을 찾지 못한다. 플레이어별 데이터를 읽어야 하는 퇴장 단계는 20 앞에 둔다.
- `Close` 에서는 구독자 뒤에 저장 파트 쓰기와 세션 등록 해제가 온다. 구독자는 세션이 아직 등록된 상태에서 돈다.

## 기능의 데이터

기능은 `Session` 에 필드를 더하지 않고 `core` 의 파트를 패키지 변수로 둔다.

- `core.NewPart[T](mk)`: 세션마다 하나, 처음 `Of(s)` 를 부를 때 만든다. 저장하지 않는다. 예: `progress` 의 `held`.
- `core.NewPlayers[T]()`: 세션 안에서 플레이어마다 하나. `Of(s)` 는 `map[*world.EntityHandle]T` 를 돌려준다. 플레이어가 나갈 때 `Leave` 의 `core.parts` 단계(`Session.Leave`)가 항목을 지운다. 예: `tinker` 의 `keptFood`.
- `core.NewSaved[T](mk)`: 저장하는 데이터. 모든 `Saved` 파트는 세이브 상태 문서(`skygear.state`) 하나의 JSON 필드이고, 필드 이름은 파트끼리 겹치지 않는다.

```go
var keptFood = core.NewPlayers[int]()

core.Death.On(30, "tinker.keepFood", func(e core.PlayerEvent) {
    keptFood.Of(e.S)[e.P.H()] = e.P.Food()
})
```

## 클라이언트 항목 고치기

클라이언트가 받는 커스텀 아이템·블록 항목은 `core.EditEntries(srv)` 가 한 번에 정한다. 서버를 만든 뒤 플레이어가 들어오기 전에 한 번 부른다.

- 아이템 항목은 `clientdef.ItemEntry` 를 거친 뒤 기능 목록 순서대로 각 기능의 `Feature.ItemEntry` 를 거친다. 블록 항목은 `clientdef.BlockEntry` 를 거친다.
- `core.CheckEntries()` 는 `srv.Listen()` 직후에 불러 모든 항목에 고치기를 미리 적용한다. 항목 선언이 잘못되면 패닉 대신 블록이나 아이템 이름이 든 오류를 돌려주고, 서버는 그 오류를 남기고 멈춘다.
- 새 아이템이 `minecraft:digger` 같은 컴포넌트가 필요하면 `clientdef` 의 인터페이스를 구현한다. `ItemEntry` 필드는 인터페이스로 되지 않는 항목 고치기에만 쓴다.

## 기능이 아직 손대는 중앙 목록

구독과 데이터는 기능 패키지 안에서 끝나지만, 다음은 기능마다 여러 곳에 이름이 오르는 목록이다. 해당하는 기능만 고친다.

- 창작 목록, 게임 아이템 레시피, 연구 트리: `content/skygear/catalog` 의 `creative.go`, `recipes.go`, `research.go`.
- 바라보는 블록의 WAILA 줄: `content/skygear/look/look.go` 의 `Info`.
- 명령어: 루트의 `content/skygear/commands.go` 의 `registerCommands`.
- 글: `content/skygear/locales` 의 `ko.json`, `en.json` 과 나뉜 파일들.

## 구조를 지키는 규칙

규칙마다 어긴 때 알려 주는 테스트가 있다.

| 규칙 | 테스트 |
|---|---|
| 루트는 아무것도 구독하지 않는다. 루트 패키지의 비테스트 파일에는 훅의 `On`·`Handle` 호출, `core.Mark` 호출, `Handle...` 메서드가 없다 | `TestRootSubscribesToNothing` |
| 모든 구독자는 목록에 오른 기능이 `Setup` 에서 단다. 이름의 접두어가 그 기능 이름이다 | `TestEverySubscriberHasItsFeature`, `core.TestInstallingPrefix` |
| `feature.go` 가 있는 패키지는 `Features` 에 정확히 한 번 오르고 `Name` 이 패키지 이름이다 | `TestEveryFeatureListed`, `TestFeatureNames` |
| 기능 패키지는 루트를 import 하지 않는다(간접도). `core` 는 기능을 import 하지 않는다. `common` 은 `content/`, `platform/` 을 import 하지 않는다 | `TestFeaturesLiftable` |
| 패키지는 `layers` 의 아래 칸만 import 하고, 같은 칸의 기능 import 는 `siblings` 가 허용한 것뿐이다 | `TestLayers`, `TestFeatureSiblings` |
| 기능 하나가 목록과 자기 패키지만으로 이벤트를 받는다 | `core.TestNewFeatureNeedsOnlyTheList` |
| 플레이어 핸들러의 모든 메서드가 `core.PlayerHandler` 에 있고 훅마다 구독자가 있으며, 루트가 같은 이름의 메서드를 두지 않는다 | `TestHandlerMethodsOneWay` |
| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다 | `core.TestEmitAllocatesNothing`, `core.TestEventsEmitAllocateNothing` |
| 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름·우선순위 겹침은 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSamePriorityPanics`, `core.TestSameNamePanics` |

핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse` 가 잰다.

```sh
go test -run xxx -bench 'HandleMove|Second|UseOnBlock|Use$' -benchmem -count 5 ./content/skygear
go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
```

구독자는 하나에 대략 3~10ns 가 든다. 할당 수와 바이트 수는 늘지 않아야 한다.

## 핀

구독 순서와 클라이언트가 받는 데이터는 골든 파일로 고정한다.

| 파일 | 담는 것 | 테스트 |
|---|---|---|
| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 도는 순서대로 `<훅> <우선순위> <이름>` | `TestPinOrder` |
| `content/skygear/testdata/pin/handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden` | 장면마다 이벤트가 거친 구독자 이름과 결과(취소 여부, 효과) | `TestPinSimple`, `TestPinBlocks`, `TestPinUse` |
| `content/skygear/testdata/pin/lifecycle.golden` | 입장·퇴장·매초·빠른 틱·새 날·자동 저장·닫기·시작의 구독자 이름과 결과 | `TestPinLifecycle` |
| `content/skygear/testdata/pin/settings.golden` | 설정이 등록된 순서 | `TestPinSettings` |
| `content/skygear/wire/testdata/*.golden` | 접속한 플레이어가 받는 블록 항목, 아이템 등록, 제작 데이터, 창작 목록의 항목별 해시 | `TestWire` |

```sh
go test ./content/skygear -run 'TestPin' -count=1     # 구독 순서와 장면
go test ./content/skygear/wire/                        # 클라이언트 데이터
```

골든이 달라지면 테스트가 어느 줄(장면)이 어떻게 달라졌는지 `want` 와 `got` 로 보여 준다. 디스패치 순서나 클라이언트 데이터가 바뀌면 골든 차이로 나타난다.
골든은 동작을 바꾸려는 변경에서만 다시 기록한다.

```sh
SKYGEAR_GOLDEN=update go test ./content/skygear -run 'TestPin' -count=1
SKYGEAR_GOLDEN=update go test ./content/skygear/wire/
```

다시 기록한 골든은 `git diff` 로 의도한 줄만 바뀌었는지 확인하고, 커밋 메시지에 왜 바꿨는지 적는다. 테스트를 통과시키려고 다시 기록하지 않는다.
