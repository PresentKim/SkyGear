# 기계

기획: [02-skygear.md §11](../design/02-skygear.md). 코드: `content/skygear/machine/machine.go`.

## 창 (Ore UI)

- 기계를 우클릭하면 Ore UI 인벤토리 화면(`machine/screen.go` `ShowMachine`)이 열린다. 레이아웃은 기계 정의(`Def`)만으로 정해진다: 왼쪽에 에너지 게이지(`EnergyCap` 이 있고 쓰거나 만드는 기계, 빨강)와 액체 게이지(`FluidCap`, 파랑), 그 오른쪽에 넣는 칸(`Input`, 9칸이면 3×3)·충전 칸(`Charge`), 진행 화살표나 링(`progressLook`, 표식이 있는 기계), 결과 칸(`Output`, 4칸이면 2×2), 아래에 인벤토리와 핫바다. 결과 줄 오른쪽 아래에 결과 모두 꺼내기, 인벤토리 오른쪽 위에 모두 넣기·있는 것만 넣기(`ui.TransferBar`)가 있다. 기계마다 따로 만든 화면이 없어 레이아웃을 고치면 모든 기계에 한 번에 들어간다.
- 화면 부품은 공용이다: 게이지 `ui.InvGauge`, 진행 `ui.InvProgress`(그림은 `progress_*`·`cycle_*` 아이템 텍스처), 버튼 줄 `InvRow.Bar`·`InventoryScreen.Tools`. 페이지는 `web/oreui/src/inventory/Gauge.tsx`, `ButtonColumn.tsx` 다.
- 열린 동안 기계 틱이 5틱마다 화면에 보이는 것(진행 10% 단위, 에너지·액체 5% 단위, 칸 변경, 막힘)이 바뀌었는지 보고 다시 그린다(`watchMachine`). 다른 화면으로 바뀌었으면 그리지 않는다(`ui.ShowingKey`).
- 아래의 상자 UI 레이아웃(`§d§m§<n>`, 진행·게이지 아이템)은 `/sgdebug ui` 의 창 미리보기(`previewWindow`)에만 남아 있다.

## 구조

- 블록 타입 `Machine{Kind}` 하나가 종류(`crusher`, `sieve`, `auto_sieve`, `auto_miner`, `harvester`, `alloy_smelter` …)마다 따로 등록된다.
  식별자는 `skygear:<종류>` 다. 모델·텍스처는 리소스팩 `assets/resourcepack/skygear/`(`models/blocks/`, `textures/`)에 있고, Go 에는 식별자·재질 이름만 둔다(`blockmodel/models.go`).
- 체 세 종류는 상태 하나(`skygear:look`, 끼운 메쉬 단계×내용물×높이의 번호 `000`~)가 있는 블록이라 메쉬(여섯 단계, 단계마다 색이 다른 `sieve_mesh_<단계>` 텍스처를 입힌 가로세로 막대 여덟 개)와 내용물·진행(8단계, 높이 6픽셀에서 1픽셀까지)이 모델에 보인다. 모델과 메쉬 텍스처는 Ex Nihilo Fabrico 의 체·메쉬 모델에서 `tools/fabricotex` 가 만든다(창 없음, [sieve.go](../../content/skygear/machine/sieve.go)). 상태를 하나로 둔 것은 클라이언트가 런타임 ID를 상태 순서대로 짝짓기 때문이다.
- 설치된 기계는 `machine.State`(포인터)를 가진다. Dragonfly 상자처럼 블록 값이 상태를 가리키고,
  블록 엔티티 NBT(`Items`, `Progress` …)로 월드와 함께 저장된다.
  설치할 때는 `HandleBlockPlace` → `machine.Place` 가 새 상태로 바꿔 놓는다.
- 기계 인벤토리는 27칸 `chestui.Storage` 그 자체다. 창의 칸이 곧 입력·출력 칸이고, 나머지는 회색 판유리(조작 불가).
  창 제목의 표식(`§d§m§<n>`)으로 리소스팩이 기계별 레이아웃을 골라 필요한 칸만 보여 준다([json-ui.md](json-ui.md)). 자동 농작물 수확기는 채굴기 레이아웃(`§d§m§3`)을 같이 쓴다. `§d§m§2`는 링·출력 4칸·게이지가 있는 레이아웃이다.
