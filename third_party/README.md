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
| `server/session/handler_player_auth_input.go` | 이동할 때 열린 엔티티 상자의 엔티티를 플레이어 위치로 옮김 |
| `server/session/handler_container_close.go` | 엔티티 상자가 열려 있으면 클라이언트의 255 창 닫기는 무시. Retry 로 여는 중에는 그 창의 닫기도 응답만 하고 무시 |
| `server/session/handler_modal_form_response.go` | 없는 폼 ID 응답(10개 초과로 버려졌거나 두 번 응답)은 킥 대신 무시 |
| `server/player/player.go` | `OpenEntityContainer`, `ContainerViewer`, `CloseOpenedContainer`, `AfterClientSync`. 블록 `Activate` 성공 후 주변 블록 재전송(클라이언트가 예측 설치한 유령 블록 제거), **`Activate` 뒤 손 아이템을 다시 읽어 차감**(원래는 `Activate` 전 값을 되써서 서랍·납품함에 넣은 아이템이 복사됨) |

아이템 이동은 기존 `invByID` 의 `ContainerLevelEntity → openedWindow` 경로를 그대로 탄다.
패치한 곳에는 `dragonfactory patch` 주석을 단다.

### 업그레이드 절차

1. 새 버전을 받는다: `go mod download -json github.com/df-mc/dragonfly@vX.Y.Z` 의 `Dir` 을 복사.
2. `.github` 를 지우고 `third_party/dragonfly` 를 교체한다.
3. `cd third_party/dragonfly && patch -p1 < ../dragonfly.patch` — 충돌은 손으로 고친다.
4. 루트 `go.mod` 의 `require` 버전을 맞추고 `go mod tidy`, `go vet ./...`, `go test ./...`, 시나리오 실행.
5. 원본과 다시 비교해 `dragonfly.patch` 를 갱신한다(`diff -ruN a b`, a=원본, b=패치본).
