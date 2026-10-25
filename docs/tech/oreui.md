# Ore UI 화면 (DDUI)

팩을 암호화하면(`[ResourcePack] Encrypt`, 기본 켬) 섬 안의 메뉴 폼은 게임의 Ore UI에 그린 서버 주도 DDUI 화면으로 보인다. 로비 폼과 수량·표 폼은 JSON UI(`docs/tech/json-ui.md`)로 남는다.

플레이어는 `/uimode` 로 자기 메뉴를 JSON UI 폼으로 바꾸거나 되돌린다. 계정 설정(`account_settings` 의 `ui_mode`, `"json"`)으로 남고, 기본값은 모든 기기가 같다(Ore UI). 애플 기기에서 Ore UI 화면이 깨진다는 보고가 있어, 그 기기에서 두 방식을 견주어 보는 데 쓴다(`internal/platform/uimode.go`, common `ui.PreferJSON`).

## 구성

- common `packs/oreui`: Ore UI 게임 중 페이지(`hbui/gameplay.html`)를 바꾸는 팩. 바닐라 번들을 그대로 불러온 뒤 렌더러(`oui-ddui.js`)와 모양(`oui-mc.css`)을 더한다. 클라이언트는 이 페이지를 **암호화 팩에서만** 가져가므로, 암호화를 켤 때만 보낸다(`main.go`).
- common `oreui`: DDUI 화면을 열고, 레이아웃과 값을 데이터 저장소로 보내고, 누름을 받는다. 클라이언트의 DDUI 패킷은 `dfhook`에서 가로챈다.
- common `ui`: `ui.OreUI`를 켜면 `Menu`(`Confirm`), `CardMenu`, `ItemBoard`, `SplitMenu`, `SieveBoard`, `ListSplitMenu`, `Input`, `ItemGrid`, `ItemGridSections`, `SlotMenu`가 같은 모양의 DDUI 화면이 된다(`ui/oreui.go`, `ui/oreforms.go`). 게임 코드는 바꾸지 않는다. 버튼의 `Run`도 폼과 같이 그 플레이어 월드에서 돈다. 누름 뒤에 새 화면을 보이지 않으면 화면이 닫힌다(폼 버튼을 누르면 폼이 닫히는 것과 같다).
- 창은 모두 바닐라 인벤토리 크기(326×166) 하나다(퀘스트 책만 따로). 그래서 한 화면에서 다른 화면으로 가는 누름(정수 추출기의 정수 고르기, 도움말 주제, 치장 탭, 친구 줄에서 확인 질문, 통계 탭)은 열린 화면 안에서 바뀐다.
- 같은 화면에 새 레이아웃이 오면 렌더러는 그려 둔 것과 견주어 달라진 노드만 고친다(`oui-ddui.js` `patch`). 개수·글·아이콘·고른 칸만 바뀌면 그 요소만 바뀌고, 창·스크롤 위치·입력칸·불러온 아이템 그림은 그대로다. 영역(`region`)도 값이 바뀌면 같은 방식으로 고친다. 게임 코드는 화면 전체를 다시 보이기만 하면 된다.
- 터치 화면에서는 툴팁이 있는 요소(아이템 칸, 도움말 `?`, 발전과제 노드, 버튼)를 처음 탭하면 툴팁만 뜨고, 같은 요소를 한 번 더 탭하면 눌린다(JSON UI 폼과 같다). 다른 곳을 탭하면 툴팁이 닫힌다. 툴팁이 이름뿐인 뒤로·쪽 넘김 버튼과 탭은 바로 눌린다(common `ui.quick`, 노드의 `quick`). 마우스는 가리키면 툴팁이 뜨고 누르면 바로 눌린다.
- 입력 화면(`ui.Input`)은 창 안의 입력칸과 확인 버튼이다. 버튼을 누르면 화면의 입력칸 글이 함께 오고, Enter 도 확인을 누른다. 돌아갈 화면이 있으면 컨트롤 막대에 뒤로 버튼이 있다.
- 소셜 화면(친구·참여자 관리)은 목록 분할 화면(`ui.ListSplitMenu`)이다. 탭은 이름이 보이는 넓은 탭으로 창 위에, 줄은 왼쪽 판에서 스크롤되고(이름과 그 아래 작은 정보 줄, 고른 줄은 밝게), 오른쪽은 본문과 동작 버튼(첫 버튼은 전체 폭, 나머지는 두 칸씩)이다. 확인 질문·친구 추가 입력도 같은 창에서 바뀐다.
- 도감의 체 탭은 Ex Deorum JEI 체 화면 배치로 그린다(`ui.SieveBoard`, `codexpages.go`).
- 도감 검색 탭은 창 안의 검색칸과 결과 한 판이다(`ui.SearchGrid`, `codexsearch.go` `showCodexSearchLive`). 글을 치면 250ms 뒤 결과 영역만 다시 그리므로 검색칸의 글과 커서가 남는다. 결과는 쪽 없이 스크롤된다. 빈 검색에서는 목표와 최근 본 아이템이 위에 나온다.

