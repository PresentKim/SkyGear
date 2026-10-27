# Ore UI 화면 (DDUI)

팩을 암호화하면(`[ResourcePack] Encrypt`, 기본 켬) 섬과 로비의 메뉴 폼은 게임의 Ore UI에 그린 서버 주도 DDUI 화면으로 보인다. 로비의 메인 메뉴는 화면 전체를 채우는 메인 화면(`ui.HomeScreen`)이 된다. 수량·표 폼은 JSON UI(`docs/tech/json-ui.md`)로 남는다.

플레이어는 `/uimode` 로 자기 메뉴를 JSON UI 폼으로 바꾸거나 되돌린다. 계정 설정(`account_settings` 의 `ui_mode`, `"json"`)으로 남고, 기본값은 모든 기기가 같다(Ore UI). Ore UI 화면이 깨지는 기기(애플 기기 등)에서 두 방식을 견주어 보는 데 쓴다(`internal/platform/uimode.go`, `common/ui.PreferJSON`).

## 구성

- `common/packs/oreui`: Ore UI 게임 중 페이지(`hbui/gameplay.html`)를 바꾸는 팩. 바닐라 번들을 그대로 불러온 뒤 렌더러(`oui-ddui.js`)와 모양(`oui-mc.css`)을 더한다. 클라이언트는 이 페이지를 **암호화 팩에서만** 가져가므로, 암호화를 켤 때만 보낸다(`main.go`).
- `common/oreui`: DDUI 화면을 열고, 레이아웃과 값을 데이터 저장소로 보내고, 누름을 받는다. 클라이언트의 DDUI 패킷은 `dfhook`에서 가로챈다.
- `common/ui`: `ui.OreUI`를 켜면 `Menu`(`Confirm`), `CardMenu`, `ItemBoard`, `SplitMenu`, `SieveBoard`, `ListSplitMenu`, `Input`, `ItemGrid`, `ItemGridSections`, `SlotMenu`, `ChoiceMenu`가 같은 모양의 DDUI 화면이 된다(`ui/oreui.go`, `ui/oreforms.go`). 게임 코드는 바꾸지 않는다. 버튼의 `Run`도 폼과 같이 그 플레이어 월드에서 돈다. 누름 뒤에 새 화면을 보이지 않으면 화면이 닫힌다(폼 버튼을 누르면 폼이 닫히는 것과 같다).
- 창은 퀘스트 책까지 모두 바닐라 인벤토리 크기(326×166) 하나다. 그래서 한 화면에서 다른 화면으로 가는 누름(정수 추출기의 정수 고르기, 도움말 주제, 치장 탭, 친구 줄에서 확인 질문, 통계 탭)은 열린 화면 안에서 바뀐다.
- 같은 화면에 새 레이아웃이 오면 렌더러는 그려 둔 것과 견주어 달라진 노드만 고친다(`oui-ddui.js` `patch`). 개수·글·아이콘·고른 칸만 바뀌면 그 요소만 바뀌고, 창·스크롤 위치·입력칸·불러온 아이템 그림은 그대로다. 영역(`region`)도 값이 바뀌면 같은 방식으로 고친다. 게임 코드는 화면 전체를 다시 보이기만 하면 된다.
- 터치 화면에서는 툴팁이 있는 요소(아이템 칸, 도움말 `?`, 발전과제 노드, 버튼)를 처음 탭하면 툴팁만 뜨고, 같은 요소를 한 번 더 탭하면 눌린다(JSON UI 폼과 같다). 다른 곳을 탭하면 툴팁이 닫힌다. 툴팁이 이름뿐인 뒤로·쪽 넘김 버튼과 탭은 바로 눌린다(`common/ui.quick`, 노드의 `quick`). 마우스는 가리키면 툴팁이 뜨고 누르면 바로 눌린다. 툴팁 글자는 창 글자와 같은 크기·Minecraft 글꼴이고 터치 화면에서는 한 단계 크다. 오른쪽에 자리가 없으면 요소 왼쪽에 뜨고, 화면 밖으로 잘리지 않게 끌어온다.
- 입력 화면(`ui.Input`)은 창 안의 입력칸과 확인 버튼이다. 버튼을 누르면 화면의 입력칸 글이 함께 오고, Enter 도 확인을 누른다. 돌아갈 화면이 있으면 컨트롤 막대에 뒤로 버튼이 있다.
- 소셜 화면(친구·참여자 관리)은 목록 분할 화면(`ui.ListSplitMenu`)이다. 탭은 다른 화면과 같은 아이콘 탭(이름은 툴팁)으로 창 위에, 줄은 왼쪽 판에서 스크롤되고(이름과 그 아래 작은 정보 줄, 고른 줄은 밝게), 오른쪽은 본문과 동작 버튼(첫 버튼은 전체 폭, 나머지는 두 칸씩)이다. 확인 질문·친구 추가 입력도 같은 창에서 바뀐다.
- 선택 판(`ui.ChoiceMenu`, 요정의 마법판)은 위의 수치 판(아이콘과 값 한 줄, 이름은 툴팁, `ui.Stat`)과 한 줄 메모, 아래에 갈래(`ui.ChoiceGroup`)마다 한 줄: 왼쪽에 갈래 이름, 오른쪽에 아이콘 위·이름 아래의 칸(효과는 툴팁)이다. 지금 것(`On`)은 초록 테두리로 눌리지 않고 툴팁에 표시(`Badge`)가 붙으며, 지금 못 하는 것(`Dim`)은 흐리다. JSON UI 폼에서는 본문에 수치, 버튼마다 칸인 목록이다(`ui/choice.go`, `oui-mc.css` `.mc-pick*`·`.mc-stat*`).
- 도감의 체 탭은 Ex Deorum JEI 체 화면 배치로 그린다(`ui.SieveBoard`, `codexpages.go`).
- 도감 검색 탭은 창 안의 검색칸과 결과 한 판이다(`ui.SearchGrid`, `codexsearch.go` `showCodexSearchLive`). 글을 치면 250ms 뒤 결과 영역만 다시 그리므로 검색칸의 글과 커서가 남는다. 결과는 쪽 없이 스크롤된다. 빈 검색에서는 즐겨찾기와 최근 본 아이템이 위에 나온다.
- 화면은 키보드 키를 받을 수 있다(`ui.Key`, `oreLayout.key`). 레이아웃 루트의 `keys`(`{up, down, left, right, ok: id}`)가 방향키·WASD, Enter·스페이스바를 그 id 의 누름으로 보낸다(`oui-ddui.js` `onKey`, 같은 키는 120ms 에 한 번까지, 입력칸에 글을 칠 때는 받지 않는다). 섬 확장 화면(`ui.Plane.Keys`)이 청크 이동과 구매에 쓴다.
- 화면이 받지 않는 방향키·WASD 는 키보드 선택을 옮긴다(`oui-ddui.js` `navKey`). 누를 수 있는 요소(버튼, 칸, 탭, 컨트롤 막대 버튼) 가운데 그 방향으로 가장 가까운 것이 마우스를 올린 것처럼 보이고(`oui-kb`, CSS 의 `:hover` 마다 같은 규칙, 툴팁도 뜬다), Enter·스페이스바가 그것을 누른다. 처음 누른 방향키는 왼쪽 위의 요소를 고르고, 스크롤 판 안의 요소는 보이게 스크롤한다. 누른 뒤 화면이 바뀌어 고른 요소가 없어지면 그 자리에서 가장 가까운 요소가 골라진다. 마우스를 움직이면 선택이 사라진다.
- 아이템 도관 설정은 가운데 블록과 여섯 면의 카드를 선으로 이은 화면이다(`ui.SidesMenu`, `conduitform.go` `showConduitSides`). 카드 자리는 서버가 정하고(`sideCard`), 포트 줄과 고르기 판은 영역(`@side<n>`, `@picker`)이라 값이 바뀌면 그 부분만 다시 그린다. 채널은 아이템이 아닌 색 네모(`PortChoice.Colour`)로 그린다. 색 네모를 우클릭하면 이전 값이 된다(노드의 `alt`, `oreLayout.alt`).
- 발전과제는 ATM10 To The Sky의 FTB Quests 퀘스트 책처럼 그린다(`ui.QuestBookMenu`, `advancementui.go` `showQuestBook`). 왼쪽은 챕터 목록(달성 수), 오른쪽은 끌어서 움직이는 어두운 판 위의 노드와 선행마다 오는 꺾은선(열 사이 틈에서 꺾여 노드 위를 지나지 않는다)이다. 노드 자리는 트리 폼과 같고(`advancement.LayoutOf`), 모양은 과제 마름모·연구 네모·도전 원, 다른 챕터에서 오는 입구 표시는 작은 점선 원(그 선행 과제의 아이콘, 누르면 그 챕터의 그 과제 패널)이다. 선행이 남은 노드는 조금 어둡게 그리되 아이콘·툴팁은 그대로 보인다. 처음에는 다음 할 과제가 가운데에 오고, 끈 자리는 다시 그려도 남는다. 노드를 누르면 판 위에 과제 패널(상태, 힌트, 아이템, 추적)이 열린다. 창은 다른 창과 같은 크기이고 컨트롤 막대(뒤로·닫기)도 다른 창처럼 창 오른쪽 위 바깥에 있다. 챕터 목록은 80u 폭이고 줄마다 아이콘, 이름, 그 아래 달성 수를 둔다. 판은 마우스 휠, 두 손가락 벌리기·오므리기, 판 오른쪽 아래의 +·- 버튼으로 확대·축소한다(0.5~3배, 휠·손가락은 가리킨 자리를 중심으로). 확대 배율과 끈 자리는 챕터마다 남고, 새로 여는 챕터는 마지막 배율로 시작한다(터치 화면은 처음부터 한 단계 확대).

