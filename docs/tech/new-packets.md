# 새 패킷

1.20 이후 생긴 패킷 가운데 Sky Gear 가 쓰는 것이다. 스위치 없이 항상 동작한다.

| 패킷 | 하는 일 | 코드 |
|---|---|---|
| `GameRulesChanged(locatorBar)`, `LocatorBar`, `PlayerLocation` | 세이브에 들어오면 위치 표시 막대를 켜고 섬 시작 위치(`island.Spawn`)를 금색 웨이포인트로 단다. 매초 같은 세이브의 다른 플레이어 위치를 보내고, 나가면 지운 뒤 막대를 끈다(로비에는 막대가 없다) | `content/skygear/locator`, `common/sghook/locator.go` |
| `CameraInstruction.Set` (완화 사인 인아웃) | 로비 카메라: 다섯 지점을 14초씩 차례로 옮겨 가며, 도착할 때마다 다음 지점으로 가는 이동을 보낸다 | `platform/lobbymotion.go`, `common/sghook/view.go` |
| `CameraInstruction.Set` (완화 사인 인아웃) | 세이브 첫 입장(그 세이브에 저장된 상태가 없을 때) 1.5초 뒤, 카메라가 섬 위 14칸·플레이어 뒤 4칸에서 60도로 내려다보는 자리에서 시작해 1.5초 동안 플레이어의 눈 위치로 내려온다. 그동안 입력을 잠그고 HUD 를 숨기며, 끝나면 카메라를 풀고 입력·HUD 를 되돌린다 | `content/skygear/intro` |
| `CameraInstruction.Spline` | 카메라 스플라인: 실제 클라이언트에서 로비 스플라인은 하늘만 비췄다. 그래서 두 카메라는 위의 `Set` 이동을 쓰고, 스플라인은 `/sgdebug spline` 으로만 시험한다([admin.md](admin.md)) | `common/sghook/spline.go`, `content/skygear/debug_spline.go` |
| `UnlockedRecipes`, `GameRulesChanged(recipesunlock)` | 연구한 레시피만 보내되, 레시피마다 해제가 필요하다고 표시하고(`UnlockRequirement`, 재료 없음) 보낸 목록을 해제로 알린다. 연구로 새로 생긴 레시피는 "새로 해제"로 보내 알림을 띄운다 | `common/sghook/recipeunlock.go` |
| `VoxelShapes` | 접속할 때 기본 복셀 모양(`minecraft:empty`, `minecraft:unit_cube`)을 보낸다. Dragonfly 는 보내지 않아 1.26 클라이언트가 모양 목록이 없다는 로그를 남긴다 | `common/sghook/voxelshapes.go` |

## 확인할 것

- **위치 표시 막대**: 게임 규칙 이름. Dragonfly 는 시작 데이터에서 `locatorBar` 를 끄는데, 글에 따라 `playerWaypoints`·`locator_bar` 라는 이름도 나온다. 막대가 뜨지 않으면 이름부터 본다. 다른 플레이어가 클라이언트에 이미 자동으로 표시되는지, `PlayerLocation` 이 필요한지도 본다.
- **스플라인**: 공식 정의(Mojang `bedrock-protocol-docs`)로는 제어점이 월드 좌표, 진행도가 곡선 전체의 0~1, `SplineType` 0 이 Catmull-Rom·1 이 직선이다. 동작하는 예(Mcblend)는 Catmull-Rom 에 서로 다른 점을 넷 이상 두고, 연속된 같은 점은 지우며(클라이언트가 깨진다), 진행도를 누적 거리 비율로 준다. gophertunnel v1.63.0 은 `SplineType`·`SplineIdentifier`·`LoadFromJson` 을 선택 필드로 써서 셋 다 비울 때만 맞는다(값을 넣으면 패킷이 어긋난다). 실기(2026-11-14): 점 두 개, 양 끝 복제, 진행도를 제어점 번호로 보낸 것은 모두 위치가 움직이지 않고 시선만 돌았다. 회전 값을 (피치, 요)로 보내면 좌우로, (요, 피치)로 보내면 위아래로 돌았다. 점 다섯 개와 거리 비율 진행도로 보내도 위에서 내려다보는 장면 없이 눈 근처에서 시선만 돌았다(2026-11-14). 패킷 인코딩은 upstream gophertunnel 과 바이트가 같아 원인이 아니다. `noset` 은 아무 반응이 없어 스플라인 앞에 자유 카메라 `Set` 이 필요하고, `rel` 도 시선만 돌았다. MiNET 은 `Set` 을 보내고 1초 뒤 스플라인을 따로 보내며(같은 틱의 카메라 명령이 무시된다는 보고, MCPE-173524), 회전은 (피치, 요, 롤)에 위쪽이 + 인 피치다. `wait` 로는 1초 동안 위에서 내려다본 뒤, 스플라인이 시작되자 위치가 한 번에 끝점(눈)으로 가고 회전만 시간에 맞춰 돌았다. 진행도 시간을 비율(`tfrac`)·틱(`tticks`)으로 보내도 같았고, 진행도 키프레임을 빼면(`noprog`) 클라이언트가 강제 종료됐다. 한 번에 가는 곳이 곡선의 끝점인지 플레이어인지 `endup`(끝점을 눈 6칸 위로)으로 가린다. Mcblend 는 점이 넷 미만이거나 부드러운 보간이 아니면 직선 스플라인을 쓴다. gophertunnel v1.63.0 은 종류를 넣으면 패킷이 어긋나므로, 스플라인 패킷은 `sghook` 이 클라이언트가 읽는 배치로 직접 쓴다(`splinePacket`, 기본 종류일 때 gophertunnel 과 바이트가 같은지 테스트한다). 직선은 `lin` 으로 시험한다. 확인되기 전까지 로비·첫 입장 카메라는 스플라인을 쓰지 않는다.
- **레시피 해제**: 새 레시피 알림이 뜨는지, 목록을 다시 보낸 뒤에도 해제가 유지되는지.

## 보류: `PlayerUpdateEntityOverrides`

엔티티의 클라이언트 속성 값(정수·실수·참거짓·열거형)을 플레이어마다 다르게 덮어쓰는 패킷이다. 다음 이유로 이번엔 만들지 않았다.

- 지금 게임에는 플레이어마다 엔티티를 여러 개 만드는 곳이 없다. 로비 캐릭터 모델만 시청자별 가짜 엔티티이고, 범위 상자·요정·거치대·도관 표시 등은 하나를 모두가 같이 본다.
- 글자(이름표)는 속성이 될 수 없다.
- Dragonfly 는 엔티티 속성을 보내지 않는다(`SyncActorProperty`, `AddActor.EntityProperties` 없음). 쓰려면 속성 정의 동기화, 엔티티 추가 때 속성 값, 리소스팩 렌더 컨트롤러의 `query.property` 를 먼저 만들어야 한다.

플레이어별 보임만 필요하면 Dragonfly 포크의 보기 층(`p.ViewVisibility` 등, `third_party/dragonfly/server/session/view_layer.go`)으로 충분하다.
