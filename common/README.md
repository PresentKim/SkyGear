# common

게임과 상관없는 Go 패키지와 리소스팩(import 경로 `github.com/presentkim/skygear/common/...`).
게임 규칙·플랫폼 규칙(로비, 세이브, 계정 저장)은 담지 않고, 어느 게임에서나 같은 동작만 둔다.

## 패키지

| 패키지 | 내용 |
|---|---|
| `sghook` | Dragonfly 를 패치하지 않고 세션에 닿는 훅(패킷 가로채기, `go:linkname`, 리플렉션): 카메라·HUD·입력 잠금·가짜 블록·제목·DDUI 패킷·폼 응답 정리 |
| `chestui` | 아이템이 실제로 오가는 창(엔티티 상자). `packs/inventoryui` 와 짝 |
| `ui` | 메뉴 화면(메뉴·카드·목록·확인·입력·칸 격자·분할·퀘스트 책 등)과 토스트. 화면은 `packs/oreui` 가 그리는 DDUI 화면이고(`oreui`), 그림은 `packs/sgui` 에 있다 |
| `oreui` | 서버 주도 DDUI 화면(레이아웃·값 보내기, 누름 받기). `packs/oreui` 와 짝 |
| `hud` | 사이드바 점수판으로 보내는 HUD 글. `packs/sgui` 의 `ui/scoreboards.json` 이 그린다(`sgui` 는 HUD 의 JSON UI 와 메뉴 화면의 그림) |
| `waila` | 바라보는 블록·엔티티 패널. `packs/waila` 와 짝 |
| `slotdeco` | 아이템 칸 장식 표식. `packs/slotdeco` 와 짝 |
| `i18n` | 플레이어 언어별 글 카탈로그. 서버는 `i18n.Register` 로 자기 카탈로그를 더한다. `i18n/i18ntest` 는 서버가 자기 코드에 돌리는 카탈로그 검사 |
| `timed` | 기간제 아이템(현실 시간 만료) |
| `perf` | 월드 틱 부하 표시 |
| `playerstate` | 플레이어 상태(인벤토리·위치·체력 등) 캡처·복원·직렬화 |
| `resourcepack` | 리소스팩 읽기(내용 해시로 버전 부여, 압축·암호화) |
| `backup` | 폴더 백업 |
| `textsearch` | 한글 초성 등 글 검색 |
| `packs` | 위 패키지들의 리소스팩(`inventoryui`, `waila`, `slotdeco`, `sgui`, `oreui`)을 실행 파일에 넣은 것 |

`packs/inventoryui` 는 [tedo0627/InventoryUIResourcePack](https://github.com/tedo0627/InventoryUIResourcePack)(MIT)을, `packs/waila` 는 [r4isen1920/WAILA](https://github.com/r4isen1920/WAILA)(MIT)를 바탕으로 한다.

## 쓰기

- 리소스팩: `packs.Sub(name)` 을 `resourcepack.LoadFS` 에 넘긴다.
- 글: 서버의 카탈로그 패키지가 `init` 에서 `i18n.Register(fsys)` 를 부른다.

## 개발

- `tools/uitextures`, `tools/slotdecotex packs/slotdeco packs/sgui` 가 팩의 임시 그림을 만든다(`common/` 에서 `go run`).
- 패키지별 설계 문서는 `docs/tech/`(chestui·waila·slotdeco·timed-items·i18n·resourcepack·backup·json-ui·oreui)에 있다.