## 패킷과 데이터 저장소

데이터 계약(`oui`, `oui_event`, `oui_vars_1_<n>`)은 `common/oreui/oreui.go` 머리 주석에, 레이아웃 노드 목록은 `oui-ddui.js` 머리 주석에 있다.

| 방향 | 패킷 | 쓰는 법 |
|---|---|---|
| 서버에서 클라이언트 | `ClientBoundDataDrivenUIShowScreen` | `ScreenID` `minecraft:custom_form`, `FormID` 는 폼마다 새 번호, `DataInstanceID` 1 |
| 서버에서 클라이언트 | `ClientBoundDataStore` | 저장소 `minecraft` 의 `custom_form_data_1`(레이아웃)과 `oui_vars_1_<n>`(값 패치)에 쓴다 |
| 서버에서 클라이언트 | `ClientBoundDataDrivenUICloseScreen` | 화면 닫기 |
| 클라이언트에서 서버 | `ServerBoundDataStore` | 렌더러가 `oui_event` 에 쓴 이벤트 |
| 클라이언트에서 서버 | `ServerBoundDataDrivenScreenClosed` | 화면이 닫힘 |

- Dragonfly 는 클라이언트의 DDUI 패킷 두 개를 처리하지 않아 `sghook` 이 연결의 패킷 읽기에서 가로챈다.
- 화면을 바꿔 열면 클라이언트는 앞 폼의 닫힘을 새 폼이 열린 뒤에 알린다. 그래서 닫힘은 `FormID` 가 지금 폼과 같을 때만 처리한다.
- 바닐라 루트는 `custom_form_data_1` 의 바닐라 필드(`title`, `closeButton`, `layout`)를 읽으므로 함께 채운다. 바닐라 폼은 렌더러가 CSS 로 숨긴다.
- 클라이언트는 한 프레임 안에서 속성의 가장 새 값만 준다. 한 속성에 연달아 쓰면 앞 값이 사라지므로 값 패치는 순번을 올리며 여러 속성(`varsSlots`)을 돌아 쓰고, 10ms 안의 변경은 한 패치로 합친다.
- 모든 쓰기는 `Change` 로 보내고 `UpdateCount` 는 서버 전체에서 하나씩 올린다. 클라이언트는 속성의 마지막 판보다 큰 판만 받는다. 경로 하나만 바꾸는 `Update` 를 쓰려면 `PropertyUpdateCount` 를 그 속성의 마지막 `Change.UpdateCount` 와 같게 두고 `PathUpdateCount` 만 올린다.
- Gameface 는 일반 `div` 에 `click` 을 보내지 않으므로 렌더러는 `mousedown` 으로 누름을 받는다.
- 팩으로 할 수 없는 것: 바닐라 DDUI 레이아웃(`ddui/root`) 바꾸기, 진짜 컨테이너(상자·화로·인벤토리)를 Ore UI 로 열기, HUD(HUD 는 Ore UI 가 아니다). 컨테이너 화면은 JSON UI 로 꾸민다([chestui.md](chestui.md)).

