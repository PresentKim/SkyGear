# third_party

## dragonfly

`github.com/df-mc/dragonfly` 의 사본에 skygear 패치를 얹은 것이다. 기준은 master `1bf4e242`(v0.11.5 이후, 2026-10-24)에
1.26.60 지원 PR [#1481](https://github.com/df-mc/dragonfly/pull/1481)(gophertunnel v1.63.0)을 더한 커밋 `d86aefd9ff9b6a357d6dfadab2093af811545256` 이다.
루트 `go.mod` 의 `replace github.com/df-mc/dragonfly => ./third_party/dragonfly` 로 쓰인다.
자체 `go.mod` 가 있어 루트의 `go vet ./...`, `go test ./...` 대상이 아니다(`.github` 는 제거).

### 왜 포크하는가

Dragonfly 의 동작 자체(블록 규칙, 패킷 처리, 세션 기능)를 바꿔야 하는 곳만 패치한다.
패킷을 가로채거나 세션 내부를 읽고 쓰기만 하면 되는 일은 패치하지 않고 `common/sghook` 으로 한다
([bedrock-gophers/inv](https://github.com/bedrock-gophers/inv) 방식: 리스너를 감싼 연결에서 패킷 훅,
`go:linkname` 으로 세션의 비공개 메서드, 리플렉션으로 비공개 필드). 엔티티 상자(`common/chestui`)와
한 플레이어에게만 보내는 화면 조작(`sghook.SetCamera`·`SetCameraFOV`·`HideHUD`·`LockInputs`·`ViewFakeBlock`·`ViewChunkCentre`·`SendTitle`,
접속 시 카메라 프리셋), `ForgetHiddenEntities`, 실험 설정(`SendBasicVoxelShapes`, `BlockActionDebug`),
폼 응답 정리(없는 폼 ID 응답은 킥 대신 버림, 응답 `null` 은 닫기)가 이렇게 동작한다. 세션이 필요한 훅을 위해 접속한 플레이어마다 `sghook.Attach` 를 부른다(`main.go`).
`sghook` 이 이름으로 기대는 내부는 `TestInternals` 가 검사한다(업그레이드 후 깨지면 여기서 보인다).

### 패치 목록 (`dragonfly.patch`)

| 파일 | 내용 |
|---|---|
| `server/session/world.go` | `FloatingText()` 메서드가 있는 엔티티 종류는 공기 블록을 든 낙하 블록처럼 그려 보냄(떠 있는 글자 엔티티용) |
| `server/session/session.go` | **패킷 처리 오류로 세션이 끝날 때 ERROR 로그 + 이유를 담은 Disconnect**(업스트림은 Debug 로그만 남기고 조용히 끊는다) |
| `server/session/player.go`(`sendRecipes`, `SetRecipeFilter`), `session.go`, `handler_crafting.go` | 플레이어별 레시피 필터: 필터가 고른 레시피만 `CraftingData` 로 보내고 받아들임(네트워크 ID 는 전체 목록 순번 그대로). `SetRecipeFilter` 는 바로 다시 보낸다(레시피 연구·잠금). 필터가 있으면 동적 레시피(장식된 도자기)는 막음 |
| `server/session/fake_chunk.go` (신규), `chunk.go`, `world.go`, `session.go` | `ViewFakeChunk`/`ClearFakeChunk`(그 플레이어에게만 보이는 청크: 그동안 그 청크의 월드 블록 갱신은 안 보내고, 청크 재전송·서브청크 요청은 가짜 청크로 응답). 섬 확장 `map` 보기용 |
| `server/listener.go`, `server/conf.go` | NetherNet 리스너 설정 `LocalIP`(연결을 묶고 알리는 로컬 주소 하나)와 `PublicIP`(로컬 주소 대신 알리는 공개 주소, 1:1 NAT·터널 뒤 서버용). `NetherNetConfig` 와 `Config` 의 NetherNet 항목에 둠 |
| `server/conf.go`, `server/server.go` | `Config.ShutdownKick`: 서버가 꺼질 때 플레이어를 내보내는 화면의 글을 정한다(기본값인 번역 키 `%disconnect.disconnected` 는 연결 끊김 화면에 키 그대로 보인다) |
| `server/block/crop.go` 외 작물·줄기 6종 `RandomTick` | `block.ScheduledCropGrowth` 가 켜져 있으면 작물·수박/호박 줄기가 무작위 틱으로 자라지 않음(게임이 인스턴스 시간 일정으로 키움) |
| `server/block/carrot.go`, `potato.go`, `wheat_seeds.go`, `beetroot_seeds.go` | 다 자란 작물을 캘 때 떨어지는 작물·씨앗을 성장 0의 새 아이템으로(업스트림은 다 자란 상태를 복사해, 심으면 바로 다 자란 작물이 된다) |
| `server/cmd/command.go` | 명령어 `Run` 의 panic 을 잡아 ERROR 로그와 오류 출력으로 바꿈(서버가 죽지 않음) |
| `server/item/digger.go` (신규), `server/internal/iteminternal/components.go` | `item.Digger`(`DestroySpeeds`: 블록 이름별)와 `item.TagDigger`(`DestroySpeedsByTag`: 블록 태그 질의별, `item.TagSpeed`)를 구현한 커스텀 아이템에 `minecraft:digger` 컴포넌트(캐기 속도)를 보냄. 없으면 클라이언트가 커스텀 도구로 맨손 속도로 캔다 |
| `server/internal/iteminternal/components.go` | `item.Usable` 을 구현한 커스텀 아이템(먹는 것 제외)에 `minecraft:use_modifiers` 를 보냄. 없으면 클라이언트가 허공 사용을 서버에 보내지 않는다 |
| `server/internal/iteminternal/components.go`, `fuel_test.go` (신규) | `item.Fuel` 을 구현한 커스텀 아이템에 `minecraft:fuel` 컴포넌트(`duration`, 초)를 보냄. 없으면 클라이언트가 커스텀 연료(나무껍질 섬유)를 화로 연료 칸에 넣지 못한다 |
| `server/internal/blockinternal/components.go`, `tooltags_test.go` (신규) | 파괴 가능한 커스텀 블록에 바닐라 블록처럼 도구 태그(`minecraft:is_<도구>_item_destructible`, 곡괭이는 필요한 등급 `stone_pick_diggable` 등)를 붙임(`toolTags`, `BreakInfo` 의 `Effective`·`Harvestable` 로 정함). `item.TagDigger` 를 구현한 도구가 그 블록을 제 속도로 캔다 |
| `server/internal/blockinternal/components.go` | `block.Activatable` 을 구현한 커스텀 블록에 `minecraft:custom_components`(`hasPlayerInteract`=1)를 보냄. 상자처럼 클라이언트가 그 블록을 클릭할 때 손에 든 블록의 설치를 예측하지 않는다(형식은 PowerNukkitX `CustomBlockDefinition.isPlayerInteractable` 참고) |
| `server/block/customblock/render_method.go` | `AlphaTestSingleSidedRenderMethod`(`alpha_test_single_sided`): 뒷면을 그리지 않는 컷아웃. 누에 나뭇잎이 바닐라 잎처럼 이웃 면과 겹쳐 깜빡이지 않게 한다 |
| `server/block/customblock/permutations.go`, `server/internal/blockinternal/components.go` | `Properties.CollisionBoxes`: 충돌 상자를 여러 개 보냄(`minecraft:collision_box` 의 `boxes`). 배럴을 퇴비통처럼 바닥과 네 벽만 있는 블록으로 만들어 안에 들어갈 수 있게 한다(`machine/netshape.go` 가 도관 망 모양에 씀) |
| `server/internal/blockinternal/components.go`, `components_test.go` | 커스텀 블록의 상태 정의(`properties`)를 `States()` 맵 순서(매번 무작위) 대신 **이름순**으로 보냄. 클라이언트는 상태 조합을 이름순으로 첫 상태가 가장 느리게 바뀌는 순서로 번호 매기고(바닐라 팔레트와 같은 순서), 서버는 등록 순서로 매긴다. 상태가 여럿인 블록이 늘 같은 정의를 보내게 한다. 상태가 하나인 블록은 바뀌는 것이 없다 |
| `server/block/customblock/permutations.go`, `server/internal/blockinternal/components.go`, `components_test.go` (신규) | `Properties.BoneVisibility`: 지오메트리의 뼈마다 보임 여부(`true`·`false`·Molang 식)를 `minecraft:geometry` 의 `bone_visibility` 로 보냄(기본 블록과 퍼뮤테이션 모두, 문자열로 인코딩). 상태 수만큼 퍼뮤테이션을 만드는 대신 뼈로 모양을 바꾸는 블록용(`machine/sieve.go`, `machine/netshape.go`) |
| `server/world/item.go`, `server/internal/packbuilder/items.go` | `world.CustomItem` 에서 `Texture()` 를 빼고 선택 인터페이스 `world.CustomItemTextured` 로 옮김. 텍스처를 리소스팩으로 따로 관리하는 아이템은 이미지가 필요 없다(자동 팩 생성은 텍스처가 있는 아이템만) |
| `server/item/recipe/recipe.go`, `server/session/player.go`, `server/session/handler_crafting.go`, `server/session/shaped_arrangement_test.go` (신규) | 정해진 모양으로만 만들어지는 레시피: `Shaped.WithoutMirror()`(좌우 대칭 배치도 안 됨, 클라이언트에 `AssumeSymmetry=false`). **서버도 모양을 검사**한다(업스트림은 클라이언트가 고른 레시피의 재료를 칸 위치와 상관없이 가져간다): 제작 칸의 아이템이 레시피 모양(격자 안 어디든, 대칭은 허용할 때만)이 아니면 거절 |
| `server/session/player.go` (`creativeBlockFitter`) | 레시피 결과의 블록 런타임 ID 를 그 아이템의 창작 인벤토리 항목(항목이 하나뿐일 때)과 같게 보냄. 클라이언트 레시피 책은 결과가 창작 항목과 같을 때만 레시피를 나열하는데, 바닐라 레시피 데이터의 블록 상태(상자 방향 등)는 창작 목록과 다를 수 있다 |
| `server/world/tx.go`, `server/world/task_test.go` | 월드 트랜잭션(블록 틱, 이벤트·패킷 핸들러)의 panic 을 잡아 ERROR 로그(스택 포함)로 남기고 월드는 계속 돈다(업스트림은 프로세스가 죽어 모든 섬이 멈춘다). `Do`/`DoAfter` 작업은 업스트림도 잡는다 |
| `server/block/hopper.go` | `HopperTransfer` 훅(`Tick` 에서 바닐라 이동 규칙 대신 부름). 게임이 모든 호퍼를 자기 아이템 입출력 인터페이스(`common/itemio`)로 움직이게 함 |
| `server/block/composter.go` | `ComposterVanillaRules`(기본 켬)를 끄면 퇴비통이 호퍼로 넣고 빼지 않고 우클릭도 하지 않음(게임이 자기 규칙으로 처리: 고정 개수, 흙) |
| `server/block/leaves.go` | 나뭇잎 묘목·막대기 드롭 확률(`LeavesSaplingChances`, `JungleLeavesSaplingChances`, `LeavesStickChances`)을 상수 대신 변수로, 원목을 잃은 나뭇잎의 시듦을 무작위 틱 대신 예약 틱으로 앞당길 수 있게 함(`LeavesDecayDelayMin/Jitter`, `ScheduledTick`; 시듦 처리는 `decay` 로 합침). 기본값은 바닐라 그대로, 게임이 바꿈. 나뭇잎이 원목을 찾을 때 `block.LeavesLike` 를 구현한 블록(누에 나뭇잎)도 나뭇잎처럼 지나감(`findLog`) |
| `server/world/tick_stats.go` (신규), `tick.go`, `world.go` | 월드 틱마다 걸린 시간(틱 예정 시각부터 끝날 때까지)을 최근 5초만 기록하고 `World.TickStats()`(TPS, 평균·최대 틱 시간)로 돌려줌. 운영자 사이드바의 서버 부하 표시용(`common/perf`) |
| `server/world/tick.go`, `world.go`, `loader.go`, `tick_cache_test.go`, `weather_leak_test.go` (신규) | 무작위·블록 엔티티 틱 대상 청크 목록을 캐시(`tickSetCache`). 로더 추가·제거·이동, 청크 추가·삭제, `SetTickRange` 때 갱신 표시를 하고, 안전망으로 100틱마다 강제 재계산. 블록 엔티티 위치 버퍼를 재사용하고 `tx.Block` 대신 `c.BlockEntities` 를 직접 읽음(언로드된 청크는 건너뜀). 날씨 패킷은 값이 바뀔 때만 보냄(월드별로 기억하고 월드가 닫히면 잊음(`weather_leak_test.go`). 접속한 뷰어는 `Loader.Load` 에서 받음) |
| `server/world/world.go`, `tx.go` 관련, `save_test.go`, `transient_test.go`, `concurrent_test.go` (신규) | 증분 저장: 1초마다 수정된 칼럼을 최대 16개(`saveBatch`)까지 오래된 순으로 저장(`Save`/`Close` 는 전체 저장). `modified` 는 저장 뒤 블록 엔티티·엔티티·예약 틱이 없는 칼럼만 지움. `Column.version` 과 서브청크 캐시. 임시 엔티티는 저장하지 않고 칼럼을 수정됨으로 표시하지도 않음. `tx.Players()` 는 플레이어 색인을 씀. `entitiesWithin` 은 청크마다 복제하지 않음(쓸 때 복사) |
| `server/world/entity.go` | `TransientEntityType`: 저장하지 않는 임시 엔티티 종류(표시용 등) |
| `server/world/chunk/encoding.go`, `server/world/block_registry.go`, `chunk/palette_cache_test.go` (신규) | 팔레트 항목 NBT 를 런타임 ID 별로 한 번만 인코딩해 재사용(저장 결과도 항상 같음) |
| `server/world/mcdb/db.go`, `skipwrite_test.go` (신규) | 값이 안 바뀐 `Put` 은 건너뜀(키별 xxhash, 배치 쓰기 성공 뒤에 갱신). 빈 서브청크는 삭제. 칼럼의 엔티티 ID 목록을 기억해 다시 읽지 않음 |
| `server/world/conf.go`, `server/world/mcdb/conf.go` | `ConcurrentProvider`: 동시 호출이 안전한 저장소는 `lockedProvider` 로 감싸지 않음(mcdb 가 해당) |
| `server/session/chunk.go`, `subchunk_cache_test.go` (신규) | 서브청크 네트워크 데이터를 칼럼 `version` 별로 캐시. 가짜 청크는 캐시를 거치지 않음 |
| `server/session/player.go`, `recipes_cache_test.go` (신규) | 레시피·창작 데이터를 블록 레지스트리당 한 번만 만들어 공유하고 레시피 네트워크 ID 를 고정함. 한 트랜잭션에서 슬롯이 여러 번 바뀌어도 UI 인벤토리는 한 번만 보냄 |
| `server/session/session.go` | 클라이언트가 청크를 받는 중(열린 blob 트랜잭션이 적음)이면 틱마다 청크를 최대 16개 보냄(그 외 4개) |
| `server/world/skygear_modified.go` (신규), `server/block/chest.go`, `barrel.go`, `hopper.go`, `smelter.go`, `furnace.go`, `blast_furnace.go`, `smoker.go` | **컨테이너 내용 저장**: 인벤토리 변경은 청크를 바뀜으로 표시하지 않아, 같은 청크에 다른 변경이 없으면 상자·통·호퍼·화로 내용이 저장되지 않았다. `Tx.MarkModified(pos)`(로드된 청크만, 패킷·이웃 갱신 없음)를 두고 각 컨테이너의 `Inventory(tx, pos)` 와 불이 켜진 화로의 `Tick` 에서 부른다 |
| `server/world/skygear_hooks.go`, `skygear_hooks_test.go` (신규), `world.go`, `chunk_request.go` | 도관 망 그래프용 훅 두 개. `BlockChangeHook`: `Tx.SetBlock` 이 블록을 저장한 뒤 이전 블록(아니면 새 블록)이 구현했으면 `BlockChanged(tx, pos, before, after)` 를 한 번 부름(값이 같으면 안 부름, 같은 칸의 훅 안에서 일어난 변경은 훅을 다시 부르지 않음). `ColumnHandler`(월드 `Handler` 가 구현하는 선택 인터페이스): 칼럼이 월드에 더해진 뒤(`addChunk`, 비동기 로드 포함) `HandleColumnLoad`, `closeChunk` 에서 저장 전에 `HandleColumnUnload`(훅이 블록 엔티티 맵에 쓴 값이 저장됨). 칼럼 훅의 panic 은 잡아 ERROR 로그로 남김 |
| `server/world/skygear_layers.go`, `skygear_layers_test.go` (신규) | `Tx.SetBlockLayer`·`BlockLayer`·`SubChunkLayers`: 블록 저장 레이어 하나에 블록을 쓰고 읽음(지금은 호출하는 곳이 없음). 업데이트·훅 없이 그 레이어로 `UpdateBlock` 만 보냄 |
| `server/world/skygear_bench_test.go` (신규) | 위 패치의 벤치마크(`BlockEntityTick`, `TickLoadersRange`, `DiskEncode`, `SaveColumn` 등). 수치는 [docs/tech/performance.md](../docs/tech/performance.md) |
| `server/block/dirt.go` | 잔디·균사체 퍼짐(`spreadDirt`): 위에 꽉 찬 고체 블록(`model.Solid`)이 있으면 그 아래 잔디는 흙으로 돌아가고, 그런 곳의 흙으로는 퍼지지 않음(바닐라와 같게). 업스트림은 빛만 봐서 빛을 막지 않는 커스텀 블록 아래에서도 잔디가 남고 퍼진다 |
| `server/session/player.go` (추가) | Nop 세션의 `SendForm` 무시(테스트 플레이어) |
| `server/player/player.go` | `SetRecipeFilter`, `ViewFakeChunk`, `ClearFakeChunk`. 블록 `Activate` 성공 후 주변 블록 재전송(클라이언트가 예측 설치한 유령 블록 제거), **`Activate` 뒤 손 아이템을 다시 읽어 차감**(업스트림은 `Activate` 전 값을 되써서 서랍·납품함에 넣은 아이템이 복사된다). **커스텀 블록은 부수기 시간이 지나면 서버가 직접 부순다**(`tickCustomBreak`: 실제 클라이언트는 금 가기를 끝내도 파괴 동작을 보내지 않는다). 같은 블록을 계속 캐는 동안(ContinueDestroyBlock 반복) 시작 시각은 유지 |

패치한 곳에는 `skygear patch` 주석을 단다. 자동 팩 아이콘(`server/internal/packbuilder/pack_icon.png`)은 새로 그린 그림으로 바꿨다(패치 파일에는 없다).

### 업그레이드 절차

1. 새 기준 커밋(현재 `d86aefd9ff9b6a357d6dfadab2093af811545256`)을 `.github` 없이 `base` 로 받는다: `git init -q df && cd df && git fetch -q --depth 1 https://github.com/df-mc/dragonfly <전체 커밋 해시> && mkdir ../base && git archive FETCH_HEAD | tar -x -C ../base && rm -rf ../base/.github`. 릴리스 태그는 `go mod download -json github.com/df-mc/dragonfly@vX.Y.Z` 의 `Dir` 을 복사한다. 풀 리퀘스트 ref 는 움직이므로 커밋 해시로만 받는다.
2. `.github` 를 지우고 `third_party/dragonfly` 를 교체한다.
3. `cd third_party/dragonfly && patch -p1 < ../dragonfly.patch` 로 패치를 얹고, 충돌은 손으로 고친다.
4. 루트 `go.mod` 의 `require` 버전을 맞추고 `go mod tidy`, `go vet ./...`, `go test ./...` 를 돌린다.
5. 원본(`base`)과 다시 비교해 `dragonfly.patch` 를 갱신한다: `cp -r base a; cp -r <저장소>/third_party/dragonfly b; diff -ruN -x pack_icon.png a b > dragonfly.patch`. 확인: `cp -r base c; (cd c && patch -p1 -s < ../dragonfly.patch); diff -r -x pack_icon.png c <저장소>/third_party/dragonfly` 가 아무것도 출력하지 않고 `grep -c '^diff ' dragonfly.patch` 가 파일 수와 같아야 한다.
6. 패치 시험은 `cd third_party/dragonfly && go test ./server/...` 로 돌린다.

## EnderIO 리소스 (도관 텍스처)

도관 텍스처(`tools/conduittex/src/` 의 64×64 원본과 거기서 만든 `textures/entity/skygear/conduit_*.png`, `textures/items/conduit_*.png`)와 도관 결합체·결합 반죽 아이콘은 Team-EnderIO/EnderIO
(Unlicense, 퍼블릭 도메인)의 텍스처(`energy`, `enhanced_energy`, `ender_energy`, `item`의 첫 프레임, `conduit_binder`, `conduit_binder_composite`)에서 가져왔다. 아이템 도관 2·3단은 색조만 바꾼 것이다. 모델 비율(3×3×3 코어, 2×2 팔)도 EnderIO 모델을 따랐다. 배치는 `tools/conduitgeo`·`tools/conduittex` 참고.

연결부 판의 화살표(`tools/conduittex/src/arrow_in.png`, `arrow_out.png`)도 EnderIO(`item_conduit_in`, `item_conduit_out`)의 것이다(채널 색으로 물들여 쓴다).

## licenses

`licenses/` 는 리소스팩 텍스처를 가져온 모드의 라이선스 전문이다(Mekanism, Tinkers' Construct, Storage Drawers, Refined Storage 2 는 MIT, Ender IO 는 Unlicense). 출처와 쓴 곳은 [docs/tech/asset-credits.md](../docs/tech/asset-credits.md) 에 있다.
