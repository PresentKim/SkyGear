# Dragonfly 확장

서버는 `third_party/dragonfly` 의 패치본 Dragonfly 위에서 돈다. 패치는 작게 두고, 클라이언트에 보내는 데이터와 세션 훅은 패치 밖에서 더한다.
패치 목록과 업그레이드 절차는 [third_party/README.md](../../third_party/README.md) 에 있다.

## 층

| 층 | 위치 | 하는 일 |
|---|---|---|
| Dragonfly 패치 | `third_party/dragonfly.patch` | Dragonfly 의 동작 자체(블록 규칙, 저장, 틱, 청크 전송, 월드 훅) |
| `sghook` | `common/sghook` | 세션과 패킷에 닿는 훅. 서버가 클라이언트에 보내는 패킷을 가로채 고치고, 세션의 비공개 메서드와 필드를 쓴다. Dragonfly 내부는 여기서만 건드린다 |
| `clientdef` | `common/clientdef` | 클라이언트가 접속 때 받는 아이템·블록 항목에 더하는 컴포넌트. 공개 레지스트리만 쓰고 게임을 모른다 |
| 게임 | `content/skygear` | 아이템과 블록이 `clientdef` 의 인터페이스를 구현한다. `content/skygear/core` 의 `core.EditEntries` 가 훅을 한 번 건다 |

위에서 아래로 시도한다. 패킷을 가로채거나 세션 내부를 읽고 쓰는 것으로 되면 `sghook` 에 둔다. 클라이언트 쪽 컴포넌트만 더하면 되면 `clientdef` 에 둔다.
Dragonfly 의 동작 자체를 바꿔야 하거나, 패치 없이 같은 결과를 내는 것을 이 환경에서 증명할 수 없을 때만 패치한다.

## 패치에 두는 것

- 저장, 청크 캐시, 틱 성능, 월드 트랜잭션의 panic 회복처럼 Dragonfly 안쪽의 일.
- 훅이 없는 곳에 훅을 다는 일(`HopperTransfer`, `BlockChangeHook`, `ColumnHandler`).
- 실제 클라이언트에서만 맞는지 볼 수 있는 일: 가짜 청크(`ViewFakeChunk`), `Activate` 뒤 두 번의 재전송, 커스텀 블록의 서버 쪽 부수기, 잔디 퍼짐 검사. 대체 방법이 맞는지 [docs/TODO.md](../TODO.md) 의 "실기 확인" 에서 본다.
- 같은 결과임을 증명할 수 없는 일: 퇴비통 규칙, 나뭇잎, NetherNet `LocalIP`·`PublicIP`, 레시피 필터, 창작 항목 맞춤, 로딩 중 틱당 청크 수, 값이 바뀔 때만 보내는 날씨 패킷.

패치한 곳에는 `skygear patch` 주석을 달고, 패치를 바꾸면 `third_party/README.md` 의 표를 함께 고친다.

## 아이템에 클라이언트 컴포넌트 더하기

`sghook.EditCustomItems` 가 서버가 보내는 아이템 항목을 고치고, `clientdef.ItemEntry` 가 등록된 커스텀 아이템에서 컴포넌트를 읽어 더한다. 아이템이 인터페이스를 구현하면 된다.

- `clientdef.Digger`: `minecraft:digger` 의 블록별 캐기 속도(예: 망치, 크룩).
- `clientdef.TagDigger`: 도구 태그별 캐기 속도(`TagSpeed`; 팅커 도구).
- `item.Usable`: `minecraft:use_modifiers`. `item.Fuel`: `minecraft:fuel`.
- 아이콘은 업스트림의 `Texture()` 를 구현한다.

## 블록에 클라이언트 컴포넌트 더하기

`sghook.EditCustomBlocks` 가 서버가 보내는 블록 항목을 고치고, `clientdef.BlockEntry` 가 등록된 커스텀 블록에서 아래를 더한다.

- `block.Breakable`: `BreakInfo` 의 도구 태그(`tag:...`)와 곡괭이 등급.
- `block.Activatable`: 클릭을 서버로 보내는 상호작용 표시.
- 속성은 이름 순으로 정렬한다(클라이언트가 상태 번호를 매기는 순서와 맞춘다).
- `clientdef.ExtraProperties` 를 구현하면 `ClientExtras()` 의 충돌 상자(`CollisionBoxes`)와 지오메트리 뼈 보임(`BoneVisibility`)을 기본 컴포넌트와 같은 조건의 퍼뮤테이션에 더한다.
- `clientdef.RenderMethods` 를 구현하면 `ClientRenderMethods()` 가 재질의 렌더 방식 이름을 바꾼다(예: 누에 나뭇잎의 `alpha_test` → `alpha_test_single_sided`).

항목의 모양이 기대와 다르면 `BlockEntry` 가 블록 이름을 담아 panic 한다. `TestWire` 가 같은 길을 타므로 Dragonfly 를 올린 뒤 먼저 여기서 걸린다.

## 클라이언트 데이터 검증

`content/skygear/wire` 의 `TestWire` 는 서버를 `cmd/skygear` 와 같은 방식으로 세워 가짜 클라이언트를 한 번 접속시키고, 받은 블록 항목·아이템 등록·제작 데이터·창작 목록을 항목마다 해시해 `testdata/*.golden` 과 맞춘다. 접속을 흉내 내는 도구는 `common/sghook/hooktest` 다(`Join`, `Canon`, `Golden`).

```sh
go test ./content/skygear/wire/                              # 골든과 비교
SKYGEAR_GOLDEN=update go test ./content/skygear/wire/        # 골든 다시 기록
SKYGEAR_GOLDEN_DUMP=/tmp/a go test ./content/skygear/wire/   # 항목마다 전체 글을 /tmp/a 에 씀
diff -r /tmp/a /tmp/b                                        # 두 트리의 덤프 비교
```

클라이언트 데이터를 옮기거나 패치를 바꾼 변경은 골든이 그대로여야 한다. 클라이언트가 받는 것을 바꾸려는 변경만 골든을 다시 기록하고, 커밋 메시지에 이유를 쓴다.
골든이 바뀌면 불일치한 항목 이름이 나온다. 무엇이 달라졌는지는 두 트리를 덤프해 `diff -r` 로 본다.

## Dragonfly 올리기

절차는 [third_party/README.md](../../third_party/README.md) 의 업그레이드 절차를 따른다. 올린 뒤에는 `TestInternals`(`sghook` 이 이름으로 기대는 내부)와 `TestWire` 를 돌린다.
