# 리소스팩 감사와 용량 개선 (2026-11-09, optimize/addon)

dev `8dcbfac` 기준으로 6개 팩을 조사하고, 쓰이지 않는 요소를 지우고, 용량이 큰 부분을 줄였다. 크기는 서버가 실제로 만드는 팩을 쟀다. `assets.Pack` → `resourcepack.LoadFS`(minify 켬, 상태 프로브 끔)로 만든 다음, 암호화 끔과 켬(`pack.Len()`) 두 가지를 기록했다.

## 결과 요약

| 팩 | 파일 수 | 전송 크기, 암호화 끔 | 전송 크기, 암호화 켬 |
|---|---:|---:|---:|
| skygear (Main) | 874 → **782** | 636 KB → **554 KB** (−13%) | 2,507 KB → **846 KB** (−66%) |
| oreui | 9 → 9 | 88 KB → **81 KB** (−8%) | 318 KB → **291 KB** (−8%) |
| waila | 36 → **13** | 19.9 KB → **11.3 KB** (−43%) | 77 KB → **66 KB** (−15%) |
| sgui | 27 → **25** | 8.2 KB → 7.7 KB | 18.4 KB → 17.6 KB |
| inventoryui | 11 | 7.4 KB (변화 없음) | 31.8 KB |
| slotdeco | 19 | 4.9 KB (설명만 바뀜) | 8.1 KB |
| **합계** | 976 → **859** | 764 KB → **666 KB** (−13%) | 2,961 KB → **1,260 KB** (−57%) |

암호화를 켜면 파일을 모두 Store(무압축)로 넣기 때문에, 압축 전 크기가 그대로 전송량이 된다. 처음 보고된 "Main 2.3 MB"는 이 크기(minify 후 압축 전 2.28 MB, 암호화 시 2.51 MB)이다. 암호화를 끄면 Deflate가 큰 JSON을 이미 잘 줄이고 있었으므로 감소폭이 작다.

### 커밋별 변화 (Main, 암호화 켬 / 끔)

| 커밋 | 내용 | 암호화 켬 | 암호화 끔 |
|---|---|---:|---:|
| `8dcbfac` (dev) | 기준 | 2,507,027 | 635,507 |
| `9b6c926` | 쓰이지 않는 텍스처·변수 제거 | 2,504,356 | 633,333 |
| `bfc330b` | oreui: 이전 렌더러 CSS 제거, CSS minify | (Main 그대로) | (Main 그대로) |
| `f61c617` | 네트워크 블록: 종류당 지오메트리 1개 | **1,059,357** | 612,134 |
| `cfce885` | 체 3종이 지오메트리 1개를 공유 | 907,353 | 608,100 |
| `e286e92` | 같은 그림인 블록 텍스처 합치기 | **845,509** | **554,025** |

## 1. 네트워크 블록 지오메트리 (`f61c617`, Main 암호화 켬 기준 −1.45 MB)

**문제**: `tools/netgeo`가 케이블·입력기·출력기·외부 저장공간의 연결 모양마다 지오메트리를 하나씩 썼다. 640개, 큐브 3,808개인데 서로 다른 큐브는 43개뿐이었다. `net.geo.json` 하나가 1,476 KB로 Main의 64.6%를 차지했다.

**변경**
- `netshape.Parts`: 부품(심 `core`, 면별 팔 `arm_<면>`, 방향별 머리 `head_<면>`)을 뼈대로 정의한다. `netshape.Visible`은 한 모양이 보이는 부품을 고르고, `netshape.Of`(서버 모델·충돌 상자)도 같은 부품에서 만든다.
- 지오메트리는 종류마다 1개(`geometry.skygear_<kind>`)이다. 퍼뮤테이션마다 `bone_visibility`(Dragonfly 패치 `Properties.BoneVisibility`, 상태 프로브에서 이미 쓰는 기능)로 그 모양의 뼈대만 켠다.
- `net.geo.json`: 1,476 KB → **31 KB**. 부동소수 잡음(`4.1000000000000005`)도 1/10000 픽셀 단위로 반올림했다.

**검증**
- 640개 모양 모두에서 "보이는 뼈대의 큐브 합"이 이전 모양별 지오메트리의 큐브와 같다(반올림 1e-4 기준, 불일치 0, 일회성 스크립트로 확인).
- `TestNetPartsDrawEveryShape`: 보이는 뼈대의 상자가 `netBoxes`(서버 충돌)와 같고, `bone_visibility`가 모든 뼈대를 지정하는지 본다.
- `TestPackHasBlockAssets`: `bone_visibility`에 적힌 뼈대가 실제 지오메트리에 있는지도 확인하도록 늘렸다.

