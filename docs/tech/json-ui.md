# JSON UI (메뉴 창·입력 폼·창고 창·HUD)

화면은 모두 JSON UI 로 그린다. 메뉴는 플레이어마다 하나인 상자 창에 그리는 **메뉴 창**이고, 글을 받는 곳은 **입력 폼**(서버 커스텀 폼)이며, 아이템이 오가는 창(상자·기계 창·필터 창)은 엔티티 상자다([chestui.md](chestui.md)).
HUD·입력 폼처럼 게임과 상관없는 것은 `assets/resourcepack/sgui` 에, 상자 창과 메뉴 창은 `assets/resourcepack/inventoryui` 에, 기계 창처럼 이 게임에만 있는 것은 `skygear` 팩에 있다. `cmd/skygear/main.go` 가 `skygear` 를 공통 팩보다 먼저(스택 위에) 싣는다.

## 파일

| 파일 | 내용 |
|---|---|
| `assets/resourcepack/inventoryui/ui/sg_menu.json` | 메뉴 창(`sg_menu`). `common/invui/layout.go` 에서 저장소 루트의 `go run ./tools/invuigen` 이 만든다. 손으로 고치지 않는다 |
| `assets/resourcepack/inventoryui/ui/chest_screen.json` | 제목 표식 `§s§g§m§u` 이면 메뉴 창, `§d§m§<n>` 이면 기계 창으로 분기(기계 창은 데스크톱 UI 프로필(`$desktop_screen`)만, 포켓 프로필은 크기 지정 상자 그대로) |
| `assets/resourcepack/sgui/ui/server_form.json`, `sg_form.json` | 입력 폼(제목이 `sg:in:` 으로 시작하는 커스텀 폼) |
| `assets/resourcepack/sgui/ui/scoreboards.json` | 사이드바 한 장을 HUD 부분으로 나눠 그린다(아래 "HUD") |
| `assets/resourcepack/sgui/ui/hud_screen.json` | 터치 포켓 배치에서 하트·배고픔을 핫바 위로(아래) |
| `assets/resourcepack/sgui/textures/ui/sg/*` | 메뉴가 쓰는 그림: 버튼·창·로비 버튼, 쪽 화살표·꺾쇠·고른 칸·로비 아이콘. 저장소 루트에서 `go run ./tools/uitextures`(전부 다시 쓰려면 `-f`)로 만든다(색은 그 파일 위쪽). 서랍 업그레이드 빈 칸(`slot_empty_*`)·메뉴 아이콘(`menu_*`)·플랫폼 아이콘은 `skygear` 팩에 있다 |
| `assets/resourcepack/skygear/ui/sg_machine.json` | 기계 창 레이아웃(`sg_machine`): 바닐라 작은 상자 화면에서 그 기계의 칸만 정해진 위치에 놓는다 |

## 메뉴 창

섬과 로비의 메뉴는 모두 메뉴 창에 그린다(`common/invui`, `common/ui`). 메뉴 창은 `invui.Size` 칸짜리 `chestui.Storage` 이고 제목에 `invui.Marker` 가 있다. 화면을 바꾸거나 다시 그리면 바뀐 칸만 다시 쓰고 창은 다시 열지 않으므로, 한 화면에서 다른 화면으로 가는 누름은 열린 창을 그 자리에서 바꾼다.

