# 운영자 도구 (`/dfadmin`)

테스트용. `platform.toml` 의 `[Admin] Operators` 에 있는 플레이어 이름만 보고 쓸 수 있다. 세이브 안에서 쓴다.
코드: `internal/games/dragonfactory/admin.go`.

| 명령 | 하는 일 |
|---|---|
| `/dfadmin level <n>` | 세이브 레벨을 n 으로(납품 수량 초기화) |
| `/dfadmin points <n>` | 코어 강화 포인트를 n 으로 |
| `/dfadmin tierspeed <티어> <n>` | 그 티어의 코어 속도 강화 Lv 를 n 으로 |
| `/dfadmin money <n>` | 내 소지금에 n 더하기 |
| `/dfadmin give <ID> [개수]` | 상점 상품 ID·기계 종류(`drawer`, `drawer_key`, `hammer`, `sieve`, `auto_sieve`, `crusher`, `auto_miner`, `sell_box`, 코어 `core_t1`~`core_t4`, 받침 `base_cobblestone`, 연구 노트 `research_note` 등) |
| `/dfadmin item <이름> [개수]` | 아무 아이템(`iron_ore`, `minecraft:diamond` 등) |
| `/dfadmin urgent` | 긴급 매입을 지금 시작(진행 중이면 끝내고 새로) |
| `/dfadmin skip <분>` | 인스턴스 시간을 n분 앞당김(날짜 변경·긴급 매입·분실물 만료 확인용) |
| `/dfadmin timed <초>` | 손에 든 아이템을 지금부터 n초(현실 시간) 기간제로. 0 이면 기간 해제 ([기간제 아이템](timed-items.md)) |
| `/dfadmin fairy` | 선물 요정을 지금 나에게 부름 |
| `/dfadmin line` | 시작 코어에 샘플 무인 라인(채굴기·분쇄기·호퍼·상자·판매함) 설치 |
| `/dfadmin status` | `level= money= points= speed1= budget= sold=` 출력(시나리오 확인용) |
