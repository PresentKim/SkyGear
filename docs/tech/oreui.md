# Ore UI 화면 (DDUI)

섬과 로비의 메뉴는 모두 게임의 Ore UI에 그린 서버 주도 DDUI 화면이다. 게임은 Ore UI 페이지를 암호화되었거나 `contents.zip` 이 있는 팩에서만 가져가므로, 암호화를 끄면 팩에 빈 `contents.zip` 을 넣는다([resourcepack.md](resourcepack.md)). 로비는 화면 전체를 채우는 메인 화면(`ui.HomeScreen`) 하나로 그린다. JSON UI 는 HUD 와 창고 창에만 남는다([json-ui.md](json-ui.md)).

## 구성

- `assets/resourcepack/oreui`: Ore UI 게임 중 페이지(`hbui/gameplay.html`)를 바꾸는 팩. 바닐라 번들을 그대로 불러온 뒤 렌더러(`oui-ddui.js`)와 모양(`oui-mc.css`)을 더한다. 클라이언트는 이 페이지를 **암호화되었거나 `contents.zip` 이 있는 팩에서만** 가져간다. 암호화하지 않은 팩에는 로더가 빈 `contents.zip` 을 넣는다(`common/resourcepack`).
- `common/oreui`: DDUI 화면을 열고, 레이아웃과 값을 데이터 저장소로 보내고, 누름을 받는다. 클라이언트의 DDUI 패킷은 `sghook`에서 가로챈다.
- `common/ui`: `Menu`(`Confirm`), `CardMenu`, `ItemBoard`, `SplitMenu`, `SieveBoard`, `ListSplitMenu`, `Input`, `ItemGrid`, `ItemGridSections`, `SearchGrid`, `SlotMenu`, `ChoiceMenu`, `SidesMenu`, `PlaneMenu`, `QuestBookMenu` 가 DDUI 화면을 그린다(`ui/oreui.go`, `ui/oreforms.go` 등). 버튼의 `Run`은 그 플레이어 월드에서 돈다. 탭·뒤로·도움말·쪽 버튼은 라벨 앞 표식(`ui.Tab`, `ui.Back`, `ui.Help`, `ui.BarPager`)으로 창 위·컨트롤 막대에 놓이고, 표식은 글에서 빠진다(`chrome`, `untoken`). 누름 뒤에 새 화면을 보이지 않으면 화면이 닫힌다(폼 버튼을 누르면 폼이 닫히는 것과 같다).
- 창은 퀘스트 책까지 모두 바닐라 인벤토리 크기(326×166) 하나다. 그래서 한 화면에서 다른 화면으로 가는 누름(정수 추출기의 정수 고르기, 도움말 주제, 치장 탭, 친구 줄에서 확인 질문, 통계 탭)은 열린 화면 안에서 바뀐다.
- 글 속의 클라이언트 표식(아이템 이름, 날짜)은 렌더러가 클라이언트 언어로 바꿔 그린다(`oui-ddui.js` `localize`, [i18n.md](i18n.md) 클라이언트 표식).
- 같은 화면에 새 레이아웃이 오면 렌더러는 그려 둔 것과 견주어 달라진 노드만 고친다(`oui-ddui.js` `patch`). 개수·글·아이콘·고른 칸만 바뀌면 그 요소만 바뀌고, 창·스크롤 위치·입력칸·불러온 아이템 그림은 그대로다. 영역(`region`)도 값이 바뀌면 같은 방식으로 고친다. 게임 코드는 화면 전체를 다시 보이기만 하면 된다.
- 터치 화면에서는 툴팁이 있는 요소(아이템 칸, 도움말 `?`, 발전과제 노드, 버튼)를 처음 탭하면 툴팁만 뜨고, 같은 요소를 한 번 더 탭하면 눌린다. 다른 곳을 탭하면 툴팁이 닫힌다. 툴팁이 이름뿐인 뒤로·쪽 넘김 버튼과 탭은 바로 눌린다(`common/ui.quick`, 노드의 `quick`). 마우스는 가리키면 툴팁이 뜨고 누르면 바로 눌린다. 툴팁 글자는 창 글자와 같은 크기·Minecraft 글꼴이고 터치 화면에서는 한 단계 크다. 오른쪽에 자리가 없으면 요소 왼쪽에 뜨고, 화면 밖으로 잘리지 않게 끌어온다.
- 입력 화면(`ui.Input`)은 창 안의 입력칸과 확인 버튼이다. 버튼을 누르면 화면의 입력칸 글이 함께 오고, Enter 도 확인을 누른다. 돌아갈 화면이 있으면 컨트롤 막대에 뒤로 버튼이 있다.
- 소셜 화면(친구·참여자 관리)은 목록 분할 화면(`ui.ListSplitMenu`)이다. 탭은 다른 화면과 같은 아이콘 탭(이름은 툴팁)으로 창 위에, 줄은 왼쪽 판에서 스크롤되고(이름과 그 아래 작은 정보 줄, 고른 줄은 밝게), 오른쪽은 본문과 동작 버튼(첫 버튼은 전체 폭, 나머지는 두 칸씩)이다. 확인 질문·친구 추가 입력도 같은 창에서 바뀐다.
- 선택 판(`ui.ChoiceMenu`, 요정의 마법판)은 위의 수치 판(아이콘과 값 한 줄, 이름은 툴팁, `ui.Stat`)과 한 줄 메모, 아래에 갈래(`ui.ChoiceGroup`)마다 한 줄: 왼쪽에 갈래 이름, 오른쪽에 아이콘 위·이름 아래의 칸(효과는 툴팁)이다. 지금 것(`On`)은 초록 테두리로 눌리지 않고 툴팁에 표시(`Badge`)가 붙으며, 지금 못 하는 것(`Dim`)은 흐리다(`ui/choice.go`, `oui-mc.css` `.mc-pick*`·`.mc-stat*`).
- 도감의 체 탭은 Ex Deorum JEI 체 화면 배치로 그린다(`ui.SieveBoard`, `codexpages.go`).
- 도감 검색 탭은 창 안의 검색칸과 결과 한 판이다(`ui.SearchGrid`, `codexsearch.go` `showCodexSearchLive`). 글을 치면 250ms 뒤 결과 영역만 다시 그리므로 검색칸의 글과 커서가 남는다. 결과는 쪽 없이 스크롤된다. 빈 검색에서는 즐겨찾기와 최근 본 아이템이 위에 나온다.
- 화면은 키보드 키를 받을 수 있다(`ui.Key`, `oreLayout.key`). 레이아웃 루트의 `keys`(`{up, down, left, right, ok: id}`)가 방향키·WASD, Enter·스페이스바를 그 id 의 누름으로 보낸다(`oui-ddui.js` `onKey`, 같은 키는 120ms 에 한 번까지, 입력칸에 글을 칠 때는 받지 않는다). 섬 확장 화면(`ui.Plane.Keys`)이 청크 이동과 구매에 쓴다.
- 화면이 받지 않는 방향키·WASD 는 키보드 선택을 옮긴다(`oui-ddui.js` `navKey`). 누를 수 있는 요소(버튼, 칸, 탭, 컨트롤 막대 버튼) 가운데 그 방향으로 가장 가까운 것이 마우스를 올린 것처럼 보이고(`oui-kb`, CSS 의 `:hover` 마다 같은 규칙, 툴팁도 뜬다), Enter·스페이스바가 그것을 누른다. 처음 누른 방향키는 왼쪽 위의 요소를 고르고, 스크롤 판 안의 요소는 보이게 스크롤한다. 누른 뒤 화면이 바뀌어 고른 요소가 없어지면 그 자리에서 가장 가까운 요소가 골라진다. 마우스를 움직이면 선택이 사라진다.
- 아이템 도관 설정은 가운데 블록과 여섯 면의 카드를 선으로 이은 화면이다(`ui.SidesMenu`, `conduitform.go` `showConduitSides`). 카드 자리는 서버가 정하고(`sideCard`), 포트 줄과 고르기 판은 영역(`@side<n>`, `@picker`)이라 값이 바뀌면 그 부분만 다시 그린다. 채널은 아이템이 아닌 색 네모(`PortChoice.Colour`)로 그린다. 색 네모를 우클릭하면 이전 값이 된다(노드의 `alt`, `oreLayout.alt`).
- 발전과제는 ATM10 To The Sky의 FTB Quests 퀘스트 책처럼 그린다(`ui.QuestBookMenu`, `advancementui.go` `showQuestBook`). 왼쪽은 챕터 목록(달성 수), 오른쪽은 끌어서 움직이는 어두운 판 위의 노드와 선행마다 오는 꺾은선(열 사이 틈에서 꺾여 노드 위를 지나지 않는다)이다. 노드 자리는 서버가 정하고(`advancement.LayoutOf`: 부모 다음 열, 연결된 묶음이 21줄을 넘으면 오른쪽에 새 띠), 모양은 과제 마름모·연구 네모·도전 원, 다른 챕터에서 오는 입구 표시는 작은 점선 원(그 선행 과제의 아이콘, 누르면 그 챕터의 그 과제 패널)이다. 선행이 남은 노드는 조금 어둡게 그리되 아이콘·툴팁은 그대로 보인다. 처음에는 다음 할 과제가 가운데에 오고, 끈 자리는 다시 그려도 남는다. 노드를 누르면 판 위에 과제 패널(상태, 힌트, 아이템, 추적)이 열린다. 창은 다른 창과 같은 크기이고 컨트롤 막대(뒤로·닫기)도 다른 창처럼 창 오른쪽 위 바깥에 있다. 챕터 목록은 80u 폭이고 줄마다 아이콘, 이름, 그 아래 달성 수를 둔다. 판은 마우스 휠, 두 손가락 벌리기·오므리기(두 번째 손가락이 나중에 닿아도 된다: 렌더러가 페이지 전체의 터치를 손가락 id 로 따로 모은다, `trackFingers`), 판 오른쪽 아래의 +·- 버튼으로 확대·축소한다(0.5~3배, 휠·손가락은 가리킨 자리를 중심으로). 판은 노드가 있는 곳까지만 끌린다: 노드들을 감싼 상자가 판보다 작으면 판 안(가장자리 2.4rem 안쪽)에 머물고, 크면 그 가장자리가 판 가장자리 안으로 들어오지 않는다(`oui-ddui.js` `clamp`, 확대·축소 뒤에도). 확대 배율과 끈 자리는 챕터마다 남고, 새로 여는 챕터는 마지막 배율로 시작한다(터치 화면은 처음부터 한 단계 확대).

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
- 렌더러는 Esc 와 함께 휴대폰 뒤로 버튼으로 올 수 있는 키(Android `KEYCODE_BACK` 4, 브라우저 뒤로 166·`GoBack`·`BrowserBack`)를 닫기로 보낸다(`oui-ddui.js` `isBack`). 클라이언트가 화면을 스스로 닫으면 서버 로그에 그 사유(`ore ui screen closed by client`, `reason`)가 남는다. 휴대폰 뒤로 버튼은 페이지에 키로 오지 않는다(`/sgdebug oreinput` 에 줄이 없다) [실측]. 숨긴 바닐라 폼(`custom_form` 루트)이 `oui_event` 밖의 경로에 쓰는 것(닫기 버튼 누름 등)은 서버 로그(`ore ui vanilla form write`)에 남기고, 경로에 `close` 가 들어 있으면 Esc 로 받는다(`common/oreui`). 뒤로 버튼이 그 길로 오는지는 확인 안 됨.
- 팩으로 할 수 없는 것: 바닐라 DDUI 레이아웃(`ddui/root`) 바꾸기, 진짜 컨테이너(상자·화로·인벤토리)를 Ore UI 로 열기, HUD(HUD 는 Ore UI 가 아니다). 컨테이너 화면은 JSON UI 로 꾸민다([chestui.md](chestui.md)).

