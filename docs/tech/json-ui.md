# JSON UI (HUD·창고 창)

JSON UI 는 HUD 와 아이템이 오가는 창(인벤토리·상자·기계 창)에만 쓴다. 메뉴는 모두 Ore UI DDUI 화면이다([oreui.md](oreui.md)).
HUD 는 게임과 상관없어 `assets/resourcepack/sgui` 에, 상자 창은 `assets/resourcepack/inventoryui` 에, 기계 창처럼 이 게임에만 있는 것은 `skygear` 팩에 있다. `cmd/skygear/main.go` 가 `skygear` 를 공통 팩보다 먼저(스택 위에) 싣는다.

## 파일

| 파일 | 내용 |
|---|---|
| `assets/resourcepack/sgui/ui/scoreboards.json` | 사이드바 한 장을 HUD 부분으로 나눠 그린다(아래 "HUD") |
| `assets/resourcepack/sgui/ui/hud_screen.json` | 터치 포켓 배치에서 하트·배고픔을 핫바 위로(아래) |
| `assets/resourcepack/sgui/textures/ui/sg/*` | 메뉴(DDUI 화면)가 쓰는 그림: 버튼·창·로비 버튼(`oui-mc.css`), 쪽 화살표·꺾쇠·고른 칸·로비 아이콘(`common/ui`), 희귀도 테두리 `deco_<희귀도>_0`(`common/ui` `slotDeco`). `common/` 에서 `go run ./tools/uitextures`, `go run ./tools/slotdecotex assets/resourcepack/slotdeco assets/resourcepack/sgui` 로 다시 만든다(색은 그 파일 위쪽). 서랍 업그레이드 빈 칸(`slot_empty_*`)·메뉴 아이콘(`menu_*`)·플랫폼 아이콘은 `skygear` 팩에 있다 |
| `assets/resourcepack/skygear/ui/sg_machine.json` | 기계 창 레이아웃(`sg_machine`): 바닐라 작은 상자 화면에서 그 기계의 칸만 정해진 위치에 |
| `assets/resourcepack/inventoryui/ui/chest_screen.json` | 제목 표식 `§d§m§<n>` 이면 기계 창으로 분기(PC 화면만, 모바일은 크기 지정 상자 그대로) |

## 기계 창

- 기계 인벤토리는 그대로 엔티티 상자(`chestui.Storage`, 27칸). 제목 표식으로 기계별 레이아웃을 골라 입력·진행·출력·회수 칸만 보인다.
- 진행도 칸에는 `skygear:progress_0`~`progress_10` 아이템(채워지는 화살표)이 들어가 화살표가 차오르는 것처럼 보인다.
- 창 오른쪽 위 어두운 컨트롤 막대(`sg_machine.control_bar`)에 닫기 버튼 왼쪽으로 26번(정보 아이콘, 툴팁이 기계 설명)이 칸 배경 없이 18×18로 놓인다.
- 모두 꺼내기·모두 넣기는 모든 창고와 같이 창 오른쪽 바깥의 세로 버튼 줄(common inventoryui `sg_quick.side_bar`)이다. 버튼 밑에 칸마다 보이지 않는 버튼을 겹쳐 두고, 각각 누름을 그 칸의 바닐라 빠른 이동(`button.container_auto_place`, `consume_event: false`)으로 바꾸므로 클릭 한 번이 모든 칸의 쉬프트 클릭이 된다(Quick Container 팩의 기법, 코드는 쓰지 않음). 기계 창은 `$sg_machine_outputs` 식으로 꺼낼 칸을 거른다. 있는 것만 넣기는 인벤토리 칸의 숨은 버튼이 이름 붙은 창고 칸 컨트롤(`sg_quick_c<n>`)의 `#item_id_aux` 를 `source_control_name` 으로 읽어(화면 루트부터 첫 일치) 같은 것이 있을 때만 보인다(`#visible`). 툴팁은 버튼 hover 상태 안의 라벨이라 번역 키가 풀린다(`common.hover_text` 의 `#hover_text` 는 번역되지 않음).
  - `collection_index` 는 `collection_name` 을 가진 요소의 바로 아래 자식에서만 먹는다. `collection_panel` 안의 `stack_panel` 자식에 쓰면 "Unknown property [collection_index]" 오류가 나므로, `stack_panel` 자체에 `"collection_name": "container_items"` 를 준다.
- 모양은 바닐라 화로 화면과 같다: 밝은 회색 창(`textures/ui/dialog_background_opaque`), 바닐라 칸(`textures/ui/cell_image`), 진한 회색 제목(바닐라 `$title_text_color` 그대로)은 화로 화면(`furnace_label`)처럼 창 위 가운데 4px 아래. 제목 표식 `§d§m§<n>` 의 마지막 코드는 색 코드라 Go 가 표식 뒤에 `§r` 을 붙인다. 출력이 하나인 기계(분쇄기·전기 화로·합금로)의 출력 칸은 화로 결과 칸처럼 26×26. 진행도 칸은 칸 배경 없이 그림만이고, 화살표·링·배터리의 빈 부분은 바닐라 화로 화살표(`arrow_inactive`)와 같은 밝은 회색, 찬 부분은 흰색·주황이다. 정보 아이콘은 창 오른쪽 위 어두운 컨트롤 막대에 흰 그림으로, 모두 꺼내기·모두 넣기는 창 오른쪽 바깥에 바닐라 밝은 버튼으로 있다.
- 기계 창 레이아웃은 PC 화면에만 있다. 모바일은 크기 지정 상자 그대로 보인다.

