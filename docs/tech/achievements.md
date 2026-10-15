# 업적·칭호·카운터

기획: [01-platform.md §5](../design/01-platform.md). 코드: `internal/account`(엔진), `internal/platform/cosmetics.go`(화면), `internal/games/dragonfactory/`의 `counter*.go`·`collect.go`·`machinecount.go`·`achcatalog.go`(드래곤팩토리의 카탈로그와 집계).

## 구조

| 층 | 하는 일 |
|---|---|
| `account.Register` / `account.Grant` | 업적을 등록하고 계정에 한 번만 준다. 준 업적은 메모리에 캐시해(`granted`) 같은 업적을 또 줄 때 DB를 부르지 않는다. 새로 얻으면 메시지·소리, 칭호가 있으면 칭호 안내 |
| `account.RegisterCounter` / `account.Count` | 키마다 단계(`Tier{Need, ID}`)를 두고, 횟수가 단계를 넘는 순간 그 업적을 준다. 등록되지 않은 키는 무시한다 |
| `account.Flush` | 쌓인 횟수를 DB `counters`(계정, 키, 값)에 더한다. `Count` 가 마지막 저장에서 20초가 지났을 때, 접속을 떠날 때(`Forget`), 서버를 끌 때(`main.go`) 부른다. DB에 없는 계정의 횟수는 버린다 |
| `account.Groups` / `SetGroupOrder` | 칭호 고르기와 업적 목록의 분류와 순서 |
| 게임의 `counterCatalog` | 키, 분류, 단계(횟수)의 표. 시작할 때 모두 등록한다 |

횟수는 계정 단위이고 세이브와 무관하다. 읽을 때(`Counter`, `Progress`)는 처음 한 번 DB에서 메모리로 불러온다.

## 카탈로그와 글

- 업적 ID는 `df_<키>_<횟수>`(예: `df_stick_dirt_100`). 글은 `ko.json`의 `achievement.<ID>.name|desc|title`, 분류 이름은 `achgroup.<분류>`이다.
- 이름은 "<행동> 첫 경험"이거나 "<행동> <횟수><단위>"이고, 설명은 "…을 N번 …한다"로 쓴다. 칭호는 14자 이하이고 전체에서 겹치지 않는다(`TestCounterCatalog`).
- 분류: 손작업(`hand`), 농사와 식물(`farm`), 체질(`sift`), 기계와 공장(`factory`), 에너지(`energy`), 장비(`gear`), 전투(`combat`), 성장(`progress`), 섬(`island`), 수집(`collect`), 생활(`life`), 계정(`account`, 플랫폼의 접속 일수).
- 카운터 없이 직접 주는 업적: 첫 기계·청크 10개·첫 연구·웨이브·장비·꽃·용의 알(`achievements.go`, `project.go`), 연구 전부·발전과제 전부(`collect.go`), 아이템을 처음 가질 때(`haveCatalog`, 1초마다 인벤토리를 본다).

## 카운터를 더하는 법

1. 행동이 일어나는 곳에서 `count(p, "키", n)`(그 플레이어) 또는 `countAll(tx, "키", n)`(기계·식물이 스스로 한 일: 섬에 접속한 모든 플레이어)을 부른다. 기계 안에서는 `st.done++`(주기 하나), `st.made += re`(발전), `st.tally("키", n)`을 쓰면 `Tick` 끝에서 한꺼번에 센다(`machinecount.go`).
2. `achcatalog.go`에 `{"키", "분류", []int64{횟수…}}`를 더한다. 종류별로 같은 규칙인 키(`place_<기계>`, `cycle_<기계>`, `kill_<몹>`, `crop_<작물>`, `sift_mesh_<메쉬>`, `pebble_<돌>`, `lab_<정수>`, `wave_level_<단계>`)는 접두사로 센다.
3. `ko.json`에 단계마다 `achievement.df_<키>_<횟수>.name|desc|title`을 쓴다.
4. 검사가 막아 준다: 모든 글이 있는지·칭호가 겹치지 않는지(`TestCounterCatalog`), 키가 코드 어딘가에서 세어지는지·종류 이름이 실제로 있는지(`counterhooks_test.go`).

## 세는 곳

| 키 | 위치 |
|---|---|
| `stick_dirt`·`stick_grass`·`stick_other`·`pebble_*` | `handwork.go` `useStick` |
| `strip_log` | `handwork.go` `stripFiber` |
| `block_break`·`log_break`·`leaves_break`·`hammer_*` | `counterhooks.go` `countBreak`(`handler.go` `HandleBlockBreak`) |
| `block_place`·`crop_plant`·`conduit_place` | `handler.go` `HandleBlockPlace` |
| `walk` | `counterhooks.go` `countWalk`(`HandleMove`, 소수 부분을 모아 한 블록마다) |
| `sneak`·`sprint`·`jump`·`punch_air`·`chat`·`eat`·`pickup`·`death`·`block_pick`·`hotbar`·`item_drop`·`hurt`·`command`·`hit_monster` | `handler.go` 이벤트 핸들러 |
| `sapling_grow`·`sapling_cmd` | `sapling.go` |
| `crop_harvest`·`crop_<작물>` | `cropinfo.go` |
| `compost_fill`·`compost_dirt`·`barrel_water`·`barrel_lava` | `composter.go`, `barrel.go` |
| `bloom_*` | `magic.go` `resetPlant` |
| `fairy_gift` | `fairy.go` |
| `sift_hand`·`sift_mesh_*` | `sieve.go` |
| `machine_place`·`place_<기계>`·`drawer_place` | `machine.go` `placeMachine`, `drawer.go` |
| `cycle_total`·`cycle_<기계>`·`energy_made`·`lab_<정수>` | `machinecount.go`와 각 기계의 `done`·`made`·`tally` |
| `tool_made`·`tool_mod`·`tool_levelup` | `tinkerform.go` |
| `kill_total`·`kill_<몹>`·`wave_win`·`wave_fail`·`wave_level_N` | `wave.go` |
| `rpg_level`·`skill_spend` | `rpg.go` |
| `research_done`·`project_done`·`essence_spent`·`adv_done` | `research.go`, `advancement.go` |
| `research_open`·`adv_view`·`adv_track`·`codex_open`·`menu_open` | 각 화면 |
| `chunk_buy`·`void_return` | `island.go` |
| `play_seconds` | `second.go` |
| `login` | `internal/platform/platform.go` `Join` |

## 화면

칭호 고르기와 업적 목록은 분류를 먼저 고른다(`cosmetics.go`). 칭호는 분류마다 한 쪽 24개씩 쪽을 넘긴다. 못 얻은 카운터 업적은 `지금/목표`를 보인다. 이름 색은 업적 수로 열린다(`account.Colors`).

## 성능

- `Count`는 전역 잠금 안에서 맵 두 번만 만지고, 단계를 넘을 때만 DB를 건드린다. 걷기는 한 블록을 넘을 때만 부른다.
- 기계의 집계(`countWork`)는 끝난 주기가 있을 때만 플레이어를 돈다. 발전량은 1초에 한 번 모은다.
- 수집 업적은 플레이어당 1초에 한 번 인벤토리를 본다.