- 진행도 칸에는 진행 화살표 아이템(`progress_0`~`progress_10`)이 10% 단위로 들어간다.
- 26번 칸은 **정보 아이콘**(`skygear:machine_info`)이다: 이름은 기계 이름, 설명(툴팁)은 하는 일. 칸 배치가 아니라 창 오른쪽 위 모서리 위의 어두운 컨트롤 막대(`sg_machine.control_bar`, 메뉴 화면의 컨트롤 막대와 같은 모양)에 닫기 버튼 왼쪽으로 그려지고, 모든 기계 창에서 같은 자리에 있다.
- **모두 꺼내기·모두 넣기**는 모든 창고와 같은, 창 오른쪽 바깥의 세로 버튼 줄이다(common inventoryui `ui/sg_quick.json`, [chestui.md](chestui.md)). 클라이언트가 칸마다 빠른 이동(쉬프트 클릭)을 하므로 서버 버튼이 없다. 기계 창에서는 레이아웃마다 꺼낼 칸을 출력 칸으로 거르고(`sg_machine.json` `$sg_machine_outputs`, `quickbar_test.go` 가 `machine.Kinds` 와 맞는지 본다), 넣기는 핫바까지 옮긴다. 클라이언트는 빈 칸 아무 데나 넣으려 하므로, 입력이 아닌 칸에 놓이는 아이템을 서버가 취소하고 기계가 받는 것이면 같은 월드 작업으로 입력 칸에 채운다(`machineHandler.HandlePlace`, `putFrom`: 같은 종류가 있는 칸에 먼저, 없으면 빈 칸 하나에, 안 들어가는 것은 플레이어에게 남음). 손으로 하는 쉬프트 클릭도 같다.
- 진행 칸(화살표)에는 진행도만 쓴다. 기계 블록을 부수면 내용물과 함께 나온다. 여러 명이 같이 열어도 같은 내용이 보인다.
- 블록 `Tick`(20틱/초)으로 동작한다. 인스턴스가 활성인 동안만 돈다. 로드 반경 밖의 소유 청크는 청크별 로더로 계속 돈다.
- 진행도는 기계 창과 WAILA 에서, 체는 모델로 본다.

| 종류 | 입력칸 | 출력칸 | 진행도 | 주기 |
|---|---|---|---|---|
| 분쇄기 | 11 | 15 | 13 | 80틱 |
| 체(손) | 10(1개) | 없음(위로 떨굼) | 모델 | 손으로 4단계 |
| 합금로 | 10, 11 | 15 | 화살표(13) | 100틱, 에너지 20 RE/t, 창 `§d§m§6` |
| 몬스터 시뮬레이터 | 10(에센스) | 13~16 | 링(12) | 200틱, 에너지 40 RE/t, 창 `§d§m§2` |
| 유체 탱크 | 없음 | 없음 | 없음 | 창 없음, 양동이·액체 도관, 16,000 mB(`fluidmachine.go`) |
| 전기 도가니 | 10(조약돌) | 없음(탱크) | 없음 | 20틱, 30 RE/t, 용암 250 mB, 창 없음 |
| 용암 발전기 | 없음 | 없음 | 없음 | 80 RE/t, 2틱 1 mB, 창 없음 |
| 광석 세척기 | 10(원석) | 13~16 | 링(12) | 60틱, 40 RE/t, 물 250 mB → 가루 2개, 창 `§d§m§2`(양동이를 들고 우클릭하면 물이 찬다) |
| 에너지 셀 | 없음 | 없음 | 없음 | 창 없음, 100,000 RE, 도관·이웃과 주고받음 |
| 전기 화로 | 11 | 15 | 화살표(13) | 100틱, 30 RE/t, 창 `§d§m§1` |
| 아이템 흡수기 | - | 13~16 | 링(10) | 10틱마다 한 묶음, 5 RE/t, 창 `§d§m§3` |
| 자동 벌목기 | 10(묘목) | 13~16 | 링(12) | 60틱, 15 RE/t, 창 `§d§m§2` |
| 자동 제작기 | 0·1·2·9·10·11·18·19·20 | 6·7·15·16 | 링(13), 조합법 칸 4 | 40틱, 25 RE/t, 창 `§d§m§7` |
| 정수 추출기 | 0·1·2·9·10·11·18·19·20 | 6·7·15·16 | 링(13) | 레시피마다 160~320틱(`State.period`), 40 RE/t, 창 대신 정수 추출기 화면(`labui.go`)으로 정수를 고르고 넣고 뺀다. 고른 레시피의 재료만 받고 `Recipe` 에 정수 종류로 저장한다. 레시피는 `machinerecipe.Labs` |
| 자동 체 | 10 | 13~16(칸 4개, 호퍼·도관이 꺼냄; 가득 차면 멈춤) | 링 게이지(12) | 60틱 |
| 마법 체 | 10(블록), 11(에센스) | 13~16 | 링 게이지(12) | 90틱, 40 RE/t, 창 `§d§m§9`(`magicsieve_slots`). 에센스 칸은 `Def.Charge`, 역할 `roleCharge`. 에센스 하나가 24번 충전하고, 충전·몬스터는 `charges`·`mob`(시뮬레이터와 같은 NBT)에 저장된다. 테마 표는 `sievetable/magicsieve_tables.go`, 굴리기는 `magicsieve.go` |
| 자동 자원 채굴기 | - | 13~16 | 10 | 40틱 |
| 자동 농작물 수확기 | - | 13~16 | 10 | 40틱 |