## 클라이언트 설정

Ore UI 페이지는 게임의 설정 화면이 쓰는 설정을 읽고 바꿀 수 있다(`vanilla.menus.settings*Query`, `vanilla_menus_update_settings`). 서버는 접속한 동안 몇 가지를 정해 둔다(`platform` `keepClientSettings`, `common/oreui.SetClientSettings`).

- 모두: UI 프로필 클래식(`video.ui_profile` 0). 서버의 창과 화면은 클래식 UI 에 맞춰 그린다.
- 운영자가 아닌 플레이어: 콘텐츠 로그 끔(`creator.contentLog.gui.enable`, `creator.contentLog.gui.showErrors`, `creator.contentLog.file.enable`, `creator.scriptDiagnostic.enableClient`).
- 설정은 다음에 열리는 DDUI 화면의 레이아웃과 함께 가고(`oui` 의 `settings`), 렌더러가 다를 때만 쓴다(`oui-ddui.js` `keepSettings`). 쓴 뒤에도 값을 지켜보아, 플레이어가 접속 중에 바꾸면 되돌린다.
- 설정은 기기에 저장되는 게임 옵션이라 서버를 나간 뒤에도 남는다.
- 설정 id 는 게임의 설정 화면 트리(`vanilla.menus.settingsGroupQuery("settings-tabs-groups")` 아래)의 이름이다.