- `common/ui` 의 화면 함수(`Menu`, `CardMenu`, `ItemBoard`, `SplitMenu`, `SieveBoard`, `ListSplitMenu`, `ItemGrid`, `SearchGrid`, `SlotMenu`, `ChoiceMenu`, `SidesMenu`, `PlaneMenu`, `QuestBookMenu`, `HomeScreen`, `LoadingScreen`, `ConsoleScreen` 등)가 화면 이름과 데이터를 만들어 `showScreen` 으로 넘기고, `menuPageOf` 가 그 데이터를 메뉴 창의 칸으로 바꾼다. 버튼의 `Run` 은 그 플레이어 월드에서 돈다.
- 칸: 0번은 화면 키(어느 레이아웃을 그릴지), 1번은 제목, 2번은 본문이다. 탭·컨트롤 막대 버튼·목록 줄은 아이콘 칸과 버튼 칸을 하나씩 쓰고, 칸 격자(`invui.Cell0` 부터)는 칸마다 아이템 하나다.
  - 버튼 칸은 글을 이름으로 단 종이이고, 이름 끝에 종류 표식(`invui.Kind*`, 보이지 않는 색 코드)이 붙는다. 팩은 표식으로 모양(줄, 탭, 스위치, 머리글, 고른 칸 등)을 고른다.
  - 아이콘 칸은 아이템 아이콘(`ui.ItemIcon`)이면 그 아이템 자체, 그림이면 그림 경로를 이름으로 단 종이다. 클라이언트는 칸의 `#hover_text` 앞에 `§f` 를 붙이므로, 팩은 이름에서 `§o`·`§r`·`§f` 와 종류 표식을 떼고 그림 경로로 쓴다.
  - 칸 격자의 아이템은 개수 글과 툴팁을 이름으로 단다. 개수 글(`invui.CountMark`)은 이름 맨 앞에 고정 길이로 붙는 색 코드 묶음이라 툴팁에는 보이지 않는다. 팩은 그 바이트 수만큼 잘라 `§` 를 떼고 칸 오른쪽 아래에 그린다. 64가 넘는 개수(438, 12k)는 아이템 묶음에 담기지 않기 때문이다. 숫자만 남은 글에서 `§` 를 떼면 빈 글이 되므로(Windows 1.26.65.1 실측) `§r` 을 앞에 두고 떼며, 남는 `r` 과 빈칸은 잘라 보이지 않게 한다. 네 글자 개수는 작게 그린다.
- 자기 레이아웃이 없는 화면은 본문과 누를 수 있는 요소마다 한 줄로 그린다(`pressRows`).
- 탭·뒤로·도움말·쪽 버튼은 라벨 앞 표식(`ui.Tab`, `ui.Back`, `ui.Help`, `ui.BarPager`)으로 창 위의 탭과 컨트롤 막대에 놓이고, 표식은 글에서 빠진다(`chromeOf`).
- 창 위의 글(본문·머리글·상세)은 회색 창에 맞게 어두운 변형 색으로 바꾼다(`§f` 는 `§0`, `§7` 은 `§8` 등, `common/ui` `inkSwap`). 글 속의 클라이언트 표식(아이템 이름, 날짜)은 대체 글로 그린다(`i18n.Plain`, [i18n.md](i18n.md)).

### 팩

- `chest_screen.json` 이 제목에 표식이 있는 창을 바닐라 작은 상자 판으로 보내고, 그 판에 `sg_menu.overlay` 를 넣어 그 아래 바닐라 창을 가린다(`sg_menu_hide_vanilla`). `$screen_content` 를 다른 판으로 바꾸면 클라이언트가 창을 바로 닫으므로 바꾸지 않는다.
- `sg_menu.overlay` 는 불러올 때 제목으로 걸러지고(`ignored`), 모달 입력 판 안에 창을 그린다. 화면마다 판이 있고 0번 칸의 화면 키로 하나만 보인다. 스스로 보이기를 정하지 않는 컨트롤의 바인딩은 보이는 동안에만 돈다.
- 터치 배치(`pocket_screen`)도 같은 메뉴 창을 그린다.

### 누름과 닫기

- 칸 버튼의 누름은 바닐라 빠른 이동(`button.container_auto_place`)으로 간다. 칸 버튼은 바닐라 칸 버튼처럼 포커스를 켜고 `button.container_slot_hovered` 를 둔다. 이것이 없으면 클라이언트가 요청을 보내지 않는다.
  - 클라이언트는 서버 답을 기다리지 않고 칸을 비워 그리므로, 누른 버튼은 답이 올 때까지 잠깐 사라진다.
  - 하나 버리기(`button.drop_one`)는 쓰지 않는다. 버리기 요청 뒤에는 서버가 성공으로 답하고 칸을 바꿔 보내도, 클라이언트가 다음 입력이 있을 때까지 창의 변화를 그리지 않았다. 창고 핸들러에서 버리기를 취소해 요청이 실패하면 칸을 하나도 그리지 않았다(Windows 1.26.65.1 실측).
