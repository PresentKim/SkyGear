# third_party

## dragonfly

`github.com/df-mc/dragonfly` **v0.11.5** 의 사본에 dragonfactory 패치를 얹은 것이다.
루트 `go.mod` 의 `replace github.com/df-mc/dragonfly => ./third_party/dragonfly` 로 쓰인다.
**[common](https://github.com/PresentKim/common) 의 `third_party/dragonfly` 와 같은 사본이어야 한다**(`replace` 는 최상위 모듈에서만 적용되어 레포마다 사본을 둔다). 패치를 바꾸면 양쪽에 똑같이 적용한다.
자체 `go.mod` 가 있어 루트의 `go vet ./...`, `go test ./...` 대상이 아니다(`.github` 는 제거).

### 왜 포크하는가

Dragonfly 의 동작 자체(블록 규칙, 패킷 처리, 세션 기능)를 바꿔야 하는 곳만 패치한다.
패킷을 가로채거나 세션 내부를 읽고 쓰기만 하면 되는 일은 패치하지 않고 `common/dfhook` 으로 한다
([bedrock-gophers/inv](https://github.com/bedrock-gophers/inv) 방식: 리스너를 감싼 연결에서 패킷 훅,
`go:linkname` 으로 세션의 비공개 메서드, 리플렉션으로 비공개 필드). 엔티티 상자(`common/chestui`)와
한 플레이어에게만 보내는 화면 조작(`dfhook.SetCamera`·`SetCameraFOV`·`HideHUD`·`LockInputs`·`ViewFakeBlock`·`ViewChunkCentre`·`SendTitle`,
접속 시 카메라 프리셋), 클라이언트가 보낸 이동 키(`dfhook.MoveInput`), `ForgetHiddenEntities`, 실험 설정(`SendBasicVoxelShapes`, `BlockActionDebug`),
폼 응답 정리(없는 폼 ID 응답은 킥 대신 버림, 응답 `null` 은 닫기)가 이렇게 동작한다. 세션이 필요한 훅을 위해 접속한 플레이어마다 `dfhook.Attach` 를 부른다(`main.go`).
`dfhook` 이 이름으로 기대는 내부는 `TestInternals` 가 검사한다(업그레이드 후 깨지면 여기서 보인다).

### 패치 목록 (`dragonfly.patch`)

| 파일 | 내용 |
|---|---|
| `server/session/session.go` | **패킷 처리 오류로 세션이 끝날 때 ERROR 로그 + 이유를 담은 Disconnect**(원래는 Debug 로그만 남기고 조용히 끊음) |
| `server/session/player.go`(`sendRecipes`, `SetRecipeFilter`), `session.go`, `handler_crafting.go` | 플레이어별 레시피 필터: 필터가 고른 레시피만 `CraftingData` 로 보내고 받아들임(네트워크 ID 는 전체 목록 순번 그대로). `SetRecipeFilter` 는 바로 다시 보낸다(레시피 연구·잠금). 필터가 있으면 동적 레시피(장식 항아리)는 막음 |
| `server/session/fake_chunk.go` (신규), `chunk.go`, `world.go`, `session.go` | `ViewFakeChunk`/`ClearFakeChunk`(그 플레이어에게만 보이는 청크: 그동안 그 청크의 월드 블록 갱신은 안 보내고, 청크 재전송·서브청크 요청은 가짜 청크로 응답). 섬 확장 `map` 보기용 |
| `server/conf.go`, `server/server.go` | `Config.ShutdownKick`: 서버가 꺼질 때 플레이어를 내보내는 화면의 글을 정한다(기본값은 번역 키 `%disconnect.disconnected` 라 연결 끊김 화면에 키가 그대로 보였다) |
| `server/block/crop.go` 외 작물·줄기 6종 `RandomTick` | `block.ScheduledCropGrowth` 가 켜져 있으면 작물·수박/호박 줄기가 무작위 틱으로 자라지 않음(게임이 인스턴스 시간 일정으로 키움) |
| `server/block/carrot.go`, `potato.go`, `wheat_seeds.go`, `beetroot_seeds.go` | 다 자란 작물을 캘 때 떨어지는 작물·씨앗을 성장 0의 새 아이템으로(원래는 다 자란 상태를 복사해 심으면 바로 다 자란 작물이 됨) |
| `server/cmd/command.go` | 명령어 `Run` 의 panic 을 잡아 ERROR 로그와 오류 출력으로 바꿈(서버가 죽지 않음) |
| `server/item/digger.go` (신규), `server/internal/iteminternal/components.go` | `item.Digger`(`DestroySpeeds`)를 구현한 커스텀 아이템에 `minecraft:digger` 컴포넌트(블록별 캐기 속도)를 보냄. 없으면 클라이언트가 커스텀 도구로 맨손 속도로 캔다 |
| `server/internal/blockinternal/components.go` | `block.Activatable` 을 구현한 커스텀 블록에 `minecraft:custom_components`(`hasPlayerInteract`=1)를 보냄. 상자처럼 클라이언트가 그 블록을 클릭할 때 손에 든 블록의 설치를 예측하지 않는다(형식은 PowerNukkitX `CustomBlockDefinition.isPlayerInteractable` 참고) |
| `server/world/item.go`, `server/internal/packbuilder/items.go` | `world.CustomItem` 에서 `Texture()` 를 빼고 선택 인터페이스 `world.CustomItemTextured` 로 옮김. 텍스처를 리소스팩으로 따로 관리하는 아이템은 이미지가 필요 없다(자동 팩 생성은 텍스처가 있는 아이템만) |
| `server/item/recipe/recipe.go`, `server/session/player.go`, `server/session/handler_crafting.go`, `server/session/shaped_arrangement_test.go` (신규) | 정해진 모양으로만 만들어지는 레시피: `Shaped.WithoutMirror()`(좌우 대칭 배치도 안 됨, 클라이언트에 `AssumeSymmetry=false`). **서버도 모양을 검사**한다(원래는 클라이언트가 고른 레시피의 재료를 칸 위치와 상관없이 가져갔다): 제작 칸의 아이템이 레시피 모양(격자 안 어디든, 대칭은 허용할 때만)이 아니면 거절 |
| `server/session/player.go` (`creativeBlockFitter`) | 레시피 결과의 블록 런타임 ID 를 그 아이템의 창작 인벤토리 항목(항목이 하나뿐일 때)과 같게 보냄. 클라이언트 레시피 책은 결과가 창작 항목과 같을 때만 레시피를 나열하는데, 바닐라 레시피 데이터의 블록 상태(상자 방향 등)가 창작 목록과 달라 상자 같은 바닐라 블록 레시피가 안 보였다 |
| `server/world/tx.go`, `server/world/task_test.go` | 월드 트랜잭션(블록 틱, 이벤트·패킷 핸들러)의 panic 을 잡아 ERROR 로그(스택 포함)로 남기고 월드는 계속 돈다(원래는 프로세스가 죽어 모든 섬이 멈췄다). `Do`/`DoAfter` 작업은 원래도 잡는다 |
| `server/block/hopper.go` | `HopperTransfer` 훅(`Tick` 에서 바닐라 이동 규칙 대신 부름). 게임이 모든 호퍼를 자기 아이템 입출력 인터페이스(`internal/itemio`)로 움직이게 함 |
| `server/block/composter.go` | `ComposterVanillaRules`(기본 켬)를 끄면 퇴비통이 호퍼로 넣고 빼지 않고 우클릭도 하지 않음(게임이 자기 규칙으로 처리: 고정 개수, 흙) |
| `server/block/leaves.go` | 나뭇잎 묘목·막대기 드롭 확률(`LeavesSaplingChances`, `JungleLeavesSaplingChances`, `LeavesStickChances`)을 상수 대신 변수로, 로그를 잃은 나뭇잎의 시듦을 무작위 틱 대신 예약 틱으로 앞당길 수 있게 함(`LeavesDecayDelayMin/Jitter`, `ScheduledTick`; 시듦 처리는 `decay` 로 합침). 기본값은 바닐라 그대로, 게임이 바꿈 |
| `server/world/tick_stats.go` (신규), `tick.go`, `world.go` | 월드 틱마다 걸린 시간(틱 예정 시각부터 끝날 때까지)을 최근 5초만 기록하고 `World.TickStats()`(TPS, 평균·최대 틱 시간)로 돌려줌. 운영자 사이드바의 서버 부하 표시용(`common/perf`) |
| `server/block/dirt.go` | 잔디·균사체 퍼짐(`spreadDirt`): 위에 꽉 찬 고체 블록(`model.Solid`)이 있으면 흙으로 퍼지지 않고, 그 아래 잔디는 흙으로 돌아감(바닐라와 같게). 원래는 빛만 봐서 빛을 막지 않는 커스텀 블록(코어 등) 아래 흙이 잔디가 됨 |
| `server/session/player.go` (추가) | Nop 세션의 `SendForm` 무시(테스트 플레이어) |
| `server/player/player.go` | `SetRecipeFilter`, `ViewFakeChunk`, `ClearFakeChunk`. 블록 `Activate` 성공 후 주변 블록 재전송(클라이언트가 예측 설치한 유령 블록 제거), **`Activate` 뒤 손 아이템을 다시 읽어 차감**(원래는 `Activate` 전 값을 되써서 서랍·납품함에 넣은 아이템이 복사됨). **커스텀 블록은 부수기 시간이 지나면 서버가 직접 부순다**(`tickCustomBreak`: 실제 클라이언트가 금 가기는 끝내도 파괴 동작을 보내지 않아 부서지지 않았음). 같은 블록을 계속 캐는 동안(ContinueDestroyBlock 반복) 시작 시각은 유지 |

패치한 곳에는 `dragonfactory patch` 주석을 단다.

### 업그레이드 절차

1. 새 버전을 받는다: `go mod download -json github.com/df-mc/dragonfly@vX.Y.Z` 의 `Dir` 을 복사.
2. `.github` 를 지우고 `third_party/dragonfly` 를 교체한다.
3. `cd third_party/dragonfly && patch -p1 < ../dragonfly.patch` — 충돌은 손으로 고친다.
4. 루트 `go.mod` 의 `require` 버전을 맞추고 `go mod tidy`, `go vet ./...`, `go test ./...`, 시나리오 실행.
5. 원본과 다시 비교해 `dragonfly.patch` 를 갱신한다(`diff -ruN a b`, a=원본, b=패치본).

## EnderIO 리소스 (도관 텍스처)

도관 텍스처(`tools/conduittex/src/` 의 64×64 원본과 거기서 만든 `textures/entity/dragonfactory/conduit_*.png`, `textures/items/conduit_*.png`)와 도관 결합체·결합 반죽 아이콘은 Team-EnderIO/EnderIO
(Unlicense, 퍼블릭 도메인)의 텍스처(`energy`, `enhanced_energy`, `ender_energy`, `item`의 첫 프레임, `conduit_binder`, `conduit_binder_composite`)에서 가져왔다. 아이템 도관 2·3단은 색조만 바꾼 것이다. 모델 비율(3×3×3 코어, 2×2 팔)도 EnderIO 모델을 따랐다. 배치는 `tools/conduitgeo`·`tools/conduittex` 참고.

연결부 판의 화살표(`tools/conduittex/src/arrow_in.png`, `arrow_out.png`)도 EnderIO(`item_conduit_in`, `item_conduit_out`)의 것이다(채널 색으로 물들여 쓴다).

## licenses

`licenses/` 는 리소스팩 텍스처를 가져온 모드의 라이선스 전문이다(Mekanism, Tinkers' Construct, Storage Drawers, Refined Storage 2 는 MIT, Ender IO 는 Unlicense). 출처와 쓴 곳은 [docs/tech/asset-credits.md](../docs/tech/asset-credits.md).