### 정의, 플래그, 소유

- 종류마다 `Def` 하나가 `Kinds` 에 이름으로 들어 있고, `Order` 가 블록과 아이템을 등록하는 순서다. 다른 기능의 종류도 여기에 같이 적어서 등록 순서(클라이언트가 받는 아이템 id)가 설치된 기능에 따라 달라지지 않는다.
- 기계 틀이 종류에 따라 다르게 하는 일은 `Def` 의 플래그로 적는다. `NoDisplay` 는 계속 갱신할 창이 없는 종류(에너지 셀, 유체 탱크, 전기 도가니, 용암·태양광·수력 발전기, 서랍 종류 셋, 팅커 테이블, 이차원 에센스 수집기, 보구 제작대)이고 틱이 창의 진행, 에너지 게이지, 정보 아이콘을 갱신하기 전에 끝난다. `NoWindow` 는 클릭해도 아무 일이 없는 종류(저장망 케이블), `Cycle` 은 일이 있든 없든 주기가 도는 종류(수확기)다. 수확기 범위가 위아래로 움직이는 것은 `tier.go` 의 `rangeLifts` 표다.
- 다른 기능의 종류는 `Def.Owner` 에 그 기능 이름이 있다. 그 기능이 `Setup` 에서 `machine.Register(kind, machine.Behaviour{…})` 로 동작을 건넨다(각 기능의 `kinds.go`, 농장은 `farm/harvester.go` 의 `registerKinds`). `Behaviour` 의 필드는 `Click`, `Run`, `Insert`, `Peek`, `Extract`, `Accepts`, `Lines`, `Info` 이고, 비운 필드는 기계 틀이 자기 종류에 하는 대로 한다. 틀은 등록된 동작을 `Def.Hooks()` 로 읽는다.
- 기능 목록을 설치할 때 기계 기능의 `Check` 인 `CheckKinds` 가 `Order` 의 종류 가운데 `Owner` 가 있는데 등록되지 않은 것을 찾는다. 하나라도 있으면 `machine kinds without behaviour: …` 오류로 시작이 멈춘다. 검사가 끝난 뒤의 `Register` 는 패닉한다(`kinds.go`).

