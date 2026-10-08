# JSON UI (드래곤팩토리 테마)

폼과 기계 창의 모양은 리소스팩 JSON UI 로 바꾼다. 서버는 폼 내용만 보내고, 모양은 클라이언트가 팩으로 그린다.

## 파일

| 파일 | 내용 |
|---|---|
| `assets/resourcepack/dragonfactory/ui/server_form.json` | 바닐라 `server_form.main_screen_content`(전체 화면으로)·`long_form`·`custom_form` 을 덮어써 모든 서버 폼을 어두운 테마로. 제목 앞 `§d§f§c§r`(Go `ui.CardToken`)이면 카드 격자 |
| `assets/resourcepack/dragonfactory/ui/df_form.json` | 테마 컨트롤(`df_form` 네임스페이스): 창(패널·제목 막대·닫기), 목록 버튼(아이콘+글자), 카드 격자, 입력 폼 |
| `assets/resourcepack/dragonfactory/ui/df_machine.json` | 기계 창 레이아웃(`df_machine`): 바닐라 작은 상자 화면에서 그 기계의 칸만 정해진 위치에 |
| `assets/resourcepack/inventoryui/ui/chest_screen.json` | 제목 표식 `§d§m§1`~`§d§m§4` 이면 기계 창으로 분기(PC 화면만, 모바일은 크기 지정 상자 그대로) |
| `assets/resourcepack/dragonfactory/textures/ui/df/*` | 테마 텍스처. `go run ./tools/uitextures` 로 다시 만든다(색은 그 파일 위쪽) |

## 폼 (메뉴·확인·입력)

- 항목을 넣고 빼지 않는 메뉴는 **폼**으로 만든다(상자 아님): 드래곤팩토리 메뉴, 상점 구매·판매, 분실물 보관함, 섬 확장 지도 등.
- 카드 격자 폼(`ui.CardMenu`): 아이콘 위·글자 아래 카드가 한 줄에 3개(90px 칸, 격자 폭 270px 고정·가운데), 버튼 수만큼 줄이 늘어난다(스크롤). 메뉴(3×3)·상점·판매·분실물.
  - 격자 항목 수는 `#form_button_length` → `#maximum_grid_items`, 격자 높이는 `100%c` 로 묶는다(`#form_button_contents` 나 `default` 높이면 버튼이 하나도 안 생긴다 — 6차 실기).
- 목록 폼(`ui.Menu`): 버튼마다 아이콘(`ui.Option.Icon`, 텍스처 경로) + 글자. 아이콘은 바닐라 텍스처 경로이거나 커스텀 아이템이면
  Dragonfly 가 만든 팩의 `textures/items/<이름>`(`dragonfactory.iconOf`).
- 확인 폼(`ui.Confirm`)은 목록 폼과 같은 모양으로 보이는 것을 기대(클라이언트가 modal 을 long_form 으로 그리는지 실기 확인 필요).
- 입력 폼(custom_form): 어두운 창 안에 바닐라 입력 칸 그대로(밝은 카드는 글자가 안 보여 뺐다).
- 폼 뒤를 어둡게 하는 배경은 화면 일부에만 그려져 뺐다.

## 기계 창

- 기계 인벤토리는 그대로 엔티티 상자(`chestui.Storage`, 27칸). 제목 표식으로 기계별 레이아웃을 골라 입력·진행·출력·회수 칸만 보인다.
- 진행도 칸에는 `dragonfactory:progress_0`~`progress_10` 아이템(채워지는 화살표)이 들어가 화살표가 차오르는 것처럼 보인다.
- 26번 칸은 "기계 회수" 버튼.

## 실기 확인 전에는 미완료

정적 검사로는 클릭 전달·스크롤·포커스가 동작하는지 알 수 없다. [TODO](../TODO.md) 의 실기 항목으로 확인한다.