- 서버의 창고 핸들러는 칸에서 꺼내기·버리기를 그 칸의 누름으로 받고, 모든 이동을 취소한다(`invui` `handler`). 누른 칸의 요소가 돌고, 새 화면을 보이지 않았으면 `ui.Idle` 이 돌며, 그것도 화면을 보이지 않으면 창이 닫힌다.
- 창을 닫으면(닫기 버튼) 화면의 닫기 동작이 있으면 그것이, 없으면 `ui.Idle` 이 돈다.
- Esc(포켓의 뒤로가기도 같다)는 창을 닫지 않고 숨은 칸 `invui.SlotEsc` 를 누른다(`sg_menu.esc_button` 의 전역 매핑 `button.menu_cancel` → `button.container_auto_place`). 서버는 창을 연 채로 닫기 동작이나 `ui.Idle` 을 돌리고, 새 화면이 없을 때만 창을 닫는다. 그래서 다음 화면이 그 자리에서 바뀌고 창이 다시 열리며 깜빡이지 않는다. 메뉴 창 전체를 덮는 모달 입력 판(`cover`)은 키를 먹으므로, Esc 가 빠져나갈 길로 `button.menu_cancel`·`button.menu_inventory_cancel` → `button.menu_exit` 매핑도 둔다.
- 메뉴 창은 열고 닫을 때 화면 애니메이션이 없다(`chest_screen.json` 이 표식이 있는 제목에 `$screen_animations`·`$background_animations` 를 비운다).
- 발전과제(`ui.QuestBookMenu`)는 넓은 창(`KeyQuest`)에 그린다: 왼쪽에 장 목록(아이콘, 이름, 개수, 새 표시), 오른쪽에 장 제목과 어두운 판. 판은 퀘스트 하나에 칸 하나(`invui.QuestGrids` 의 11×10 또는 7×15)이고, 퀘스트는 칸 왼쪽 위 18픽셀에 모양(사각·마름모·원)과 상태(열림 흰색·완료 초록·잠김 회색과 빨간 금지 표시)로 그린다. 선은 칸 오른쪽 틈의 세로 길(lane)을 지나 다음 열로 간다. 칸 이름이 길면 끝이 잘리므로 종류 표식은 개수 글자 바로 뒤에 둔다.
- 창은 플레이어마다 하나다. `ui.FormOpen` 은 메뉴 창이나 입력 폼이 열려 있는지, `ui.ShowingKey` 는 열린 화면의 키를 알린다. 메뉴 창이 아닌 화면을 열면 커서에 든 아이템은 인벤토리로 돌아간다(`ui.ReturnCarried`).

### 로비와 로딩 화면

- 로비에 있는 동안(`ui.InLobby`) 화면 키에 `invui.KeyLobby` 가 붙고, 창 위에 게임 로고와 이름(`ui.LobbyHeader`)이 그려진다.
- 로비의 메인 화면(`ui.HomeScreen`, `platform/home.go`)은 로비에 있는 동안 닫히지 않는다. 칸은 `invui.Home*` 이다: 왼쪽 위 프로필, 오른쪽 위 아이콘 버튼(알림 수 표시), 왼쪽 타일, 새 알림·접속 중인 친구 줄, 세이브 카드(꺾쇠로 다른 세이브), `섬으로 출발`, `세이브 관리 · 새 게임`. Esc 는 시스템 메뉴(`ui.Home.Overlay`)를 메인 화면 위에 띄운다. 하위 화면(세이브 선택, 멀티 플레이, 친구, 칭호·치장, 설정, 안내, 입력·확인)에서 누른 뒤 새 화면이 없거나 창을 닫으면 `ui.Idle` 이 메인 화면을 다시 그린다.
- 운영자는 로비에서 명령을 콘솔 화면(`ui.ConsoleScreen`, `platform/console.go`)에 친다. 로비에는 채팅창이 없다.
- 세이브에 들어갈 때(`platform` `enter`) 로딩 화면(`ui.LoadingScreen`)을 띄우고, 섬의 청크가 와서 플레이어가 땅에 놓이면 닫는다(`ui.CloseLoading`). 깜빡이지 않게 최소 1.2초(`minLoading`)는 보인다. Esc 는 닫지 않고 같은 화면을 다시 보인다. 들어가지 못하면 메인 화면으로 돌아오고 사유는 토스트로 보낸다.

### 알림 (토스트)

메뉴 창이 열려 있으면 채팅이 보이지 않으므로 알림은 **토스트**(화면 위 알림창)로 보낸다. 창 위에도 보이고 다음 화면을 막지 않는다(`common/ui`).