**남은 확인**: 실제 클라이언트에서 네트워크 블록 모양, 접속·로딩 시간, 프레임을 확인해야 한다. 상태 프로브(`stateprobe.go`)가 같은 방식(정적 `bone_visibility`)을 실기기에서 써 본 적이 있다.

## 2. 체 지오메트리 공유 (`cfce885`, −152 KB)

**문제**: `sieve`·`auto_sieve`·`magic_sieve.geo.json`이 식별자만 다르고 내용이 같았다. 각 파일에 지오메트리 9개(틀 + 채움 단계별 내용물)가 있었고, 합계 161 KB였다.

**변경**
- 세 종류가 `geometry.skygear_sieve` 하나를 쓴다. 틀·메쉬는 뼈대 `root`, 채움 N의 내용물은 뼈대 `content_N`이고 `bone_visibility`로 하나만 켠다.
- 종류별 차이는 텍스처(`*` 재질)뿐이라 그대로 둔다.
- `tools/fabricotex`가 공유 지오메트리를 쓰도록 바꿨다.
- 파일 3개(161 KB) → 1개(45 KB, minify 후 약 6 KB).

**검증**
- 세 종류 × 9단계 모두 그려지는 큐브가 이전과 같다(불일치 0).
- Ex Nihilo Fabrico를 클론해 `fabricotex`를 다시 돌린 결과가 커밋된 파일과 같다. PNG는 픽셀 기준, JSON은 내용 기준으로 비교했다.

## 3. 같은 그림인 블록 텍스처 합치기 (`e286e92`, 파일 −85개, −62 KB)

**문제**: 기계 텍스처 도구(`machinetex`, `tierblocks`)가 만든 면 가운데 픽셀이 같은 것이 많았다. 대부분의 기계 아랫면·윗면, 무늬 없는 기계의 모든 면, 단계별(`_t1`~`_t3`) 사본이다. 블록 PNG 85개가 사본이었다.

**변경**
- `tools/texdedup`: `terrain_texture.json`에서 같은 그림을 가리키는 항목을 한 파일로 모으고 사본을 지운다. 다른 팩 파일(flipbook, 엔티티)이 이름으로 부르는 파일은 남기고 그 파일을 기준으로 삼는다.
- 아이템 텍스처는 대상에서 뺐다. 폼 아이콘이 경로를 동적으로 만들기도 하기 때문이다(`textures/items/cycle_N`).
- 문서(`asset-credits.md` "다시 만들 때")의 생성 순서에 `texdedup`을 추가했다.

**검증**
- 다시 가리키게 된 85개 항목 모두 이전과 같은 픽셀을 가리킨다(불일치 0).
- `TestPackBlockTexturesUnique`: 텍스처 도구를 다시 돌려 사본이 생기면 실패하고 `texdedup` 실행을 안내한다. 기준 커밋에서는 85건을 잡아낸다.

**주의**: 합친 면 하나만 손으로 다시 그리려면, 그 항목에 다시 고유 파일을 지정해야 한다.

## 4. Ore UI 팩 (`bfc330b`, 암호화 켬 기준 −27 KB)

**분석**
- `oui-app.js`(235 KB)를 esbuild metafile로 분석했다. react-reconciler(production, 91 KB), @react-facet(49 KB), react·scheduler(11 KB), 앱 화면 코드(약 80 KB)로 구성된다. 모두 production 빌드이고 쓰이지 않는 모듈은 없었다.
- `probe` 화면은 `/sgdebug` 입력 프로브라서 남겼다.
- 번들은 소스보다 오래된 상태였다(`TestOreUIBundleFresh` 실패). 새로 빌드해 해결했다. 크기는 같다.

