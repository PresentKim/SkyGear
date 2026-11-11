# WAILA

바라보는 블록의 이름·아이콘·정보를 화면 위에 띄우는 HUD 패널이다. 리소스팩(`assets/resourcepack/waila/`)이 그리고, 서버(Go)가 데이터를 제목 문자열로 보낸다. Dragonfly 는 Script API 를 실행하지 않으므로 시선 추적·제목 만들기·보내기는 서버가 한다. 엔티티 표시는 필요해지면 더한다.

## 구성

- `common/waila`: 패널 값(`Info`), 제목 형식(`Format`), 보내기(`Show`: 바뀐 것만, 30초마다 다시; `Clear`), 도구 아이콘(`ToolsFor`: Dragonfly `BreakInfo` 로 효과 있는 도구와 최소 등급, 손에 든 것이 맞으면 `a`·필요한데 틀리면 `b`), 부제목 꼬리(`tail.go`). 블록이 자기 줄을 채우는 방법도 여기 있다(`provider.go`: `View`, `Provider`, `Adapter`, `RegisterAdapter`, `Describe`).
- 리소스팩 `assets/resourcepack/waila/`: 아이콘을 부제목에서 읽는 `sg_subtitle_item_renderer`, `tile.json`(`sg_marks`·`sg_item_line`·`item_inventory`), `main.json`(위치, `sg_name_row`).
- 스카이기어 `content/skygear/look/look.go`: 시선 추적과 기본 패널(`Info`), 숨기기(`Hidden`)와 따라가기(`Follow`).
- 블록의 줄은 그 블록의 패키지에 있다: `machine/look.go`, `storage/look.go`, `farm/look.go`, `hopper/look.go`, `conduit/look.go`.
- 확인: `common/waila/waila_test.go`, `common/waila/provider_test.go`, `content/skygear/look_test.go`, `content/skygear/pin_look_test.go`(`TestPinLook`, 골든 `testdata/pin/look.golden`).

## 패널을 만드는 차례

`look.Info` 가 바라본 블록 하나의 패널을 만든다.

1. `look` 이 기본 값을 채운다: 이름(`naming.BlockName`, 아이템이기도 한 블록은 `naming.ItemName`)과 번역 키(`naming.BlockNameKey`), 아이콘(블록 아이템, 불은 부싯돌과 부시), 도구 아이콘(`waila.ToolsFor`), 바닐라 보관 블록의 내용(상자·통·셜커 상자, 화로류, 호퍼, 양조기).
2. `waila.Describe` 가 블록에 묻는다. 블록에 `Waila(v, info)` 메서드가 있으면(`waila.Provider`) 그 메서드가 줄과 아이템 줄과 보관 칸을 채운다. `v`(`waila.View`)는 보는 사람의 언어, 웅크림 여부, 트랜잭션과 위치다. 세션의 데이터가 필요한 블록은 `core.At(v.Tx)` 로 세션을 찾는다.
3. 이어서 `waila.RegisterAdapter` 로 단 어댑터가 단 순서대로 돈다. 어댑터는 `Waila` 메서드가 없는 블록(바닐라 블록)을 채우고 다른 블록에는 값을 그대로 돌려준다. 어댑터는 기능의 `Setup` 에서 달리므로 기능 목록 순서로 돈다. 지금은 `machine` 의 어댑터(화로·용광로·훈연기의 진행도 화살표)가 먼저, `farm` 의 어댑터(퇴비통 내용, 작물 이름과 시간)가 다음이다.

`look` 은 게임 패키지 가운데 `core` 와 `naming` 만 import 한다(`TestCapabilityImports`). 블록 종류를 아는 코드는 `look` 에 두지 않고 블록의 패키지에 둔다.

## 숨기기와 따라가기

