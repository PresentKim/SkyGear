# common

[Dragonfly](https://github.com/df-mc/dragonfly) 기반 서버들(dragonfactory, 마인팜 …)이 함께 쓰는 Go 패키지와 리소스팩.
게임 규칙·플랫폼 규칙(로비, 세이브, 계정 저장)은 담지 않고, 어느 서버에서나 같은 동작만 둔다.

## 패키지

| 패키지 | 내용 |
|---|---|
| `dfhook` | Dragonfly 를 패치하지 않고 세션에 닿는 훅(패킷 가로채기, `go:linkname`, 리플렉션): 카메라·HUD·입력 잠금·가짜 블록·제목·폼 응답 정리 |
| `chestui` | 아이템이 실제로 오가는 창(엔티티 상자). `packs/inventoryui` 와 짝 |
| `ui` | 폼 헬퍼(메뉴·카드·목록·확인·토스트). 모양은 `packs/dfui` 의 JSON UI 테마(`ui/df_form.json`)가 그린다 |
| `hud` | 사이드바 점수판으로 보내는 HUD 글. `packs/dfui` 의 `ui/scoreboards.json` 이 그린다 |
| `waila` | 바라보는 블록·엔티티 패널. `packs/waila` 와 짝 |
| `slotdeco` | 아이템 칸 장식 표식. `packs/slotdeco` 와 짝 |
| `i18n` | 플레이어 언어별 글 카탈로그. 서버는 `i18n.Register` 로 자기 카탈로그를 더한다. `i18n/i18ntest` 는 서버가 자기 코드에 돌리는 카탈로그 검사 |
| `timed` | 기간제 아이템(현실 시간 만료) |
| `perf` | 월드 틱 부하 표시 |
| `playerstate` | 플레이어 상태(인벤토리·위치·체력 등) 캡처·복원·직렬화 |
| `resourcepack` | 리소스팩 읽기(내용 해시로 버전 부여, 압축·암호화) |
| `backup` | 폴더 백업 |
| `textsearch` | 한글 초성 등 글 검색 |
| `packs` | 위 패키지들의 리소스팩(`inventoryui`, `waila`, `slotdeco`, `dfui`)을 실행 파일에 넣은 것 |

`packs/inventoryui` 는 [tedo0627/InventoryUIResourcePack](https://github.com/tedo0627/InventoryUIResourcePack)(MIT)을 바탕으로 한다.

## 서버에서 쓰기

```go
// go.mod
require github.com/presentkim/dragonfactory/common v0.0.0-...
replace github.com/df-mc/dragonfly => ./third_party/dragonfly // 서버 레포의 사본
```

- Dragonfly 는 패치본을 쓴다([third_party/README.md](third_party/README.md)). 서버 레포도 같은 사본을 `third_party/dragonfly` 에 두고 `replace` 한다.
- 비공개 레포이므로 받는 쪽에 `GOPRIVATE=github.com/presentkim/*` 가 필요하다.
- 여러 레포를 함께 고칠 때는 상위 폴더에 `go.work` 를 두고 `use ./common ./dragonfactory …` 와 Dragonfly `replace` 를 적는다(모듈마다 `replace` 가 달라 `go.work` 에 하나로 정해야 한다).
- 리소스팩: `packs.Sub(name)` 을 `resourcepack.LoadFS` 에 넘긴다.
- `dfui` 테마는 서버 팩이 채우는 자리(`ui/dfui_hooks.json`, 네임스페이스 `dfui_hooks`)를 둔다. 로비 화면 틀(`lobby_frame`)처럼 게임마다 다른 것은 서버 팩이 같은 경로의 `ui/dfui_hooks.json` 에 다시 정의하고, 서버 팩을 `dfui` 보다 먼저(위에) 싣는다(`conf.Resources` 의 앞쪽이 위).
- 글: 서버의 카탈로그 패키지가 `init` 에서 `i18n.Register(fsys)` 를 부른다.

## 개발

- `tools/uitextures`, `tools/treetex`, `tools/slotdecotex packs/slotdeco packs/dfui` 가 팩의 임시 그림을 만든다(저장소 루트에서 `go run`).
- 커밋 전 `go vet ./...`, `go test ./...`, `gofmt -l .`(third_party 제외)가 깨끗해야 한다.
- 패키지별 설계 문서는 아직 dragonfactory 의 `docs/tech/`(chestui·waila·slotdeco·timed-items·i18n·resourcepack·backup·json-ui)에 있다.
