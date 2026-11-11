# common

게임과 상관없는 Go 패키지(import 경로 `github.com/presentkim/skygear/common/...`).
게임 규칙·플랫폼 규칙(로비, 세이브, 계정 저장)은 담지 않고, 어느 게임에서나 같은 동작만 둔다.

## 패키지

| 패키지 | 내용 |
|---|---|
| `sghook` | Dragonfly 를 패치하지 않고 세션에 닿는 훅(패킷 가로채기, `go:linkname`, 리플렉션): 카메라·HUD·입력 잠금·가짜 블록·제목·DDUI 패킷·폼 응답 정리·`minecraft:falling_block` 으로 보내는 커스텀 엔티티(떠 있는 글자)에 공기 블록 지정 |
| `clientdef` | 커스텀 아이템이 클라이언트에 보내는 컴포넌트를 Dragonfly 패치 없이 더함(`ItemEntry`, `sghook.EditCustomItems` 안에서): `Digger`·`TagDigger` 를 구현하면 `minecraft:digger`(캐기 속도), `item.Usable` 이면 `minecraft:use_modifiers`, `item.Fuel` 이면 `minecraft:fuel` |
| `chestui` | 아이템이 실제로 오가는 창(엔티티 상자). `assets/resourcepack/inventoryui` 와 짝 |
| `ui` | 메뉴 화면(메뉴·카드·목록·확인·입력·칸 격자·분할·퀘스트 책 등)과 토스트. 화면은 `assets/resourcepack/oreui` 가 그리는 DDUI 화면이고(`oreui`), 그림은 `assets/resourcepack/sgui` 에 있다 |
| `oreui` | 서버 주도 DDUI 화면(레이아웃·값 보내기, 누름 받기). `assets/resourcepack/oreui` 와 짝 |
| `hud` | 사이드바 점수판으로 보내는 HUD 글. `assets/resourcepack/sgui` 의 `ui/scoreboards.json` 이 그린다(`sgui` 는 HUD 의 JSON UI 와 메뉴 화면의 그림) |
| `waila` | 바라보는 블록·엔티티 패널. `assets/resourcepack/waila` 와 짝 |
| `slotdeco` | 아이템 칸 장식 표식. `assets/resourcepack/slotdeco` 와 짝 |
| `i18n` | 플레이어 언어별 글 카탈로그. 서버는 `i18n.Register` 로 자기 카탈로그를 더한다. `i18n/i18ntest` 는 서버가 자기 코드에 돌리는 카탈로그 검사 |
| `timed` | 기간제 아이템(현실 시간 만료) |
| `perf` | 월드 틱 부하 표시 |
| `playerstate` | 플레이어 상태(인벤토리·위치·체력 등) 캡처·복원·직렬화 |
| `resourcepack` | 리소스팩 읽기(내용 해시로 버전 부여, 압축·암호화) |
| `backup` | 폴더 백업 |
| `textsearch` | 한글 초성 등 글 검색 |
| `itemio` | 블록이 아이템을 주고받는 방법(호퍼·도관) |
| `energy` | RE 에너지를 공급원에서 소비처로 나르는 망 |
| `itemnet` | 맞닿은 아이템 도관 망의 아이템 이동 |
| `fluidnet` | 맞닿은 액체 도관 망의 유체 이동 |
| `netgraph` | 도관 망을 이벤트로만 바뀌는 그래프로 둠(노드를 더하면 망을 합치고, 빼면 가르며, 망마다 공유 버퍼 `Store` 를 가짐) |
| `storagenet` | 저장망·제작망의 엔진 형식(키 있는 아이템 묶음, 저장소, 제작 패턴, 계획). Dragonfly 에 기대지 않는다 |
| `nbtcanon` | 아이템 하나의 NBT 를 늘 같은 바이트로 쓰는 정규 형식(저장망 키와 세이브 키 변환이 함께 쓴다) |
| `nbtrewrite` | 저장 데이터(리틀엔디언 NBT, LevelDB 하위 청크 팔레트)의 글자열을 함수로 바꿈 |
| `invclick` | 바닐라 인벤토리의 클릭 동작(커서 아이템, 칸 누름·끌기·모으기) |
| `netcompress` | 네트워크 계층의 패킷 압축 |
| `internal/slottext` | 칸 폼, WAILA 패널, HUD 줄이 같은 방식으로 만드는 글 조각(`common` 안에서만 import) |

패키지들이 그리는 리소스팩(`inventoryui`, `waila`, `slotdeco`, `sgui`, `oreui`)은 `assets/resourcepack/` 에 있다(`assets` 패키지가 실행 파일에 넣는다). `assets/resourcepack/inventoryui` 는 [tedo0627/InventoryUIResourcePack](https://github.com/tedo0627/InventoryUIResourcePack)(MIT)을, `assets/resourcepack/waila` 는 [r4isen1920/WAILA](https://github.com/r4isen1920/WAILA)(MIT)를 바탕으로 한다.

## 쓰기

- 리소스팩: `assets.Pack(name)` 을 `resourcepack.LoadFS` 에 넘긴다.
- 글: 서버의 카탈로그 패키지가 `init` 에서 `i18n.Register(fsys)` 를 부른다.

## 개발

- `tools/uitextures`, `tools/slotdecotex assets/resourcepack/slotdeco assets/resourcepack/sgui` 가 팩의 임시 그림을 만든다(저장소 루트에서 `go run`).
- 패키지별 설계 문서는 `docs/tech/`(chestui·waila·slotdeco·timed-items·i18n·resourcepack·backup·json-ui·oreui·storagenet·conduits·itemio)에 있다.
