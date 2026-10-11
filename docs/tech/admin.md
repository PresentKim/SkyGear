# 운영자 도구 (`/dfadmin`)

테스트용. `platform.toml` 의 `[Admin] Operators` 에 있는 플레이어 이름만 보고 쓸 수 있다. 세이브 안에서 쓴다.
코드: `internal/games/dragonfactory/admin.go`.

| 명령 | 하는 일 |
|---|---|
| `/dfadmin level <n>` | 세이브 레벨을 n 으로(납품 수량 초기화) |
| `/dfadmin points <n>` | 코어 강화 포인트를 n 으로 |
| `/dfadmin tierspeed <티어> <n>` | 그 티어의 코어 속도 강화 Lv 를 n 으로 |
| `/dfadmin money <n>` | 내 소지금에 n 더하기 |
| `/dfadmin give <ID> [개수]` | 상점 상품 ID·기계 종류(`drawer`, `drawer_key`, `hammer`, `sieve`, `auto_sieve`, `crusher`, `auto_miner`, `sell_box`, 코어 `core_t1`~`core_t4`, 코어 기반 `base_cobblestone`, 연구 노트 `research_note` 등) |
| `/dfadmin item <이름> [개수]` | 아무 아이템(`iron_ore`, `minecraft:diamond` 등) |
| `/dfadmin urgent` | 긴급 매입을 지금 시작(진행 중이면 끝내고 새로) |
| `/dfadmin skip <분>` | 인스턴스 시간을 n분 앞당김(날짜 변경·긴급 매입·분실물 만료 확인용) |
| `/dfadmin timed <초>` | 손에 든 아이템을 지금부터 n초(현실 시간) 기간제로. 0 이면 기간 해제 ([기간제 아이템](timed-items.md)) |
| `/dfadmin fairy` | 선물 요정을 지금 나에게 부름 |
| `/dfadmin line` | 시작 코어에 샘플 무인 라인(채굴기·분쇄기·호퍼·상자·판매함) 설치 |
| `/dfadmin look <yaw> <pitch>` | 시선을 돌리고(yaw 0 남쪽, pitch 90 아래) WAILA 패널을 바로 갱신, 보는 블록 출력(시나리오·실기 확인용) |
| `/dfadmin status` | `level= money= points= speed1= budget= sold=` 출력(시나리오 확인용) |
| `/dfadmin tps` | 내 HUD 왼쪽 위 서버 부하 줄 켜기·끄기(아래, 로비에서도) |

## 서버 부하 표시

운영자가 `/dfadmin tps` 로 켜고 끈다(플레이어마다, 서버를 다시 켜면 꺼짐; 로비에서도 됨). 켜면 HUD 왼쪽 위(일차 줄 아래)에 1초마다 한 줄로 보인다(로비에서는 그 줄만). 코드: `internal/perf`(`Line`, `Toggle`), `platform/load.go`, `dragonfactory/sidebar.go`.

`TPS 20.0 · 부하 12% · 틱 6.1ms/9.7ms · 메모리 76MB`

| 항목 | 뜻 |
|---|---|
| TPS | 지금 있는 월드의 최근 5초 초당 틱 수(정상 20). 19 아래 노랑, 15 아래 빨강 |
| 부하 | 평균 틱 시간 ÷ 50ms. 50% 아래 초록, 100% 아래 노랑, 넘으면 빨강(틱이 밀려 TPS 가 떨어짐) |
| 틱 | 최근 5초 평균/최대 틱 시간(그 월드의 다른 작업에 밀려 기다린 시간 포함). 읽는 법은 [performance.md](performance.md) |
| 메모리 | Go 힙 메모리 |

## 부하 시험

`/dfadmin factory <개수>`(1~100): 운영자를 가운데로 공장 모듈(코어 줄·체 줄·농장과 수확기)을 지어 부하를 본다(`/dfadmin tps` 로 켜고). 결과와 Dragonfly 스레딩은 [performance.md](performance.md).

`/dfadmin dense <개수>`(1~4000): 운영자를 가운데로 2×1 칸짜리 밀집 셀(1티어 코어, 그 출력 위 자동 자원 채굴기, 옆 분쇄기 → 호퍼 → 서랍)을 빽빽하게 짓는다. 셀 하나에 블록 엔티티 5개(+서랍 표시 엔티티). 코어·기계를 많이 둘 때의 부하를 게임 안에서 잰다(측정은 운영자가 직접, 결과는 [performance.md](performance.md)에 적는다).

`/dfadmin expandview <map|floor>`: 이 섬의 확장 모드 표시 방식을 바꾼다(멀리 보낸 평면 지도(기본) / 섬 높이 유리 바닥). 확장 모드 중인 플레이어는 바로 다시 그려진다. 저장하지 않는다(서버를 다시 켜면 `map`, [기획](../design/02-dragonfactory.md) 섬 확장).
