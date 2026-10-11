# third_party

## dragonfly

`github.com/df-mc/dragonfly` **v0.11.5** 의 사본에 dragonfactory 패치를 얹은 것이다.
루트 `go.mod` 의 `replace github.com/df-mc/dragonfly => ./third_party/dragonfly` 로 쓰인다.
자체 `go.mod` 가 있어 루트의 `go vet ./...`, `go test ./...` 대상이 아니다(`.github` 는 제거).

### 왜 포크하는가

Dragonfly 에는 블록 없이 임의 크기의 창고 창을 여는 API 가 없다.
PocketMine 의 [CustomSizedInvMenu](https://github.com/Muqsit/CustomSizedInvMenu) 처럼
보이지 않는 클라이언트 전용 엔티티를 컨테이너로 쓰려면 세션 내부(`openedWindow`, 창 ID,
패킷 전송)를 건드려야 해서 공개 API 만으로는 불가능하다.

### 패치 목록 (`dragonfly.patch`)

| 파일 | 내용 |
|---|---|
| `server/session/entity_container.go` (신규) | 엔티티 상자: `OpenEntityContainer`(같은 인벤토리면 무시, Nop 세션 무시), 투명 엔티티 `AddActor` 후 바로 `ContainerOpen`. `EntityContainerOptions`(Ride·WaitLatency·Retry·Debug)로 InvMenu `PlayerWindowDispatcher` 단계(탑승 링크, `NetworkStackLatency` 응답 대기, `PacketViolationWarning` 확인까지 매 틱 재전송)를 켤 수 있다. 창이 닫힌 지 1초 안의 열기는 `AfterClientSync`(NetworkStackLatency 왕복) 뒤에 연다. `EntityContainerViewer`(그 인벤토리가 열린 창일 때만 슬롯 갱신 전송), `AfterClientSync`, 엔티티 따라가기. 관련 핸들러 |
| `server/session/session.go` | `entityContainer` 필드, `AvailableActorIdentifiers` 에 `dragonfly:entity_container`, 위 핸들러 등록. **패킷 처리 오류로 세션이 끝날 때 ERROR 로그 + 이유를 담은 Disconnect**(원래는 Debug 로그만 남기고 조용히 끊음) |
| `server/session/player.go` | `closeCurrentContainer` 가 열리는 중인 엔티티 상자도 정리 |
| `server/session/session.go` (추가) | 실험: `SendBasicVoxelShapes` 면 접속 시 최소 `VoxelShapes`(empty, unit_cube) 전송 |
| `server/session/camera.go` (신규) | 접속 시 카메라 프리셋(`minecraft:free`, 카메라 기준 이동) 전송, `SetCamera`/`ClearCamera`(CameraInstruction), `SetCameraFOV`/`ClearCameraFOV`(시야각), `ViewFakeBlock`(그 플레이어에게만 보이는 블록), `MoveInput`, `ViewFakeEntity`/`MoveFakeEntity`/`RemoveFakeEntity`(그 플레이어에게만 보이는 엔티티) |
| `server/session/fake_chunk.go` (신규), `chunk.go`, `world.go`, `session.go` | `ViewFakeChunk`/`ClearFakeChunk`(그 플레이어에게만 보이는 청크: 그동안 그 청크의 월드 블록 갱신은 안 보내고, 청크 재전송·서브청크 요청은 가짜 청크로 응답), `ViewChunkCentre`(텔레포트 직후 보낸 가짜 청크를 클라이언트가 버리지 않게 청크 중심을 바로 알림). 섬 확장 `map` 보기용 |
| `server/session/handler_player_auth_input.go` | 이동할 때 열린 엔티티 상자의 엔티티를 플레이어 위치로 옮김. 누른 이동 키(`RawMoveVector`, 움직일 수 없을 때도 옴)를 저장 → `Player.MoveInput` |
| `server/session/handler_container_close.go` | 엔티티 상자가 열려 있으면 클라이언트의 255 창 닫기는 무시. Retry 로 여는 중에는 그 창의 닫기도 응답만 하고 무시 |
| `server/session/handler_player_action.go` | `BlockActionDebug` 면 클라이언트가 보낸 블록 부수기 동작(시작·계속·중단·예측 파괴)을 INFO 로그로 |
| `server/block/crop.go` 외 작물·줄기 6종 `RandomTick` | `block.ScheduledCropGrowth` 가 켜져 있으면 작물·수박/호박 줄기가 무작위 틱으로 자라지 않음(게임이 인스턴스 시간 일정으로 키움) |
| `server/block/carrot.go`, `potato.go`, `wheat_seeds.go`, `beetroot_seeds.go` | 다 자란 작물을 캘 때 떨어지는 작물·씨앗을 성장 0의 새 아이템으로(원래는 다 자란 상태를 복사해 심으면 바로 다 자란 작물이 됨) |
| `server/cmd/command.go` | 명령어 `Run` 의 panic 을 잡아 ERROR 로그와 오류 출력으로 바꿈(서버가 죽지 않음) |
| `server/item/digger.go` (신규), `server/internal/iteminternal/components.go` | `item.Digger`(`DestroySpeeds`)를 구현한 커스텀 아이템에 `minecraft:digger` 컴포넌트(블록별 캐기 속도)를 보냄. 없으면 클라이언트가 커스텀 도구로 맨손 속도로 캔다 |
| `server/internal/blockinternal/components.go` | `block.Activatable` 을 구현한 커스텀 블록에 `minecraft:custom_components`(`hasPlayerInteract`=1)를 보냄. 상자처럼 클라이언트가 그 블록을 클릭할 때 손에 든 블록의 설치를 예측하지 않는다(형식은 PowerNukkitX `CustomBlockDefinition.isPlayerInteractable` 참고) |
| `server/world/item.go`, `server/internal/packbuilder/items.go` | `world.CustomItem` 에서 `Texture()` 를 빼고 선택 인터페이스 `world.CustomItemTextured` 로 옮김. 텍스처를 리소스팩으로 따로 관리하는 아이템은 이미지가 필요 없다(자동 팩 생성은 텍스처가 있는 아이템만) |
| `server/world/tick_stats.go` (신규), `tick.go`, `world.go` | 월드 틱마다 걸린 시간(틱 예정 시각부터 끝날 때까지)을 최근 5초만 기록하고 `World.TickStats()`(TPS, 평균·최대 틱 시간)로 돌려줌. 운영자 사이드바의 서버 부하 표시용(`internal/perf`) |
| `server/block/dirt.go` | 잔디·균사체 퍼짐(`spreadDirt`): 위에 꽉 찬 고체 블록(`model.Solid`)이 있으면 흙으로 퍼지지 않고, 그 아래 잔디는 흙으로 돌아감(바닐라와 같게). 원래는 빛만 봐서 빛을 막지 않는 커스텀 블록(코어 등) 아래 흙이 잔디가 됨 |
| `server/player/player.go` (`SendTitle`) | 부제목을 제목보다 먼저 보냄. 클라이언트는 부제목을 다음 제목과 함께 보여서, 제목 뒤에 보내면 한 제목 늦게 나왔다(WAILA 패널이 부제목의 아이콘 값을 읽음) |
| `server/session/handler_modal_form_response.go` | 없는 폼 ID 응답(10개 초과로 버려졌거나 두 번 응답)은 킥 대신 무시. 응답 데이터가 `null` 이면 닫기로 처리(원래는 메뉴 폼에서 0번 버튼이 눌림) |
| `server/session/player.go` (추가) | Nop 세션의 `SendForm` 무시(테스트 플레이어) |
| `server/player/player.go` | `OpenEntityContainer`, `ContainerViewer`, `CloseOpenedContainer`, `AfterClientSync`, `SetCamera`, `ClearCamera`, `SetCameraFOV`, `ClearCameraFOV`, `ViewFakeBlock`, `ViewFakeChunk`, `ClearFakeChunk`, `ViewChunkCentre`. 블록 `Activate` 성공 후 주변 블록 재전송(클라이언트가 예측 설치한 유령 블록 제거), **`Activate` 뒤 손 아이템을 다시 읽어 차감**(원래는 `Activate` 전 값을 되써서 서랍·납품함에 넣은 아이템이 복사됨). **커스텀 블록은 부수기 시간이 지나면 서버가 직접 부순다**(`tickCustomBreak`: 실제 클라이언트가 금 가기는 끝내도 파괴 동작을 보내지 않아 부서지지 않았음). 같은 블록을 계속 캐는 동안(ContinueDestroyBlock 반복) 시작 시각은 유지 |

아이템 이동은 기존 `invByID` 의 `ContainerLevelEntity → openedWindow` 경로를 그대로 탄다.
패치한 곳에는 `dragonfactory patch` 주석을 단다.

### 업그레이드 절차

1. 새 버전을 받는다: `go mod download -json github.com/df-mc/dragonfly@vX.Y.Z` 의 `Dir` 을 복사.
2. `.github` 를 지우고 `third_party/dragonfly` 를 교체한다.
3. `cd third_party/dragonfly && patch -p1 < ../dragonfly.patch` — 충돌은 손으로 고친다.
4. 루트 `go.mod` 의 `require` 버전을 맞추고 `go mod tidy`, `go vet ./...`, `go test ./...`, 시나리오 실행.
5. 원본과 다시 비교해 `dragonfly.patch` 를 갱신한다(`diff -ruN a b`, a=원본, b=패치본).
