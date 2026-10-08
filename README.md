# dragonfactory — 드래곤팩토리

[Dragonfly](https://github.com/df-mc/dragonfly) 기반 Minecraft Bedrock 멀티 게임 섬 서버입니다.
첫 게임은 스카이팩토리/알케미 팩토리 스타일의 공장 만들기 게임 **드래곤팩토리**입니다.

- 기획: [docs/design](docs/design/README.md)
- 기술 검증·애드온 운용: [docs/tech](docs/tech)

## 실행

```sh
go run .
```

처음 실행하면 설정 파일이 생성됩니다.

| 파일 | 내용 |
|---|---|
| `config.toml` | Dragonfly 서버 설정(포트, 서버 이름, 인증 등) |
| `platform.toml` | 플랫폼 설정(데이터 폴더, 인스턴스 하루 길이, 로드 반경, 동시 인스턴스 수, 자동 저장 주기) |

데이터는 `data/` 아래에 저장됩니다: `data/platform.db`(SQLite), `data/saves/<세이브 ID>/world`(세이브별 월드).

## 현재 구현 (M1: 플랫폼 뼈대)

- 첫 접속 시 서버 안내, 로비(나침반 = 세이브 선택, 책 = 안내)
- 계정당 세이브 슬롯 3개: 새 게임 / 입장 / 이름 변경 / 삭제(2회 확인)
- 세이브마다 별도 월드(인스턴스). 호스트가 들어오면 활성화, 나가면 종료
- 세이브별 인벤토리·체력·허기·경험치·효과·위치 저장과 복원
- 인스턴스 시간과 경과 일수(호스트가 있을 때만 흐름), 자동 저장
- 낙하 피해 없음, 사망 시 인벤토리 유지
- 명령어: `/lobby`(`/spawn`), `/saves`, `/guide`

## 구조

```
main.go                         서버 부트스트랩, 종료 시 인스턴스 저장
internal/db/                    SQLite (계정, 세이브, 참여자, 세이브별 플레이어 상태, 게임 상태)
internal/playerstate/           플레이어 상태 캡처/복원/직렬화
internal/instance/              인스턴스 매니저(활성화/종료, 로비↔인스턴스 이동, 시계, 기본 핸들러)
internal/platform/              로비, 서버 안내, 세이브 선택 UI, 명령어
internal/games/dragonfactory/   드래곤팩토리 (M1: 시작 섬만)
internal/oneblock, internal/farm  원블럭 프로토타입에서 남긴 참고 데이터(현재 미사용)
spike/                          기술 검증 코드와 테스트 클라이언트
assets/resourcepack/            수동 리소스팩 원본
```

## 테스트

```sh
go test ./...
# 실제 접속 흐름 확인(서버를 인증 끔으로 실행한 뒤)
go run ./spike/cmd/platformbot -name Tester 'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=테스트' 'expect:입장했습니다'
```
