# content/skygear

스카이기어 게임 자체(import 경로 `github.com/presentkim/skygear/content/skygear/...`). `platform/instance` 의
`Game`·`Session` 을 구현해 플랫폼에 붙고, `common/` 과 `platform/` 을 쓴다. 리소스팩은 `assets/resourcepack/skygear`.

## 하위 패키지

| 패키지 | 내용 |
|---|---|
| `skygear` | 세션에 묶인 코드: 게임·상태·핸들러, 기계·서랍·배럴·도관 블록, 연구, 발전과제 화면, 도감, 섬 확장, 웨이브 … |
| `sgitem` | 세션이 필요 없는 아이템과 단순 블록(조약돌, 가루, 광석, 금속, 메쉬, 망치 …) |
| `naming` | 아이템·블록 이름 키와 이름(카탈로그, 팩 글, vanillakeys.json) |
| `textfmt` | 수·시간 표기, 팩 글꼴 기호 |
| `blockmodel` | 커스텀 블록 재질과 큐브 속성 |
| `formicon` | 메뉴 버튼 아이콘 |
| `wintext` | 기계 창 아이템의 글(보는 사람 언어로) |
| `sievetable` | 체 드롭 표와 마법 체 테마 |
| `machinerecipe` | 합금 제련기·정수 추출기·자동 제작기 레시피 |
| `tinker` | Tinkers 식 도구 재료·레벨·경험치·개조 |
| `hopper` | 나무·역방향 호퍼 |
| `advancement` | 발전과제 트리·장·배치 |
| `achievement` | 계정 업적 목록과 ID |
| `nsmigrate` | 이름 변경(DragonFactory → Sky Gear) 전 세이브의 한 번짜리 변환 |

## 문서

- 설계: [docs/design/02-skygear.md](../../docs/design/02-skygear.md) 이후의 `docs/design/*.md`
- 기술: `docs/tech/*.md`
