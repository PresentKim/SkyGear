# 텍스처 출처와 라이선스

직접 그리지 않은 텍스처는 라이선스가 허락하는 오픈소스 모드의 것을 쓴다. 가져오는 일은 `tools/modtex` 가 하고(어느 그림을 어디에 쓰는지는 그 파일의 `entries`), 라이선스 전문은 `third_party/licenses/` 에 둔다. 작업물(PNG)은 팩에 이미 들어 있으므로 다시 돌릴 필요는 없다.

## 쓰는 모드

저장소를 `git clone --depth 1` 로 받은 시점 기준이다. 모두 저장소 자체의 라이선스가 텍스처까지 덮는지 README·LICENSE 로 확인했다.

| 모드 | 저장소 | 확인한 커밋 | 라이선스 | 쓴 곳 |
|---|---|---|---|---|
| Mekanism | https://github.com/mekanism/Mekanism | `90bb3de4c7` | MIT (README: 모드 전체) | 합금(인퓨즈·강화·원자), 제어 회로 I~IV, 강화 키트(티어 인스톨러), 설정 복사기, 주괴·가루·덩어리(주석·납·청동·강철·구리·금·철), 전기 화로·전기 도가니·유체 탱크 면 |
| Ender IO | https://github.com/Team-EnderIO/EnderIO | `a25cd60d65` | Unlicense, 외부 기여 그림은 CC0(`docs/license/`) | 합금로·자동 제작기·몬스터 시뮬레이터·광석 세척기·연구소·발전기류·흡수기·벌목기·에너지 셀·차원 게이트 면, 도관 접합제, 도관 텍스처(기존) |
| Tinkers' Construct | https://github.com/SlimeKnights/TinkersConstruct | `f4e77d43f3` | MIT (README: 코드와 텍스처 모두) | 장비 제작대 |
| Storage Drawers | https://github.com/jaquadro/StorageDrawers | `62a33ab1aa` | MIT | 스토리지 컨트롤러 |
| Refined Storage 2 | https://github.com/refinedmods/refinedstorage2 | `d33a173017` | MIT | 스토리지 링크 |

주석: 은·니켈·인바·일렉트럼 주괴·가루·덩어리는 Mekanism의 회색 바탕 그림(`ingot`, `dust`, `clump`)에 색만 곱해 만든 파생물이다.

## 쓰지 않는 모드

| 모드 | 이유 |
|---|---|
| Ex Nihilo(Sequentia 등) | 그림 라이선스가 All Rights Reserved(저장소 README) |
| Create | 에셋 All Rights Reserved |
| Thermal 시리즈 | 저장소 라이선스 없음(All Rights Reserved) |
| Applied Energistics 2 | 텍스처 CC BY-NC-SA 3.0(상업 이용 금지) |
| Sophisticated Storage | All Rights Reserved |
| Botania, Immersive Engineering | 에셋 조건이 단순 허용형이 아니어서 쓰지 않음 |
| Industrial Foregoing | 저장소는 MIT 이지만 그림이 외부 텍스처 팩(Unity)의 것이라 그 팩의 허락이 확인되지 않음 |

그래서 체·메쉬·통(배럴)·마법 식물·꽃잎·정수·나무 양동이처럼 그런 모드의 요소를 따온 텍스처는 가져오지 않았고, 기존 그림 그대로다.

## 손대지 않은 그림

직접 그리거나 고친 텍스처(커밋 제목이 `Texture:` 인 것 등)는 건드리지 않는다. `modtex` 의 `entries` 에 없는 파일이 그렇다.

## 다시 만들 때

```
git clone --depth 1 <저장소> <모드 이름>      # tools/modtex 의 repos 이름대로
go run ./tools/modtex <클론 폴더> assets/resourcepack/dragonfactory
go run ./tools/animtex assets/resourcepack/dragonfactory          # 작동 띠
rm assets/resourcepack/dragonfactory/textures/blocks/*_t[123].png
go run ./tools/tierblocks assets/resourcepack/dragonfactory       # 단계 텍스처
```

저작권 표시는 MIT 조건에 따라 `third_party/licenses/` 의 파일과 이 문서가 맡는다.
