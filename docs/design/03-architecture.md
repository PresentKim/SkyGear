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
| 소유 청크 상시 틱 | 인스턴스별 `world.Loader`로 소유 청크 전체를 로드 상태로 유지 |
| 분실물 수집 | 아이템 엔티티의 보이드 낙하 감지 → 세이브 분실물 목록에 병합 |
| 낙하 피해 제거 | 플레이어 핸들러 `HandleHurt`에서 `entity.FallDamageSource` 취소 |
| 인벤토리 열람(InvSee) | 온라인: 대상 플레이어 인벤토리를 상자형 UI로 열기 / 오프라인: 저장된 상태 편집 |

**검증 필요(스파이크)**
- 커스텀 블록 + 블록 엔티티 + 틱이 함께 동작하는지, 리소스팩 자동 생성 결과.
- 플레이어의 월드 간 이동 시 인벤토리/효과/게임모드 처리.
- 로드된 월드 수·소유 청크 수에 따른 메모리/틱 비용(동시 인스턴스 상한, 섬 확장 가격의 근거).
- 아이템 엔티티가 보이드로 떨어져 사라지는 시점을 감지할 수 있는지(분실물 보관함).

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

## 3. 패키지 구조(목표)

```
main.go
internal/platform/      접속 흐름, 로비, 계정, 친구, 초대, 세이브 슬롯 UI
internal/instance/      인스턴스 로드/언로드, 플레이어 이동, 상태 저장/복원
internal/db/            SQLite 접근 계층
internal/games/         게임 종류 인터페이스와 등록
internal/games/dragonfactory/  드래곤팩토리: core, mission, shop, urgent, cheque, machine …
```

게임 종류 인터페이스(초안):

```go
type Game interface {
    ID() string                          // "dragonfactory"
    Name() string                        // "드래곤팩토리"
    NewSave(tx *world.Tx, s *Save)       // 섬/초기 상태 생성
    Load(inst *Instance) Session         // 인스턴스 로드 시 게임 로직 시작
}

type Session interface {
    Join(p *player.Player)               // 플레이어 입장(핸들러 부착 등)
    Leave(p *player.Player)
    Close()                              // 언로드 전 상태 저장
}
```

## 4. 기존 프로토타입 처리

| 기존 | 처리 |
|---|---|
| `internal/oneblock` 단계 테이블 | 코어 블록 테이블의 참고 데이터로 재사용 |
| `internal/farm` 수확 판정·레벨 | 생명 코어 시대/농사 기계에서 재사용 검토 |
| `internal/island` 격자 | 인스턴스마다 월드가 따로라 불필요 → 제거 |
| `internal/store` JSON | SQLite로 대체 |
| `internal/game` | 플랫폼/게임 구조로 재작성 후 제거 |
