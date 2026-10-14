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
| `platform.toml` | 플랫폼 설정(데이터 폴더, 인스턴스 하루 길이, 로드 반경, 동시 인스턴스 수, 자동 저장 주기, 운영자 이름 `Admin.Operators`) |

데이터는 `data/` 아래에 저장됩니다: `data/platform.db`(SQLite), `data/saves/<세이브 ID>/world`(세이브별 월드).

## 현재 구현

### 플랫폼
- 첫 접속 시 서버 안내, 로비(나침반 = 세이브 선택, 책 = 안내)
- 계정당 세이브 슬롯 3개: 새 게임 / 입장 / 이름 변경 / 삭제(2회 확인)
- 세이브마다 별도 월드(인스턴스). 호스트가 들어오면 활성화, 나가면 종료
- 세이브별 인벤토리·체력·허기·경험치·효과·위치 저장과 복원
- 인스턴스 시간과 경과 일수(호스트가 있을 때만 흐름), 자동 저장
- 낙하 피해 없음, 사망 시 인벤토리 유지

### 협동
- 친구: 요청(오프라인 대상 포함)·수락·거절·삭제·차단, 접속 상태 표시(오프라인/로비/플레이 중)
- 초대 코드(6자리, 재발급 가능)와 친구 초대(세이브 안에서), 세이브당 최대 4명
- 참여 중인 세이브 목록(입장·탈퇴), 호스트가 플레이 중일 때만 입장
- 호스트의 참여자 관리: 추방, 인벤토리 보기(엔티티 상자, 접속 중이면 실시간·아니면 저장된 인벤토리)

### 드래곤팩토리
- 3×3×3 하늘섬 + 나무 한 그루, 맨손 시작([기획 05](docs/design/05-sky-progression.md))
- **연구**: 참나무 판자·조합대·연구대 외 모든 조합법이 잠겨 있고, 연구대에서 재료를 한 번에 써서 조합법을 연다. 섬의 모두에게 바로 열린다(Dragonfly 패치 레시피 필터)
- **발전과제**: 다음 할 일을 힌트로 안내하고 사이드바 퀘스트 창에 보여 준다
- **손작업 자원**: 돌멩이, 나무껍질 섬유·실, 망치 분해(조약돌 → 자갈 → 모래 → 먼지), 체와 메쉬 등급, 퇴비통(흙), 배럴(물)과 돌 배럴(용암)
- **자동화**: 호퍼, 에너지(RE)와 발전기, 에너지·아이템·액체 도관, 분쇄기·자동 체·자동 자원 채굴기·자동 농작물 수확기·합금로·광석 세척기 등. 기계 창은 엔티티 상자이고 호퍼·도관으로 넣고 뺀다
- **서랍과 스토리지 컨트롤러**: 한 종류 대용량 보관, 앞면 넣기/꺼내기, 업그레이드·잠금, 여러 서랍을 한 저장소로 묶기
- **차원 게이트 웨이브**: 몬스터 8단계, 정수 드롭, 몬스터 시뮬레이터
- **장비 제작(팅커스풍)과 캐릭터 성장**: 머리·손잡이 조합, 도구 레벨과 강화, 캐릭터 레벨과 스킬
- **마법 식물**: 꽃·광석꽃·보석꽃·정수꽃, 꽃잎 염색
- **섬 확장**: 위에서 내려다보는 지도에서 맞닿은 청크를 조약돌로 구매, 소유 청크 밖 건축 금지
- **보이드 복귀**(가장 가까운 블록 위 높은 곳에서 다시 낙하), **분실물 보관함**(공허로 떨어진 아이템, 하루 보관)
- 선물 요정(가끔 찾아와 원목·묘목 선물)
- HUD(왼쪽 위 일차·플레이 시간, 오른쪽 퀘스트 창)

### 명령어
| 위치 | 명령어 |
|---|---|
| 어디서나 | `/lobby`(`/spawn`), `/guide`, `/friends` |
| 로비 | `/saves`, `/join [코드]` |
| 세이브 안 | `/menu`(`/df`), `/invite`·`/members`(호스트) |
| 운영자 | `/dfadmin give·research·advance·wave·factory·status …` ([docs/tech/admin.md](docs/tech/admin.md), `platform.toml`의 `Admin.Operators`) |

## 구조

```
main.go                         서버 부트스트랩, 종료 시 인스턴스 저장
internal/db/                    SQLite (계정, 세이브, 참여자, 세이브별 플레이어 상태, 게임 상태)
internal/playerstate/           플레이어 상태 캡처/복원/직렬화
internal/instance/              인스턴스 매니저(활성화/종료, 로비↔인스턴스 이동, 시계, 기본 핸들러)
internal/platform/              로비, 서버 안내, 세이브 선택 UI, 친구·초대·참여자, 칭호·치장 메뉴, 명령어
internal/account/               업적 등록·지급, 칭호·이름 색(채팅·이름표 표시)
internal/timed/                 기간제 아이템(현실 시간 만료, 정리)
internal/games/dragonfactory/   드래곤팩토리 (연구, 발전과제, 섬 확장, 서랍, 기계, 도관, 웨이브, 장비, 마법 식물, 도움말 …)
internal/i18n/                  문구 카탈로그
internal/ui/                    폼 헬퍼(버튼 아이콘, 격자 폼)
internal/waila/                 WAILA 패널(바라보는 블록 정보 HUD) 제목 형식·보내기, 도구 아이콘
internal/itemio/                아이템 입출력 인터페이스(Inserter·Extractor·Transfer, 바닐라 어댑터): 호퍼와 도관이 쓴다
internal/energy, itemnet, fluidnet/  에너지·아이템·액체 연결망
internal/perf/                  서버 부하(TPS·틱·CPU·메모리) 측정, 운영자 HUD 줄과 켜기·끄기
internal/hud/                   HUD 글자(왼쪽 위·오른쪽 위·퀘스트 창)를 사이드바 줄로 보내기
internal/chestui/               엔티티 상자(크기 자유 상자 창): 아이템이 오가는 창고
internal/resourcepack/          수동 리소스팩 로더(내용 해시로 버전 자동 부여, JSON minify, 암호화: docs/tech/resourcepack.md)
internal/oneblock, internal/farm  원블럭 프로토타입의 참고 데이터
spike/                          기술 검증 코드와 테스트 클라이언트
assets/resourcepack/            리소스팩 원본 (dragonfactory: 블록·아이템 텍스처·모델·이름, 엔티티, JSON UI 테마 / inventoryui / waila: WAILA HUD)
tools/uitextures/               JSON UI 테마 텍스처 생성기
third_party/dragonfly/          엔티티 상자 패치를 얹은 Dragonfly v0.11.5 (third_party/README.md)
```

## 테스트

```sh
go test ./...
# 실제 접속 흐름(서버 빌드·인증 끔 실행·봇 시나리오·종료까지 자동)
spike/scenarios/run.sh spike/scenarios/advanced.sh
```