## 로비 화면

로비의 메인 화면(`ui.HomeScreen`, `platform` `showLobbyMenu`)은 일반 게임의 첫 화면처럼 화면 전체를 채운다. 하위 DDUI 화면(세이브 선택, 멀티 플레이, 친구, 칭호·치장, 안내, 입력·확인)은 같은 모양으로 그 자리에 열린다(`ui.lobbySkin`, `oui-mc.css` `.mc-lobby`).

- 화면 전체를 어둡게 하지 않아 로비 섬(천천히 도는 카메라)이 배경으로 보이고, 왼쪽은 그늘(`.mc-lobby-shade`, 왼쪽에서 오른쪽으로 옅어지는 그라데이션)이 진다. 왼쪽 위에 로비 틀과 같은 자리에 게임 로고·이름(`ui.LobbyHeader`, 플랫폼이 `name.Game` 과 풀 블록 옆면으로 채운다)과 그 아래 옅은 선을 그린다. 그늘과 이름은 모든 로비 화면에 같은 자리로 들어가, 메인 화면과 하위 화면 사이에서는 메뉴와 창만 바뀐다.
- 메인 화면은 이름 아래(왼쪽 16u, 위 58u)의 큰 버튼 세로줄(170×28u, 간격 4u: 아이콘·글·꺾쇠, `.mc-home-*`)과 오른쪽 아래의 플레이어 이름이다. 버튼 글은 `platform.home.<이름>`(팩 키 `df.lobby.<이름>` 과 짝)이다.
- 메인 화면은 창과 같은 크기 키(`common/ui.windowW`·`windowH`)라, 하위 화면이 열린 화면을 그대로 바꾸고 돌아올 때도 그렇다(화면이 닫혔다 열리지 않는다). Esc 는 메인 화면을 닫고(`ui.Home.Close`, 1.5초 뒤 또는 다음 입력에 다시 열린다), 하위 화면의 Esc·닫기 버튼은 메인 화면으로 돌아간다.
- Ore UI 를 쓰는 플레이어에게는 HUD 의 로비 틀(`hud.Board.Frame`)을 보내지 않는다(`platform/load.go`). 메인 화면이 닫힌 동안에는 로비 섬만 보인다.
- 창·버튼·탭·컨트롤 막대는 메인 메뉴 버튼 그림(`launcher_button`, 호버·누름은 `_hover`·`_pressed`)이고, 글은 밝은 색이다(회색 창의 짙은 글씨 변환을 하지 않는다).
- 창이나 탭·컨트롤 막대가 이름 자리(34×10rem)를 덮으면 렌더러가 아래 공간이 있을 때 창을 내리고, 없으면(짧은 화면) 이름을 숨긴다(`oui-ddui.js` `fitLobbyHead`).
- 하위 화면의 Esc·닫기 버튼이나 누름 뒤에 새 화면이 없으면 화면을 닫기 전에 `ui.Idle` 이 불려(`common/ui.oreEnd`) 메인 화면이 열린 화면에 그려진다. 클라이언트가 화면을 스스로 닫으면 `ui.Idle` 이 메인 화면을 새로 연다. DDUI 화면이 열려 있는 동안은 이동 키 등의 입력이 메인 화면을 다시 그리지 않는다(`platform` `wakeLobbyMenu`).

