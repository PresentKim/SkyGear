# 텍스처 출처와 라이선스

직접 그리지 않은 텍스처는 라이선스가 허락하는 오픈소스 모드의 것을 쓴다. 가져오는 일은 `tools/modtex` 가 하고(어느 그림을 어디에 쓰는지는 그 파일의 `entries`), 라이선스 전문은 `third_party/licenses/` 에 둔다. 작업물(PNG)은 팩에 이미 들어 있으므로 다시 돌릴 필요는 없다.

## 쓰는 모드

저장소를 `git clone --depth 1` 로 받은 시점 기준이다. 모두 저장소 자체의 라이선스가 텍스처까지 덮는지 README·LICENSE 로 확인했다.

| 모드 | 저장소 | 확인한 커밋 | 라이선스 | 쓴 곳 |
|---|---|---|---|---|
| Mekanism | https://github.com/mekanism/Mekanism | `90bb3de4c7` | MIT (README: 모드 전체) | 합금(인퓨즈·강화·원자), 제어 회로 I~IV, 강화 키트(티어 인스톨러), 설정 복사기, 주괴·가루·덩어리(주석·납·청동·강철·구리·금·철), 전기 화로·전기 도가니·유체 탱크 면 |
| Ender IO | https://github.com/Team-EnderIO/EnderIO | `a25cd60d65` | Unlicense, 외부 기여 그림은 CC0(`docs/license/`) | 합금로·자동 제작기·몬스터 시뮬레이터·광석 세척기·정수 추출기·발전기류·흡수기·벌목기·에너지 셀 면, 도관 접합제, 도관 텍스처(기존) |
| Tinkers' Construct | https://github.com/SlimeKnights/TinkersConstruct | `f4e77d43f3` | MIT (README: 코드와 텍스처 모두) | 장비 제작대 |
| Storage Drawers | https://github.com/jaquadro/StorageDrawers | `62a33ab1aa` | MIT | 스토리지 컨트롤러 |
| Refined Storage 2 | https://github.com/refinedmods/refinedstorage2 | `d33a173017` | MIT | 스토리지 링크 |
| Ex Nihilo Fabrico | https://github.com/SirLyle/ExNihiloFabrico (기본 브랜치 `1.14.4-Fabric`) | `5224a14b7b` | MIT (`LICENSE`: 원작 MikeLydeamore, 이식·텍스처 기여자 포함) | 체 세 종류의 모델(다리·틀, 메쉬 막대 여덟 개, 내용물 높이)과 메쉬 텍스처(블록 `sieve_mesh*`, 아이템 `mesh_*`). `tools/fabricotex` 가 모델 JSON(`models/block/sieve.json`, `models/item/mesh.json`)에서 만든다 |

주석: 은·니켈·인바·일렉트럼 주괴·가루·덩어리는 Mekanism의 회색 바탕 그림(`ingot`, `dust`, `clump`)에 색만 곱해 만든 파생물이다.

주석: Ex Nihilo Fabrico 에는 메쉬 그림 파일이 없다(메쉬 모델이 바닐라 흰 양털에 메쉬마다 색을 입혀 그린다). 그래서 메쉬 모델의 막대 배치를 위에서 본 16×16 흰 바탕 그림(`sieve_mesh`)으로 옮기고 Fabrico 가 메쉬마다 주는 색(실 `FFFFFF`, 부싯돌 `555555`, 철 `777777`, 금 `FFAA00`, 다이아몬드 `00AAAA`)을 곱했다. 네더라이트 메쉬는 Fabrico 에 없어 같은 바탕에 우리 색(`524248`)을 곱했다. 체 틀은 Fabrico 처럼 바닐라 판자를 경로로 가리킨다.

## 쓰지 않는 모드

