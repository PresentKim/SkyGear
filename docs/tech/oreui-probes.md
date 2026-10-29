# Ore UI 시험 화면

Ore UI(Gameface) 페이지가 무엇을 받고 무엇을 불러오는지 실제 클라이언트에서 보는 운영자 명령이다([admin.md](admin.md)). 화면은 `common/ui.ProbeScreen` 의 창이고, 렌더러(`oui-ddui.js`)의 시험 노드가 결과를 그린 뒤 줄마다 서버로 보낸다(이벤트 종류 `probe`, 서버 로그 `ore ui probe`). 창 위의 줄은 서버가 받은 것이다: 받은 줄 수와 마지막 줄, 화면이 열린 동안 클라이언트가 보낸 패킷 종류와 수(늘 오는 `PlayerAuthInput`·`NetworkStackLatency`·`SubChunkRequest`·`Animate` 는 뺀다). 코드는 `content/skygear/debug_oreui.go` 다.

표기: **[실측]** 게임에서 확인, **[코드]** 설치본 번들에서 읽음, **[문서]** 외부 문서, **[추론]** 위 근거로 판단, **확인 안 됨** 근거 없음.

## 서버가 보낸 리소스 (`/sgdebug oreres`)

서버가 Ore UI 페이지에 HTML·CSS·JS 를 줄 수 있는 길은 셋이다. 시험 화면은 길마다 방법을 하나씩 차례로 해 보고(방법마다 4초), 줄마다 `ok`·`fail`·사유를 그려 서버로 보낸다. 스크립트는 `window.__ouiRes(태그)` 를 부르면, 스타일시트는 `.oui-res-probe` 의 폭을 13px 로 만들면, HTML 은 `oui-res-html` 이 들어 있으면 통과다. 맨 앞 줄은 페이지 주소(`location.href`, `origin`)와 `fetch`·`XMLHttpRequest`·`Blob`·`URL.createObjectURL` 이 있는지다.

Windows 클라이언트에서 본 결과(2026-10-27) [실측]:

- 페이지 주소는 `ui://./hbui/gameplay.html`, `origin` 은 `ui://.` 이다. `fetch` 는 없고(`undefined`), `XMLHttpRequest`·`Blob`·`URL.createObjectURL` 은 있다.

| 길 | 방법 | 결과 |
|---|---|---|
| 패킷(데이터 저장소 글) | `eval`(전역) | 된다 |
| | `new Function` | 된다 |
| | 인라인 `<script>`(글을 `textContent` 로) | 된다(`onload` 도 온다) |
| | `data:` 스크립트 | 안 된다(`onerror`) |
| | `blob:` 스크립트 | `new Blob([글], {type})` 이 `TypeError`(인자 둘을 받는 생성자 없음). 인자 하나로 바꾼 시험은 확인 안 됨 |
| | `<style>` | 확인 안 됨(첫 시험이 스타일을 붙이자마자 쟀다. Gameface 는 다음 프레임에 배치하므로 지금은 0.3초 뒤에 잰다) |
| | `data:` 스타일시트 | 안 된다(`onerror`) |
| | `innerHTML` | 된다 |
| 리소스팩(암호화, `hbui/oui-probe.*`) | `<script src="/hbui/…">` | 된다 |
| | `<link href="/hbui/…">` | `onload` 는 온다. 스타일이 들었는지는 `<style>` 과 같은 까닭으로 확인 안 됨 |
| | `fetch` | 없다 |
| | `XMLHttpRequest` `/hbui/oui-probe.html` | 된다(200) |
| | `XMLHttpRequest` `pack://hbui/oui-probe.html` | 된다(200) |
| HTTP(접속한 호스트의 게임 포트, 다음 포트) | `<script src>`, `<link>`, `<img>` | 안 된다(`onerror`) |
| | `XMLHttpRequest` | 안 된다(상태 404, 0바이트) |
| | `fetch` | 없다 |

