# 아이템 입출력 인터페이스 (`internal/itemio`)

블록이 아이템을 주고받는 **하나의 길**이다. 지금은 호퍼가 쓰고, 나중에 EnderIO 를 참고한 자원 이동 파이프가 같은 길을 쓴다.
블록은 호퍼 같은 특정 이동 블록의 종류를 알 필요 없이 이 인터페이스만 구현하면 된다.

## 인터페이스

```go
type Inserter interface {
    InsertItems(tx, pos, face, st) int            // 받은 개수(0이면 거절)
}
type Extractor interface {
    PeekItems(tx, pos, face, want) item.Stack     // 꺼낼 수 있는 첫 묶음(꺼내지 않음)
    ExtractItems(tx, pos, face, st, n) item.Stack // st 와 같은 종류를 n 개까지 꺼냄
}
type Adapter func(tx, pos, b) (Inserter, Extractor) // 메서드를 못 붙이는 블록(바닐라 퇴비통 등)
```

- `pos` 는 주거나 받는 블록, `face` 는 그 **블록의 어느 면**으로 아이템이 지나가는지다. 위에서 들어오면 `cube.FaceUp`, 아래로 나가면 `cube.FaceDown`.
- 함수: `Insert`, `Peek`, `Extract`, `Transfer(tx, from, fromFace, to, toFace, max, want)`.
  `Transfer` 는 받는 쪽이 거절한 종류를 건너뛰고 다음 종류를 시도한다(바닐라 호퍼처럼).
- `itemio.IO(tx, pos)` 가 블록의 `Inserter`·`Extractor` 를 찾는다: 블록 자신의 메서드 → 등록된 어댑터 → 바닐라 규칙.

## 바닐라 블록 (어댑터 내장)

| 블록 | 넣기 | 꺼내기 |
|---|---|---|
| 상자·통·셜커·호퍼 등 `block.Container` | 아무 면, 인벤토리에 | 아무 면, 아무 칸 |
| 화로·용광로·훈연기 | 바닐라 호퍼 규칙(위 = 재료 칸, 옆 = 연료 칸) | **아래 면만**: 결과 칸(+연료 칸의 연료 아닌 것) |
| 양조대 | 바닐라 호퍼 규칙 | 아래 면만: 병 세 칸 |
| 그 밖의 `block.HopperInsertable`(단지·레코드판 등) | 바닐라 규칙(임시 호퍼를 통해) | 안 함 |

바닐라 규칙을 쓰는 블록에는 아이템 하나를 든 임시 호퍼를 만들어 그 블록의 `InsertItem` 을 부른다(그 블록들은 호퍼가 어느 쪽을 보는지로 칸을 고른다).

## 우리 블록

| 블록 | 넣기 | 꺼내기 |
|---|---|---|
| 기계(분쇄기·자동 체·채굴기·수확기, `machine.go`) | **아래 면을 뺀** 모든 면(손 체는 메쉬가 있고 비었을 때 한 개씩) | **아래 면**: 결과 칸 |
| 서랍(`drawer.go`) | 모든 면(잠긴 서랍·다른 종류는 거절) | 아래 면 |
| 퇴비통(`composter.go`, 어댑터) | 모든 면: 식물 아이템 | 아래 면: 가득 찼을 때 흙 1개 |
| 배럴(`barrel.go`) | 모든 면: 식물 아이템 | 없음(물은 양동이) |
| 나무 호퍼·역방향 호퍼(`hopper.go`) | 호퍼 규칙 | 호퍼 규칙 |

기계의 출력(자동 채굴기·수확기가 옆에 밀어 넣기, 자동 체의 부산물)도 이 길로 이웃 블록에 넣는다(`pushOutput`, `storeBesideOrBelow`).

## 호퍼

- 바닐라 호퍼는 Dragonfly 패치의 `block.HopperTransfer` 훅(`dragonfly.patch`, `third_party/README.md`)으로 **같은 길**을 탄다: 한 걸음마다 자기 칸의 아이템 하나를 바라보는 블록에 넣고(`Insert`), 위 블록에서 하나를 꺼낸다(`Peek` → `Extract`). 쿨타임은 바닐라 8틱.
- 이동 한 걸음은 `stepHopper`(`itemlink.go`) 하나이고 우리 호퍼들도 이것을 쓴다. 방향만 다르다: 바닐라·나무 호퍼는 `out = 바라보는 면`, `in = 위`; 역방향 호퍼는 `out = 위`, `in = 아래`.

## 새 블록을 붙이려면

1. 블록 타입에 `Inserter`·`Extractor` 를 구현한다(또는 어댑터를 등록한다: `registerItemIO`).
2. 면 규칙을 정한다(어느 면이 입력이고 어느 면이 출력인가). 이웃 블록이 누구든 상관없이 동작해야 한다.
3. 상태가 세이브에 있어야 하면 값 갱신 뒤 블록 표시도 갱신한다(서랍 `setDrawer`, 배럴 `showBarrel`).
4. 테스트: 호퍼 위·아래에 두고 틱을 돌려 확인한다(`TestHoppersFeedComposterAndBarrel`, `TestHandsFreeLine`).