## 로비 화면

로비의 메인 화면(`ui.HomeScreen`, `platform/home.go`)은 게임 로비처럼 화면 전체를 채우고, 로비에 있는 동안 닫히지 않는다. 하위 DDUI 화면(세이브 선택, 멀티 플레이, 친구, 칭호·치장, 설정, 안내, 입력·확인)은 같은 모양으로 그 자리에 열린다(`ui.lobbySkin`, `oui-mc.css` `.mc-lobby`, `.mc-home-*`).

- 로비 섬(천천히 도는 카메라와 손 흔드는 캐릭터)이 가운데에 보이도록 버튼은 가장자리에 둔다. 메인 화면은 그늘 없이 섬이 그대로 보이고(`.mc-home .mc-lobby-shade`), 하위 화면은 전체가 어두워진다. 위 가운데에 게임 로고·이름(`ui.LobbyHeader`, 플랫폼이 `name.Game` 과 풀 블록 옆면으로 채운다)이 있다.
- 휴대폰·태블릿(접속 기기 OS 가 Android·iOS·Fire OS, `common/ui.touchDevice`)에서는 루트에 `mc-touch` 를 붙여, 화면 가장자리의 카메라 구멍·시스템 핸들(엣지 패널 등)이 누름을 가로채지 않게 버튼을 양옆에서 8rem 안쪽에 두고 왼쪽 타일·오른쪽 위 아이콘 버튼을 크게 그린다. 게임 페이지의 안전 영역 쿼리(`core.safeZone`)는 답이 오지 않아 쓰지 않는다([oreui-probes.md](oreui-probes.md)).
- 왼쪽 위: 프로필(얼굴, 이름 색을 입힌 이름, 칭호, 업적 수 막대). 누르면 칭호·치장. 얼굴은 플레이어 스킨의 머리 앞면(모자 겹 포함, 8×8)을 서버가 읽어 `pixels` 노드로 보낸다(`platform/skinface.go`). 스킨 모델(Persona·사용자 모델)이 있으면 `head`·`hat` 뼈의 첫 큐브 박스 UV 에서, 없으면 표준 스킨 자리(64px 기준 (8,8), 모자 (40,8))에서 읽는다. 얼굴이 비어 있으면 바닐라 `textures/ui/icon_steve` 를 그린다. 페이지에 캔버스가 없어(`getContext`) 그림은 8줄×8칸 상자다(`oui-ddui.js` `pixels`). Gameface 는 빈 flex 항목을 늘리지 않으므로 줄과 칸의 크기는 퍼센트로 정한다.
- 오른쪽 위: 알림(친구 요청·세이브 초대 등 알림 수, `platform/notify.go`), 설정, 서버 나가기 아이콘. 이름은 툴팁, 수는 빨간 표시(`ui.HomeButton.Badge`).
- 운영자에게는 오른쪽 위 맨 앞과 시스템 메뉴에 **명령어**가 더 있다. 로비에서는 메인 화면이 늘 열려 있고 페이지에서 게임 채팅창을 여는 엔진 명령도 없어([oreui-probes.md](oreui-probes.md)), 명령은 콘솔 화면(`ui.ConsoleScreen`, `platform/console.go`)에 친다. 입력칸에 친 명령(앞의 `/` 는 없어도 된다)을 그 플레이어로 실행하고(`ExecuteCommand`), 콘솔이 열린 동안 그 연결로 가는 명령 결과(`CommandOutput`)와 채팅 줄(`Text`)을 기록(최근 80줄)에 더해 다시 그린다. 번역 키는 클라이언트 표식으로 바꾼다. 기록은 접속한 동안 남는다. 기록 상자는 늘 끝으로 스크롤된다(노드의 `end`).
- 왼쪽: 멀티 플레이 · 친구 · 칭호·치장 · 도움말 타일. 왼쪽 아래: 새 알림, 접속 중인 친구 최대 3줄. 새 알림이 오면 보고 있는 메인 화면을 다시 그린다.
- 오른쪽 아래: 세이브 카드(그 세이브의 일차·플레이 시간·참여자·마지막 플레이, 꺾쇠로 다른 세이브), 초록 `섬으로 출발`(고른 세이브에 들어간다, 세이브가 없으면 `새 게임`), `세이브 관리 · 새 게임`. 처음 보이는 세이브는 마지막으로 플레이한 것이다.
- Esc 는 화면을 닫지 않고 시스템 메뉴(계속하기 · 설정 · 서버 안내 · 서버 나가기)를 메인 화면 위에 띄운다(`ui.Home.Overlay`). 렌더러가 Esc 를 서버로 보내고 서버가 다시 그리므로 화면이 닫히지 않는다. 서버 나가기는 확인 뒤 접속을 끊는다.
- 메인 화면은 창과 같은 크기 키(`common/ui.windowW`·`windowH`)라, 하위 화면이 열린 화면을 그대로 바꾸고 돌아올 때도 그렇다. 하위 화면의 Esc·닫기 버튼이나 누름 뒤에 새 화면이 없으면 화면을 닫기 전에 `ui.Idle` 이 불려(`common/ui.oreEnd`) 메인 화면이 열린 화면에 그려진다. 클라이언트가 화면을 스스로 닫으면 `ui.Idle` 이 바로 다시 연다.
- 창·버튼·탭·컨트롤 막대는 남색 버튼 그림(`launcher_button`, 호버·누름은 `_hover`·`_pressed`)이고, 글은 밝은 색이다(회색 창의 짙은 글씨 변환을 하지 않는다).
- 창이나 탭·컨트롤 막대가 로고·이름을 덮으면 렌더러가 아래 공간이 있을 때 창을 내리고, 없으면(짧은 화면) 이름을 숨긴다(`oui-ddui.js` `fitLobbyHead`).