- `ui.Notify`(실패 등 지금만 필요한 알림: 재료 부족 …): 토스트만. 제목 `알림`(빨강).
- `ui.Flash`(결과가 중요한 알림): 토스트와 같은 줄을 채팅에도(나중에 다시 읽기). 제목 `완료`(초록).
- 로비에는 채팅이 없어 `ui.Flash` 는 토스트만 보내고, 채팅으로 보낼 글은 `ui.Tell`(세이브 안이면 채팅, 로비면 토스트 `알림`)로 보낸다.
- 클라이언트는 토스트를 하나씩 차례로 보여 주므로, 플레이어마다 **`ui.ToastGap`(3초)에 하나**만 보낸다: 그 사이에 온 알림은 하나로 합쳐 앞 토스트가 끝날 때 가장 최근 메시지와 `(외 n건)` 으로 보낸다.
- 클라이언트는 아이템 사용 버튼을 누르고 있는 동안 사용 패킷을 되풀이해 보낸다. 메뉴 아이템 등으로 화면을 여는 사용은 0.5초 디바운스한다(`core.Session.Debounced`).

## 입력 폼

글을 받는 곳(`ui.Input`)은 커스텀 폼(`ModalFormRequest`) 하나다(`common/formui`, `common/ui` `openInput`). 메뉴 창은 글을 받을 수 없어서, 입력을 열면 메뉴 창을 닫고 폼을 연다.

- 제목은 `sg:in:` 과 화면 제목이다. `server_form.json` 이 제목이 이 접두어로 시작하는 커스텀 폼을 `sg_form.input_screen`(326×166 회색 창, 제목, 닫기 버튼, 라벨, 입력칸, 확인 버튼)으로 그리고, 다른 커스텀 폼은 바닐라대로 둔다.
- 확인은 입력한 글로 화면의 동작을 돌리고, 새 화면이 없으면 `ui.Idle` 을 부른다. 제목에 `ui.CreateToken` 이 있으면 확인 버튼이 `생성하기` 다.
- 폼을 닫으면 돌아갈 화면(`ui.Input` 의 `back`)이 있으면 그것을, 없으면 화면의 닫기 동작이나 `ui.Idle` 을 돈다.
- 검색(`ui.SearchGrid`)과 콘솔은 검색·실행 버튼이 입력 폼을 열고, 확인하면 결과를 담은 화면을 메뉴 창에 다시 그린다.

## 기계 창

- 기계 인벤토리는 그대로 엔티티 상자(`chestui.Storage`, 27칸)다. 제목 표식으로 기계별 레이아웃을 골라 입력·진행·출력·회수 칸만 보인다.
- 진행도 칸에는 `skygear:progress_0`~`progress_10` 아이템(채워지는 화살표)이 들어가 화살표가 차오르는 것처럼 보인다.
- 창 오른쪽 위 어두운 컨트롤 막대(`sg_machine.control_bar`)에 닫기 버튼 왼쪽으로 26번(정보 아이콘, 툴팁이 기계 설명)이 칸 배경 없이 18×18로 놓인다.
- 모두 꺼내기·모두 넣기는 모든 창고와 같이 창 오른쪽 바깥의 세로 버튼 줄(common inventoryui `sg_quick.side_bar`)이다. 버튼 밑에 칸마다 보이지 않는 버튼을 겹쳐 두고, 각각 누름을 그 칸의 바닐라 빠른 이동(`button.container_auto_place`, `consume_event: false`)으로 바꾸므로 클릭 한 번이 모든 칸의 쉬프트 클릭이 된다(Quick Container 팩의 기법, 코드는 쓰지 않음). 기계 창은 `$sg_machine_outputs` 식으로 꺼낼 칸을 거른다. 있는 것만 넣기는 창고 칸 컨트롤(`sg_quick_c<n>`)들을 담은 스택(`sg_quick_ids`)이 그 칸들의 `#item_id_aux` 를 한 번 읽어 `|` 로 이은 글자(`#sg_ids`, 9칸씩 묶어 잇는다)를 만들고, 인벤토리 칸의 숨은 버튼은 그 글자 하나만 `source_control_name` 으로 읽어(화면 루트부터 첫 일치) 자기 `|<id>|` 를 뺀 결과가 달라질 때만 보인다(`#visible`). 버튼마다 54개를 비교하지 않으므로 바인딩이 칸당 5개다. 툴팁은 버튼 hover 상태 안의 라벨이라 번역 키가 풀린다(`common.hover_text` 의 `#hover_text` 는 번역되지 않음).
  - `collection_index` 는 `collection_name` 을 가진 요소의 바로 아래 자식에서만 먹는다. `collection_panel` 안의 `stack_panel` 자식에 쓰면 "Unknown property [collection_index]" 오류가 나므로, `stack_panel` 자체에 `"collection_name": "container_items"` 를 준다.
