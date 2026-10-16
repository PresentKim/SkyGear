# 생산 통계와 정지 알림 (구현)

기획은 [12](../design/12-production-monitor.md)다. 이 문서는 코드의 구조와 성능을 적는다.

## 구조

| 파일 | 역할 |
|---|---|
| `stats.go` | `prodStats`(1분 칸 15개 고리, 아이템 종류별 개수와 발전·소비 RE), `machineState.emit`(출력 + 기록)·`roomFor`(출력 공간 검사 + 막힘 기록)·`tickWork`(에너지 쓰기 + 진행), 화면 세 쪽(`showStats`) |
| `monitor.go` | `machineStatus`·`stuckKind`, 기계 목록 판정(`judge`), 알림 상태기계(`tickMonitor`), 알림 보내기, 정지 목록 화면, `/통계` |
| `state.go` | `State.Produced`(누적, 아이템 키별), `State.AlertsOn`(알림을 켠 계정) |

- 기계의 출력 지점은 `put(출력)` 대신 `emit`을, `fits(출력)` 대신 `roomFor`를 쓴다. 막힘은 `st.stuck`에 남고 `Tick` 시작에서 지운다.
- 발전기는 연료가 없으면 `stuckFuel`, 용암 발전기는 용암이 없으면 `stuckFuel`로 적는다. 한 번도 일하지 않은 기계는 연료 부족을 알리지 않는다.
- `Tick`이 기계를 `session.machines`(위치 → 상태)에 올리고, `Second`의 `tickMonitor`가 1초마다 목록을 한 번 돈다. 10초 넘게 틱이 없는 기계는 건너뛰고 5분 넘으면 지운다. 부술 때(`breakMachine`) 지운다.
- 상태는 `judge`가 정한다: 지난 1초에 일했으면 작동, 아니면 마지막 틱의 막힘, 아니면 5분 안에 일했던 기계는 재료 부족, 아니면 대기.
- 알림은 알릴 상태가 30초 이어지고 같은 기계의 마지막 알림에서 5분이 지났을 때, 한 번에 세 줄까지 보낸다. 복구는 작동으로 돌아올 때 센다.

## 성능

- 기록은 맵 더하기와 정수 더하기뿐이다. 기계마다 락을 잡지 않는다.
- 계정 카운터와 업적은 초마다 `prodStats.fresh`를 `countAll`로 한 번 넘긴다.
- 1초마다 기계 목록을 한 번 도는 일(기계 150대 안팎)이 전부다.
