# 엔티티 상자 (chestui)

아이템이 실제로 오가는 가상 창고는 `internal/chestui` 의 **엔티티 상자**로 띄운다.
[CustomSizedInvMenu](https://github.com/Muqsit/CustomSizedInvMenu) 와 같은 방식이다.

## 동작

1. 보이지 않는 클라이언트 전용 엔티티(`dragonfly:entity_container`)를 `AddActor` 로 보낸다.
   메타데이터: 투명·NoAI, `ContainerType`=INVENTORY(-1), `ContainerSize`=칸 수, 이름=창 제목.
   식별자는 `AvailableActorIdentifiers` 에 등록(엔티티 정의 파일 불필요).
2. 바로 `ContainerOpen`(`ContainerTypeContainer` + `ContainerEntityUniqueID`)과 내용물을 보낸다. **기본값이며 1차 실기에서 동작한 방식.**
3. 엔티티 상자가 열려 있는 동안 클라이언트가 보내는 255 창 닫기(폼이 닫히거나 블록 클릭과 겹칠 때)는 무시하고,
   같은 인벤토리를 다시 열면 무시한다(블록 연타 대비).
4. **상자 → 상자 전환**: 창이 닫힌 지 1초 안에 새 창을 열면, 클라이언트가 닫기를 처리했는지 `NetworkStackLatency` 왕복으로
   확인한 뒤 엔티티를 만들고 연다(InvMenu 와 같은 원리. 닫히는 중의 열기는 클라이언트가 무시함).
5. 열린 동안 플레이어가 움직이면 엔티티도 플레이어 위치로 옮긴다(떨어지거나 밀려 멀어지면 클라이언트가 창을 닫음).

`platform.toml` `[ChestUI]` 로 [Muqsit/InvMenu](https://github.com/Muqsit/InvMenu) `PlayerWindowDispatcher` 의 단계를 하나씩 켤 수 있다.
2차 실기에서 셋을 모두 켠 상태로 "모든 상자가 바로 닫힘" 이 나와 기본은 꺼 두었다. 어느 것이 원인인지는 실기로 가린다.

| 설정 | 동작 |
|---|---|
| `Ride` | 엔티티를 플레이어에 탑승 링크로 붙인다 |
| `WaitLatency` | `AddActor` 뒤 `NetworkStackLatency` 응답을 받고 연다(최대 1초) |
| `Retry` | 클라이언트가 초과 열기에 대한 `PacketViolationWarning`(type -1)으로 확인할 때까지 매 틱 닫기+열기 재전송(최대 20회). 그동안 그 창의 닫기는 무시 |
| `Debug` | 열기·닫기·클라이언트 닫기·위반 경고를 INFO 로그로 남긴다 |

6. 리소스팩 `assets/resourcepack/inventoryui` (tedo0627/InventoryUIResourcePack, MIT)가
   제목 앞의 `§{줄 수 1~6}§{스크롤 0/1}` + `§r` 9개를 읽어 창 크기를 정한다.
   54칸을 넘으면 6줄 높이로 스크롤한다. 접두어 없는 일반 상자는 바닐라 화면 그대로.
7. 아이템 이동은 Dragonfly 의 `ContainerLevelEntity → openedWindow` 경로를 그대로 탄다.
   창이 닫히면(클라이언트·다른 창 열기·월드 이동·접속 종료) 엔티티를 지우고 닫힘 콜백을 부른다.

테스트 봇(`spike/cmd/platformbot`)은 클라이언트처럼 `NetworkStackLatency` 에 응답하고, 이미 열린 창의 `ContainerOpen` 에
`PacketViolationWarning` 을 보낸다. 실제 클라이언트의 정확한 동작은 실기로 확인한다.

Dragonfly 에 이 기능이 없어 `third_party/dragonfly` 에 패치했다([third_party/README.md](../../third_party/README.md)).
리소스팩은 필수(`ResourcesRequired`)로 보낸다. 팩이 없으면 27칸 상자로만 보인다.

## 사용

| 타입 | 용도 |
|---|---|
| `chestui.Storage` | 실제 아이템이 오가는 창고. 여러 명이 같이 봐도 변경이 모두에게 보인다. `Inventory().Handle` 로 이동 검증, `OnChange` 로 변경 반영 |

- 버튼 동작은 플레이어의 현재 월드 트랜잭션에서 실행된다. 게임 세션 상태를 만지는 버튼은
  그 세션 월드인지 확인한다(드래곤팩토리 `session.button`).
- 인스턴스 이동(`instance.Manager` Enter/leave) 전에 열린 창을 닫는다.

## 적용 화면

메뉴(넣고 빼는 것이 없는 화면)는 상자가 아니라 폼이다([json-ui.md](json-ui.md)). 상자는 실제 아이템이 오가는 곳에만 쓴다.


- **인벤토리 보기(InvSee)** — 72칸(8줄, 스크롤): 인벤 27 + 핫바 9 / 갑옷 4 + 보조손 1 / 엔더상자 27.
  같은 세이브 월드에 있는 대상은 실시간(1초마다 + 이동 직전 동기화, 화면과 대상이 다르면 그 이동은 취소),
  그 외에는 저장된 상태를 `LockState` 로 잠그고 바뀔 때마다 DB 에 쓴다. 잠글 수 없으면 읽기 전용.
- **기계 창**([machines](machines.md)): 기계별 레이아웃은 [json-ui.md](json-ui.md).
  수표 발행(입력), 미션·코어·섬 확장(확인 창)은 폼으로 이어진다.

## 실기 확인 필요

[TODO](../TODO.md) 의 "엔티티 상자" 항목.
