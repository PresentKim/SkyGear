# 애드온(Add-on) 구현 방법

Bedrock 애드온은 **리소스팩(RP)** 과 **비헤이비어팩(BP, 스크립트 포함)** 으로 이루어진다.
이 서버는 Mojang의 BDS가 아니라 **Dragonfly(Go)** 이므로, 두 팩의 역할이 아래처럼 나뉜다.

## 1. 결론

| 구분 | BDS에서 | 이 서버(Dragonfly)에서 |
|---|---|---|
| 리소스팩 (텍스처, 모델, 애니메이션, 사운드, 언어, 파티클, JSON UI) | 클라이언트에 전송 | **그대로 사용 가능** — 클라이언트에 전송되어 적용됨 |
| 비헤이비어팩 JSON (블록/아이템/엔티티 정의, 루트 테이블, 레시피, 스폰 규칙) | 서버가 실행 | **실행되지 않음** → 같은 내용을 **Go 코드로 정의** |
| Script API (`@minecraft/server` JS) | 서버가 실행 | **실행되지 않음** → **Go 코드로 구현** |
| 커스텀 블록/아이템 정의 | BP JSON | Go 타입이 정의하고, 서버가 컴포넌트를 클라이언트에 전송 |

즉, **"애드온 = Go로 짠 게임 로직 + 리소스팩"** 이다. 비헤이비어팩은 만들지 않는다.

## 2. 커스텀 블록

Go에서 `world.CustomBlock`을 구현하면 Dragonfly가 블록 정의를 클라이언트에 보낸다.

| 기능 | 지원 | 방법 |
|---|---|---|
| 텍스처(면별 포함), 렌더 방식 | ✅ | `Properties().Textures` (`customblock.Material`) |
| 커스텀 모델(geometry) | ✅ | `Properties().Geometry` + `Geometry()`가 `.geo.json` 반환 |
| 충돌/선택 박스, 회전·크기·이동 | ✅ | `Properties()` |
| 상태별 외형(permutation) | ✅ | `States()` / `Permutations()` 구현 |
| 빛 방출/차단, 채굴 시간, 마찰, 가연성 | ✅ | 블록이 해당 인터페이스(`LightEmitter`, `BreakInfo` 등) 구현 |
| 블록 엔티티 데이터, 매 틱 로직, 상호작용 | ✅ | `NBTer`, `TickerBlock`, `Activatable` 등 Go 인터페이스 |
| 위 목록에 없는 BP 블록 컴포넌트 | ⚠️ | Dragonfly 내부 패키지(`internal/blockinternal`)에서 컴포넌트를 만들므로 확장하려면 Dragonfly 수정(포크) 필요 |
| 클라이언트측 블록 애니메이션 | ❌ | 블록은 애니메이션 미지원. 움직이는 연출은 엔티티로 대체 |

## 3. 커스텀 아이템

- `world.CustomItem`(이름, 텍스처, 크리에이티브 분류) 구현 + `world.RegisterItem`.
- 내구도, 음식, 갑옷, 쿨다운, 투척물 등은 아이템이 해당 인터페이스를 구현하면 자동으로 컴포넌트가 붙는다.
- 텍스처는 Go 코드의 `image.Image`로 넘기면 자동 리소스팩에 포함된다.

## 4. 커스텀 엔티티

- Go: `world.EntityType`(식별자 `dragonfactory:xxx`, 히트박스, NBT)을 만들고 서버 엔티티 레지스트리에 등록한다. 등록된 식별자는 접속 시 클라이언트로 전송된다.
- RP: 같은 식별자로 **클라이언트 엔티티 정의**(`entity/*.entity.json`), 지오메트리, 텍스처, 애니메이션, 렌더 컨트롤러를 만든다.
- AI·이동·상호작용은 전부 Go로 구현한다.
- 엔티티가 `item.Carrier`를 구현하면 들고 있는 아이템이 클라이언트로 전송되고, 지오메트리의 `rightItem` 본에 그려진다.
- 첫 사례: 서랍 표시용 `dragonfactory:item_holder` ([drawer.md](drawer.md)). 실제 클라이언트에서 표시 확인 완료.
- 주의: 지오메트리를 바꿔도 팩 버전이 같으면 클라이언트가 캐시를 쓴다 → `resourcepack.Load`의 자동 버전으로 해결(§6).

