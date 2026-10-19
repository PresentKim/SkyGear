# 03. 기술 설계

## 1. Dragonfly에서의 구현 방식

| 요구사항 | 구현 방식 |
|---|---|
| 로비 / 인스턴스 분리 | 인스턴스마다 별도의 `world.World` (`world.Config{...}.New()`), 세이브별 `mcdb` 폴더 |
| 인스턴스 간 이동 | `tx.RemoveEntity(p)`로 핸들 회수 → 대상 월드 `w.Do(func(tx){ tx.AddEntityAt(handle, pos) })` |
| 인스턴스별 인벤토리·상태 | 기본 `playerdb`는 플레이어당 1벌이라 사용하지 않음. 이동 시 직접 직렬화/복원 |
| 기계 | `world.CustomBlock` + `NBTer`(블록 엔티티) + `TickerBlock`. 모양·텍스처·이름은 리소스팩 파일(`assets/resourcepack/dragonfactory/`), Go는 모델 식별자와 재질 키만 적는다 |
| 아이템 값 | `item.Stack` 값(`WithValue`)에 서랍 내용, 도구 경험·강화 같은 값을 저장 |
| 메뉴 | 폼(`internal/ui`, 리소스팩 JSON UI 테마), 아이템이 오가는 창은 엔티티 상자(`internal/chestui`) |
| 시간 기반 이벤트 | 인스턴스 시간(활성 누적 시간) 기준 스케줄러. 비활성화 시 멈춤 |
| 소유 청크 상시 틱 | 인스턴스별 `world.NewLoader(r, w, world.NopViewer{})`. 로더 반경과 `TickRange`가 소유 청크 전체를 덮도록 설정 ([M0 결과](../tech/m0-verification.md) §4) |
| 분실물 수집 | 인스턴스 월드 핸들러 `HandleEntityDespawn`에서 Y < 월드 최저 높이인 아이템 엔티티 → 분실물 목록에 병합 |
| 낙하 피해 제거 | 플레이어 핸들러 `HandleHurt`에서 `entity.FallDamageSource` 취소 |
| 인벤토리 열람(InvSee) | 엔티티 상자(`internal/chestui`, Dragonfly 패치)로 대상 인벤토리를 표시·동기화. 오프라인 대상은 저장된 상태 편집 |

**검증 결과**: [docs/tech/m0-verification.md](../tech/m0-verification.md). 애드온(리소스팩) 운용: [docs/tech/addon.md](../tech/addon.md).
부하 시험: [performance.md](../tech/performance.md).

## 2. 데이터 저장

| 데이터 | 저장소 |
|---|---|
| 계정, 친구, 업적, 세이브 메타, 세이브 × 플레이어 상태, 세이브 상태 | **SQLite** (`modernc.org/sqlite`, CGO 불필요) |
| 월드 블록 | 세이브별 `mcdb` (LevelDB) 폴더: `saves/<saveID>/world` |

- 테이블(`internal/db/db.go`):

| 테이블 | 내용 |
|---|---|
| `accounts` | 계정(XUID), 이름, 칭호, 이름 색, 안내 확인 버전, 접속 시각 |
| `friends`, `friend_requests`, `blocks` | 친구 관계, 요청, 차단 |
| `achievements` | 계정별 획득한 업적 |
| `saves`, `save_members` | 세이브 메타(게임 종류, 소유자, 슬롯, 이름, 초대 코드, 인스턴스 시간, 일수)와 참여자(역할 `host/cohost/guest`, 탈퇴·추방 뒤에도 행 유지) |
| `save_players` | 세이브 × 플레이어 상태(인벤토리·위치·체력 등 직렬화) |
| `save_state` | 게임 종류별 키-값 상태(연구, 발전과제, 청크, 분실물 …) |

- `save_state`는 게임 종류가 자유롭게 쓰는 키-값이라 플랫폼이 게임 내용을 몰라도 된다.
- 스키마 변경은 `internal/db/db.go`의 `migrations`에 새 항목을 더하는 방식으로 하고, 기존 항목은 고치지 않는다.

## 3. 패키지 구조

```
main.go
internal/db/                   SQLite 접근 계층 (버전별 마이그레이션)
internal/playerstate/          세이브별 플레이어 상태 캡처/복원/직렬화
internal/instance/             인스턴스 매니저, Game/Session 인터페이스, 기본 플레이어 핸들러, 인스턴스 시계
internal/platform/             접속 흐름, 로비, 서버 안내, 세이브 슬롯 UI, 친구·초대, 칭호·치장 메뉴
internal/account/              업적 등록·지급(게임이 호출), 칭호·이름 색 표시
internal/games/dragonfactory/  드래곤팩토리: research, advancement, machine, sieve, hammer, drawer, bundle(도관), storage, tinker, wave, monster, magic, expand, achievements, help …
internal/energy, itemio, itemnet, fluidnet/  에너지·아이템·액체 연결망과 아이템 입출력 인터페이스
internal/chestui, ui, waila, hud/  엔티티 상자, 폼 헬퍼, 바라보기 패널, HUD
internal/i18n/                 문구 카탈로그(`locales/ko.json`)
internal/resourcepack/         리소스팩 로드(내용 해시로 버전 부여)
```

게임 종류 인터페이스(`internal/instance/game.go`):

```go
type Game interface {
    ID() string                         // "dragonfactory"
    Name() string                       // "드래곤팩토리"
    Init(tx *world.Tx, inst *Instance)  // 새 세이브의 월드를 처음 한 번 만든다
    Start(inst *Instance) Session       // 인스턴스 활성화 시 게임 로직 시작
}

type Session interface {
    Spawn() mgl64.Vec3                          // 위치 기록이 없는 플레이어의 시작 위치
    Join(p *player.Player, first bool)          // 입장 후(상태 복원 뒤). first = 이 세이브 첫 입장
    Leave(p *player.Player)                     // 퇴장 전(상태 저장 전)
    Handler(p *player.Player) player.Handler    // 인스턴스 안에서의 게임 이벤트 핸들러
    NewDay(tx *world.Tx, day int)               // 인스턴스 날짜가 바뀔 때
    Autosave(tx *world.Tx)                      // 자동 저장 주기마다
    Close(tx *world.Tx)                         // 월드 저장·종료 전
}
```

### 플레이어 이벤트 흐름
모든 플레이어는 접속 내내 `instance.Handler` 하나를 가진다. 이 핸들러는 플랫폼 공통 규칙(낙하 피해 제거, 사망 시 인벤토리 유지, 보이드 복귀, 접속 종료 시 저장)을 처리하고,
나머지 이벤트는 현재 위치의 핸들러(로비 핸들러 또는 `Session.Handler`)로 전달한다.

### 인스턴스 수명
1. 호스트가 입장 → mcdb 월드 열기, 소유 영역을 덮는 NopViewer 로더 생성, 처음이면 `Game.Init`, `Game.Start`, 시계 시작.
2. 입장/퇴장마다 세이브별 플레이어 상태를 DB에 저장·복원. 자동 저장 주기마다 시계와 접속 중인 플레이어 상태 저장.
3. 호스트가 로비로 가거나 접속 종료 → 남은 플레이어를 로비로 이동 → `Session.Close` → 시계 저장 → 월드 저장·종료.
4. 서버 종료 시 모든 인스턴스를 먼저 종료한 뒤 서버를 닫는다.