- 모양은 바닐라 화로 화면과 같다: 밝은 회색 창(`textures/ui/dialog_background_opaque`), 바닐라 칸(`textures/ui/cell_image`), 진한 회색 제목(바닐라 `$title_text_color` 그대로)은 화로 화면(`furnace_label`)처럼 창 위 가운데 4px 아래에 있다. 제목 표식 `§d§m§<n>` 의 마지막 코드는 색 코드라 Go 가 표식 뒤에 `§r` 을 붙인다. 출력이 하나인 기계(분쇄기·전기 화로·합금로)의 출력 칸은 화로 결과 칸처럼 26×26이다. 진행도 칸은 칸 배경 없이 그림만이고, 화살표·링·배터리의 빈 부분은 바닐라 화로 화살표(`arrow_inactive`)와 같은 밝은 회색, 찬 부분은 흰색·주황이다. 정보 아이콘은 창 오른쪽 위 어두운 컨트롤 막대에 흰 그림으로, 모두 꺼내기·모두 넣기는 창 오른쪽 바깥에 바닐라 밝은 버튼으로 있다.
- 기계 창 레이아웃은 데스크톱 UI 프로필에만 있다. 포켓 프로필은 크기 지정 상자 그대로 보인다. `$desktop_screen`·`$pocket_screen` 은 기기가 아니라 UI 프로필을 따르므로, 클래식 프로필의 휴대폰은 기계 창과 버튼 줄을 터치로 쓰고 포켓 프로필의 PC 는 크기 지정 상자를 본다.

## 필터 창

스토리지 망의 가져오기·내보내기·외부 저장소를 누르면 필터 창이 열린다(`content/skygear/netstore/filterui.go`). 두 줄 상자로, 위 줄은 필터 칸, 아래 줄은 업그레이드 칸(칸마다 업그레이드 하나)·정보·모드(와 우선순위) 버튼이다. 필터 칸에 아이템을 놓거나 인벤토리에서 쉬프트 클릭하면 그 종류가 칸에 더해지고 아이템은 플레이어에게 남는다. 필터 칸을 꺼내면 그 칸이 비워진다.

## 요령

- 팩토리 `control_ids` 에 변수를 쓰면 "UI control reference not found" 오류가 난다. 크기 배열 안의 숫자 변수(`[ "$w", "100%c" ]`)는 "Dangling number" 오류가 나므로 크기는 배열 전체를 변수로(`"size": "$size"`) 둔다. `grid_item_template`·상속한 `$scrolling_content` 같은 자리도 변수 대신 컨트롤을 따로 정의한다.
- 가로 스택의 `100%c` 는 자식 높이를 모두 더한다. 높이를 가장 큰 자식에 맞추려면 `100%cm`, 왼쪽·오른쪽 판을 나란히 둘 때는 `panel` 안에 `offset` 으로 둔다.
- 글자 비교로 보이기를 정할 때는 먼저 글자를 `#변수`에 담고 `(not (#변수 = 'z'))` 로 비교한다. `(not ((A - B) = 'z'))` 처럼 식 안에서 바로 비교하면 참·거짓이 뒤집힌다.
- 표식은 식에 글자로 적는다(변수는 식 안에서 바뀌지 않는다). 판단은 Go 가 표식으로 미리 정하고 UI 는 단순 비교만 한다.
- 정적 검사로는 클릭 전달·스크롤·포커스를 알 수 없으므로 실제 클라이언트에서 확인한다([TODO](../TODO.md)).

## HUD(사이드바 줄을 나눠 그리기)

