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
| `core` | `content/skygear/core` | 기능의 커널. `Session`과 그 파트(`Part`, `Players`, `Saved`), 훅(`Hook`)과 이벤트, 핸들러(`PlayerHandler`, `WorldHandler`), 수명 호출(`Lifecycle`), 기능 서술자(`Feature`)와 `Install`, 창작 목록 순서(`CreativeItem`, `CreativeItems`), 기능의 명령어(`Commands`, `PlayerSession`), 플레이어를 붙잡는 모드(`RegisterTakeover`, `Takeover`) |
| 기능 패키지 | `content/skygear/<기능>` | 한 기능의 블록·아이템·화면·규칙과, 그 기능이 훅에 거는 구독자 |
| 루트 | `content/skygear` | 기능 목록(`features.go`)과 플랫폼 접착제(`Game`, `session`), 운영자 도구 |

`content/skygear` 안의 패키지는 `content/skygear/layers_test.go` 의 `layers` 에 아래에서 위로 적혀 있다.

- 데이터: `locales`, `naming`, `textfmt`, `formicon`, `blockmodel`, `netshape`, `sgitem`, `sievetable`, `wintext`, `machinerecipe`, `advancement`, `hopper`, `migrate`, `wire`
- 핵심: `core`, `network`, `achievement`, `preview`, `look`, `autofill`, `decor`
- 진행: `progress`, `island`, `climate`
- 기능: `tinker`, `machine`, `conduit`, `storage`, `farm`, `fairy`, `dimension`, `handwork`, `expand`, `copier`, `wand`, `netstore`, `elevator`
- 내용 목록: `catalog`
- 화면: `codex`, `screens`

`look`(WAILA 패널), `autofill`(손에 든 묶음 다시 채우기), `decor`(아이템 칸 장식)는 핵심 칸에 있어서 기능 칸의 패키지가 셋을 부를 수 있다.

루트 패키지에 남은 파일은 다음과 같다.

| 파일 | 하는 일 |
|---|---|
| `features.go` | 기능 목록 `Features` 와 `core.Install`. 루트 자신의 기능 `itemsFeature` 는 아이템 항목의 이름을 번역 키로 바꾸고(`naming.LocalItemName`) 클라이언트가 빈 나무 양동이로 액체를 겨누게 한다(`sgitem.ClipWoodenBucket`) |
| `game.go` | `Game` 과 `session`. 플랫폼의 `instance.Game`·`instance.Session` 을 구현한다. `New` 가 기능의 명령어(`core.Commands()`)와 창작 목록(`catalog.RegisterCreativeItems`)과 운영자 명령을 등록한다 |
| `state.go` | 세이브 상태 문서의 버전(`State`, `core.NewSaved`) |
| `hopper.go` | `hopper` 는 기능이 아닌 데이터 칸 패키지라 루트가 `init` 에서 호퍼 블록과 아이템을 등록한다(`hopper.Register`) |
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

   팅커 테이블은 `Owner` 가 비어 있고 `machine/tinkertable.go` 가 동작을 정한다. `tinker` 가 `machine` 을 import 하면 `machine` 이 `tinker` 보다 먼저 초기화되어 기계 아이템이 팅커 도구보다 먼저 등록되고, 클라이언트가 받는 아이템 id 가 바뀌기 때문이다. 루트가 `hopper` 를 등록하는 것(`hopper.go`)과 같은 사정이다.
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
14. 구독자를 달았으면 골든이 달라진다. `order.golden` 에는 새 줄(`<훅> <우선순위> <이름>`)이 생기고, 그 훅을 거치는 장면이 있으면 `handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden`, `lifecycle.golden` 의 그 장면 줄에도 새 이름이 들어간다. 의도한 변경이므로 `SKYGEAR_GOLDEN=update go test ./content/skygear -run TestPin -count=1` 로 다시 기록하고 `git diff` 로 새 단계의 이름만 늘었는지 확인한다. 커밋 메시지에 어느 단계를 더했는지 적는다.
15. `go test ./content/skygear -run 'TestLayers|TestFeature|TestCapability|TestLook|TestCreative|TestItemDecor|TestEvery|TestRoot|TestPin|TestHandlerMethodsOneWay|TestKind|TestMachine' -count=1` 로 확인한다. 위 순서를 빠뜨리면 이 테스트들이 `TestLayers`(`package <기능> is not in layers`), `TestFeatureNames`, `TestPinOrder`, `TestEverySubscriberHasItsFeature`, `TestPinSimple`, `TestCreativeListNamesNoFeature`, `TestFeatureCommands` 따위로 어긋난 곳을 알려 준다.

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
- `Leave` 에서 `dimension.leave`(10)와 `expand.stop`(15)은 `core.parts`(20)보다 앞이다. `expand.stop` 은 `expanding` 항목에 둔 게임 모드를 되돌리므로 항목이 지워지기 전에 돈다. `core.parts` 는 `Players` 의 플레이어 항목을 지우므로, 그 뒤(`look.clear` 40 등)에 도는 구독자는 `Players` 에서 그 플레이어의 항목을 찾지 못한다. 플레이어별 데이터를 읽어야 하는 퇴장 단계는 20 앞에 둔다.
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