## 5. UI

| 방식 | 용도 | 비고 |
|---|---|---|
| 폼(`form.Menu`, `form.Custom`, `form.Modal`) | 메뉴, 상점, 설정, 세이브 선택 | 기본 수단 |
| JSON UI로 폼 꾸미기 | 상점·메뉴를 게임 UI처럼 표현 | RP의 `ui/server_form.json` 등을 수정. 폼 제목에 태그를 넣어 화면별로 다른 레이아웃을 적용하는 방식이 일반적 |
| 상자형 인벤토리 UI | InvSee, 분실물 보관함, 기계 인벤토리 | Dragonfly는 실제 컨테이너 블록만 열 수 있음(`OpenBlockContainer`) → 플레이어가 볼 수 없는 위치(섬 아래 등)에 컨테이너 블록을 두고 여는 방식으로 구현 |
| 사이드바, 보스바, 타이틀, 액션바 | 미션·긴급 미션 표시 | Dragonfly API 지원 |

## 6. 리소스팩 운영 방식

두 가지 팩을 함께 쓴다.

1. **자동 생성 팩**: Dragonfly가 커스텀 블록·아이템의 텍스처/모델/이름으로 서버 시작 시 생성(`Resources.AutoBuildPack`).
2. **수동 팩**: 엔티티 모델, 애니메이션, 사운드, JSON UI, 파티클 등 자동 생성이 다루지 않는 것.
   - 저장소에 원본 폴더로 관리: `assets/resourcepack/dragonfactory/` (manifest.json 포함)
   - 서버 시작 시 `internal/resourcepack.Load`로 읽어 `Config.Resources`에 추가한다(`resources/`는 실행 시 생성되는 폴더라 git에서 제외됨).
   - **버전은 자동**: 클라이언트는 팩을 UUID+버전으로 캐시하고, 서버는 클라이언트가 이미 가진 UUID+버전의 팩을 보내지 않는다.
     버전을 안 올리면 바뀐 팩이 전달되지 않으므로, `resourcepack.Load`가 팩 내용 해시로 manifest 버전을 매번 계산해 넣는다. `manifest.json`의 버전은 손대지 않는다.
   - 접속 시 팩 적용을 강제하려면 `Resources.Required = true`.

규칙:
- 식별자 네임스페이스는 `dragonfactory:` 로 통일한다.
- 블록/아이템 텍스처는 가능하면 Go 코드(자동 생성 팩) 쪽에 두고, 수동 팩과 같은 파일 경로를 쓰지 않는다(충돌 방지).

## 7. 바닐라 기능 제약

Dragonfly에 없는 바닐라 블록: 피스톤, 옵저버, 드로퍼, 디스펜서, 리피터, 비교기.
있는 것: 호퍼, 상자, 화로, 레드스톤 가루·횃불·레버 등 일부.
→ 공장 자동화는 커스텀 기계 블록(Go)으로 만든다([02-dragonfactory.md](../design/02-dragonfactory.md) §6).

## 8. 애드온 기능 → 구현 위치 대응표

| 애드온에서 하던 것 | 여기서 할 곳 |
|---|---|
| `blocks/*.json` | `internal/games/dragonfactory/...` Go 블록 타입 |
| `items/*.json` | Go 아이템 타입 |
| `entities/*.json` (BP) | Go `EntityType` + 동작 코드 |
| `entity/*.entity.json` (RP), 모델, 애니메이션 | `assets/resourcepack/dragonfactory/` |
| `loot_tables`, `recipes` | Go 데이터(또는 설정 파일). 바닐라 레시피는 Dragonfly 내장 |
| Script API 이벤트 | Dragonfly 플레이어/월드 핸들러 |
| `ui/*.json` | `assets/resourcepack/dragonfactory/ui/` |
| `sounds`, `particles`, `texts` | `assets/resourcepack/dragonfactory/` |