## 세이브 로딩 화면

세이브에 들어갈 때(`platform` `enter`) 화면 전체를 채우는 로딩 화면(`ui.LoadingScreen`)을 띄우고, 섬의 청크가 서버와 클라이언트에 모두 와서 플레이어가 땅에 놓이면(`instance.PlaceAt` 이 움직임을 풀 때, 시간 초과 포함) 닫는다(`ui.CloseLoading`). 입장 채팅 메시지는 보내지 않는다.

- 화면: 짙은 하늘에서 밝은 하늘로 가는 배경, 가로지르는 네모 구름 5개, 가운데에서 위아래로 떠다니는 세이브 그림과 그 뒤의 빛, 세이브 이름과 일차·플레이 시간, 왼쪽에서 오른쪽으로 미끄러지는 막대, 점이 차례로 깜빡이는 `섬으로 가는 중`, 아래에 `/menu` 안내(`oui-mc.css` `.mc-load*`, CSS 애니메이션).
- 창과 같은 크기 키라 메인 화면 자리에서 바뀌고, 플레이어가 세이브로 옮겨 가도 그대로 열려 있다. Esc 는 닫지 않고 같은 화면을 다시 보인다.
- 너무 빨리 끝나 깜빡이지 않게 최소 1.2초(`minLoading`)는 보인다. 그 사이 다른 화면이 열리면 닫지 않는다(`oreScreen.loading`).
- 들어가지 못하면(호스트 오프라인, 서버 바쁨 등) 메인 화면으로 돌아오고 사유는 토스트로 보낸다(`enterFailed`).