- `look.Hidden` 이 true 인 플레이어는 패널이 비워지고 갱신되지 않는다. 섬 확장 모드의 `expand` 가 이 함수를 채운다.
- `look.Follow` 로 단 단계는 패널을 갱신할 때마다 단 순서대로 돈다. 지금은 `tinker.vein`(빠른 틱마다 광맥 윤곽), `wand.preview`(건축 지팡이 미리보기) 순서다(`TestLookFollowers`). 단계는 `look.Target`(세션, 플레이어, 트랜잭션, 바라본 블록, 빠른 틱 갱신인지)을 받는다.
- 움직임으로 갱신할 때 패널이 숨겨져 있으면 따라가는 단계도 돌지 않는다. 빠른 틱의 갱신에서는 패널이 숨겨져 있어도 돈다(`TestLookExpandingWithTool`).

## 제목과 부제목

- 접두어·토큰은 `_waila:` / `__waila:` 다. 팩의 제목 위치 자르기(`'%.Ns' * $var_title_text`)는 접두어 길이에 맞춘다. `'%.Ns'` 는 바이트로 자른다.
- **제목**: `_waila:A:` + 도구 칸 10자(`aa,a;zz,z:`) + `§f이름` + `\n§7정보 줄…` + `__waila:anchor.top_middle__`. 출처 줄(스카이기어/Minecraft)은 쓰지 않는다.
- **부제목**: `i` + 아이콘 칸 11자(아이템 런타임 ID × 65536 을 왼쪽에 `x` 로 채움, 빈 것은 `xxxxxxxxxx0`) + (보관 블록이면 칸들) + `e`. 숫자만 보내면 팩이 부제목을 문자열로 다루는 블록 상태 검사가 깨져(회색 칸, "expected a string" 경고) 문자열 형태를 지킨다. 블록 상태 칸(`main.json` `block_states`)은 쓰지 않아 끈다.
- 팩은 제목 팩토리가 주는 **변수** `$var_subtitle_text` 에서 `'%.11s'` 로 칸을 잘라 `x` 를 지우고 `property_bag` 의 `#item_id_aux` 에 넣는다. 패널 글자도 모두 이 변수로 그린다.
- 부제목을 제목보다 먼저 보낸다(`sghook.SendTitle`). 일반 제목(레벨 업·긴급 매입)은 `waila.SendTitle` 로 보내고, 그 제목이 끝날 때까지 패널을 멈춘다.
- 위치: 위 가운데, 보스바 아래로 26 내림(`main.json`). 이름 줄 최소 높이는 10이다.

### 부제목 꼬리 (`common/waila/tail.go`)

칸 28개 뒤에 줄 아이콘 칸(11자), 표시 4자(`L`/`-`, 용량 업그레이드 `a`+단계, `V`/`-`, 줄 게이지 `a`+10분의 몇 또는 `-`), 줄 글자 24바이트(`~` 로 채움, `§f` 로 시작), 나머지가 줄 이름이다. 한글은 줄 이름에만 둔다(팩이 위치로 자르므로 앞쪽은 모두 ASCII). 팩은 `tile.json` `sg_marks`(이름 옆)와 `sg_item_line`(제목에 `__waila:sg.line__` 이 있을 때)으로 그린다.

### JSON UI 에서 피할 것

- **바인딩** 식(`#hud_subtitle_text_string`)으로 칸을 꺼내면 어떤 형식(자르기, 고정 폭, 더하고 빼기, `x` 채우고 지우기)으로도 아이콘이 비어 있다. 변수 `$var_subtitle_text` 를 쓴다.
- 숫자로 바뀐 값은 글자(label)에 보이지 않는다(`" 219"`, `"1,532/4,096"`). 수량·줄 글자는 `§f` 로 시작하는 바이트 칸으로 보내고, 변수 식 안에서 `'§r' + 값` 으로 문자열로 만든다. 개수 식은 라벨 자신의 변수 블록에 둔다(부모 패널의 식이나 text 안의 `('§r' + $변수)` 는 계산되지 않는다).
- 숫자 한 글자 플래그는 팩에서 숫자가 되어 글자 비교가 맞지 않는다. 플래그는 글자(`a`~`k`)로 쓰고, 변수 블록에 식이 아닌 고정 글자로 고른다(변수 블록의 식을 다른 식 안에 쓰면 계산되지 않는다).
- 바인딩 식이 `#바인딩` 을 하나도 쓰지 않으면 "Must define a source property name" 오류가 난다.

