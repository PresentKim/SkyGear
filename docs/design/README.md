# 기획 문서

> 수치(확률, 시간, 비용)는 모두 밸런싱 전 초안이다. 결정이 필요한 항목과 수치 표는 [open-questions.md](open-questions.md)에 모은다.

## 비전

하나의 서버 안에서 여러 종류의 **섬 게임**을 골라 플레이하는 플랫폼.
첫 번째 게임은 스카이팩토리 / ATM10 Sky 스타일의 **공장 만들기** 게임 **드래곤팩토리**다.
게임 종류를 늘릴 수 있는 구조로 만들되, 지금은 사용자에게 게임 선택 단계를 보여 주지 않는다.

## 문서 구성

| 문서 | 내용 |
|---|---|
| [01-platform.md](01-platform.md) | 접속 흐름, 계정, 세이브 슬롯, 인스턴스(호스트 기준 활성화), 친구·초대, 참여자 관리, 기간제 아이템 |
| [02-dragonfactory.md](02-dragonfactory.md) | 드래곤팩토리: 시작 상태, 손작업 자원, 퇴비통·배럴, 체, 연구와 발전과제, 기계, 작물, 섬 영역, 안전 장치, 서랍, HUD, 메뉴 |
| [03-architecture.md](03-architecture.md) | 기술 설계, Dragonfly 구현 방식, 데이터 저장, 패키지 구조 |
| [04-roadmap.md](04-roadmap.md) | 개발 단계와 방향 |
| [05-sky-progression.md](05-sky-progression.md) | 스카이블록 진행 설계(스카이팩토리 3·ATM10 Sky): 손작업 초반, 체 그물, 발전과제 안내, 차원 게이트 웨이브 |
| [06-power-and-conduits.md](06-power-and-conduits.md) | 에너지(RE)·발전기·도관(에너지/아이템/액체, EnderIO 참고) |
| [07-progression-expansion.md](07-progression-expansion.md) | 금속·합금로·에너지 셀·몬스터 시뮬레이터·액체 네트워크·차원 게이트, 연구·발전과제 구성 |
| [08-storage-gear-magic.md](08-storage-gear-magic.md) | 스토리지 컨트롤러·장비 제작(팅커스풍)·캐릭터 성장·마법 식물 |
| [09-automation-equipment.md](09-automation-equipment.md) | 자동화 설비 확장: 전기 화로·자동 제작기·아이템 흡수기·자동 벌목기 |
| [open-questions.md](open-questions.md) | 결정이 필요한 사항과 밸런싱 수치 표 |

## 용어

| 용어 | 정의 |
|---|---|
| 계정(Account) | Xbox XUID 기준 플레이어. 치장·칭호·친구 등 게임과 무관한 데이터를 가진다. |
| 게임 종류(Game Type) | 섬 게임의 종류. 지금은 드래곤팩토리 하나다. |
| 세이브(Save) | 한 게임 종류의 진행 데이터 한 벌. 계정당 3슬롯. |
| 인스턴스(Instance) | 세이브를 서버에 불러온 실행 단위. 호스트가 접속해 있는 동안만 활성화된다. |
| 인스턴스 시간 | 인스턴스가 활성화되어 있던 누적 시간. 게임 내 모든 주기의 기준. |
| 호스트(Host) | 세이브의 소유자. 초대된 플레이어는 게스트. 게스트는 호스트 없이 입장할 수 없다. |
| 연구 | 재료를 내고 조합법을 여는 진행 단위. 세이브의 모두가 공유한다. |
| 발전과제 | 다음에 할 일을 힌트로 안내하는 진행 단위. 세이브의 모두가 공유한다. |
| RE | 기계가 쓰고 발전기가 만드는 에너지 단위(RedstoneEnergy). |