### 플레이어를 붙잡는 모드

한 기능이 플레이어의 게임 모드, 카메라, 몸을 맡아 두었다가 모드가 끝날 때 돌려주는 모드를 붙잡는 모드(takeover)라 한다. 섬 확장 모드(`expand`), 사냥(`dimension.hunt`), 사냥 관전(`dimension.watch`)이 그렇다. 둘은 겹치지 않는다. 저마다 시작할 때 돌려줄 것을 저장해 두므로, 겹치면 끝날 때 엉뚱한 모드로 돌아간다.

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
- 기능 셋 이상이 설치되어야 통과하는 테스트, 루트가 등록하는 호퍼(`hopper.go`)가 필요한 테스트, 자기가 import 하지 않는 기능이 등록한 아이템이 필요한 테스트.
- 루트의 내보내지 않은 이름(`Game`, `session`, `giveItems` 등)을 쓰는 테스트와 런타임 id 를 보는 테스트.

루트의 `helpers_test.go` 는 같은 도우미를 루트의 `session` 위에서 준다(`testSession`, `loadAround`, `spawnPlayer`, `withPlayer`, `tick`, `placeTestMachine`, `machinePos`). `testSession` 은 데이터베이스와 `Game` 도 만든다.

지킬 것이 셋 있다.

- 패키지의 테스트는 루트 패키지를 import 하지 않고, 내용 목록 아래 칸의 패키지 테스트는 `catalog`, `codex`, `screens` 를 import 하지 않는다(`TestPackageTestsStayBelow`).
- 테스트 지원 패키지는 테스트 파일만 import 한다(`TestSupportPackagesAreTestOnly`).
- `dimension` 의 테스트 바이너리에서는 `popup_wire_test.go` 가 서버에 접속해 `testdata/popup.golden` 의 아이템 id 를 본다. 이 패키지의 테스트에 블록이나 아이템을 등록하는 패키지를 새로 import 하면 id 가 바뀐다.

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
| 루트는 아무것도 구독하지 않는다. 루트 패키지의 비테스트 파일에는 훅의 `On`·`Handle` 호출, `core.Mark` 호출, `Handle...` 메서드가 없다 | `TestRootSubscribesToNothing` |
| 모든 구독자는 목록에 오른 기능이 `Setup` 에서 단다. 이름의 접두어가 그 기능 이름이다 | `TestEverySubscriberHasItsFeature`, `core.TestInstallingPrefix` |
| `feature.go` 가 있는 패키지는 `Features` 에 정확히 한 번 오르고 `Name` 이 패키지 이름이다 | `TestEveryFeatureListed`, `TestFeatureNames` |
| 기능 패키지는 루트를 import 하지 않는다(간접도). `core` 는 기능을 import 하지 않는다. `common` 은 `content/`, `platform/` 을 import 하지 않는다 | `TestFeaturesLiftable` |
| 패키지는 `layers` 의 아래 칸만 import 하고, 같은 칸의 기능 import 는 `siblings` 가 허용한 것뿐이다. `featureLayer` 는 `layers` 의 기능 칸과 같다 | `TestLayers`, `TestFeatureSiblings` |
| `look` 은 `core`, `naming` 만, `decor` 는 `naming` 만 import 한다. 블록과 아이템은 인터페이스로 자기를 설명한다 | `TestCapabilityImports` |
| `machine` 은 기능 칸의 패키지를 import 하지 않는다. 예외는 `machine/tinkertable.go` 의 `tinker` 하나다(`machineFeatureImports`) | `TestMachineImportsNoFeature` |
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
| 핫 경로는 할당하지 않는다. 이벤트마다 `AllocsPerRun` 으로 0 을 확인한다 | `core.TestEmitAllocatesNothing`, `core.TestEventsEmitAllocateNothing` |
| 구독자는 `Install` 동안에만 단다. 첫 `Emit` 뒤의 등록, 같은 훅 안의 이름·우선순위 겹침은 패닉이다 | `core.TestRegisterAfterEmitPanics`, `core.TestSamePriorityPanics`, `core.TestSameNamePanics` |

