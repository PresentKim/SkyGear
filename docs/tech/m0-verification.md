# M0 기술 검증 결과

> 대상: Dragonfly v0.11.5 / gophertunnel v1.62.0 / Minecraft Bedrock 1.26.50
> 코드: [`spike/`](../../spike) — `go test ./spike/` 로 재현, `spike/cmd/hopserver`·`hopbot`으로 실제 접속 확인

## 요약

| # | 검증 항목 | 결과 | 근거 |
|---|---|---|---|
| 1 | 월드 간 플레이어 이동 | ✅ 가능 | `TestPlayerMovesBetweenWorlds`, hopserver/hopbot 실접속 |
| 2 | 인스턴스별 인벤토리 저장·복원 | ✅ 가능 (직접 구현) | `TestItemStackRoundTrip`, hopserver |
| 3 | 커스텀 블록 + 블록 엔티티(NBT) + 매 틱 처리 | ✅ 가능 | `TestMachineTicksWithoutPlayersAndPersists` |
| 4 | 플레이어 없이 공장(블록 엔티티) 가동 | ✅ 가능 (로더 필요) | 위 테스트, `TestTickRangeLimitsMachines` |
| 5 | 보이드로 떨어진 아이템 감지 | ✅ 가능 | `TestVoidItemsAreDetectable` |
| 6 | 커스텀 블록 리소스팩 자동 생성·전송 | ✅ 가능 | hopbot에서 자동 생성 팩 다운로드 확인 |

## 1. 월드 간 이동

- 인스턴스 월드는 `world.Config{...}.New()`로 서버 기본 월드와 별개로 만들 수 있다.
- 이동: 현재 월드 트랜잭션에서 `h := tx.RemoveEntity(p)` → 대상 월드에서 `w.Do(func(tx){ tx.AddEntityAt(h, pos) })`.
- 실제 클라이언트로 로비 ↔ 인스턴스를 3회 왕복했고 연결 끊김 없이 청크를 다시 받는다(차원 변경 패킷 없이 동일 차원 내 전환).
- **인벤토리·체력 등은 엔티티 핸들과 함께 이동한다.** 따라서 인스턴스별 인벤토리는 이동 시점에 직접 저장(비우기) → 복원해야 한다.

## 2. 인벤토리 직렬화

- `item.WriteNBT(stack, true)` / `item.ReadNBT(m, nil)`(공개 API)로 아이템을 NBT 맵으로 변환할 수 있다.
  이름, 인챈트, 내구도, **커스텀 값(`WithValue`)**, 커스텀 아이템까지 그대로 왕복된다.
- 서랍 내용, 도구 경험 같은 데이터는 `WithValue`로 아이템에 싣는다.
- 저장 형식: 슬롯별 NBT를 리틀엔디언 NBT 바이트로 인코딩해 DB에 저장(Dragonfly `playerdb`와 같은 방식).
- 플레이어 상태 전체는 `player.Config`(체력, 허기, 경험치, 효과, 갑옷, 보조손, 엔더상자 등)를 기준으로 저장 항목을 정한다.

## 3. 커스텀 블록 · 블록 엔티티 · 틱

- 커스텀 블록 = `world.CustomBlock`(+ 리소스팩 자동 생성용 `CustomBlockBuildable`) 구현 후 `world.RegisterBlock`, `world.RegisterItem`.
- `NBTer`(`EncodeNBT`/`DecodeNBT`)를 구현하면 블록 엔티티로 저장되고, `TickerBlock`(`Tick`)을 구현하면 **매 틱** 호출된다.
- 블록 엔티티 데이터는 mcdb(LevelDB) 월드에 저장되고, 월드를 닫았다 다시 열어도 유지된다.
- 블록 해시: 타입 기본값은 `block.NextHash()`, 상태 해시에는 블록 엔티티 데이터를 넣지 않는다.

## 4. 플레이어 없이 공장 가동 (중요)

- 월드는 **뷰어(로더)가 하나라도 있으면** 틱이 돈다. `world.NewLoader(r, w, world.NopViewer{})`로 플레이어 없는 로더를 만들 수 있다.
- **블록 엔티티 틱 범위 = 로더가 있는 청크 기준 `TickRange` 이내의 로드된 청크.**
  - 비동기(실제) 월드에서는 이 범위 밖의 기계는 로드되어 있어도 틱되지 않는다.
  - 동기(`Synchronous`, 테스트용) 월드는 로드된 모든 청크를 틱한다. **테스트 결과를 실제 동작으로 일반화하지 말 것.**
- 로더 반경 밖 청크는 주기적으로 언로드된다.
- ⇒ 인스턴스 활성화 시: **섬 중심에 로더를 두고, 반경과 `TickRange`를 소유 청크 전체를 덮도록 설정**한다.
  (섬이 커지면 로더 여러 개를 두는 방식도 가능.) 비활성화 시 로더를 닫는다.
- 무작위 틱(작물 성장 등)도 같은 범위 규칙을 따른다.

## 5. 보이드 아이템 감지

- Dragonfly는 아이템 엔티티가 월드 최저 높이 아래로 떨어지면 엔티티를 닫는다.
- 이때 월드 핸들러의 `HandleEntityDespawn(tx, e)`가 호출되며, 엔티티 위치가 `tx.Range()[0]` 아래에 있다.
  플레이어가 줍거나 일반 디스폰된 경우는 위치가 그보다 위이므로 **보이드 소실만 정확히 구분**할 수 있다.
- 아이템은 `e.(*entity.Ent).Behaviour().(*entity.ItemBehaviour).Item()`으로 얻는다.

## 6. 리소스팩 자동 생성

- 서버 설정 `Resources.AutoBuildPack = true`이면 Dragonfly가 커스텀 블록·아이템의 텍스처, 지오메트리, 언어 파일로 리소스팩을 만들어 접속 시 전송한다(테스트에서 8KB 팩 다운로드 확인).
- 애드온 전반의 구현 방법은 [addon.md](addon.md) 참고.

## 설계에 반영할 점

1. 인스턴스 = 별도 `world.World` + mcdb 폴더. 활성화 시 NopViewer 로더로 소유 청크 전체를 덮는다.
2. 인스턴스 이동 시 인벤토리·상태를 직접 저장/복원한다(엔티티 핸들은 그대로 옮긴다).
3. 테스트는 동기 월드로 쓰되, 틱 범위처럼 동작이 다른 부분은 비동기 월드로 검증한다.
4. 분실물 보관함은 인스턴스 월드 핸들러의 `HandleEntityDespawn`에서 수집한다.

## 남은 확인 사항 (M1 이후)

- 커스텀 엔티티(드론 등)를 리소스팩 클라이언트 정의와 함께 실제 클라이언트에 표시하는 것.
- 로드된 인스턴스 수·소유 청크 수에 따른 메모리/CPU 측정(동시 인스턴스 상한, 섬 확장 가격 근거).
- Xbox 인증을 켠 상태의 실제 Bedrock 클라이언트 플레이 테스트(이 개발 환경은 인증 서버 접근이 막혀 있음).
