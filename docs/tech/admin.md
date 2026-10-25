# 운영자 도구 (`/dfadmin`, `/dfdebug`)

`platform.toml` 의 `[Admin] Operators` 에 있는 플레이어(XUID, 숫자)만 보고 쓸 수 있다. 이름을 적어도 되지만 Xbox 닉네임은 다른 사람에게 넘어갈 수 있어, 이름으로만 운영자가 된 플레이어가 접속하면 서버 로그에 XUID 와 함께 경고가 남는다. 인증을 끈 시험 서버에서는 XUID 가 비어 있어 이름으로만 판별한다. 세이브 안에서 쓴다(`tps` 는 로비에서도).

운영자는 들어가 있는 어느 세이브에서든 호스트처럼 참여자를 관리한다: `/members`, 또는 플레이어를 웅크린 채 우클릭(인벤토리 보기·추방, [01](../design/01-platform.md) 참여자 관리).

## 관리 (`/dfadmin`)

서버를 운영하며 쓰는 명령. 코드: `internal/games/dragonfactory/admin.go`.

| 명령 | 하는 일 |
|---|---|
| `/dfadmin give <ID> [개수]` | 게임 아이템·기계·도구(`research_table`, `drawer`, `drawer_key`, `barrel`, `hammer`, `sieve`, `auto_sieve`, `crusher`, `auto_miner`, `harvester`, `dimension_gate`, `storage_controller`, `tinker_table`, `ingot_<금속>`, `essence_<몹>`, `seed_<씨앗>`, `petal_<색>` 등. 자동 완성에 전체 목록이 나온다) |
| `/dfadmin item <이름> [개수]` | 아무 아이템(`iron_ore`, `minecraft:diamond` 등) |
| `/dfadmin research <연구 ID\|all\|reset\|free>` | 연구 하나·전부 완료, 연구 모두 지우기, `free`는 모든 조합법을 인스턴스가 닫힐 때까지 열기(다시 하면 잠금) |
| `/dfadmin advance <발전과제 ID\|all\|reset>` | 발전과제 하나(앞 단계 포함)·전부 달성, 모두 지우기 |
| `/dfadmin tps` | 내 HUD 왼쪽 위 서버 부하 줄 켜기·끄기(아래, 로비에서도) |

## 디버그 (`/dfdebug`)

시나리오와 실기 확인에 쓰는 명령: 일을 지금 일으키거나 상태를 출력한다. 코드: `debug.go`, `debug_ui.go`, `debug_factory.go`, `debug_oreui.go`.