## 지킬 것

- **게임이 업데이트되면 `packs/oreui/hbui/gameplay.html`의 바닐라 번들 파일 이름(해시)을 설치본 `data/gui/dist/hbui/gameplay.html`과 맞춘다.** 틀리면 게임 중 Ore UI 전체가 뜨지 않는다. `oui-mc.css` 의 `@font-face` 글꼴 파일 이름(설치본 바닐라 gameplay CSS 의 `ui://hbui/fonts/...`)도 함께 맞춘다.
- 창 크기가 바뀌면 Gameface가 누름을 바뀌기 전 자리로 판정한다. 그래서 크기가 다른 창은 화면을 새로 연다(`showOre`). 새 화면도 같은 창 크기(`common/ui.windowW`·`windowH`)로 만든다.
- 화면 크기 상자(`#oui-ddui`, `.mc-screen`, 창의 최대 크기)는 퍼센트가 아닌 화면 단위(vw·vh)로 잡는다. Gameface 는 절대 위치 상자의 퍼센트를 부모(body) 크기로 풀고, iPad 에서는 body 높이가 0이라 창이 위쪽 띠로 접힌다.
- 화면은 바닐라 CSS·JS 에 기대지 않는다. iOS 에서는 팩의 `hbui` 폴더가 게임의 것을 통째로 대신해, `gameplay.html` 이 잇는 바닐라 CSS 가 불러와지지 않는다(그 `@font-face`, `body` 높이, 번들이 정하는 `--localeSpecificNotoSansFonts` 가 없다). 그래서 글꼴은 팩 CSS 가 게임 파일(`ui://hbui/fonts`, iOS 에서도 열린다)로 직접 선언하고 글꼴 목록에 이름으로 적는다(한글 줄은 Noto Sans KR 먼저), 화면 크기는 vw·vh 로 잡는다.
- `/uidebug`(플레이어 자신의 화면, `common/ui.SetOreDebug`)를 쓰면 화면 위에 렌더러의 디버그 줄이 뜬다: 화면 상태, 지역 글꼴 변수, 페이지 언어, 바닐라 루트·body 의 글꼴, `@font-face` 글꼴 목록(바닐라 CSS 가 불러와졌는지), 화면 크기.
- 화면은 플레이어마다 하나다. JSON UI 폼을 보낼 때는 먼저 닫는다.
- 페이지가 받는 입력과 불러올 수 있는 리소스를 보는 운영자 시험 화면은 [oreui-probes.md](oreui-probes.md) 에 있다.

## 바닐라 컨테이너 화면

- 게임 번들(`gameplay-*.js`)과 바닐라 팩에는 컨테이너 DDUI 화면 18개(`minecraft:inventory_screen`, `workbench_screen`, `furnace_screen`, 상자류, `base_layout`)가 있다. 레이아웃은 데이터 저장소 `minecraft` / `container` 의 `slots.<모음>[i]`(`armorContainer`, `offhandContainer`, `craftingInputContainer` 등)를 읽고, 바닐라 슬롯(`Container.Slot`)은 엔진 쿼리 `vanillaGameplayContainerItemQuery(모음 번호, 칸)`(핫바 28, 인벤토리 29, 갑옷 6, 보조손 34, 커서 59)로 아이템을 읽는다. 누름은 엔진 명령 `vanillaGameplayContainerAPICommands`(`handleSlotMousePress(버튼, 모음, 칸, …)`, `takeAllItems`, `placeOneItem`, `splitMultipleItems` 등)로 보낸다.