왼쪽 위(일차·플레이 시간, 운영자 부하 줄), 오른쪽 가운데 퀘스트 창(다음 발전과제)을 **사이드바 스코어보드 한 장**으로 보낸다(`common/hud`). 제목(title)은 WAILA 가 쓰고 한 번에 하나만 담겨서, 늘 떠 있어야 하는 글자를 같이 실으면 서로 덮는다. 사이드바 줄은 바꾸기 전까지 남아 있다.

- 서버: 목표(objective) 이름 `§d§h§u§d`, 줄마다 부분 표시를 앞에 붙인다: `§d§1` 왼쪽 위, `§d§2` 오른쪽 위, `§d§3` 퀘스트 제목, `§d§4` 퀘스트 줄. 줄 순서는 부분 안에서 그대로. Dragonfly 의 앞 공백(padding)은 끈다. 같은 글자 줄은 클라이언트가 합치므로 뒤에 `§r` 를 붙여 구분한다. 최대 15줄.
- 팩: `assets/resourcepack/sgui/ui/scoreboards.json` 이 바닐라 `scoreboard.scoreboard_sidebar` 를 화면 전체 패널로 바꾸고, 목표 이름에 표시가 있으면 `sg_hud`, 없으면 단순한 상자(`sg_plain_sidebar`, 다른 보드용)를 보인다. 부분마다 `scoreboard_players` 컬렉션 스택(`sg_row_stack`, 줄 라벨 `sg_row`)을 두는데, 길이(`#collection_length`)는 목표 이름에 표시가 있을 때만 `#scoreboard_sidebar_size` 이고 단순한 상자는 그 반대라 보이지 않는 쪽은 줄을 만들지 않는다(`#visible` 은 그리기만 막는다). 줄 글자 식은 식 전체를 변수(`$sg_row_text`)로 넘기고(식 일부에 넣은 변수는 풀리지 않는다), 줄 라벨은 `#player_name_sidebar` 에 그 표시가 있을 때만 보이며 표시를 뺀 글자를 쓴다. 바닐라 이름은 Mojang `bedrock-samples` `resource_pack/ui/scoreboards.json` 에서 확인했다. 라벨은 `localize: false`(번역을 거치면 `%` 가 사라진다).
- 숨겨진 줄은 스택에서 자리를 차지하지 않는다.
- 터치 화면(`$touch`)은 채팅을 왼쪽 위에, 점프·웅크리기 버튼을 오른쪽 가운데에 그리므로, 왼쪽 위 줄을 오른쪽 위(오른쪽 맞춤, 오른쪽 위 줄 위)에 그리고(`top_left_touch`, `sg_row_stack` 에 `$sg_row_anchor: top_right`, 바깥 `top_right` 스택은 `use_child_anchors: true` 라 자식 앵커를 따른다) 퀘스트 창을 그 아래에 둔다(`quest_touch`).
- 경험치 바는 바닐라 그대로다.

## HUD (터치의 포켓 배치)

터치 화면에서 UI 프로필이 포켓이면 바닐라 HUD 가 `not_centered_gui_elements` 를 써서 하트를 왼쪽 위, 배고픔을 오른쪽 위에 그린다. 그 자리는 이 서버의 일차·플레이 시간 줄과 겹치므로, `assets/resourcepack/sgui/ui/hud_screen.json` 이 경로 키로 그 판의 하트·갑옷·배고픔·공기 방울·탈것 하트를 `ignored` 로 끄고, `modifications` 의 `insert_back` 으로 같은 렌더러를 핫바 위(클래식 터치 배치, 바닐라 `centered_gui_elements_at_bottom_middle_touch` 와 같은 위치)에 둔 패널을 끼운다. 바닐라 `controls` 배열은 다시 쓰지 않으므로 종이 인형은 왼쪽 위에 그대로 남고, 바닐라가 나중에 더하는 컨트롤도 지워지지 않는다. 핫바는 포켓 배치가 따로 그리므로 넣지 않는다.

## 흑백 PNG

클라이언트는 흑백(grayscale) PNG 를 반투명하게 그린다. 텍스처 최적화 도구가 회색 그림을 흑백으로 바꾸므로, 팩의 PNG 는 RGB·RGBA 로 둔다. 도구는 `imgio.NRGBA` 로 바꿔 쓰고, `assets` 의 `TestPNGNotGrayscale` 이 흑백·흑백+알파 PNG 를 이름과 함께 잡는다.
