# 바닐라 UI에서 배운 것

게임 기본 화면(인벤토리·제작·화로)의 모양은 `Mojang/bedrock-samples` 의 `resource_pack/ui` 를 읽어 확인했다(`furnace_screen_pocket.json`, `inventory_screen.json`, `ui_common.json`, `ui_template_tabs.json`, `_global_variables.json`). **그림은 클라이언트에 이미 있으므로 우리 팩은 `textures/ui/...` 경로로 가리키기만 한다**(파일을 복사하지 않는다). 작은 설명은 [ui-style.md](ui-style.md).

## 화면 조각과 그림

| 조각 | 바닐라 그림 / 쓰임 |
|---|---|
| 창 바탕(둥근 밝은 회색) | `textures/ui/dialog_background_opaque` (`common.common_panel` 의 기본 `$dialog_background`) |
| 위쪽 컨트롤 막대(탭 토글·도움말·닫기를 담는 어두운 둥근 상자) | `textures/ui/toolbar_background` (`common.toolbar_background_image`, 크기 `["100%cm","100% + 8px"]`) |
| 닫기 버튼(밝은 바탕) | `common.light_close_button`: `close_button_default_light` / `_hover_light` / `_pressed_light`, 그림 15×15, 버튼 21×21 |
| 도움말 버튼 | `common.help_button`(15×15): `how_to_play_button_default_light` / `_hover_light` / `_pressed_light` |
| 위 탭 | `TabTopBack`(안 고른 탭) · `TabTopFront`(고른 탭), 호버 `...Hover`, 가장 왼쪽·오른쪽은 `...LeftMost` · `...RightMost`. 옆 탭은 `TabLeft*` · `TabRight*`. 아이콘만 있는 24×25 |
| 레이아웃 토글(레시피북·인벤토리) | `common.layout_toggle`(25×18), `recipe_book_button_borderless_light...` |
| 아이템 칸 | `textures/ui/cell_image`, 선택 `highlight_slot` · `pocket_ui_highlight_slot` |
| 레시피 목록의 안쪽 영역 | `recipe_book_pane_bg`, `recipe_back_panel` |
| 검색 칸 | `edit_box_indent`(`_hover`) |
| 스크롤 | `ScrollHandle`, `ScrollRail`, `ScrollGutterWithBG` |
| 구분선 | `dialog_divider`, `divider2`, `divider3` |

## 글자색 변수 (`_global_variables.json`)

- 제목 글자 `$pocket_title_text_color` = `[0.3, 0.3, 0.3]`(그림자 없음, 가운데 정렬).
- 탭 글자: 안 고른 `$tab_unchecked_text_color` = `[0.8784 ×3]`(밝음), 고른 `$tab_checked_text_color` = `[0.3, 0.3, 0.3]`. 안 고른 탭 그림이 어둡고 고른 탭 그림이 밝다.
- 툴팁/상세 글자 `$pocket_item_details_color` = 흰색.

## 구조에서 배운 것

- **탭은 토글이다**(`common_tabs.base_tab` 안의 `common.toggle`): 상태마다(unchecked, unchecked_hover, locked, checked …) 같은 `tab_panel` 에 그림과 글자색·아이콘색 변수만 바꿔 넣는다. 우리 폼은 서버가 탭 버튼을 다시 보내므로 토글 대신 선택 표식(`§0§2§r`)으로 같은 모양을 낸다.
- 컨트롤 막대는 가로 스택에 `패딩 → 레이아웃 토글들 → 도움말 → 닫기 → 패딩` 순서다(`inventory_screen.json` 의 toolbar_panel). 닫기 버튼 칸 폭 15, 도움말은 `100%c`.
- 창 위 제목은 `anchor center` 로 가운데에 쓴다. 헤더 높이는 28(`furnace_screen_pocket.json` 의 `header`).
- 슬롯은 `common.pocket_ui_container_item` 로 `$background_images` 에 `cell_image` 를 둔다. 이름 지을 때 `$item_collection_name` 으로 컬렉션을 가리킨다(상자 UI는 [chestui.md](chestui.md)).

## 우리 폼에 적용한 것

창(`df_form.window`)·기계 창 배경(`df_machine.dark_bg`)은 `dialog_background_opaque`, 컨트롤 막대는 `toolbar_background`, 닫기는 `light_close_button`, 도움말은 `how_to_play_button_*_light`, 위 탭은 `TabTopBack`/`TabTopFront`, 아이템 칸은 `cell_image`. 안쪽 판(연구 분할의 왼쪽·오른쪽 등)은 우리 그림 `textures/ui/df/panel`(회색 판)이다.
