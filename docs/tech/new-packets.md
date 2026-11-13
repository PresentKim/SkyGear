# 새 패킷 시험

1.20 이후 생긴 패킷 가운데 Sky Gear 에 쓸 만한 것을 시험하는 기능이다. 실제 클라이언트에서 아직 확인하지 않았으므로 모두 `platform.toml` 의 `[Experimental]` 스위치로 켠다(기본값은 꺼짐). 첫 입장 카메라만 스위치와 상관없이 동작하고, 스위치는 그 방식만 바꾼다.

| 스위치 | 패킷 | 하는 일 | 코드 |
|---|---|---|---|
| `LocatorBar = true` | `GameRulesChanged(locatorBar)`, `LocatorBar`, `PlayerLocation` | 세이브에 들어오면 위치 표시 막대를 켜고 섬 시작 위치(`island.Spawn`)를 금색 웨이포인트로 단다. 매초 같은 세이브의 다른 플레이어 위치를 보내고, 나가면 지운 뒤 막대를 끈다(로비에는 막대가 없다) | `content/skygear/locator`, `common/sghook/locator.go` |
| `CameraSpline = true` | `CameraInstruction.Spline` | 로비 카메라: 14초마다 다음 지점으로 보내던 이동 대신, 다섯 지점을 한 바퀴 도는 스플라인(70초)을 한 번에 보내 클라이언트가 재생한다. 첫 입장 카메라도 스플라인으로 재생 | `platform/lobbymotion.go`, `common/sghook/spline.go` |
| (항상) | `CameraInstruction.Set` (+`Spline`) | 세이브 첫 입장(그 세이브에 저장된 상태가 없을 때) 1.5초 뒤, 카메라가 눈 위치에서 뒤로 10칸·위로 5칸 빠져 섬을 보여 주고(1.3초), 0.6초 머문 뒤 돌아온다(1.3초). 그동안 입력을 잠그고 HUD 를 숨긴다. 스위치가 꺼져 있으면 감속 이동 두 번으로 한다 | `content/skygear/intro` |
| `RecipeUnlock = "toast"` | `UnlockedRecipes`, `GameRulesChanged(recipesunlock)` | 지금처럼 연구한 레시피만 보내되, 레시피마다 해제가 필요하다고 표시하고(`UnlockRequirement`, 재료 없음) 보낸 목록을 해제로 알린다. 연구로 새로 생긴 레시피는 "새로 해제"로 보내 알림이 뜨는지 본다 | `common/sghook/recipeunlock.go` |
| `RecipeUnlock = "book"` | 위와 같음 | 모든 레시피를 한 번만 보내고, 이후 연구는 `UnlockedRecipes` 로만 알린다(전체 `CraftingData` 재전송 없음). 서버는 연구하지 않은 레시피의 제작을 계속 거절한다 | 같음 |

## 확인할 것

- **위치 표시 막대**: 게임 규칙 이름. Dragonfly 는 시작 데이터에서 `locatorBar` 를 끄는데, 글에 따라 `playerWaypoints`·`locator_bar` 라는 이름도 나온다. 막대가 뜨지 않으면 이름부터 본다. 다른 플레이어가 클라이언트에 이미 자동으로 표시되는지, `PlayerLocation` 이 필요한지도 본다.
- **스플라인**: 회전 키프레임 값의 순서(지금은 피치, 요, 롤)와 진행도 키프레임(제어점마다 균등하게 0~1)이 맞는지. 맞지 않으면 카메라가 엉뚱한 곳을 본다. 스플라인 시작 전에 같은 위치로 `Set` 을 먼저 보낸다.
- **레시피 해제 toast**: 새 레시피 알림이 뜨는지, 목록을 다시 보낸 뒤에도 해제가 유지되는지.
- **레시피 해제 book**: 해제하지 않은 레시피를 제작 격자에 직접 놓았을 때 클라이언트가 결과를 보여 주는지. 바닐라는 레시피 해제가 레시피 책에만 영향을 주는 것으로 알려져 있다. 결과가 보이면 서버가 거절해 아이템이 깜빡이므로, 이 방식은 쓸 수 없고 toast 처럼 거른 목록을 유지해야 한다.

## 보류: `PlayerUpdateEntityOverrides`

엔티티의 클라이언트 속성 값(정수·실수·참거짓·열거형)을 플레이어마다 다르게 덮어쓰는 패킷이다. 다음 이유로 이번엔 만들지 않았다.

- 지금 게임에는 플레이어마다 엔티티를 여러 개 만드는 곳이 없다. 로비 캐릭터 모델만 시청자별 가짜 엔티티이고, 범위 상자·요정·거치대·도관 표시 등은 하나를 모두가 같이 본다.
- 글자(이름표)는 속성이 될 수 없다.
- Dragonfly 는 엔티티 속성을 보내지 않는다(`SyncActorProperty`, `AddActor.EntityProperties` 없음). 쓰려면 속성 정의 동기화, 엔티티 추가 때 속성 값, 리소스팩 렌더 컨트롤러의 `query.property` 를 먼저 만들어야 한다.

플레이어별 보임만 필요하면 Dragonfly 포크의 보기 층(`p.ViewVisibility` 등, `third_party/dragonfly/server/session/view_layer.go`)으로 충분하다.
