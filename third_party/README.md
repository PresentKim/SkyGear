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
| `server/session/entity_container.go` (신규) | `OpenEntityContainer`(Nop 세션은 무시, 투명 엔티티 `AddActor` + `ContainerOpen` 의 `ContainerEntityUniqueID`), `CloseOpenedContainer`, 닫힐 때 엔티티 제거·`onClose` 콜백 |
| `server/session/session.go` | `entityContainer` 필드, `AvailableActorIdentifiers` 에 `dragonfly:entity_container` 추가 |
| `server/session/player.go` | `closeCurrentContainer` 가 엔티티 컨테이너면 블록 조회 없이 정리 |
| `server/player/player.go` | `Player.OpenEntityContainer`, `ContainerViewer`(트랜잭션 밖에서 슬롯 갱신 전송), `CloseOpenedContainer` |

아이템 이동은 기존 `invByID` 의 `ContainerLevelEntity → openedWindow` 경로를 그대로 탄다.
패치한 곳에는 `dragonfactory patch` 주석을 단다.

### 업그레이드 절차

1. 새 버전을 받는다: `go mod download -json github.com/df-mc/dragonfly@vX.Y.Z` 의 `Dir` 을 복사.
2. `.github` 를 지우고 `third_party/dragonfly` 를 교체한다.
3. `cd third_party/dragonfly && patch -p1 < ../dragonfly.patch` — 충돌은 손으로 고친다.
4. 루트 `go.mod` 의 `require` 버전을 맞추고 `go mod tidy`, `go vet ./...`, `go test ./...`, 시나리오 실행.
5. 원본과 다시 비교해 `dragonfly.patch` 를 갱신한다(`diff -ruN a b`, a=원본, b=패치본).
