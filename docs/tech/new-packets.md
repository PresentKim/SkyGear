# 새 패킷

1.20 이후 생긴 패킷 가운데 Sky Gear 가 쓰는 것이다. 스위치 없이 항상 동작한다.

| 패킷 | 하는 일 | 코드 |
|---|---|---|
| `GameRulesChanged(locatorBar)`, `LocatorBar`, `PlayerLocation` | 세이브에 들어오면 위치 표시 막대를 켜고 섬 시작 위치(`island.Spawn`)를 금색 웨이포인트로 단다. 매초 같은 세이브의 다른 플레이어 위치를 보내고, 나가면 지운 뒤 막대를 끈다(로비에는 막대가 없다) | `content/skygear/locator`, `common/sghook/locator.go` |
| `CameraInstruction.Set` (완화 사인 인아웃) | 로비 카메라: 다섯 지점을 14초씩 차례로 옮겨 가며, 도착할 때마다 다음 지점으로 가는 이동을 보낸다 | `platform/lobbymotion.go`, `common/sghook/view.go` |
| `CameraInstruction.Set` (완화 사인 인아웃) | 세이브 첫 입장(그 세이브에 저장된 상태가 없을 때) 1.5초 뒤, 카메라가 눈 위치에서 뒤로 10칸·위로 5칸 빠져 섬을 보여 주고(1.3초), 0.6초 머문 뒤 돌아온다(1.3초). 그동안 입력을 잠그고 HUD 를 숨긴다. 시작 위치로 `Set` 을 먼저 보내 카메라를 고정한다 | `content/skygear/intro` |
| `CameraInstruction.Spline` | 카메라 스플라인: 실제 클라이언트에서 로비 스플라인은 하늘만 비췄다. 그래서 두 카메라는 위의 `Set` 이동을 쓰고, 스플라인은 `/sgdebug spline` 으로만 시험한다([admin.md](admin.md)) | `common/sghook/spline.go`, `content/skygear/debug_spline.go` |
| `UnlockedRecipes`, `GameRulesChanged(recipesunlock)` | 연구한 레시피만 보내되, 레시피마다 해제가 필요하다고 표시하고(`UnlockRequirement`, 재료 없음) 보낸 목록을 해제로 알린다. 연구로 새로 생긴 레시피는 "새로 해제"로 보내 알림을 띄운다 | `common/sghook/recipeunlock.go` |
| `VoxelShapes` | 접속할 때 기본 복셀 모양(`minecraft:empty`, `minecraft:unit_cube`)을 보낸다. Dragonfly 는 보내지 않아 1.26 클라이언트가 모양 목록이 없다는 로그를 남긴다 | `common/sghook/voxelshapes.go` |

## 확인할 것

- **위치 표시 막대**: 게임 규칙 이름. Dragonfly 는 시작 데이터에서 `locatorBar` 를 끄는데, 글에 따라 `playerWaypoints`·`locator_bar` 라는 이름도 나온다. 막대가 뜨지 않으면 이름부터 본다. 다른 플레이어가 클라이언트에 이미 자동으로 표시되는지, `PlayerLocation` 이 필요한지도 본다.
- **스플라인**: 회전 키프레임 값의 순서(지금은 피치, 요, 롤)와 진행도 키프레임(제어점마다 균등하게 0~1)이 맞는지. 맞지 않으면 카메라가 엉뚱한 곳을 본다. `/sgdebug spline` 의 `rot=yp`(요, 피치 순서), `pad`(양 끝 제어점 복제), `type=linear` 로 원인을 가려 본다.
- **레시피 해제**: 새 레시피 알림이 뜨는지, 목록을 다시 보낸 뒤에도 해제가 유지되는지.

## 보류: `PlayerUpdateEntityOverrides`

엔티티의 클라이언트 속성 값(정수·실수·참거짓·열거형)을 플레이어마다 다르게 덮어쓰는 패킷이다. 다음 이유로 이번엔 만들지 않았다.

- 지금 게임에는 플레이어마다 엔티티를 여러 개 만드는 곳이 없다. 로비 캐릭터 모델만 시청자별 가짜 엔티티이고, 범위 상자·요정·거치대·도관 표시 등은 하나를 모두가 같이 본다.
- 글자(이름표)는 속성이 될 수 없다.
- Dragonfly 는 엔티티 속성을 보내지 않는다(`SyncActorProperty`, `AddActor.EntityProperties` 없음). 쓰려면 속성 정의 동기화, 엔티티 추가 때 속성 값, 리소스팩 렌더 컨트롤러의 `query.property` 를 먼저 만들어야 한다.

플레이어별 보임만 필요하면 Dragonfly 포크의 보기 층(`p.ViewVisibility` 등, `third_party/dragonfly/server/session/view_layer.go`)으로 충분하다.