- 아이템 도관 설정은 가운데 블록과 여섯 면의 카드를 선으로 이은 화면이다(`ui.SidesMenu`, `conduitform.go` `showConduitSides`). 카드 자리는 서버가 정하고(`sideCard`), 포트 줄과 고르기 판은 영역(`@side<n>`, `@picker`)이라 값이 바뀌면 그 부분만 다시 그린다. 채널은 아이템이 아닌 색 네모(`PortChoice.Colour`)로 그린다.
- 발전과제는 ATM10 To The Sky의 FTB Quests 퀘스트 책처럼 그린다(`ui.QuestBookMenu`, `advancementui.go` `showQuestBook`). 왼쪽은 갈래 목록(달성 수), 오른쪽은 끌어서 움직이는 어두운 판 위의 노드와 따르는 과제에서 오는 직선이다. 노드 자리는 트리 폼과 같고(`advancementLayout`), 모양은 과제 마름모·연구 네모·도전 원이다. 처음에는 다음 할 과제가 가운데에 오고, 끈 자리는 다시 그려도 남는다. 노드를 누르면 판 위에 과제 패널(상태, 힌트, 아이템, 추적)이 열린다. 창은 화면의 82%이고 컨트롤 막대(뒤로·닫기)는 창 밖이 아니라 판 제목 줄 오른쪽에 있어 짧은 폰 화면에서도 벗어나지 않는다. 갈래 목록은 화면 폭의 24%를 넘지 않는다. 판은 마우스 휠, 두 손가락 벌리기·오므리기, 판 오른쪽 아래의 +·- 버튼으로 확대·축소한다(0.5~3배, 휠·손가락은 가리킨 자리를 중심으로). 확대 배율과 끈 자리는 갈래마다 남고, 새로 여는 갈래는 마지막 배율로 시작한다(터치 화면은 처음부터 한 단계 확대).

## 지킬 것

- **게임이 업데이트되면 `packs/oreui/hbui/gameplay.html`의 바닐라 번들 파일 이름(해시)을 설치본 `data/gui/dist/hbui/gameplay.html`과 맞춘다.** 틀리면 게임 중 Ore UI 전체가 뜨지 않는다. `oui-mc.css` 의 `@font-face` 글꼴 파일 이름(설치본 바닐라 gameplay CSS 의 `ui://hbui/fonts/...`)도 함께 맞춘다.
- 창 크기가 바뀌면 Gameface가 누름을 바뀌기 전 자리로 판정한다. 그래서 크기가 다른 창은 화면을 새로 연다(`showOre`). 새 화면도 같은 창 크기(common `ui.windowW`·`windowH`)로 만든다.
- 화면 크기 상자(`#oui-ddui`, `.mc-screen`, 창의 최대 크기, 퀘스트 책 창)는 퍼센트가 아닌 화면 단위(vw·vh)로 잡는다. Gameface 는 절대 위치 상자의 퍼센트를 부모(body) 크기로 풀고, iPad 에서는 body 높이가 0이라 창이 위쪽 띠로 접힌다.
- 화면은 바닐라 CSS·JS 에 기대지 않는다. iOS 에서는 팩의 `hbui` 폴더가 게임의 것을 통째로 대신해, `gameplay.html` 이 잇는 바닐라 CSS 가 불러와지지 않는다(그 `@font-face`, `body` 높이, 번들이 정하는 `--localeSpecificNotoSansFonts` 가 없다). 그래서 글꼴은 팩 CSS 가 게임 파일(`ui://hbui/fonts`, iOS 에서도 열린다)로 직접 선언하고 글꼴 목록에 이름으로 적는다(한글 줄은 Noto Sans KR 먼저), 화면 크기는 vw·vh 로 잡는다.
- `/uidebug`(플레이어 자신의 화면, common `ui.SetOreDebug`)를 쓰면 화면 위에 렌더러의 디버그 줄이 뜬다: 화면 상태, 지역 글꼴 변수, 페이지 언어, 바닐라 루트·body 의 글꼴, `@font-face` 글꼴 목록(바닐라 CSS 가 불러와졌는지), 화면 크기.
- 화면은 플레이어마다 하나다. JSON UI 폼을 보낼 때는 먼저 닫는다.