## 요령

- 팩토리 `control_ids` 에 변수를 쓰면 "UI control reference not found". 크기 배열 안의 숫자 변수(`[ "$w", "100%c" ]`)는 "Dangling number" 이므로 크기는 배열 전체를 변수로(`"size": "$size"`) 둔다. `grid_item_template`·상속한 `$scrolling_content` 같은 자리도 변수 대신 컨트롤을 따로 정의한다.
- 가로 스택의 `100%c` 는 자식 높이를 모두 더한다. 높이를 가장 큰 자식에 맞추려면 `100%cm`, 왼쪽·오른쪽 판을 나란히 둘 때는 `panel` 안에 `offset` 으로 둔다.
- 글자 비교로 보이기를 정할 때는 먼저 글자를 `#변수`에 담고 `(not (#변수 = 'z'))` 로 비교한다. `(not ((A - B) = 'z'))` 처럼 식 안에서 바로 비교하면 참·거짓이 뒤집힌다.
- 표식은 식에 글자로 적는다(변수는 식 안에서 바뀌지 않는다). 판단은 Go 가 표식으로 미리 정하고 UI 는 단순 비교만 한다.
- 정적 검사로는 클릭 전달·스크롤·포커스를 알 수 없으므로 실제 클라이언트에서 확인한다([TODO](../TODO.md)).

## HUD(사이드바 줄을 나눠 그리기)

왼쪽 위(일차·플레이 시간, 운영자 부하 줄), 오른쪽 가운데 퀘스트 창(다음 발전과제)을 **사이드바 스코어보드 한 장**으로 보낸다(`common/hud`). 제목(title)은 WAILA 가 쓰고 한 번에 하나만 담겨서, 늘 떠 있어야 하는 글자를 같이 실으면 서로 덮는다. 사이드바 줄은 바꾸기 전까지 남아 있다.

- 서버: 목표(objective) 이름 `§d§h§u§d`, 줄마다 부분 표시를 앞에 붙인다: `§d§1` 왼쪽 위, `§d§2` 오른쪽 위, `§d§3` 퀘스트 제목, `§d§4` 퀘스트 줄. 줄 순서는 부분 안에서 그대로. Dragonfly 의 앞 공백(padding)은 끈다. 같은 글자 줄은 클라이언트가 합치므로 뒤에 `§r` 를 붙여 구분한다. 최대 15줄.
- 팩: `assets/resourcepack/sgui/ui/scoreboards.json` 이 바닐라 `scoreboard.scoreboard_sidebar` 를 화면 전체 패널로 바꾸고, 목표 이름에 표시가 있으면 `sg_hud`, 없으면 단순한 상자(`sg_plain_sidebar`, 다른 보드용)를 보인다. 부분마다 `scoreboard_players` 컬렉션 스택(`#scoreboard_sidebar_size` → `#collection_length`)을 두고, 줄 라벨은 `#player_name_sidebar` 에 그 표시가 있을 때만 보이며 표시를 뺀 글자를 쓴다. 바닐라 이름은 Mojang `bedrock-samples` `resource_pack/ui/scoreboards.json` 에서 확인했다. 라벨은 `localize: false`(번역을 거치면 `%` 가 사라진다).
- 숨겨진 줄은 스택에서 자리를 차지하지 않는다.
- 터치 화면(`$touch`)은 채팅을 왼쪽 위에, 점프·웅크리기 버튼을 오른쪽 가운데에 그리므로, 왼쪽 위 줄을 오른쪽 위(오른쪽 맞춤, 오른쪽 위 줄 위)에 그리고(`sg_top_left_touch_stack`) 퀘스트 창을 그 아래에 둔다(`quest_touch`).
- 경험치 바는 바닐라 그대로다.

## HUD (터치의 포켓 배치)

터치 화면에서 UI 프로필이 포켓이면 바닐라 HUD 가 `not_centered_gui_elements` 를 써서 하트를 왼쪽 위, 배고픔을 오른쪽 위에 그린다. 그 자리는 이 서버의 일차·플레이 시간 줄과 겹치므로, `assets/resourcepack/sgui/ui/hud_screen.json` 이 그 판을 다시 정의해 하트·갑옷·배고픔·공기 방울·탈것 하트를 핫바 위(클래식 터치 배치, 바닐라 `centered_gui_elements_at_bottom_middle_touch` 와 같은 위치)에 둔다. 핫바는 포켓 배치가 따로 그리므로 넣지 않고, 종이 인형만 왼쪽 위에 남는다.

## 흑백 PNG

클라이언트는 흑백(grayscale) PNG 를 반투명하게 그린다. 텍스처 최적화 도구가 회색 그림을 흑백으로 바꾸므로, 팩의 PNG 는 RGB·RGBA 로 둔다. 최적화한 뒤 `file` 로 grayscale 이 없는지 본다.
