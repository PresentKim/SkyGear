# 리소스팩 감사 (2026-11-09, optimize/addon)

dev `8dcbfac` 기준으로 6개 팩을 조사했다. 측정은 서버가 실제로 만드는 방식을 그대로 따랐다(`assets.Pack` → minify → `zipDir`, 상태 프로브 끔).

## 1. 팩별 크기

| 팩 | 전송 zip(암호화 끔) | 비고 |
|---|---:|---|
| skygear (Main) | 635 KB | 압축 전(minify 후) 2,284 KB, 파일 875개 |
| oreui | 88 KB | `oui-app.js` 235 KB(원본) |
| waila | 20 KB | |
| sgui | 8 KB | |
| inventoryui | 7 KB | |
| slotdeco | 5 KB | |

**Main이 2.3 MB로 보이는 이유**: 압축하지 않은 크기가 2.28 MB이다. 클라이언트는 이 크기를 보여 주고, `Encrypt = true`이면 모든 파일이 Store(무압축)로 들어가서 전송량도 이 크기가 된다. 암호화를 끈 상태의 전송량은 635 KB이고, 그중 약 140 KB는 파일 875개의 zip 헤더(로컬 헤더와 중앙 디렉터리)이다.

## 2. Main(2,284 KB, 압축 전) 내역

| 항목 | 크기 | 비율 |
|---|---:|---:|
| `models/blocks/net.geo.json` | 1,476 KB | **64.6%** |
| `models/blocks/{sieve,auto_sieve,magic_sieve}.geo.json` | 54 KB × 3 = 161 KB | 7.1% |
| 블록 텍스처 451개 | 220 KB | 9.6% |
| 아이템 텍스처 281개 | 97 KB | 4.3% |
| 파티클 텍스처 13개 | 84 KB | 3.7% (`fx_blast` 30 KB, `fx_lightning` 21 KB) |
| `terrain_texture.json` | 37 KB | 1.6% |
| `entity/conduit_visual.geo.json` 등 엔티티 모델 | 26 KB | 1.1% |
| 언어 파일 2개 | 43 KB | 1.9% |
| `item_texture.json` | 21 KB | 0.9% |
| 그 외(ui, 렌더 컨트롤러, 파티클 JSON 등) | 약 120 KB | |

### 2.1 net.geo.json: 가장 큰 개선 여지 (약 −1.45 MB)

- `tools/netgeo`가 네트워크 블록의 모양(연결 방향 조합)마다 지오메트리를 하나씩 쓴다. 케이블 64개, 임포터·익스포터·외부 저장소 각 192개로 모두 **640개**이고, 큐브는 3,808개이다.
- 서로 다른 큐브는 **43개뿐**이다. 같은 팔(arm) 큐브가 모양마다 반복된다.
- 개선안: 종류마다 지오메트리 하나에 `core`와 방향별 팔 bone을 두고, 퍼뮤테이션은 `bone_visibility`로 팔을 켜고 끈다. 우리 Dragonfly 패치에 `customblock.Properties.BoneVisibility`가 이미 있다(`content/skygear/stateprobe.go`가 사용). 이렇게 하면 지오메트리 640개가 4개(약 20 KB)로 줄어 **Main 압축 전 크기가 약 2.28 MB → 0.83 MB**가 된다. 바꿔야 할 곳은 `machine.NetGeometry`/`NetShapes`, `tools/netgeo`, 블록 퍼뮤테이션이다. 클라이언트 렌더링·로딩 시간은 실기기에서 확인해야 한다.
- 보조안(구조를 바꾸지 않을 때): 부동소수 잡음(`-2.0500000000000007`)을 반올림하면 42 KB만 준다. 효과가 작다.

### 2.2 체 지오메트리 3벌 중복 (−108 KB)

- `sieve`/`auto_sieve`/`magic_sieve.geo.json`의 bone이 **바이트 단위로 같고** 식별자만 다르다(각 9개).
- 개선안: 자동·마법 체도 `geometry.skygear_sieve_N`을 참조하고 두 파일을 지운다. 더 나아가 9단계(채움 정도)도 큐브 24종을 `bone_visibility`로 바꾸면 파일 하나가 수 KB로 준다.

### 2.3 텍스처 중복 (약 −50 KB, 그리고 파일 수 감소)

- 픽셀이 같은 PNG 그룹이 66개, 중복분이 약 50 KB이다. 대부분 `tools/machinetex` 결과물이다. 기계들의 `*_down*`, `*_up*`, `*_side*`(티어별)가 6벌·4벌씩 같고, `energy_cell`·`essence_collector`·`drawer_controller`·`net_autocrafter`는 면마다 같다. `items/mesh_*`와 `blocks/sieve_mesh_*`도 같다.
- 개선안: `terrain_texture.json` 항목이 공용 경로 하나를 가리키게 하고 중복 PNG를 지운다. 바이트보다 **zip 항목 수**(헤더 약 160 B/개)와 클라이언트 텍스처 로드 수에서 이득이 크다.
- PNG는 이미 최적화되어 있다(`8dcbfac`). 16×16이 721개라 개별 압축 여지는 거의 없다.