- HTTP 는 두 포트 모두 서버에 요청이 하나도 오지 않았다(창 위에 `http` 줄이 없다). XHR 의 404 는 서버가 아니라 게임의 리소스 처리기가 낸 것이다. 클라이언트는 페이지에서 http 주소로 네트워크 요청을 보내지 않는다. "Ore UI 는 접속한 서버와 같은 출처의 리소스를 허용한다"는 말은 이 결과와 맞지 않고, 출처도 찾지 못했다. 페이지의 출처(`ui://.`)는 서버 호스트와 다르다. 앞서 인터넷 주소의 `<img>`·`<video>` 도 모두 실패했다(Windows 1.26.52).
- 정리: 서버가 쓸 수 있는 길은 둘이다. (1) 암호화 팩의 `hbui/` 파일을 `<script src>`·XHR(`/hbui/…` 나 `pack://…`)로 불러오기, (2) 데이터 저장소로 보낸 글을 `eval`·`new Function`·인라인 `<script>`·`innerHTML` 로 쓰기. (2)는 팩을 바꾸지 않고 접속 중에 코드를 보낼 수 있다. 스타일시트를 나중에 붙이는 것은 다시 확인한다.
- Gameface 문서: `XMLHttpRequest` 는 있고 요청은 게임의 리소스 처리기로 간다 [문서](https://docs.coherent-labs.com/cpp-gameface/api_reference/classes/interface_x_m_l_http_request/), `fetch` 는 Window API 목록에 없다 [문서](https://docs.coherent-labs.com/cpp-gameface/api_reference/classes/interface_window/).

- HTTP 서버는 화면이 열려 있는 동안만 돈다. 받은 요청(포트, 경로, `Origin`·`Referer`·`User-Agent` 헤더)은 창 위의 `http` 줄과 서버 로그(`ore ui probe http`)에 남는다. 요청이 서버에 닿았는데 페이지에서 실패하면 네트워크가 아니라 페이지 쪽(출처 규칙, 처리기)이 막은 것이고, 요청이 오지 않으면 클라이언트가 보내지도 않은 것이다. 응답에는 `Access-Control-Allow-Origin: *` 를 붙인다.
- 포트를 열지 못하면(다른 프로그램이 쓰는 중) 그 사유가 `http` 줄에 보인다. 게임 포트는 UDP 라서 같은 번호의 TCP 는 보통 비어 있다.
- `https` 와 웹소켓은 시험하지 않는다(서버에 인증서가 없다).

## 입력 이벤트 (`/sgdebug oreinput`)

- 화면에는 입력칸 하나와 이벤트 목록이 있다. 페이지(`document`, 캡처 단계)의 마우스(`mousedown`·`mouseup`·`click`·`dblclick`·`contextmenu`), 휠, 키(`keydown`·`keyup`·`keypress`), 터치, 포인터(`pointerdown` 등), 포커스(`focus`·`blur`·`focusin`·`focusout`), 글 입력(`input`·`change`·`composition*`), `visibilitychange` 와 창(`window`)의 `resize`·`gamepadconnected`·`gamepaddisconnected` 를 받는다. 이동(`mousemove`·`touchmove`·`pointermove`·`scroll`)은 개수만 센다. 게임패드는 0.1초마다 `navigator.getGamepads()` 로 읽어 눌린 버튼과 움직인 축이 바뀔 때 줄을 남긴다.
- 줄에는 버튼(`button`, `buttons`, `detail`), 누른 자리, 포인터 종류, 휠 양, 키(`key`, `code`, `keyCode`, `repeat`), 입력 글, 누른 시간(`held=`), 수정키, 대상 요소의 첫 클래스가 붙는다. 처음에 한 번 환경 줄(`env`: `engine`, 화면 크기, `getGamepads`·`PointerEvent` 가 있는지, 페이지 주소 `location.href`, 엔진 명령 그룹 목록)을 보낸다.
- 같은 줄이 이어지면(누르고 있는 키의 반복) 마지막 줄에 `xN` 으로 세고 서버로 다시 보내지 않는다.
- Windows 클라이언트에서 본 결과(2026-10-27, 마우스·키보드) [실측]:
  - `mousemove` 는 온다(2371회). `pointermove` 는 0회라 포인터 이벤트는 오지 않는 것으로 보인다. `scroll` 은 온다.
  - `click` 이 온다(`button=0 buttons=0 detail=undefined`, 스크롤 손잡이). 렌더러는 `click` 이 일반 `div` 에 오지 않는다고 보고 `mousedown` 으로 누름을 받는다(`onPress`). 어느 요소에 `click` 이 오는지는 확인 안 됨.
  - Alt 를 누르고 있으면 `keydown key= code=Alt keyCode=18 repeat=true` 가 이어진다. `key` 는 비어 있다.
  - 휠, 포커스, 글 입력·IME 조합, 터치, 게임패드는 Alt 반복 줄에 밀려 결과가 남지 않았다(확인 안 됨).
  - 서버가 받은 줄 수가 이벤트와 함께 늘었다(700줄). 페이지의 이벤트를 서버로 보내는 길은 된다.
  - 화면이 열린 동안 클라이언트는 `ServerBoundDiagnostics` 를 계속 보냈다(157개). `ScriptMessage` 는 오지 않았다.
  - 실행 중 `window.__commands__` 에는 명령 그룹이 40개 있지만, 게임 채팅, 슬래시 명령, `ScriptMessage` 를 보내는 이름은 없다.
- **서버로 보내기**: 렌더러는 줄마다 데이터 저장소 쓰기(`vanillaDduiDataStoreSetCommandGroup.setDataStorePathString`)를 부르고, 클라이언트는 이것을 `ServerBoundDataStore` 패킷으로 보낸다. 서버는 받은 줄을 창 위에 다시 그리므로 오가는 것이 한 화면에서 보인다. 이 길은 다른 DDUI 화면의 누름이 이미 쓰고 있다 [실측].

### 스크립트 이벤트 패킷 (`ScriptMessage`)

- 프로토콜에 `ScriptMessage`(ID 177, 예전 이름 `ScriptCustomEvent`) 패킷이 있다. 필드는 식별자와 문자열 값이다. Mojang 문서는 "클라이언트와 서버 사이의 사용자 메시지"라고만 적었고 [문서](https://github.com/Mojang/bedrock-protocol-docs/blob/main/json/ScriptMessagePacket.json), gophertunnel 은 양방향으로 두고 GameTest 용으로 추정한다 [문서](https://github.com/Sandertv/gophertunnel/blob/master/minecraft/protocol/packet/script_message.go). PrismarineJS 는 클라이언트가 보내는 패킷으로 적는다 [문서](https://github.com/PrismarineJS/minecraft-data/blob/master/data/bedrock/latest/proto.yml).
- 서버는 받을 수 있다: Dragonfly 는 이 패킷을 처리하지 않지만 `sghook.OnClientPacket` 으로 가로챌 수 있다. 시험 화면이 열린 동안 오는 `ScriptMessage` 는 식별자와 값이 창 위에 보이고 서버 로그(`ore ui probe script message`)에 남는다.
- Ore UI 페이지에서 보낼 길은 찾지 못했다. Windows 1.26.52 설치본의 게임 중 번들(`gameplay-*.js`)이 쓰는 엔진 명령 그룹은 `coreTranslate`·`coreScreenReader`·`sound`·`coreApplication`·`coreHaptics`·`coreOptions`·`vanillaGameplayContainerAPICommands`·`vanillaGameplayLocalPlayerRespawnCommand`·`vanillaGameplayLeaveGameCommandGroup`·`vanillaGameplayLocalPlayerWakeUpCommand`·`vanillaDduiDataStoreSetCommandGroup`(`setDataStorePathString`·`Number`·`Bool`, `dataStoreButtonPress`, `dataStoreCloseButtonPress`)·`vanillaCoreDduiContentLogCommandGroup` 이고, 메뉴 번들(`index-*.js`)에도 스크립트 이벤트·채팅·슬래시 명령을 보내는 그룹은 없다 [코드]. 8Crafter 의 Ore UI 형 정의에도 게임 채팅·명령·`ScriptMessage` 를 보내는 그룹은 없다(파티 채팅 `vanilla_partyChatCommandGroup` 만 있다) [문서](https://github.com/8Crafter-Studios/Ore-UI-Types/blob/main/index.d.ts). 실행 중 `window.__commands__` 의 40개 그룹에도 그런 이름은 없다(위 Windows 결과) [실측].
- 바닐라 클라이언트가 `ScriptMessage` 를 보내는 경우가 있는지, `/scriptevent` 와 이 패킷이 이어지는지는 확인 안 됨. 클라이언트가 친 `/scriptevent` 는 다른 명령처럼 `CommandRequest` 로 서버에 간다.
- 정리: 페이지에서 서버로 가는 길은 지금 데이터 저장소(`ServerBoundDataStore`)와 DDUI 화면 닫기(`ServerBoundDataDrivenScreenClosed`)다.

## 게임 중 페이지가 쓸 수 있는 엔진 기능

게임 중 Ore UI 페이지(`hbui/gameplay.html`)에서 서버가 보낸 JS 로 불러 본 결과다(Windows 1.26.60.5, 2026-10-28).

- **설정**: 게임 설정 화면의 설정을 읽고(`vanilla.menus.settings{Boolean,Option,Number,String}Query(id)`) 쓴다(`vanilla_menus_update_settings` 의 `updateBoolean`·`updateOption`·`updateNumber`·`updateString`(id, 값)) [실측]. 트리는 `vanilla.menus.settingsGroupQuery("settings-tabs-groups")` 아래 설정 664개다. 서버가 쓰는 방식은 [oreui.md](oreui.md) 클라이언트 설정.
- **효과음**: `soundCommandGroup.play(이름, 볼륨, 피치)` 가 소리를 낸다(`random.click`, `note.bass`, `ui.drawer_open`) [실측]. 팩 사운드 이름은 확인 안 됨.
- **번역**: `coreTranslateCommandGroup.translate(키)` 가 클라이언트 언어의 글을 돌려준다(`item.diamond.name`, 팩 `texts/*.lang` 키 모두) [실측]. 인자 배열을 둘째로 준 호출은 `undefined` 였다. `formatDate(초)` 는 `10/28/26`, `getHowLongAgoAsString(초)` 는 `1h ago` 를 돌려준다 [실측].
- **클립보드**: facet `vanilla.clipboard` 의 `copyToClipboard(글)` 이 복사한다(Windows) [실측]. 다른 기기는 확인 안 됨.
- **큰 UI**: `vanillaExtraLargeUICommandGroup.setExtraLargeUIEnabled(bool)` 이 바로 바꾼다 [실측].
- **안 되는 것**: 화면 낭독 `coreScreenReaderCommandGroup.read`(TTS 꺼짐), 캔버스(`getContext` 없음)는 안 됐다. `core.input`·`core.device.display`·`core.locale`·`core.safeZone`·`core.deviceInformation` 쿼리와 `vanilla.userAccount`·`vanilla.editor`(`navigateUri`)·`vanilla.share`·`vanilla.webBrowserFacet` facet 은 답이 오지 않았다 [실측].
- **읽을 수 있는 것**: `core.device.platform` 은 `{type, inputMethods}`, `vanilla.inGame` 은 `isInGame`·`isInMultiplayerSession`, `vanilla.buildSettings` 는 게임 버전과 빌드 종류다. `core.staticFeatureFlag(id)` 는 `{enabled}` 를 돌려준다 [실측].
- **페이지**: SVG 요소는 만들어진다. `requestAnimationFrame`·`performance.now` 가 있다. `localStorage` 는 서버를 나갔다 다시 들어와도 남지만 게임을 다시 켜면 사라진다 [실측]. `engine` 에는 `on`·`off`·`trigger`·`call`·`translate`·`createJSModel`·`updateWholeModel`·`synchronizeModels`·`createVirtualList`·`registerBindingAttribute` 등이 있다(Gameface 데이터 바인딩) [실측].