## 시선 추적 (`look/look.go`)

- 움직이거나 돌 때(0.1초에 한 번까지)와 0.2초마다(`instance.Session.Fast`, `instance.FastInterval`, 게이지가 끊기지 않는 주기) 눈에서 6블록까지 광선 검사를 한다(공기·액체 제외). 섬 확장 모드에서는 패널을 숨긴다(`look.Hidden`).
- 작물은 단계에 따라 높이가 자라는 상자((단계+1)/8)로 판정해, 낮은 작물 위 빈 곳을 지나는 시선은 뒤 블록을 본다(클라이언트와 같다). 충돌 상자 없는 다른 블록(꽃 등)은 칸 전체로 판정한다.
- 우리 블록도 재료에 맞는 도구로 빨리 캐지므로(`blockmodel.AxeEffective`·`PickaxeEffective`·`ShovelEffective`) 바닐라 블록처럼 `ToolsFor` 가 도구를 보인다(나무로 만든 서랍·연구대·나무 배럴·나무 호퍼·체는 도끼, 기계·돌 배럴·철 호퍼·도관은 곡괭이, 먼지·부순 블록은 삽, 들고 있으면 ✓).

## 정보 줄

- **기계**(`machine/look.go`): 아이템 줄 "흙 [아이콘] ×3 ◔"(넣은 아이템·개수·진행 게이지, 비면 회색 "비어있음"), 결과 칸에 든 것은 "결과" 줄. 진행·채굴·수확 주기는 원형 게이지 글리프(`glyph_E1` 둘째 줄 U+E110~E11A 의 `cycle_0..10`, 칸의 약 60%인 10px)로 보인다. 웅크리면 작업 주기 길이와 남은 틱 줄을 더하고 기계의 범위를 잠시 보인다.
- **서랍**(`storage/look.go`): 아이콘은 서랍 블록, 든 아이템은 칸 하나에 아이콘과 그 아래 개수, 비면 회색 "비어있음". 잠금·업그레이드는 서랍 앞면에 보이고, 웅크리면 용량 업그레이드·공허·잠금 줄이 패널에도 나온다.
- **작물**(`farm/look.go` 의 어댑터): "남은 시간"·"다음 열매"만(단계 숫자 없음). 작물의 남은 시간은 패널에만 나온다.
- **보관 블록 내용**: 상자·통·셜커 상자(같은 아이템을 묶어 한 줄 9종류씩 세 줄 27종류까지), 자동 자원 채굴기(결과 1칸), 화로류(입력·연료·결과, 화살표 자리에 진행도 화살표 `progress_0..10`, 불이 꺼져 있으면 빈 화살표), 호퍼, 양조기.
  - 제목에 `__waila:inv.<chest|furnace|hopper|brewstand>__` 를 붙이고, 부제목 아이콘 칸 뒤에 패널 칸 28개(8번은 아이콘 자리로 비움, 상자 줄은 0~7·9, 10~18, 19~27)마다 아이템 11바이트(아이콘과 같은 형식) + 수량 8바이트(`§f74` 처럼 색 코드로 시작해 `~` 로 채움, 1개면 `~` 만, 1만부터 `k`)를 붙인다.
  - `tile.json` `item_inventory` 는 플레이어 인벤토리 대신 부제목에서 칸을 읽는다(바인딩 식에 변수를 못 써서 칸 번호마다 고정 식을 생성해 변수로 골라 끼운다). 칸 10·19번이 비면 둘째·셋째 줄을 만들지 않는다(`ignored`). 칸 테두리는 없다.