| 소유 | 종류 | 등록하는 동작 |
|---|---|---|
| `farm` | `harvester` | `Run` |
| `storage` | `drawer_controller`, `drawer_interface` | `Click`, `Insert`, `Peek`, `Extract`, `Lines` |
| `storage` | `drawer_trim` | `Click`(아무 일도 안 함), `Lines: machine.NoLines` |
| `dimension` | `essence_collector` | `Click`, `Info`(받는 몬스터) |
| `dimension` | `relic_table` | `Click`, `Lines: machine.NoLines` |
| `netstore` | `net_controller` | `Click`, `Run`, `Insert`, `Peek`, `Extract` |
| `netstore` | `net_cable` | 없음(`machine.Behaviour{}`) |
| `netstore` | `disk_drive` | `Accepts` |
| `netstore` | `net_terminal` | `Click`, `Insert`, `Peek`, `Extract` |
| `netstore` | `net_importer`, `net_exporter` | `Click`, `Run`, `Accepts` |
| `netstore` | `net_external` | `Click`, `Run` |
| `netstore` | `pattern_encoder` | `Click`, `Accepts` |
| `netstore` | `net_autocrafter` | `Run`, `Accepts` |
| 없음 | `tinker_table` | `machine/tinkertable.go` 가 정한다(`Click`, `Lines: NoLines`) |

팅커 테이블은 `tinker` 가 `machine` 을 import 할 수 없어서 `machine` 이 직접 정한다. 그렇게 하면 기계 아이템이 팅커 도구보다 먼저 등록되어 클라이언트가 받는 아이템 id 가 바뀐다. `machine` 이 기능 칸의 패키지를 import 하는 곳은 이 파일 하나다.

### 클라이언트 모양

종류마다 모양이 하나다(`shape.go` 의 `shapeOf`). 정육면체(`shapeCube`)는 면마다 텍스처를 붙이고 일부는 단계나 작동 모습을 상태로 둔다(`tierlook.go`, `animate.go`). 체(`shapeSieve`)는 메쉬 위의 내용물을 상태로 두는 체 모델이다(`sieve.go`). 저장망 블록(`shapeNet`)은 팔과 방향을 상태로 두는 얇은 블록이다(`netshape.go`). 상태, 순열, 기본 속성, 추가 컴포넌트가 모두 이 모양에서 나온다.

## 아이템 이동

- 호퍼: 위·옆 호퍼 → `InsertItem`(입력에 받을 수 있는 아이템만 1개씩), 아래 호퍼 ← `ExtractItem`(출력에서 1개씩).
- 받을 수 없는 아이템(분쇄기에 흙 등)은 호퍼에 남는다.
- 기계는 결과를 이웃에 밀어 넣지 않는다. 결과 칸에 쌓이고, 아래 호퍼나 어느 면의 아이템 도관이 꺼낸다.
- 체의 부산물은 칸에 두지 않는다: 손 체는 위로 떨군다. 자동 체는 창의 출력 칸 4개에 쌓고(손 체 클릭으로는 진행하지 않는다, 클릭하면 창이 열린다), 칸에 자리가 없으면 멈춘다. 창 배치는 `sieve_slots`(제목 표식 §d§m§2)다: 입력 10, 링 12, 출력 13~16, 에너지 게이지 17. 출력 칸에 자리가 없으면 걸러 낸 물건은 기계 안(pending)에 두고 자리가 날 때까지 기다린다(부술 때 같이 나온다). 종류가 4개보다 많은 체 표도 돈다.
- 자동 농작물 수확기: 같은 높이 9×9에서 다 자란 밀·당근·감자·비트를 거둬 다시 어린 작물로 두고, 수박·호박 블록을 자르고, 핀 마법 식물을 거둔다.
- 자동 자원 채굴기: 캐는 동안 주변 플레이어에게 그 블록이 금 가는 모습(`StartCrackAction`, 남은 시간)을 보내고 다 캐면 부서지는 파티클을 낸다.
  아래 블록이 일반 블록(블록 엔티티·액체·부술 수 없는 블록 제외)이면 돌곡괭이로 캔 결과를 출력 칸에 넣는다.

## 기계의 한 틱과 강화 단계

- `Machine.Tick`이 상태를 `factory.machines`(`FactoryOf(s)`)에 올리고 `st.Stuck`을 지운 뒤, `st.Reps()`번 `runMachine`(종류별 동작)을 돌린다. 막히거나(`Stuck`) 아무 진전이 없는 반복은 멈춘다. 에너지는 `TickWork()`가 `consume()`으로 쓴다.
- 강화 단계(`State.Tier`, NBT `Tier`)는 `tier.go`의 `TierSpecs`가 정한다: 반복 횟수(속도), 작업당 에너지(`useCost`), 버퍼(`EnergyCap`), 발전량·연료 속도, 범위(`Area`). 발전기·에너지 셀은 반복하지 않는다.
- 출력은 `Emit`·`RoomFor`를 쓴다([monitor.md](monitor.md)).
- 마법 체는 자동 체 코드(`runAutoSieve`)를 같이 쓴다.