### 2.4 기타 소소한 항목

- `entity/conduit_visual.geo.json`: 부동소수 반올림으로 −1.6 KB.
- 언어 파일의 이름에 든 slotdeco 표식(`§k§r§a` 등): 이름마다 6~11 B. 표식이 필요하므로 유지한다.

## 3. 사용하지 않는 요소

### 3.1 이번 브랜치에서 지운 것 (확인 완료)

| 팩 | 항목 | 근거 |
|---|---|---|
| waila | `border_*` 11종(png+json, `border_center` 제외), `arrow.png` | 칸 테두리를 없앤 뒤(11차 플레이테스트) 아무 데서도 참조하지 않는다 |
| waila | `tile.json`의 `$border_type` 지정 40곳과 기본값 1곳 | 위 테두리를 고르던 변수인데 읽는 곳이 없다 |
| sgui | `launcher_button_hover.png`, `launcher_button_pressed.png` | CSS는 `launcher_button.png`만 쓴다. `tools/uitextures`의 생성 줄도 지웠다 |
| skygear | `items/metal_dust_{bronze,steel,invar,electrum}.png` | 합금 가루 아이템은 없고 atlas 항목도 없다. `tools/metaltex`, `tools/modtex`가 다시 만들지 않게 고쳤다 |
| skygear | `terrain_texture.json`의 `skygear_sieve_mesh`와 `blocks/sieve_mesh.png` | 등록된 블록·퍼뮤테이션 어디에서도 쓰지 않는다(`sieve_mesh_string`과 픽셀이 같다) |
| slotdeco | manifest 설명 "Slot backgrounds by item rarity…" | 칸 배경 등급은 `7e51baa8`에서 철회되었다. 현재 기능에 맞게 고쳤다 |

모델: 등록된 블록이 쓰는 지오메트리는 전부 쓰이고 있다(`state_probe_four`는 프로브를 끄면 팩에서 빠진다). 아이템 atlas 항목도 전부 쓰인다.

### 3.2 SlotDeco에 남은 것 (결정 필요)

`7e51baa8`("show item rarity on the tooltip border instead of the slot")은 **칸 뒤 등급 배경만** 지우고 아래는 남겼다.

| 남은 것 | 위치 | 현재 동작 |
|---|---|---|
| 전면 단계 점 `front_1..4` | slotdeco 팩, `decor.ItemDecor`(합금·키트·제어 회로의 tier) | 바닐라 컨테이너 칸에 그린다. Ore UI 칸에는 그리지 않는다 |
| 경고·잠금 배지 `front_5/6` | `machine.go`(정지한 기계의 정보 아이콘) | 바닐라 컨테이너 칸에 그린다 |
| 선택 아이템 말풍선 등급 테두리 `tip_1..4` | slotdeco 팩 `item_panel_image` | 터치 UI에서 그린다 |
| 등급 표식 생성·해석 | `common/slotdeco`, `tools/slotmarks`, 언어 파일 | Ore UI 툴팁 테두리 색에도 쓰므로 **지우면 안 된다** |

"슬롯에 등급 표시 철회"가 단계 점까지 포함하는 뜻이면 slotdeco 팩 전체(전면·말풍선)와 `TierMark`·`tier` 경로, `tools/slotdecotex`, `[ResourcePack] SlotDeco` 설정, `docs/tech/slotdeco.md`를 정리한다. 경고·잠금 배지는 기계 창 Ore UI 쪽에 대체 표시가 있는지 먼저 확인해야 한다.

## 4. 그 밖의 개선 후보

1. **zip 항목 수**: Main 파일이 875개이고 헤더만 약 140 KB이다. 2.3절의 중복 제거와 2.1절(모델)로 줄인다. 기계 티어 텍스처를 플립북이나 아틀라스로 묶는 방법도 있지만 효과에 비해 작업이 크다.
2. **암호화 시 압축 없음**: `zipDir`은 암호화하면 전부 Store로 저장한다(암호문은 압축되지 않음). 암호화를 쓰는 서버에서는 2.1절의 효과가 그대로 전송량 감소(−1.45 MB)로 이어진다.
3. **oreui `oui-app.js` 235 KB**: 이미 minify된 번들이다. 쓰지 않는 의존성이 있는지 번들 분석(esbuild metafile)으로 확인할 만하다. 또한 dev에서 `TestOreUIBundleFresh`가 실패한다(번들이 소스보다 오래됨, `npm run build` 필요). 이번 변경과는 무관하다.
4. **waila `tile.json`**: 상자 칸 54개를 손으로 펼쳐 둔 구조이다. 크기 영향은 작지만(압축 후 수 KB) 템플릿과 `$a_index`만 남았으므로 정리할 수 있다.

## 5. 검증

- `go test ./...`: 이번 변경으로 새로 실패하는 테스트는 없다. dev 원본에서도 실패하는 것: `assets.TestOreUIBundleFresh`(stale 번들), `content/skygear.TestDebugFill`(간헐적, 원본에서도 4회 중 1회 실패).