핫 경로(이동, 매초, 빠른 틱)를 건드리면 벤치마크로 재 본다. 이벤트 하나의 비용은 `core` 의 `BenchmarkEmit`, `BenchmarkEmitMove` 와 루트의 `BenchmarkHandleMove`, `BenchmarkSecond`, `BenchmarkUseOnBlock`, `BenchmarkUse` 가 잰다. 패널 하나의 비용은 루트의 `BenchmarkLookMachine`(시선 추적과 패널 보내기), `BenchmarkLookInfoMachine`, `BenchmarkLookInfoStone`(패널 만들기)이 잰다. 다른 기능이 동작을 등록한 기계의 틱 한 번은 `BenchmarkTickHarvester`, `BenchmarkTickDrawerController` 가 잰다.

```sh
go test -run xxx -bench 'HandleMove|Second|UseOnBlock|Use$|LookMachine|LookInfo|TickHarvester|TickDrawerController' -benchmem -count 5 ./content/skygear
go test -run xxx -bench Emit -benchmem -count 5 ./content/skygear/core
```

구독자는 하나에 대략 3~10ns 가 든다. 할당 수와 바이트 수는 늘지 않아야 한다.

## 핀

구독 순서, 패널, 기계 종류의 동작, 창작 목록, 클라이언트가 받는 데이터는 골든 파일로 고정한다.

| 파일 | 담는 것 | 테스트 |
|---|---|---|
| `content/skygear/testdata/pin/order.golden` | 모든 훅의 구독자를 도는 순서대로 `<훅> <우선순위> <이름>` | `TestPinOrder` |
| `content/skygear/testdata/pin/handler-simple.golden`, `handler-blocks.golden`, `handler-use.golden` | 장면마다 이벤트가 거친 구독자 이름과 결과(취소 여부, 효과) | `TestPinSimple`, `TestPinBlocks`, `TestPinUse` |
| `content/skygear/testdata/pin/lifecycle.golden` | 입장·퇴장·매초·빠른 틱·새 날·자동 저장·닫기·시작의 구독자 이름과 결과 | `TestPinLifecycle` |
| `content/skygear/testdata/pin/settings.golden` | 설정이 등록된 순서 | `TestPinSettings` |
| `content/skygear/testdata/pin/look.golden` | 장면마다 바라본 블록의 패널 한 줄(이름, 번역 키, 아이콘, 줄, 아이템 줄, 보관 칸, 도구, 표시, 출처), 143개 | `TestPinLook` |
| `content/skygear/testdata/pin/kinds.golden` | 기계 종류마다 패널 밖에서 하는 일: 서서·웅크려 클릭한 결과와 열린 화면, 창의 정보 아이콘, 창 27칸의 역할과 칸이 받는 견본 아이템, 145줄 | `TestPinKinds` |
| `content/skygear/testdata/pin/creative.golden` | 게임이 더한 창작 목록 항목을 클라이언트가 보이는 순서대로 그룹과 함께, 243개 | `TestPinCreative` |
| `content/skygear/wire/testdata/*.golden` | 접속한 플레이어가 받는 블록 항목, 아이템 등록, 제작 데이터, 창작 목록의 항목별 해시 | `TestWire` |

```sh
go test ./content/skygear -run 'TestPin' -count=1     # 구독 순서, 장면, 패널, 기계 종류, 창작 목록
go test ./content/skygear/wire/                        # 클라이언트 데이터
```

골든이 달라지면 테스트가 어느 줄(장면)이 어떻게 달라졌는지 `want` 와 `got` 로 보여 준다. 디스패치 순서나 클라이언트 데이터가 바뀌면 골든 차이로 나타난다.
골든은 동작을 바꾸려는 변경에서만 다시 기록한다.

```sh
SKYGEAR_GOLDEN=update go test ./content/skygear -run 'TestPin' -count=1
SKYGEAR_GOLDEN=update go test ./content/skygear/wire/
```

다시 기록한 골든은 `git diff` 로 의도한 줄만 바뀌었는지 확인하고, 커밋 메시지에 왜 바꿨는지 적는다. 테스트를 통과시키려고 다시 기록하지 않는다.