**변경**
- `oui-ddui.css`: 이전 렌더러 `oui-ddui.js`가 쓰던 규칙 75개를 지웠다. 이 렌더러는 React 페이지로 바뀔 때(`93871eb2`) 지워졌지만, CSS(버튼·스위치·막대·슬롯·옛 연구 화면 `rs-*`)는 남아 있었다. 10.5 KB → 3.4 KB.
- `oui-mc.css`: 그 클래스들에 대한 초기화 규칙 3개와, 쓰이지 않는 `mc-huge`·`mc-toast` 규칙 3개를 지웠다.
- 팩 빌더가 `.css`도 JSON처럼 minify한다(`minifyCSS`, 빌더 버전 4). 주석과 공백만 줄인다. 문자열·`url()`은 그대로 두고, `:` 앞 공백(`.a :hover`)도 남긴다. `oui-mc.css`는 58 KB → 41 KB이다. 원본 파일과 주석은 저장소에 그대로 있다.

**검증**
- `web/oreui/visual` 하니스로 27개 화면을 실제 브라우저에서 그려 비교했다. 기존 CSS끼리 두 번 찍으면 0.00%(잡음 없음)이다. 규칙을 지운 뒤, 그리고 minify한 CSS 모두 27개 화면이 **0.00%로 같다**.
- `vitest` 165개 통과. `TestMinifyCSS`를 추가했고, `TestRepositoryPacksMinify`에 oreui를 넣었다.

**보류**
- WAILA `tile.json`: 상자 칸 54개를 손으로 펼쳐 둔 구조이다. JSON UI는 반복을 만들 수 없고 압축 후 몇 KB라서, 실기기 확인 없이 바꾸지 않았다.

## 5. 쓰이지 않는 요소 제거 (`9b6c926`)

| 팩 | 항목 | 근거 |
|---|---|---|
| waila | `border_*` 11종(png+json, `border_center` 제외), `arrow.png` | 칸 테두리를 없앤 뒤(11차 플레이테스트) 아무 데서도 참조하지 않는다 |
| waila | `tile.json`의 `$border_type` 지정 40곳과 기본값 1곳 | 위 테두리를 고르던 변수인데 읽는 곳이 없다 |
| sgui | `launcher_button_hover.png`, `launcher_button_pressed.png` | CSS는 `launcher_button.png`만 쓴다. `tools/uitextures`의 생성 줄도 지웠다 |
| skygear | `items/metal_dust_{bronze,steel,invar,electrum}.png` | 합금 가루 아이템은 없고 atlas 항목도 없다. `tools/metaltex`, `tools/modtex`가 다시 만들지 않게 고쳤다 |
| skygear | `terrain_texture.json`의 `skygear_sieve_mesh`와 `blocks/sieve_mesh.png` | 등록된 블록·퍼뮤테이션 어디에서도 쓰지 않는다. `fabricotex`도 더는 쓰지 않는다 |
| slotdeco | manifest 설명 "Slot backgrounds by item rarity…" | 칸 배경 등급은 `7e51baa8`에서 철회되었다. 현재 기능에 맞게 고쳤다 |

**찾은 방법**
- 팩 PNG마다 경로가 코드·JSON·CSS에 나오는지 검색했다. 동적으로 만드는 경로(`front_` + 번호, `tags/` + 글자 등)는 하나씩 확인했다.
- 등록된 커스텀 블록·아이템·퍼뮤테이션이 쓰는 atlas 항목과 지오메트리를 거꾸로 모아 대조했다. 지오메트리와 아이템 atlas에는 남는 것이 없었다.

**SlotDeco는 유지한다**: 전면 단계 점·경고/잠금 배지(바닐라 컨테이너 칸)와 선택 아이템 말풍선의 등급 테두리는 지금도 쓰인다. 칸 뒤 등급 배경은 `7e51baa8`에서 이미 지워졌다.

## 6. 테스트

- `go test ./...`: 이번 변경으로 실패하는 테스트는 없다.
- `content/skygear.TestDebugFill`: dev 원본에서도 간헐적으로 실패한다(원본에서 4회 중 1회).
- `gofmt -l`은 dev에서도 몇 파일(`common/ui/home.go`, `content/skygear/game.go` 등)을 지적한다. 이번 변경과 무관하다.

## 7. 다음 후보

- 기계 티어 텍스처를 플립북이나 아틀라스로 묶어 파일 수를 더 줄일 수 있다(Main 782개, zip 항목마다 헤더 약 150 B). 작업량이 크다.
- 아이템 텍스처의 같은 그림(`mesh_*`와 `blocks/sieve_mesh_*`, `cycle_9`/`cycle_10` 등, 약 10건): 폼 아이콘 경로를 확인한 뒤 합칠 수 있다.
- 실기기 확인: 네트워크 블록·체 모양, 접속 시 팩 다운로드 시간.
