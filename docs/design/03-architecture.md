# 03. 기술 설계

## 1. Dragonfly에서의 구현 방식

| 요구사항 | 구현 방식 |
|---|---|
| 로비 / 인스턴스 분리 | 인스턴스마다 별도의 `world.World` (`world.Config{...}.New()`), 세이브별 `mcdb` 폴더 |
| 인스턴스 간 이동 | `tx.RemoveEntity(p)`로 핸들 회수 → 대상 월드 `w.Do(func(tx){ tx.AddEntityAt(handle, pos) })` |
| 인스턴스별 인벤토리·상태 | 기본 `playerdb`는 플레이어당 1벌이라 사용하지 않음. 이동 시 직접 직렬화/복원 |
| 코어·기계 | `world.CustomBlock` + `NBTer`(블록 엔티티) + `TickerBlock`. 텍스처는 Dragonfly 자동 리소스팩 |
| 코어 아이템 | `item.Stack` NBT 값(`WithValue`)에 `(CoreID, Generation)` 저장 |
| 메뉴 | `form.Menu` / `form.Custom`, 필요 시 상자형 가짜 인벤토리 |
| 시간 기반 이벤트 | 인스턴스 시간(활성 누적 시간) 기준 스케줄러. 비활성화 시 멈춤 |
| 소유 청크 상시 틱 | 인스턴스별 `world.NewLoader(r, w, world.NopViewer{})`. 로더 반경과 `TickRange`가 소유 청크 전체를 덮도록 설정 ([M0 결과](../tech/m0-verification.md) §4) |
| 분실물 수집 | 인스턴스 월드 핸들러 `HandleEntityDespawn`에서 Y < 월드 최저 높이인 아이템 엔티티 → 분실물 목록에 병합 |
| 낙하 피해 제거 | 플레이어 핸들러 `HandleHurt`에서 `entity.FallDamageSource` 취소 |
| 인벤토리 열람(InvSee) | 엔티티 상자(`internal/chestui`, Dragonfly 패치)로 대상 인벤토리를 표시·동기화. 오프라인 대상은 저장된 상태 편집 |

**검증 결과**: [docs/tech/m0-verification.md](../tech/m0-verification.md). 애드온(리소스팩) 운용: [docs/tech/addon.md](../tech/addon.md).
남은 확인: 커스텀 엔티티 표시, 인스턴스·청크 수에 따른 메모리/틱 비용.

## 2. 데이터 저장

| 데이터 | 저장소 |
|---|---|
| 계정, 친구, 세이브 메타, 세이브 × 플레이어 상태, 상점/미션/코어 레코드 | **SQLite** (`modernc.org/sqlite`, CGO 불필요) |
| 월드 블록 | 세이브별 `mcdb` (LevelDB) 폴더: `saves/<saveID>/world` |

- 현재 프로토타입의 단일 JSON 저장소(`internal/store`)는 SQLite로 대체한다.
- 스키마 초안:

```
accounts(xuid PK, name, title, cosmetics_json, guide_version, created_at, last_seen)
friends(a, b, state, created_at)            -- state: pending/accepted/blocked
saves(id PK, game_type, owner_xuid, slot, name, invite_code, instance_time, day, created_at, last_played)
save_members(save_id, xuid, role, joined_at, left_at)   -- role: host/cohost/guest, 탈퇴·추방 후에도 행 유지
save_players(save_id, xuid, money, state_blob)  -- 인벤토리/위치/체력 등 직렬화
save_state(save_id, key, value_json)        -- 게임 종류별 상태(레벨, 상점, 코어, 분실물 …)
cheques(id PK, save_id, amount, issued_by, issued_at, redeemed_by, redeemed_at)
```

- `save_state`는 게임 종류가 자유롭게 쓰는 키-값으로 두어 플랫폼이 게임 내용을 몰라도 되게 한다.

## 3. 패키지 구조

```
main.go
internal/db/                   SQLite 접근 계층 (버전별 마이그레이션)
internal/playerstate/          세이브별 플레이어 상태 캡처/복원/직렬화
internal/instance/             인스턴스 매니저, Game/Session 인터페이스, 기본 플레이어 핸들러, 인스턴스 시계
internal/platform/             접속 흐름, 로비, 서버 안내, 세이브 슬롯 UI (M3: 친구, 초대)
internal/games/dragonfactory/  드래곤팩토리: core, mission, shop, urgent, cheque, machine …
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

## 4. 기존 프로토타입 처리

| 기존 | 처리 |
|---|---|
| `internal/oneblock` 단계 테이블 | 남겨 둠. 코어 블록 테이블의 참고 데이터 |
| `internal/farm` 수확 판정·레벨 | 남겨 둠. 생명 코어 시대/농사 기계에서 재사용 검토 |
| `internal/island`, `internal/store`, `internal/game` | M1에서 제거 (인스턴스·SQLite·플랫폼으로 대체) |
