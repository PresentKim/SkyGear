# 서랍 구현

기획: [02-skygear.md](../design/02-skygear.md) §15. 코드: `internal/games/skygear/drawer.go`, `holder.go`.

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
  - 엔티티 타입 `skygear:item_holder`, 히트박스 없음, 저장하지 않음(`EncodeNBT` → nil, 서랍이 다시 생성).
  - `item.Carrier`(`HeldItems()`)를 구현하면 Dragonfly가 보는 사람에게 `MobEquipment` 패킷으로 들고 있는 아이템을 보낸다.
  - 서버 엔티티 레지스트리에 추가하면(`EntityTypes`) 접속 시 엔티티 식별자 목록에 포함되어 전송된다.
  - 엔티티는 블록 중앙에서 앞면 쪽으로 조금 나온 곳에 두고, yaw로 앞면을 바라보게 한다(북 180, 동 -90, 서 90, 남 0).
  - **꽉 찬 큐브 모델 블록은 보조 손, 그 외(일반 아이템, 문·묘목·꽃·레일처럼 평면으로 보이는 블록)는 주 손**에 쥐여 준다(`isCubeItem`). 손에 든 블록은 입체로 그려져 각도가 다르다.
  - 깊이(블록 중앙에서 앞면 방향): 주 손 0.05, 보조 손 0.025(`mainHandDepth`, `offHandDepth`). 0 이면 블록 안에 숨는다.
- 리소스팩(`assets/resourcepack/skygear/`)
  - `entity/item_holder.entity.json`: 클라이언트 엔티티 정의(`enable_attachables`, 기본 렌더 컨트롤러).
  - `models/entity/item_holder.geo.json`: `root → body → rightarm → rightitem_pose → rightitem_handheld → rightitem`, `body → leftarm → leftitem`. 손 아이템 본에 3인칭 손 자세를 상쇄하는 피벗·회전을 준다(`rightitem_pose`: pivot [-1, -6, -6], rotation [-75, -12, -16], `leftitem`: pivot [5.5, 1, -3], rotation [-20, 45, 0]). `rightitem_handheld`·`rightitem` 은 같은 피벗에 회전 없이 두고, 손에 쥐는 모양 아이템일 때만 애니메이션이 `rightitem_handheld` 를 돌린다(아래). 구조는 Storage Drawers Bedrock 애드온을 참고했다.
  - `animations/item_holder.animation.json`, `entity/item_holder.entity.json` 의 `scripts.animate`: 손에 쥐는 모양 아이템 보정.
  - 투명 텍스처.
- 지오메트리를 바꾸면 팩 버전이 바뀌어야 클라이언트가 다시 받는다. `common/resourcepack.Load`가 내용 해시로 버전을 부여한다.
- 수량·잠금·업그레이드는 앞면 글자(아래)로 표시한다.

## 손에 쥐는 모양 아이템

클라이언트는 `hand_equipped` 아이템(도구처럼 쥐는 아이템)을 3인칭 손에서 다른 아이템과 다른 회전·크기로 그린다. 홀더 본의 보정은 일반 아이템 기준이라 막대기·뼈는 크게 비스듬히 튀어나온다. 서랍은 겹쳐지지 않는 아이템(도구 등)을 받지 않으므로 우리 도구(`Hammer`, `tinker.Tool`, `HandEquipped() true`)는 서랍에 들어오지 않고, 문제는 겹쳐지는 바닐라 아이템뿐이다.

어떤 아이템이 해당하는가:
- 데이터로 정해진 바닐라 아이템: Dragonfly 의 `server/world/vanilla_items.nbt`(서버가 보내는 아이템 구성 요소)와 Mojang `bedrock-samples` `behavior_pack/items` 에서 `minecraft:hand_equipped` 가 참인 것은 `breeze_rod` 와 창(`*_spear`, 겹쳐지지 않음)뿐이다.
- 막대기·뼈는 이 데이터에 구성 요소가 없다(`component_based` 0). 클라이언트 코드에 박혀 있어 공개된 목록이 없고, 서랍 스크린샷으로 확인했다. 블레이즈 막대는 막대기와 같은 막대 아이템이라 목록에 넣었고 실기 확인이 필요하다.

검토한 방법:
- 서버가 바닐라 아이템 구성 요소를 바꿔 `hand_equipped` 를 끄기: 아이템 등록 데이터는 모든 플레이어 손에 적용되어 플레이어가 막대기를 드는 모습까지 바뀐다. 막대기·뼈는 구성 요소 기반이 아니라 바꾸면 다른 동작이 깨질 수 있다. 쓰지 않는다.
- 아이템 엔티티·액자: 아이템 엔티티는 돌고 합쳐진다(위). 액자는 블록이라 서랍 앞 칸을 차지한다. 쓰지 않는다.
- 엔티티 텍스처에 아이템 그림을 그리기: 클라이언트 엔티티 텍스처는 아이템마다 따로 고를 수 없어(렌더 컨트롤러 배열과 서버 변형 값이 아이템 수만큼 필요) 모든 아이템에 쓸 수 없다.
- **리소스팩 Molang 으로 보정(사용 중)**: 홀더가 `query.is_item_name_any('slot.weapon.mainhand', 0, ...)` 로 주 손 아이템 이름을 보고, 목록에 있으면 `animation.skygear.item_holder.handheld` 가 `rightitem_handheld` 를 돌리고 옮기고 줄인다. 보정 본을 기본 자세 본(`rightitem_pose`) 아래에 따로 두어 애니메이션 회전이 기본 회전에 더해지지 않고 따로 곱해진다.

보정 값: Bedrock 의 3인칭 아이템 변환 값은 공개 자료에 없어 실제 클라이언트에서 맞춘다. 기본은 크기 0.65 만 준다(Java 판 아이템 모델의 3인칭 크기 `item/generated` 0.55 와 `item/handheld` 0.85 의 비, 스크린샷의 막대기가 약 1.5배 큰 것과 맞다). Java 판 회전 차이([0, -90, 55])를 되돌린 값은 막대기를 납작하게 아래로 숨겨 맞지 않았다.

맞추는 법: `/sgdebug holderpose <x> <y> <z> <크기%> [dx dy dz]` 가 그 세이브의 모든 서랍 홀더에 회전(도, 5도 단위, -180~180)·크기(퍼센트, 1~255)·위치(픽셀, -16~15)를 보낸다. 홀더 엔티티의 variant(회전 셋)와 mark variant(크기·위치)에 담아 보내고 애니메이션이 풀어 쓴다(`newHolderPose`, Molang 수는 32비트 실수라 두 값 모두 2^24 미만). 팩을 다시 받지 않고 바로 바뀐다. 맞는 값을 찾으면 애니메이션의 기본값(`query.variant > 0 ? ... : 기본`)에 적는다. 다른 아이템이 같은 증상이면 `item_holder.entity.json` 의 이름 목록에 더한다.

## 앞면 글자 방향 (디버그 셰이프)

수량·잠금·업그레이드 글자는 회전을 고정한 디버그 셰이프 글자(`shapes.go` `faceText`)다. 셰이프의 `Rotation` 은 [pitch, yaw, roll] **도(°)** 단위이고, yaw 0 이면 **북쪽에서 읽히는** 방향이다.
앞면 방향별 yaw: 북 0, 동 -90, 남 180, 서 90(`faceTextYaw`, `TestFaceTextRotation`).
- 엔티티 yaw(남 0, 북 180)와 기준이 달라 그대로 쓰면 남·북이 뒤집힌다.
- 라디안 값(0~3.14)을 넣으면 사실상 0°라 동·서가 90° 돌아간다.
