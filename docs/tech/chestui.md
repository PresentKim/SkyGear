# 엔티티 상자 (chestui)

아이템이 실제로 오가는 가상 창고는 `common/chestui` 의 **엔티티 상자**로 띄운다.
[CustomSizedInvMenu](https://github.com/Muqsit/CustomSizedInvMenu) 와 같은 방식이다.
Dragonfly 를 패치하지 않고 `common/dfhook` 으로 세션에 닿는다([bedrock-gophers/inv](https://github.com/bedrock-gophers/inv) 의 `entityinv` 와 같은 기술):

- `dfhook.Wrap` 이 서버 리스너를 감싸, 연결마다 클라이언트→서버·서버→클라이언트 패킷이 훅을 먼저 지난다(바꾸기·버리기 가능). 연결이 끊기면 닫힘 훅이 돈다.
- 세션의 비공개 메서드(`writePacket`, `sendInv`, `nextWindowID`, `closeCurrentContainer`)는 `go:linkname`, 비공개 필드(`openedWindow` 등)는 리플렉션으로 쓴다. 이름이 바뀌면 `dfhook` 의 `TestInternals` 가 깨진다.

## 동작

1. 보이지 않는 클라이언트 전용 엔티티(`dragonfly:entity_container`)를 `AddActor` 로 보낸다.
   메타데이터: 투명·NoAI, `ContainerType`=INVENTORY(-1), `ContainerSize`=칸 수, 이름=창 제목.
   식별자는 나가는 `AvailableActorIdentifiers` 패킷에 훅으로 덧붙인다(엔티티 정의 파일 불필요).
2. 바로 `ContainerOpen`(`ContainerTypeContainer` + `ContainerEntityUniqueID`)과 내용물을 보낸다. **기본값이며 1차 실기에서 동작한 방식.**
3. 엔티티 상자가 열려 있는 동안 클라이언트가 보내는 255 창 닫기(폼이 닫히거나 블록 클릭과 겹칠 때)는 무시하고,
   같은 인벤토리를 다시 열면 무시한다(블록 연타 대비).
4. **상자 → 상자 전환**: 창이 닫힌 지 1초 안에 새 창을 열면, 클라이언트가 닫기를 처리했는지 `NetworkStackLatency` 왕복으로
   확인한 뒤 엔티티를 만들고 연다(InvMenu 와 같은 원리. 닫히는 중의 열기는 클라이언트가 무시함).
5. 열린 동안 플레이어가 움직이면 엔티티도 플레이어 위치로 옮긴다(떨어지거나 밀려 멀어지면 클라이언트가 창을 닫음).

`platform.toml` `[ChestUI]` 로 [Muqsit/InvMenu](https://github.com/Muqsit/InvMenu) `PlayerWindowDispatcher` 의 단계를 하나씩 켤 수 있다.
2차 실기에서 셋을 모두 켠 상태로 "모든 상자가 바로 닫힘" 이 나와 하나씩 켜 보며 확인한다. `Ride` 는 5차 실기에서 정상이라 기본으로 켰다.

| 설정 | 동작 |
|---|---|
| `Ride` | 엔티티를 플레이어에 탑승 링크로 붙인다. **기본 켜짐**(5차 실기: 이동해도 안 닫힘) |
| `WaitLatency` | `AddActor` 뒤 `NetworkStackLatency` 응답을 받고 연다(최대 1초) |
| `Retry` | 클라이언트가 초과 열기에 대한 `PacketViolationWarning`(type -1)으로 확인할 때까지 매 틱 닫기+열기 재전송(최대 20회). 그동안 그 창의 닫기는 무시 |
| `Debug` | 열기·닫기·클라이언트 닫기·위반 경고를 INFO 로그로 남긴다 |

6. 리소스팩 `common/packs/inventoryui` (tedo0627/InventoryUIResourcePack, MIT)가
   제목 앞의 `§{줄 수 1~6}§{스크롤 0/1}` + `§r` 9개를 읽어 창 크기를 정한다.
   54칸을 넘으면 6줄 높이로 스크롤한다. 접두어 없는 일반 상자는 바닐라 화면 그대로.
7. 세션의 열린 창(`openedWindow`)을 그 인벤토리로 두므로 아이템 이동은 Dragonfly 의 `ContainerLevelEntity → openedWindow` 경로를 그대로 탄다.
   창이 닫히면 엔티티를 지우고 닫힘 콜백을 부른다. 서버가 닫으면(`chestui.Close`, 다른 창 열기) 그 트랜잭션에서,
   Dragonfly 가 닫으면(클라이언트 닫기 응답 `ContainerClose` 를 훅이 봄) 플레이어 트랜잭션(`Do`)에서, 접속이 끊기면 닫힘 훅에서 플레이어 없이 부른다.

테스트 봇(`spike/cmd/platformbot`)은 클라이언트처럼 `NetworkStackLatency` 에 응답하고, 이미 열린 창의 `ContainerOpen` 에
`PacketViolationWarning` 을 보낸다. 실제 클라이언트의 정확한 동작은 실기로 확인한다.

리소스팩은 필수(`ResourcesRequired`)로 보낸다. 팩이 없으면 27칸 상자로만 보인다.

같은 팩의 `ui/df_quick.json` 은 모든 창고 창(바닐라 상자·큰 상자·엔더 상자·셜커 상자·통, chestui 세로·스크롤 창, 기계 창) 오른쪽 바깥에 **모두 꺼내기·모두 넣기** 버튼을 세로로 붙인다. 버튼 밑에 칸 수만큼 보이지 않는 버튼이 겹쳐 있고, 누름을 각 칸의 바닐라 빠른 이동(쉬프트 클릭)으로 바꾼다. 그래서 서버에는 평범한 아이템 이동 요청이 칸마다 오고 서버 코드는 없다. 꺼내기는 창고 칸(`$quick_container_size`, 창 표식마다 `chest_screen.json` 이 정함), 넣기는 인벤토리 27칸(`$quick_put_hotbar` 면 핫바까지)을 옮긴다. 세 번째 **있는 것만 넣기**는 넣기와 같되, 창고 앞 54칸 중 어느 칸에 같은 아이템(`#item_id_aux`)이 있는 인벤토리 칸만 옮긴다. JSON UI 에는 반복이 없어 창고 칸마다 이름 붙은 컨트롤(`df_quick_c<n>`)과 인벤토리 칸마다 54개의 비교 바인딩을 common `tools/quickmatch` 가 `ui/df_quick_match.json` 으로 써 둔다. 아이콘은 바닐라 `up_arrow`·`down_arrow`(어둡게 칠함)·`store_filter_icon` 이다.

## 사용

| 타입 | 용도 |
|---|---|
| `chestui.Storage` | 실제 아이템이 오가는 창고. 여러 명이 같이 봐도 변경이 모두에게 보인다. `Inventory().Handle` 로 이동 검증, `OnChange` 로 변경 반영 |

- 버튼 동작은 플레이어의 현재 월드 트랜잭션에서 실행된다. 게임 세션 상태를 만지는 버튼은
  그 세션 월드인지 확인한다(드래곤팩토리 `session.button`).
- 닫힘 콜백의 `p`·`tx` 는 접속이 끊겨 닫힌 경우 nil 이다.
- 열린 창(상자든 블록 컨테이너든)은 `chestui.Close(p)` 로 닫는다. 인스턴스 이동(`instance.Manager` Enter/leave) 전에 닫는다.
- 클라이언트가 보낸 것을 다 처리한 뒤에 할 일은 `dfhook.AfterClientSync`(NetworkStackLatency 왕복, 최대 1초).

## 적용 화면

메뉴(넣고 빼는 것이 없는 화면)는 상자가 아니라 폼이다([json-ui.md](json-ui.md)). 상자는 실제 아이템이 오가는 곳에만 쓴다.


- **인벤토리 보기(InvSee)** — 72칸(8줄, 스크롤): 인벤 27 + 핫바 9 / 갑옷 4 + 보조손 1 / 엔더상자 27.
  같은 세이브 월드에 있는 대상은 실시간(1초마다 + 이동 직전 동기화, 화면과 대상이 다르면 그 이동은 취소),
  그 외에는 저장된 상태를 `LockState` 로 잠그고 바뀔 때마다 DB 에 쓴다. 잠글 수 없으면 읽기 전용.
- **기계 창**([machines](machines.md)): 기계별 레이아웃은 [json-ui.md](json-ui.md).
  섬 확장(확인 창)은 폼으로 이어진다.

## 실기 확인 필요

[TODO](../TODO.md) 의 "엔티티 상자" 항목.
