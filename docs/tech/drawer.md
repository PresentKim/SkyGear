# 서랍 구현

기획: [02-dragonfactory.md](../design/02-dragonfactory.md) §15. 코드: `internal/games/dragonfactory/drawer.go`, `holder.go`.

## 블록
- `world.CustomBlock` + `CustomItem`(서랍 아이템) + `NBTer`. 비바닐라 블록은 `CustomBlock`과 `CustomItem`을 구현해야 등록된다.
- 블록 엔티티 NBT에 수량(int64)과 아이템 NBT를 저장한다.
- 호퍼 연동: `block.HopperInsertable`(`InsertItem`), `block.HopperExtractable`(`ExtractItem`). 한 번에 1개씩 이동(바닐라 호퍼 속도), 다른 아이템은 거부한다.
- 앞면 방향: 방향 상태(`States()`/`Permutations()`)로 앞면 텍스처를 돌린다.

## 조작
- 넣기: `Activatable.Activate`(면 정보 포함). 앞면일 때만 처리, 같은 플레이어의 직전 사용 시각으로 더블 클릭 판정.
- 꺼내기: `Punchable.Punch(pos, face, tx, user)`. 앞면이면 꺼내고 "앞면을 때림"을 플레이어별로 기록.
- 앞면 파괴 방지: `HandleBlockBreak`에는 면 정보가 없으므로, 위 기록을 보고 앞면이었으면 파괴를 취소한다.

## 표시: 아이템 홀더 엔티티

아이템 엔티티는 클라이언트에서 항상 회전·흔들리고, 근처에 떨어진 같은 아이템과 합쳐지며, 아래 칸 호퍼에 흡수된다. 그래서 **아이템을 손에 든 커스텀 엔티티**로 표시한다. 아이템 엔티티가 아니므로 합쳐짐·호퍼 흡수 문제가 없다.

- 서버(Go, `holder.go`)
  - 엔티티 타입 `dragonfactory:item_holder`, 히트박스 없음, 저장하지 않음(`EncodeNBT` → nil, 서랍이 다시 생성).
  - `item.Carrier`(`HeldItems()`)를 구현하면 Dragonfly가 보는 사람에게 `MobEquipment` 패킷으로 들고 있는 아이템을 보낸다.
  - 서버 엔티티 레지스트리에 추가하면(`EntityTypes`) 접속 시 엔티티 식별자 목록에 포함되어 전송된다.
  - 엔티티는 블록 중앙에서 앞면 쪽으로 조금 나온 곳에 두고, yaw로 앞면을 바라보게 한다(북 180, 동 -90, 서 90, 남 0).
  - **꽉 찬 큐브 모델 블록은 보조 손, 그 외(일반 아이템, 문·묘목·꽃·레일처럼 평면으로 보이는 블록)는 주 손**에 쥐여 준다(`isCubeItem`). 손에 든 블록은 입체로 그려져 각도가 다르다.
  - 깊이(블록 중앙에서 앞면 방향): 주 손 0.05, 보조 손 0.025(`mainHandDepth`, `offHandDepth`). 0 이면 블록 안에 숨는다.
- 리소스팩(`assets/resourcepack/dragonfactory/`)
  - `entity/item_holder.entity.json`: 클라이언트 엔티티 정의(`enable_attachables`, 기본 렌더 컨트롤러).
  - `models/entity/item_holder.geo.json`: `root → body → rightarm/leftarm → rightitem/leftitem`. 손 아이템 본에 3인칭 손 자세를 상쇄하는 피벗·회전을 준다(`rightitem`: pivot [-1, -6, -6], rotation [-75, -12, -16], `leftitem`: pivot [5.5, 1, -3], rotation [-20, 45, 0]). 구조는 Storage Drawers Bedrock 애드온을 참고했다.
  - 투명 텍스처.
- 지오메트리를 바꾸면 팩 버전이 바뀌어야 클라이언트가 다시 받는다. `common/resourcepack.Load`가 내용 해시로 버전을 부여한다.
- 수량·잠금·업그레이드는 앞면 글자(아래)로 표시한다.

## 앞면 글자 방향 (디버그 셰이프)

수량·잠금·업그레이드 글자는 회전을 고정한 디버그 셰이프 글자(`shapes.go` `faceText`)다. 셰이프의 `Rotation` 은 [pitch, yaw, roll] **도(°)** 단위이고, yaw 0 이면 **북쪽에서 읽히는** 방향이다.
앞면 방향별 yaw: 북 0, 동 -90, 남 180, 서 90(`faceTextYaw`, `TestFaceTextRotation`).
- 엔티티 yaw(남 0, 북 180)와 기준이 달라 그대로 쓰면 남·북이 뒤집힌다.
- 라디안 값(0~3.14)을 넣으면 사실상 0°라 동·서가 90° 돌아간다.
