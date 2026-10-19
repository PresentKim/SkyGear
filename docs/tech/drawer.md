# 서랍 구현 검증

기획: [02-dragonfactory.md](../design/02-dragonfactory.md) §10. 검증 코드: `spike/drawer.go`, `spike/drawer_test.go`, `spike/display_test.go`.

## 검증 결과

| 항목 | 결과 | 근거 |
|---|---|---|
| 호퍼 → 서랍 → 호퍼 연결 | ✅ | `TestDrawerHopperChain`: 밀 10개가 위 호퍼 → 서랍 → 아래 호퍼로 이동, 다른 아이템(사과)은 거부 |
| 대용량 수량·아이템 종류 저장 | ✅ | 블록 엔티티 NBT에 `Count`(int64) + 아이템 NBT 저장 (`TestMachineTicksWithoutPlayersAndPersists`와 같은 방식) |
| 앞면 아이템 표시(아이템 엔티티 방식) | ✅ 가능하나 **채택하지 않음** | `TestDisplayItemEntity`: 제자리 유지는 되지만 클라이언트에서 회전·흔들림 |
| 앞면 아이템 표시(커스텀 엔티티 방식) | ✅ 실제 클라이언트 확인 완료 | 아래 "아이템 홀더 엔티티" |
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
  - 방향은 엔티티 yaw로 서랍 앞면 쪽을 향하게 한다(북 180, 동 -90, 서 90, 남 0).
- 리소스팩: `assets/resourcepack/dragonfactory/`
  - `entity/item_holder.entity.json`: 클라이언트 엔티티 정의(`enable_attachables`, 기본 렌더 컨트롤러).
  - `models/entity/item_holder.geo.json`: 큐브 없는 `root` 본과 아이템이 붙는 `rightItem` 본.
  - 투명 텍스처.
- 확인된 것(hopserver/hopbot): 리소스팩 전송, 식별자 목록에 포함, `AddActor(dragonfactory:item_holder)`, 들고 있는 아이템 `MobEquipment` 수신.
- 수량은 `entity.NewText`(떠 있는 글자)로 표시한다.

#### 1차 화면 확인 결과 (사용자 스크린샷)
- 아이템은 보인다 → 커스텀 엔티티 + `rightItem` 본 방식은 동작한다.
- 다만 손에 든 자세(3인칭 손 변환)가 그대로 적용되어 비스듬히 기울고 앞면 왼쪽 아래로 삐져나온다.
- 테스트 블록의 텍스처는 면을 구분하는 테두리가 있는 임시 그림이다.

#### 회전값 보정 1차 (실패)
`rightItem` 본 하나에 회전값만 다르게 준 후보 A~H가 **모두 같은 각도**로 보였다 → 단일 본의 회전이 적용되지 않음.

#### 회전값 보정 2차 결과
- 본 이름 대소문자(B), 팔 본 유무(C)와 관계없이 세 변형 모두 같은 모습 → 참고 구조(A) 하나만 사용, 보정용 팩은 삭제.
- **각도는 맞음**: 돌은 앞면에 평평한 사각형으로, 밀은 앞을 향해 선 모습.
- **깊이가 문제**: 엔티티를 블록 중앙에 두면 돌은 앞면에 거의 붙어 너무 깊고, 밀·묘목은 블록 안에 숨는다.

#### 보정 결과가 계속 같았던 원인: 리소스팩 캐시
- 수동 팩의 manifest 버전을 계속 `0.0.1`로 두어, 클라이언트가 **처음 받은 팩을 캐시에서 계속 사용**했을 가능성이 높다
  (서버는 클라이언트가 이미 가진 UUID+버전의 팩을 보내지 않는다).
- 따라서 2차·3차 보정은 새 지오메트리가 클라이언트에 전달되지 않은 상태의 결과로 보고 무효 처리한다.
- 수정: `internal/resourcepack.Load`가 팩 내용 해시로 버전을 자동 부여한다. 이후 팩이 바뀌면 클라이언트가 다시 받는다.

#### 깊이 보정 1회차 결과
- 참고 애드온 지오메트리가 처음으로 전달됨(자동 버전).
- 깊이 0(블록 중앙): 아이템이 블록 안에 완전히 숨음.
- 주 0.10 / 보조 0.05: 앞면에서 살짝 떨어져 보임 → 적정값은 그 사이.

#### 깊이 보정 2회차 결과 — 확정
- 0.02~0.10(주) / 0.01~0.05(보조) 비교 결과, 기둥 2(0.04/0.02)와 3(0.06/0.03) 사이로 확정.
- **확정값: 주 손 0.05, 보조 손 0.025** (블록 중앙에서 앞면 방향, `spike.MainHandDepth`, `spike.OffHandDepth`).
- 각도는 참고 애드온 구조의 지오메트리 그대로(`assets/resourcepack/dragonfactory/models/entity/item_holder.geo.json`).
- 서랍 구현 시 `spike.HolderPosition`/`IsCubeItem`/아이템 홀더 엔티티를 게임 패키지로 옮겨 사용한다.

#### 실제 클라이언트로 확인하는 방법
1. 저장소 루트에서 `go run ./spike/cmd/hopserver` (인증 꺼짐, 포트 19132).
2. Bedrock 클라이언트로 접속 → `/hop` 으로 인스턴스 월드 이동.
3. Windows에서 같은 PC로 접속할 때는 UWP 루프백 예외가 필요할 수 있다.
4. 보정판: 스폰 지점에서 남쪽(+Z)을 보면 1~5 기둥이 보인다.

### (참고) 아이템 엔티티 방식에서 확인한 것
- 아이템: `entity.ItemType` 엔티티, `Gravity: 0`, `PickupDelay`·`ExistenceDuration` 최대값, 아이템에 `drawer_display` 값.
- 같은 아이템이 근처에 떨어지면 합쳐져 사라지므로 고유 값이 필요하고, 아래 칸 호퍼에 흡수된다.
  커스텀 엔티티 방식은 아이템 엔티티가 아니므로 **합쳐짐·호퍼 흡수 문제가 없다.**

## 앞면 글자 방향 (디버그 셰이프)

수량·잠금·업그레이드 글자는 회전을 고정한 디버그 셰이프 글자(`shapes.go` `faceText`)다. 셰이프의 `Rotation` 은 [pitch, yaw, roll] **도(°)** 단위이고, yaw 0 이면 **북쪽에서 읽히는** 방향이다.
앞면 방향별 yaw: 북 0, 동 -90, 남 180, 서 90(`faceTextYaw`, `TestFaceTextRotation`). 10차 실기에서 찾았다:
- 엔티티 yaw(남 0, 북 180, 동 -90, 서 90)를 넣었더니 동·서는 맞고 남·북만 뒤집혀 보였다.
- 회전 단위는 도(°)이고 0°가 북쪽이다. 라디안 값(0~3.14)을 넣으면 사실상 0°라 동·서가 90° 돌아간다.
