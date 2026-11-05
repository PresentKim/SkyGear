# 스토리지 망 구현

기획: [18-storage-network.md](../design/18-storage-network.md). 코드: 엔진 `common/storagenet/`, 게임 쪽 `content/skygear/netstore/`.

## 패키지와 파일

엔진(`common/storagenet`)은 Dragonfly를 가져오지 않는 순수 Go다. 아이템은 `Key`(문자열)와 `int64` 개수로만 다룬다.

| 파일 | 하는 일 |
|---|---|
| `types.go` | `Key`, `Stack`, `Storage`, `Ingredient`, `Pattern`, `Step`, `Plan` |
| `disk.go` | 개수 상한이 있는 디스크 저장소 |
| `root.go` | 우선순위 합성 저장소 `Root`. 합계, 보유 저장소 색인, 변화 알림 |
| `external.go` | 바깥 인벤토리(`Source`)를 읽어 망 목록에 맞추는 `External` |
| `sched.go` | 일에 맞춰 간격이 바뀌는 `Scheduler` |
| `calc.go` | 패턴 저장소 `Repo`와 자동 제작 계산 `Calculate` |
| `task.go` | 제작 작업 `Task`: 재료 예약, 단계 진행, 저장·복원 |

게임 쪽(`content/skygear/netstore`)이 엔진을 블록·창·세이브에 잇는다.

| 파일 | 하는 일 |
|---|---|
| `items.go` | 디스크·빈 패턴 아이템, 디스크 용량 표, 디스크 ID 값 |
| `keys.go` | `KeyOf`(아이템 → 키), `Proto`(키 → 개수 1인 묶음, 세션별 캐시) |
| `network.go`, `nodes.go` | 망 찾기·무효화(`At`, `Invalidate`), 컨트롤러 틱, 노드별 설정(`Config`) |
| `diskstore.go` | 디스크 내용의 저장 |
| `terminal.go` | 단말·컨트롤러 클릭, 넣기·꺼내기, 호퍼·도관 연결 |
| `terminalui.go` | 단말 창(인벤토리 화면, [oreui.md](oreui.md)): 망 목록, 3×3 제작 칸(단말 칸 0~8), 클릭 규칙 |
| `io.go`, `external.go`, `filterui.go` | 입력기·출력기·외부 저장공간과 필터 창 |
| `pattern.go`, `encoder.go` | 패턴 읽기·쓰기, 패턴 인코더 |
| `craft.go`, `craftui.go` | 자동 제작 장치, 계산 요청, 작업 실행, 제작 창 |

## 데이터 흐름

1. 블록을 놓거나 부수면 `Invalidate`가 그 칸과 이웃 칸의 망을 버린다. 다음 `At`이 맞닿은 노드를 따라가 `Network`를 새로 만든다.
2. 만든 직후 `onBuild` 훅이 돈다. 드라이브의 디스크를 `Root`에 꽂고, 입력기·출력기·외부 저장공간을 스케줄러에 올리고, 자동 제작 장치의 패턴을 읽는다.
3. 컨트롤러가 틱마다 `runController`로 RE를 내고 망을 켠다. 켜져 있으면 드라이브를 다시 맞추고 스케줄러를 돌리고 변경분을 저장한다.
4. 아이템은 모두 `netInsert`(`terminal.go`)로 들어간다. 제작 작업이 기다리는 결과는 `InsertHook`(`Intercept`)이 먼저 가져가고, 나머지는 `Root.Insert`로 간다. 꺼내기는 `Root.Extract`다.
5. 입력기·출력기·외부 저장공간의 앞면은 설치할 때 누른 블록 쪽이다(`machine.State.Facing`, NBT `Facing`; 누른 면 없이 놓였거나 예전 세이브면 남쪽). 자동 제작 장치의 앞면은 앞면 텍스처가 있는 **남쪽 이웃**이다. 모두 `front` 에서 정한다.
6. 케이블·입력기·출력기·외부 저장공간은 Refined Storage 2 모델을 옮긴 가는 모양이다(`netshape`): 4픽셀 케이블 심에 이웃한 망 블록마다 팔이 붙고, 장치는 바라보는 면에 머리가 있다. 블록 상태 `skygear:net_shape` 하나가 팔 6비트(장치는 바라보는 방향 ×64)를 담고, 모양마다 지오메트리(`models/blocks/net.geo.json`, `go run ./tools/netgeo assets/resourcepack/skygear/models/blocks/net.geo.json`)와 충돌·선택 상자가 있다. 이웃이 바뀌면(`NeighbourUpdateTick`)과 40틱마다(불러온 블록) 모양을 다시 맞춘다. 다른 망 블록에 머리를 댄 장치 쪽으로는 팔을 내지 않는다. 모든 상자의 여섯 면을 다 그린다(알파 테스트 재질은 뒷면도 그려서, 빠진 면은 모델을 뚫는 구멍이 된다). 블록 끝에 닿는 면(팔 끝, 머리 판의 바깥면)은 0.01픽셀 안쪽에 두어 옆 블록 면과 같은 평면에서 깜빡이지 않게 한다.