| 모드 | 이유 |
|---|---|
| Ex Nihilo(Sequentia 등, Fabrico 외의 이식판) | 그림 라이선스가 All Rights Reserved(저장소 README). MIT 인 Ex Nihilo Fabrico 만 위 표대로 쓴다 |
| Create | 에셋 All Rights Reserved |
| Thermal 시리즈 | 저장소 라이선스 없음(All Rights Reserved) |
| Applied Energistics 2 | 텍스처 CC BY-NC-SA 3.0(상업 이용 금지) |
| Sophisticated Storage | All Rights Reserved |
| Botania, Immersive Engineering | 에셋 조건이 단순 허용형이 아니어서 쓰지 않음 |
| Industrial Foregoing | 저장소는 MIT 이지만 그림이 외부 텍스처 팩(Unity)의 것이라 그 팩의 허락이 확인되지 않음 |

그래서 통(배럴)처럼 그런 모드의 요소는 가져오지 않고 바닐라 텍스처를 경로로 가리킨다(아래 "바닐라 텍스처"). 체의 모델과 메쉬는 Ex Nihilo Fabrico 의 것이다.

## 가져온 UI

| 저장소 | 확인한 커밋 | 라이선스 | 쓴 곳 |
|---|---|---|---|
| https://github.com/tedo0627/InventoryUIResourcePack | `394fb0a935` | MIT (`third_party/licenses/InventoryUIResourcePack-MIT.txt`) | `common/packs/inventoryui/ui/` 상자 화면 JSON UI |
| https://github.com/r4isen1920/WAILA | | MIT (`third_party/licenses/WAILA-MIT.txt`) | `common/packs/waila/` WAILA 패널 JSON UI·테두리 |

## 바닐라 텍스처

마인크래프트 바닐라 텍스처는 팩에 복사하지 않는다. 클라이언트가 가진 그림을 경로로 가리킨다.

- 드래곤팩토리 아틀라스(`textures/terrain_texture.json`)의 일부 키는 `textures/blocks/<바닐라>` 를 가리킨다: 체·나무 깔때기·참나무 통(`planks_oak`), 철 깔때기(`hopper_outside`), 자동 체(`planks_big_oak`), 마법 체(`huge_fungus/crimson_planks`), 연구대 옆면(`bookshelf`), 차원 게이트 면(`portal_placeholder`). `pack_test.go` 의 `vanillaTerrain` 이 이 목록이다.
- WAILA 팩의 도구 아이콘·하트·갑옷·물방울·테두리는 `textures/items/*`, `textures/ui/*` 바닐라 경로를 쓴다(`common/packs/waila/ui/waila/tile.json` 의 `block_tags`, `common.json` 의 `icon_renderer`).

## 생성한 그림

몹 정수(`essence_*`), 꽃잎(`petal_*`), 씨앗(`seed_*`), 장비 제작대 도구(`tool_*`), 나무 양동이(`wooden_*bucket`), 마법 식물(`blocks/plant_*`), 가루 블록(`blocks/dust`)은 deep-pixels.com 의 픽셀아트 모델로 만든 16×16 그림이다(아이템·식물은 `ultimate-extreme-turbo-lv2-opiniated-item-16px-v1`, 가루는 `ultimate-extreme-mc-block-hq-16-v1`). 모델이 한 요청에 내는 네 안 중 하나를 골라 배경을 지우고, 식물은 블록 바닥에 닿게 가운데 맞췄다. PNG 가 원본이므로 바꿀 때는 그림을 직접 갈아 끼운다.

## 손대지 않은 그림

직접 그리거나 고친 텍스처(커밋 제목이 `Texture:` 인 것 등)는 건드리지 않는다. `modtex` 의 `entries` 에 없는 파일이 그렇다.

## 다시 만들 때

```
git clone --depth 1 <저장소> <모드 이름>      # tools/modtex 의 repos 이름대로
go run ./tools/modtex <클론 폴더> assets/resourcepack/dragonfactory
go run ./tools/animtex assets/resourcepack/dragonfactory          # 작동 띠
rm assets/resourcepack/dragonfactory/textures/blocks/*_t[123].png
go run ./tools/tierblocks assets/resourcepack/dragonfactory       # 단계 텍스처
go run ./tools/fabricotex <ExNihiloFabrico 클론> assets/resourcepack/dragonfactory  # 체 모델·메쉬
```

저작권 표시는 MIT 조건에 따라 `third_party/licenses/` 의 파일과 이 문서가 맡는다.