## 지킬 것

- **게임이 업데이트되면 `assets/resourcepack/oreui/hbui/gameplay.html`의 바닐라 번들 파일 이름(해시)을 설치본 `data/gui/dist/hbui/gameplay.html`과 맞춘다.** 틀리면 게임 중 Ore UI 전체가 뜨지 않는다. `oui-mc.css` 의 `@font-face` 글꼴 파일 이름(설치본 바닐라 gameplay CSS 의 `ui://hbui/fonts/...`)도 함께 맞춘다.
- 창 크기가 바뀌면 Gameface가 누름을 바뀌기 전 자리로 판정한다. 그래서 크기가 다른 창은 화면을 새로 연다(`showOre`). 새 화면도 같은 창 크기(`common/ui.windowW`·`windowH`)로 만든다.
- 화면 크기 상자(`#oui-ddui`, `.mc-screen`, 창의 최대 크기)는 퍼센트가 아닌 화면 단위(vw·vh)로 잡는다. Gameface 는 절대 위치 상자의 퍼센트를 부모(body) 크기로 풀고, iPad 에서는 body 높이가 0이라 창이 위쪽 띠로 접힌다.
- 화면은 바닐라 CSS·JS 에 기대지 않는다. iOS 에서는 팩의 `hbui` 폴더가 게임의 것을 통째로 대신해, `gameplay.html` 이 잇는 바닐라 CSS 가 불러와지지 않는다(그 `@font-face`, `body` 높이, 번들이 정하는 `--localeSpecificNotoSansFonts` 가 없다). 그래서 글꼴은 팩 CSS 가 게임 파일(`ui://hbui/fonts`, iOS 에서도 열린다)로 직접 선언하고 글꼴 목록에 이름으로 적는다(한글 줄은 Noto Sans KR 먼저), 화면 크기는 vw·vh 로 잡는다.
- `/uidebug`(플레이어 자신의 화면, `common/ui.SetOreDebug`)를 쓰면 화면 위에 렌더러의 디버그 줄이 뜬다: 화면 상태, 지역 글꼴 변수, 페이지 언어, 바닐라 루트·body 의 글꼴, `@font-face` 글꼴 목록(바닐라 CSS 가 불러와졌는지), 화면 크기.
- 화면은 플레이어마다 하나다. `ui.FormOpen` 은 화면이 열려 있는지 알린다.
- 페이지가 받는 입력과 불러올 수 있는 리소스를 보는 운영자 시험 화면은 [oreui-probes.md](oreui-probes.md) 에 있다.

