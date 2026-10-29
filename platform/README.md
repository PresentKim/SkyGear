# platform

게임을 둘러싼 서버(import 경로 `github.com/presentkim/skygear/platform/...`). 로비, 계정, 세이브, 친구, 인스턴스를 맡고,
게임은 `instance.Game`·`instance.Session` 인터페이스로만 부른다. 게임 규칙(블록·아이템·기계)은 담지 않는다.
`common/` 만 import 하고 `content/` 는 import 하지 않는다.

## 패키지

| 패키지 | 내용 |
|---|---|
| `platform` | 접속 흐름, 로비 메인 화면, 서버 안내, 세이브 슬롯과 불러오기, 친구·초대·참여자, 알림, 칭호·치장 메뉴, 설정 화면, 운영자 콘솔, 명령어 |
| `account` | 계정의 부가 정보: 업적(게임이 등록·지급), 칭호, 이름 색, 횟수 집계, 계정 설정 |
| `db` | SQLite 저장(계정, 친구, 세이브, 참여자, 세이브별 플레이어 상태, 게임 키-값 상태)과 버전별 마이그레이션 |
| `instance` | 세이브를 인스턴스(세이브마다 월드 하나)로 돌리는 매니저, 로비 ↔ 인스턴스 이동과 플레이어 상태 교체, 인스턴스 시계, 게임 인터페이스 |

## 쓰기

다른 게임 서버는 `instance.Game` 을 구현하고, 명령(`cmd/<게임>/main.go`)에서 `instance.NewManager` 에 넘긴 뒤 `platform.New` 로 플랫폼을 띄운다(예: `cmd/skygear`).

## 문서

- 설계: [docs/design/01-platform.md](../docs/design/01-platform.md), [docs/design/03-architecture.md](../docs/design/03-architecture.md)
- 기술: `docs/tech/`(admin·backup·achievements 등)