| 명령 | 하는 일 |
|---|---|
| `/dfdebug oreinput` | Ore UI 페이지가 받는 입력을 모두 보이는 화면. 마우스 버튼(`button`, `buttons`, `detail`), 더블 클릭, 우클릭 메뉴, 휠, 키(`key`, `code`, `keyCode`), 터치, 누른 시간, 수정키를 화면에 줄로 보이고 서버 로그(`ore ui probe`)에 남긴다. 페이지가 부를 수 있는 엔진 명령 그룹 목록도 보인다 ([oreui.md](oreui.md) "인벤토리 시험") |
| `/dfdebug oreengine` | 바닐라 Ore UI 슬롯이 쓰는 엔진 쿼리(`vanillaGameplayContainerItemQuery`·`SizeQuery`)로 핫바·인벤토리·갑옷·보조손·커서 칸을 읽고, 엔진 명령(`vanillaGameplayContainerAPICommands`)을 버튼으로 부른다. 그동안 클라이언트가 보낸 패킷을 화면과 로그(`ore ui probe packet`)에 남긴다 |
| `/dfdebug orescreen <화면> [fill]` | 게임이 아는 DDUI 화면(`minecraft:inventory_screen` 등 번들 레이아웃과 바닐라 루트)을 게임이 직접 그리게 연다. `fill` 이면 먼저 데이터 저장소 `minecraft` / `container` 에 내 인벤토리를 번들 레이아웃이 읽는 모양으로 넣는다. `close` 는 열린 DDUI 화면을 모두 닫는다 |
| `/dfdebug oreinv` | 렌더러로 그린 내 인벤토리(핫바, 인벤토리 27칸, 갑옷, 보조손). 서버가 바닐라처럼 처리한다: 커서 아이템, 왼쪽 집기·놓기·합치기·바꾸기, 오른쪽 반 집기·하나 놓기, 시프트 이동, 더블 클릭 모으기, 끌어서 나누기(왼쪽 고르게, 오른쪽 하나씩), 숫자 키 핫바 교환, Q(Ctrl+Q 묶음) 버리기, 창 밖 누르기 버리기. 맨 위 줄에 서버가 한 일이 보인다. 닫으면 커서 아이템은 인벤토리로 돌아간다 |
| `/dfdebug research` | 연구 화면을 연다(연구대를 쓴 것과 같다, 게임 메뉴에는 연구 카드가 없다). 시나리오용 |
| `/dfdebug ui [기계]` | 기계를 놓지 않고 그 기계 창을 연다(입력·출력 칸에 예시 아이템, 진행도·에너지 절반). 기계 없이 쓰면 창이 있는 기계 목록 폼이 열리고 버튼으로 차례로 본다. 창 모양 실기 확인용(시나리오 `window_preview.sh`) |
| `/dfdebug place <아이템> <dx> <dy> <dz>` | 내 발 위치에서 그만큼 떨어진 칸에 블록을 놓는다. 그 아이템을 든 채 아래 블록 윗면을 누른 것처럼 게임의 놓기 처리를 그대로 거친다(기계 상태·방향). 아이템은 `give` ID 나 바닐라 이름. 실기에서 조준 없이 짓는 용도 |
| `/dfdebug floor <반지름>` | 발밑 높이에 반지름만큼 돌 바닥을 깐다(있는 블록은 그대로). 빈 공간에 실기용 자리를 만든다 |
| `/dfdebug look <yaw> <pitch>` | 시선을 돌리고(yaw 0 남쪽, pitch 90 아래) WAILA 패널을 바로 갱신, 보는 블록 출력 |
| `/dfdebug status` | `researched= chunks= advanced=` 출력(시나리오 확인용) |
| `/dfdebug skip <분>` | 인스턴스 시간을 n분 앞당김(날짜 변경·작물·분실물 만료 확인용) |
| `/dfdebug timed <초>` | 손에 든 아이템을 지금부터 n초(현실 시간) 기간제로. 0 이면 기간 해제 ([기간제 아이템](timed-items.md)) |
| `/dfdebug fairy` | 선물 요정을 지금 나에게 부름 |
| `/dfdebug wave <단계>` | 옆에 차원 게이트를 놓고 그 단계의 웨이브를 바로 시작(아래 단계는 열어 준다) |
| `/dfdebug research random [퍼센트]` | 연구를 무작위로 그만큼 완료(연구 화면의 두 탭 채우기, [research-screen.md](research-screen.md)) |
| `/dfdebug research load <개수>` | 연구 복사본을 개수만큼 만든다(긴 목록 시험, 저장하지 않음, 0 이면 없앰) |
| `/dfdebug factory <개수>`, `/dfdebug dense <개수>` | 부하 시험용 공장을 짓는다(아래) |

## 서버 부하 표시

운영자가 `/dfadmin tps` 로 켜고 끈다(플레이어마다, 서버를 다시 켜면 꺼짐; 로비에서도 됨). 켜면 HUD 왼쪽 위(일차 줄 아래)에 1초마다 한 줄로 보인다(로비에서는 그 줄만). 코드: `common/perf`(`Line`, `Toggle`), `platform/load.go`, `dragonfactory/sidebar.go`.

`TPS 20.0 · 부하 12% · 틱 6.1ms/9.7ms · 메모리 76MB`

| 항목 | 뜻 |
|---|---|
| TPS | 지금 있는 월드의 최근 5초 초당 틱 수(정상 20). 19 아래 노랑, 15 아래 빨강 |
| 부하 | 평균 틱 시간 ÷ 50ms. 50% 아래 초록, 100% 아래 노랑, 넘으면 빨강(틱이 밀려 TPS 가 떨어짐) |
| 틱 | 최근 5초 평균/최대 틱 시간(그 월드의 다른 작업에 밀려 기다린 시간 포함). 읽는 법은 [performance.md](performance.md) |
| 메모리 | Go 힙 메모리 |

## 부하 시험

`/dfdebug factory <개수>`(1~100): 운영자를 가운데로 공장 모듈(채굴기 줄·체 줄·농장과 수확기)을 지어 부하를 본다(`/dfadmin tps` 로 켜고). 결과와 Dragonfly 스레딩은 [performance.md](performance.md).

`/dfdebug dense <개수>`(1~4000): 운영자를 가운데로 2×1 칸짜리 밀집 셀(자동 자원 채굴기, 옆 분쇄기 → 호퍼 → 서랍)을 빽빽하게 짓는다. 기계를 많이 둘 때의 부하를 게임 안에서 잰다(결과는 [performance.md](performance.md)).
