# content/skygear

스카이기어 게임 자체(import 경로 `github.com/presentkim/skygear/content/skygear/...`). `platform/instance` 의
`Game`·`Session` 을 구현해 플랫폼에 붙고, `common/` 과 `platform/` 을 쓴다. 리소스팩은 `assets/resourcepack/skygear`.

## 구조

기능마다 패키지가 하나다. 패키지는 아래 표에서 **자기보다 위 칸**의 패키지만 import 한다(같은 칸끼리는 표의 순서대로).
위 칸이 아래 칸을 불러야 할 때(메뉴 열기, 사이드바 갱신, 다른 기능의 기계 종류)는 위 칸이 함수 변수(훅)를 두고,
아래 칸이 `init` 에서 채운다(예: `core.ShowMenu` 는 `screens` 가, `machine.Kinds[KindGate].Click` 는 `wave` 가).

| 층 | 패키지 | 내용 |
|---|---|---|
| 데이터 | `locales` | 게임 글 카탈로그(`<언어>[.<부분>].json`, 바닐라 이름 `<언어>.vanilla.json`) |
| | `naming` | 아이템·블록 이름 키와 이름(카탈로그, 팩 글, `vanillakeys.json`) |
| | `textfmt` | 수·시간 표기, 색 코드 지우기, 팩 글꼴 기호 |
| | `formicon` | 메뉴 버튼 아이콘 |
| | `blockmodel` | 커스텀 블록 재질과 큐브 속성 |
| | `sgitem` | 세션이 필요 없는 아이템과 단순 블록(조약돌, 가루, 광석, 금속, 메쉬, 망치 …)과 그 등록 |
| | `sievetable` | 체 드롭 표와 마법 체 테마 |
| | `wintext` | 기계 창 아이템의 글(보는 사람 언어로) |
| | `machinerecipe` | 합금 제련기·실험실·자동 제작기 레시피 |
| | `advancement` | 발전과제 트리·장·배치 |
| | `hopper` | 나무·역방향 호퍼 |
| | `nsmigrate` | 이름 변경(DragonFactory → Sky Gear) 전 세이브의 한 번짜리 변환 |
| 핵심 | `core` | 세이브 하나의 실행 상태(`Session`: 월드·시계·난수·세이브 정보)와 기능이 자기 데이터를 두는 방법(`Part`·`Players`·`Saved`), 공용 도우미(아이템, 메뉴 버튼, 디버그 도형) |
| | `network` | 세이브의 에너지·아이템·유체 연결망 |
| | `achievement` | 계정 업적 목록·등록과 횟수 집계(`Count` 등) |
| 진행 | `progress` | 연구·발전과제 규칙, 레시피 잠금, 연구대 블록 |
| | `island` | 소유 청크, 시작 위치, 청크 구매, 공허 복귀, 분실물 보관함 |
| | `climate` | 날씨·해·성장 시계 |
| 기능 | `tinker` | 팅커 도구(재료·레벨·개조), 팅커 테이블, 광맥 채굴, 자동 도구 |
| | `decor` | 아이템 칸 장식(희귀도·단계 표시)과 툴팁 이름 |
| | `machine` | 기계 블록과 종류(`Kinds`), 창, 에너지·유체, 단계, 체, 가공 기계, 생산 기록과 정지 알림 |
| | `conduit` | 도관 묶음 블록, 연결망 잇기, 표시 엔티티, 연결판과 그 화면 |
| | `storage` | 서랍과 아이템 거치대, 저장고 컨트롤러와 목록 화면 |
| | `farm` | 작물, 묘목·나무, 잎, 퇴비통, 배럴과 마녀물, 수확기, 마법 식물 |
| | `fairy` | 선물 요정, 요정 판 주문, 모래시계, 요정 뼛가루 |
| | `wave` | 차원문 웨이브와 몬스터 |
| | `handwork` | 시작의 손작업(막대기로 조약돌·섬유·씨앗, 상자 정리) |
| | `expand` | 섬 확장 모드(위에서 내려다보는 화면) |
| | `copier` | 설정 복사기 |
| 내용 목록 | `catalog` | 연구 트리, 게임 아이템 레시피, 크리에이티브 목록(여러 기능의 아이템을 이름으로 나열하는 데이터) |
| 화면 | `codex` | 도감(JEI 같은 레시피 보기) |
| | `look` | WAILA 줄(바라보는 블록 정보) |
| | `screens` | 게임 메뉴, 섬 관리, 분실물, 도움말, 연구·발전과제 화면, 사이드바와 퀘스트 판 |
| 게임 | `skygear` | 위 패키지를 잇는 곳: `Game`·`session`(플랫폼 인터페이스 구현), 이벤트 핸들러, 매초·빠른 틱의 차례(`second.go`), 명령어, 운영자 도구(`admin.go`, `debug*.go`) |

## 기능의 데이터

기능은 `session` 에 필드를 더하지 않고 `core` 의 파트를 패키지 변수로 둔다.

- `core.NewPart`: 세이브가 도는 동안의 데이터(예: `network` 의 연결망, `conduit` 의 표시 엔티티).
- `core.NewPlayers`: 플레이어별 데이터. 플레이어가 나가면 지워진다(예: `tinker` 의 광맥 미리보기).
- `core.NewSaved`: 저장되는 데이터. 모든 `Saved` 파트는 세이브 상태 문서(`skygear.state`) 하나의 필드들이고,
  필드 이름은 파트끼리 겹치지 않는다(겹치면 시작할 때 패닉). 그래서 기능을 옮겨도 세이브 형식은 그대로다.

## 문서

- 설계: [docs/design/02-skygear.md](../../docs/design/02-skygear.md) 이후의 `docs/design/*.md`
- 기술: `docs/tech/*.md`