오프라인 망(컨트롤러 없음·둘 이상·상한 초과·RE 부족)은 아이템을 그대로 두고 넣기·빼기만 거절한다.

## 저장

세이브 문서에 세 가지를 쓴다.

- **디스크**(`diskstore.go`): 디스크 ID별 `Key → 개수` 지도. 아이템에는 ID만 있다. 변경이 있으면 20번째 컨트롤러 틱마다, 망을 버릴 때 반드시 `Flush`한다.
- **노드 설정**(`nodes.go`의 `configSaved`): 위치별 우선순위, 필터 목록, 허용/차단.
- **제작 작업**(`craft.go`의 `craftDoc`): 컨트롤러 위치별 `TaskData`. 20틱마다와 작업이 끝날 때 쓴다. 패턴이 사라진 작업은 `unloaded`에 남아 아이템을 지키고, 패턴이 돌아오면 이어진다.

## 성능 (기획 §7)

| 기법 | 코드 |
|---|---|
| 바뀔 때만 다시 계산 | `network.go` `Invalidate`, `build`. 안전망 `verify`를 200 컨트롤러 틱마다 |
| 합계 목록 유지 | `root.go` `Root.apply`가 합계를 고치고 `Count`·`Totals`·`Each`가 읽는다 |
| 보유 저장소 색인 | `root.go` `holders`: 넣기는 가진 저장소부터, 빼기는 가진 저장소만 |
| 틱 관리 | `sched.go`: 간격 4~64틱(외부 저장공간 20~200), 일하면 줄고 놀면 늘며 `Wake`로 깨운다. `io.go`의 `ioJob` |
| 변화만 알림 | `Root.Listen`의 `(Key, 차이)`. 출력기 깨우기(`io.go` `buildIO`), 저장 표시(`network.go`) |
| 사본으로 틱 밖 계산 | `craft.go` `launch`가 고루틴에서 `Calculate`를 돌리고, `applyJobs`가 틱 안에서 재고를 다시 확인한다 |
| 재료 예약 | `task.go`: 시작할 때 `Root`에서 재료를 모두 빼 작업이 들고 있는다 |

벤치마크: `integration_test.go`의 `BenchmarkRootInsert`(디스크 64장, 5000종)는 넣기 한 번에 약 160 ns다(4코어 샌드박스 기준).

## 클린룸과 라이선스

- AE2는 코드도 그림도 열지 않았다. 쓴 것은 공개 문서와 널리 알려진 동작 방식이다: 그리드를 바뀔 때만 다시 계산하기, 저장소 캐시, 일에 맞춰 바뀌는 틱 간격, 사본으로 하는 제작 계산, 재료 예약. 모두 우리 구조에 맞게 새로 짰다.
- Refined Storage 2(MIT)에서 가져온 것: `common/storagenet/calc.go`(제작 계산, 머리 주석에 출처)와 일부 텍스처. 고지는 `NOTICE.md`, [asset-credits.md](asset-credits.md), `third_party/licenses/RefinedStorage2-MIT.txt`.

## 알려진 한계와 다음 일

- 단말 창은 누를 때만 다시 그려진다. 망이 바뀔 때 0.5초마다 저절로 다시 그리기는 아직 없다.
- 커서에 든 아이템(`ui.Carried`)은 서버 메모리에만 있다. 창을 닫거나 나가거나 서버가 정상 종료하면 인벤토리로 돌아가지만, 서버가 강제로 죽으면 사라진다.
- 자동 제작 장치는 회전하지 않아 앞면이 늘 남쪽이다.
- 컨트롤러·단말·패턴 인코더·자동 제작 장치 면의 구멍은 RS2 의 빛 층(`cutouts/light_blue`)을 밑에 깔아 메운다(`tools/modtex`). 구멍 난 면은 옆 블록 면이 비쳐 깜빡인다.
- 무선 단말, 감지기, 액체 디스크는 없다.
- 드라이브·자동 제작 장치·패턴 인코더의 상자 창을 나중에 칸 배치까지 다듬는다.
