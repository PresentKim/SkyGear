# 서랍 구현 검증

기획: [02-dragonfactory.md](../design/02-dragonfactory.md) §10. 검증 코드: `spike/drawer.go`, `spike/drawer_test.go`, `spike/display_test.go`.

## 검증 결과

| 항목 | 결과 | 근거 |
|---|---|---|
| 호퍼 → 서랍 → 호퍼 연결 | ✅ | `TestDrawerHopperChain`: 밀 10개가 위 호퍼 → 서랍 → 아래 호퍼로 이동, 다른 아이템(사과)은 거부 |
| 대용량 수량·아이템 종류 저장 | ✅ | 블록 엔티티 NBT에 `Count`(int64) + 아이템 NBT 저장 (`TestMachineTicksWithoutPlayersAndPersists`와 같은 방식) |
| 앞면 아이템 표시(아이템 엔티티 방식) | ✅ 가능하나 **채택하지 않음** | `TestDisplayItemEntity`: 제자리 유지는 되지만 클라이언트에서 회전·흔들림 |
| 앞면 아이템 표시(커스텀 엔티티 방식) | ✅ 서버측 확인 / 화면 확인 필요 | 아래 "아이템 홀더 엔티티" |
| 표시 아이템과 떨어진 아이템 합쳐짐 | ⚠️ 대응 필요 → 해결 | 같은 아이템이 근처에 떨어지면 서로 합쳐져 둘 다 사라짐. 표시 아이템에 고유 값(`WithValue("drawer_display", true)`)을 넣으면 합쳐지지 않음 |
| 표시 아이템을 호퍼가 흡수 | ⚠️ 대응 필요 | `TestHopperAbsorbsDisplayItem`: 표시 아이템이 있는 칸 바로 아래에 호퍼가 있으면 일반 드롭처럼 빨려 들어감 |
| 때린 면 판별 | ✅ (API 확인) | `block.Punchable.Punch(pos, face, tx, user)`로 면을 받음 |

## 구현 방식

### 블록
- `world.CustomBlock` + `CustomItem`(서랍 아이템) + `NBTer`. 비바닐라 블록은 `CustomBlock`과 `CustomItem`을 구현해야 등록된다.
- 호퍼 연동: `block.HopperInsertable`(`InsertItem`), `block.HopperExtractable`(`ExtractItem`) 구현. 한 번에 1개씩 이동(바닐라 호퍼 속도).
- 앞면 방향: 방향 상태(`States()`/`Permutations()`)로 앞면 텍스처를 돌린다. 런타임 표시는 M2에서 실제 클라이언트로 확인.

### 조작
- 넣기: `Activatable.Activate`(면 정보 포함). 앞면일 때만 처리, 같은 플레이어의 직전 사용 시각으로 더블 클릭 판정.
- 꺼내기: `Punchable.Punch`(면 정보 포함). 앞면이면 꺼내고 "앞면을 때림"을 플레이어별로 기록.
- 앞면 파괴 방지: `HandleBlockBreak`에는 면 정보가 없으므로, 위 기록을 보고 앞면이었으면 파괴를 취소한다.

### 표시 — 아이템 홀더 엔티티 (채택)

아이템 엔티티는 클라이언트에서 항상 회전·흔들리므로, **아이템을 손에 든 커스텀 엔티티**로 표시한다.

- 서버(Go): `spike/holder.go`
  - 엔티티 타입 `dragonfactory:item_holder`, 히트박스 없음, 저장하지 않음(`EncodeNBT` → nil, 서랍이 다시 생성).
  - `item.Carrier`(`HeldItems()`)를 구현하면 Dragonfly가 보는 사람에게 `MobEquipment` 패킷으로 들고 있는 아이템을 보낸다.
  - 서버 엔티티 레지스트리에 추가: `entity.DefaultRegistry.Config().New(append(기본 타입들, ItemHolderType))` → 접속 시 엔티티 식별자 목록에 포함되어 전송된다.
  - 방향은 엔티티 yaw로 서랍 앞면 쪽을 향하게 한다.
- 리소스팩: `assets/resourcepack/dragonfactory/`
  - `entity/item_holder.entity.json`: 클라이언트 엔티티 정의(`enable_attachables`, 기본 렌더 컨트롤러).
  - `models/entity/item_holder.geo.json`: 큐브 없는 `root` 본과 아이템이 붙는 `rightItem` 본.
  - 투명 텍스처.
- 확인된 것(hopserver/hopbot): 리소스팩 전송, 식별자 목록에 포함, `AddActor(dragonfactory:item_holder)`, 들고 있는 아이템 `MobEquipment` 수신.
- **실제 클라이언트에서 확인할 것**: 아이템이 보이는지, `rightItem` 본의 위치·회전(앞면에 평평하게 붙도록), 블록 아이템과 일반 아이템의 크기 차이. 본 회전값은 화면을 보며 조정한다.
- 수량은 `entity.NewText`(떠 있는 글자)로 표시한다.

#### 실제 클라이언트로 확인하는 방법
1. 저장소 루트에서 `go run ./spike/cmd/hopserver` (인증 꺼짐, 포트 19132).
2. Bedrock 클라이언트로 접속 → `/hop` 으로 인스턴스 월드 이동 → (1, 64, 0)의 테스트 블록 앞면(북쪽)에 밀이 보이는지 확인.
3. Windows에서 같은 PC로 접속할 때는 UWP 루프백 예외가 필요할 수 있다.

### (참고) 아이템 엔티티 방식에서 확인한 것
- 아이템: `entity.ItemType` 엔티티, `Gravity: 0`, `PickupDelay`·`ExistenceDuration` 최대값, 아이템에 `drawer_display` 값.
- 같은 아이템이 근처에 떨어지면 합쳐져 사라지므로 고유 값이 필요하고, 아래 칸 호퍼에 흡수된다.
  커스텀 엔티티 방식은 아이템 엔티티가 아니므로 **합쳐짐·호퍼 흡수 문제가 없다.**
