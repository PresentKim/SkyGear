# 엔티티 상자 (chestui)

가상 창고·메뉴 화면은 모두 `internal/chestui` 의 **엔티티 상자**로 띄운다.
[CustomSizedInvMenu](https://github.com/Muqsit/CustomSizedInvMenu) 와 같은 방식이다.

## 동작

1. 보이지 않는 클라이언트 전용 엔티티(`dragonfly:entity_container`)를 `AddActor` 로 보낸다.
   메타데이터: 투명·NoAI, `ContainerType`=INVENTORY(-1), `ContainerSize`=칸 수, 이름=창 제목.
   식별자는 `AvailableActorIdentifiers` 에 등록되어 있다(엔티티 정의 파일은 필요 없음).
2. `ContainerOpen` 을 `ContainerTypeContainer` + `ContainerEntityUniqueID`=그 엔티티로 보낸다.
   블록이 필요 없어 어디서나 열린다.
3. 리소스팩 `assets/resourcepack/inventoryui` (tedo0627/InventoryUIResourcePack, MIT)가
   제목 앞의 `§{줄 수 1~6}§{스크롤 0/1}` + `§r` 9개를 읽어 창 크기를 정한다.
   54칸을 넘으면 6줄 높이로 스크롤한다. 접두어 없는 일반 상자는 바닐라 화면 그대로.
4. 아이템 이동은 Dragonfly 의 `ContainerLevelEntity → openedWindow` 경로를 그대로 탄다.
   창이 닫히면(클라이언트·다른 창 열기·월드 이동·접속 종료) 엔티티를 지우고 닫힘 콜백을 부른다.

Dragonfly 에 이 기능이 없어 `third_party/dragonfly` 에 패치했다([third_party/README.md](../../third_party/README.md)).
리소스팩은 필수(`ResourcesRequired`)로 보낸다. 팩이 없으면 27칸 상자로만 보인다.

## 사용

| 타입 | 용도 |
|---|---|
| `chestui.Storage` | 실제 아이템이 오가는 창고. 여러 명이 같이 봐도 변경이 모두에게 보인다. `Inventory().Handle` 로 이동 검증, `OnChange` 로 변경 반영 |
| `chestui.Menu` | 버튼 메뉴. 아이템을 집으면(클릭) 이동은 취소되고 버튼 동작이 다음 트랜잭션에서 실행된다. 열린 채로 `Set` 하면 바로 갱신 |
| `chestui.CloseThen` | 창을 닫고 잠시 뒤(250ms) 폼 등으로 이어간다. 닫히는 중 보낸 폼은 무시될 수 있어서 |

- 버튼 동작은 플레이어의 현재 월드 트랜잭션에서 실행된다. 게임 세션 상태를 만지는 버튼은
  그 세션 월드인지 확인한다(드래곤팩토리 `session.button`).
- 인스턴스 이동(`instance.Manager` Enter/leave) 전에 열린 창을 닫는다.

## 적용 화면

- **인벤토리 보기(InvSee)** — 72칸(8줄, 스크롤): 인벤 27 + 핫바 9 / 갑옷 4 + 보조손 1 / 엔더상자 27.
  같은 세이브 월드에 있는 대상은 실시간(1초마다 + 이동 직전 동기화, 화면과 대상이 다르면 그 이동은 취소),
  그 외에는 저장된 상태를 `LockState` 로 잠그고 바뀔 때마다 DB 에 쓴다. 잠글 수 없으면 읽기 전용.
- **드래곤팩토리 메뉴**(`/menu`), **상점 구매·판매**, **분실물 보관함**.
  수표 발행(입력), 미션·코어·섬 확장(확인 창)은 폼으로 이어진다.

## 실기 확인 필요

[TODO](../TODO.md) 의 "엔티티 상자" 항목.