## 검증

- 단위: `machine/machine_test.go`(분쇄·자동 체·채굴·창 조작 제한·NBT 왕복·부술 때 드롭), `machine/sieve_test.go`, `farm/harvester_test.go`, `tinker/hammer_test.go`. 호퍼가 낀 무인 라인(`TestHandsFreeLine`)과 도감·자동 채움이 낀 체 테스트는 루트의 `machine_test.go`, `sieve_test.go` 에 있다.
- 종류 등록: `machine/kinds_test.go` 가 등록되지 않은 종류의 오류, `Register` 가 패닉하는 경우, `Order` 와 `Kinds` 가 같은지, 플래그가 붙은 종류, 종류마다 모양이 하나인지 본다. 루트의 `kinds_test.go`(`TestKindOwners`)는 종류마다 소유 기능이 `kindOwners` 와 같고 목록에 있으며 모두 등록되었는지 본다.
- 핀: 루트의 `pin_kinds_test.go`(`TestPinKinds`, `testdata/pin/kinds.golden`)가 종류마다 서서·웅크려 클릭한 결과, 창의 정보 아이콘, 창 칸의 역할과 받는 아이템을 고정한다. `TestForeignKindsRoundTrip` 은 다른 기능의 종류가 이름으로 저장되고 다시 읽히는지 본다.
- 실기 확인 항목은 [TODO](../TODO.md).

## 주기형 기계의 링 게이지

자동 농작물 수확기(2초)처럼 주기마다 반복하는 기계는 진행 칸에 화살표 대신 **채워지는 빈 원(링)**(`skygear:cycle_0`~`cycle_10`, `textures/items/cycle_*.png`)을 보인다. 주기는 할 일이 있든 없든 계속 돈다(수확할 작물이 없어도 100%에서 멈추지 않는다). 진행 칸 아이템은 10%마다 바뀌어 링이 도는 것처럼 보인다.

## 정육면체 블록 (모델링 없음)

자동 자원 채굴기·자동 농작물 수확기는 모델 없이 바닐라 정육면체(`minecraft:geometry.full_block`)에 텍스처를 붙인다(`blockmodel/models.go` `CubeProperties`).
텍스처는 `textures/blocks/<이름>_<side|up|down>.png` 3장: 네 옆면은 `side` 하나를 같이 쓴다(방향 없음).
이름은 `auto_miner`, `harvester`다. 면 그림만 바꾸려면 PNG를 교체한다.
놓는 방향으로 돌지 않는다.

분쇄기도 모델 없는 정육면체지만 정면이 따로 있다(`blockmodel.FrontCubeProperties`): `textures/blocks/<이름>_<front|back|side|up|down>.png` 5장, 정면 **남쪽**·후면 북쪽·좌우 두 옆면은 `side` 공통. 블록 아이템은 인벤토리에서 남쪽 면이 보이게 그려지므로 정면을 남쪽에 둔다(서랍은 기본 상태를 남쪽으로: `storage.DrawerFacings`). 블록 아이템은 인벤토리·폼에서 블록 모양 그대로 보이므로 아이템 아이콘 텍스처는 두지 않는다.

## 에너지 게이지

에너지를 담는 기계의 창에는 **에너지 게이지**가 있다: 창 오른쪽 열(슬롯 17, `EnergyGaugeSlot`)에 건전지 그림이 10단계로 차오르고(`EnergyGauge`, `textures/items/energy_gauge_N.png`, 임시 그림), **마우스를 올리면** "에너지 보유량/최대량 RE"가 뜬다(`State.ShowEnergy`, 0.5초마다 갱신). 레이아웃은 `ui/sg_machine.json`의 `s17@sg_machine.indicator` 다. 운영자 시험 공장(`unlimited`)의 기계는 가득 찬 것으로 보인다.