## 알림 (토스트)

화면이 열려 있으면 채팅이 보이지 않으므로 알림은 **토스트**(화면 위 알림창)로 보낸다. 화면 위에도 보이고 다음 화면을 막지 않는다(`common/ui`).

- `ui.Notify`(실패 등 지금만 필요한 알림: 재료 부족 …): 토스트만. 제목 `알림`(빨강).
- `ui.Flash`(결과가 중요한 알림): 토스트 + 같은 줄을 채팅에도(나중에 다시 읽기). 제목 `완료`(초록).
- 화면 동작 안이든 밖(명령어·아이템 사용)이든 바로 보낸다.
- 로비에는 채팅이 없어 `ui.Flash` 는 토스트만 보내고, 채팅으로 보낼 글은 `ui.Tell`(세이브 안이면 채팅, 로비면 토스트 `알림`)로 보낸다. 업적 달성도 로비에서는 토스트(`업적 달성`)다.
- 클라이언트는 토스트를 하나씩 차례로 보여 주므로, 플레이어마다 **`ui.ToastGap`(3초)에 하나**만 보낸다: 그 사이에 온 알림은 하나로 합쳐 앞 토스트가 끝날 때 가장 최근 메시지 + `(외 n건)` 으로 보낸다. 성공 메시지는 채팅에 전부 남는다.
- 클라이언트는 아이템 사용 버튼을 누르고 있는 동안 사용 패킷을 되풀이해 보낸다. 메뉴 아이템 등으로 화면을 여는 사용은 0.5초 디바운스한다(`skygear` `debounced`). 같은 크기 창의 화면을 다시 보이면 열린 화면이 그 자리에서 바뀐다.

## 바닐라 컨테이너 화면

- 게임 번들(`gameplay-*.js`)과 바닐라 팩에는 컨테이너 DDUI 화면 18개(`minecraft:inventory_screen`, `workbench_screen`, `furnace_screen`, 상자류, `base_layout`)가 있다. 레이아웃은 데이터 저장소 `minecraft` / `container` 의 `slots.<모음>[i]`(`armorContainer`, `offhandContainer`, `craftingInputContainer` 등)를 읽고, 바닐라 슬롯(`Container.Slot`)은 엔진 쿼리 `vanillaGameplayContainerItemQuery(모음 번호, 칸)`(핫바 28, 인벤토리 29, 갑옷 6, 보조손 34, 커서 59)로 아이템을 읽는다. 누름은 엔진 명령 `vanillaGameplayContainerAPICommands`(`handleSlotMousePress(버튼, 모음, 칸, …)`, `takeAllItems`, `placeOneItem`, `splitMultipleItems` 등)로 보낸다.