## 인벤토리 시험

DDUI 로 진짜 인벤토리처럼 도는 화면을 만들 수 있는지 보는 운영자 명령이다([admin.md](admin.md)). 결과는 이 절에 적는다.

- 게임 번들(`gameplay-*.js`)과 바닐라 팩에는 컨테이너 DDUI 화면 18개(`minecraft:inventory_screen`, `workbench_screen`, `furnace_screen`, 상자류, `base_layout`)가 있다. 레이아웃은 데이터 저장소 `minecraft` / `container` 의 `slots.<모음>[i]`(`armorContainer`, `offhandContainer`, `craftingInputContainer` 등)를 읽고, 바닐라 슬롯(`Container.Slot`)은 엔진 쿼리 `vanillaGameplayContainerItemQuery(모음 번호, 칸)`(핫바 28, 인벤토리 29, 갑옷 6, 보조손 34, 커서 59)로 아이템을 읽는다. 누름은 엔진 명령 `vanillaGameplayContainerAPICommands`(`handleSlotMousePress(버튼, 모음, 칸, …)`, `takeAllItems`, `placeOneItem`, `splitMultipleItems` 등)로 보낸다. `/dfdebug orescreen`·`oreengine` 이 이것들을 그대로 시험한다.
- `/dfdebug oreinput` 은 Gameface 가 오른쪽 버튼·더블 클릭·숫자 키·길게 누르기를 페이지에 주는지 본다. 바닐라 슬롯은 `button` 0 과 2 를 쓴다.
- `/dfdebug oreinv` 는 엔진 없이 서버가 바닐라 조작을 다시 만든 화면이다. 렌더러의 `raw` 슬롯(누른 버튼·수정키·더블 클릭, 버튼을 놓을 때 지나간 칸, 위에서 누른 키), `cursor` 노드(커서 아이템), `self` 상자(창 밖 누름)를 쓴다. 누를 때마다 서버를 다녀오므로 지연이 그대로 보인다. 터치로 끌기는 지나간 칸을 모으지 않는다(손가락이 움직일 때는 마우스 이벤트가 오지 않는다, `oui-ddui.js` `pointOf`).

## 실기 확인

- 2026-10-24 실제 세이브: 메뉴 카드(색 코드 없는 글도 가운데), 연구(분할 화면, 선택, 뒤로), 도감 검색(초성 검색, 결과 누르고 뒤로 오면 검색어 유지)·가공·체 탭, 아이템 쪽 보드(글 칸 왼쪽 맞춤, 뒤 칸 제자리), 도움말 목록·주제, 스크롤 막대 끌기, 창 크기가 바뀐 뒤 바로 누른 탭.
- 2026-10-24 실제 세이브: 퀘스트 책(갈래 전환, 판 끌기, 직선 연결선과 회전한 마름모, 원, 과제 패널 열고 닫기, 끈 자리 유지).
- 모바일은 확인 전이다([TODO](../TODO.md)).
