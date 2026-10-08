# inventoryui

[tedo0627/InventoryUIResourcePack](https://github.com/tedo0627/InventoryUIResourcePack)
(MIT, `LICENSE`) 의 `ui/` 를 그대로 가져왔다 (커밋 `394fb0a`, 2025-07-27).
manifest 의 UUID 는 원본 팩과 캐시가 섞이지 않도록 새로 만들었다.

`internal/chestui` 가 여는 엔티티 상자 창의 제목 앞에 붙는
`§{줄 수}§{스크롤 0/1}` + `§r` 9개 접두어를 읽어 창 크기를 정한다.
접두어가 없는 일반 상자는 바닐라 화면 그대로다.

- PC: 1~6줄 고정 그리드, 54칸 초과면 6줄 높이 스크롤
- 모바일(pocket): 접두어가 있으면 스크롤 그리드

## dragonfactory 수정

- `ui/chest_screen.json` 끝에 드래곤팩토리 기계 창 분기 4개를 추가했다(제목 표식 `§d§m§1`~`§d§m§4`, PC 화면만).
  기계 창 레이아웃은 `assets/resourcepack/dragonfactory/ui/df_machine.json`.
