# 블록 애니메이션 텍스처

기계가 동작 중(활성)일 때만 움직이는 텍스처를 보여 준다. 정면이 있는 기계와 정육면체 기계 전부에 적용했다(`animate.go`). 모델이 있는 체(자동 체·마법 체·손 체)는 제외한다.

## Bedrock 의 방법: flipbook 텍스처

- 블록 텍스처 애니메이션은 리소스팩의 `textures/flipbook_textures.json` 으로 한다(Java 의 `.mcmeta` 대신 파일 하나에 모음).
- 이미지는 **세로 띠**: 폭 16, 높이 16×프레임 수. 위에서부터 0, 1, 2… 프레임.
- 항목 하나가 아틀라스 타일 하나(`terrain_texture.json` 의 키)를 움직인다.

```json
[
  {
    "flipbook_texture": "textures/blocks/harvester_side_on",
    "atlas_tile": "dragonfactory_harvester_side_on",
    "ticks_per_frame": 2,
    "blend_frames": false
  }
]
```

  - `ticks_per_frame`: 프레임당 틱(20틱=1초). `frames`(프레임 순서 배열)로 순서·반복을 직접 줄 수도 있다.
  - `blend_frames`: 프레임 사이를 섞어 부드럽게 한다(기본값 켜짐. 픽셀 느낌을 원하면 `false`).
  - 그 밖에 `atlas_index`, `atlas_tile_variant`, `replicate` 가 있다(쓸 일 없음).
- 커스텀 블록은 `material_instances` 의 텍스처 이름이 그 아틀라스 타일이면 그대로 움직인다. 블록 쪽에 추가 컴포넌트는 없다.
- `terrain_texture.json`·`flipbook_textures.json` 은 팩끼리 **병합**된다. 다른 팩의 타일을 우리 팩(`assets/resourcepack`)의 flipbook 이 가리켜도 된다.

## 활성 시에만 움직이게

flipbook 은 클라이언트가 항상 돌린다. 켜고 끄는 것은 **블록 상태(permutation)로 텍스처를 바꿔서** 한다.

1. 기계에 상태 `dragonfactory:active`(`false`/`true`)를 둔다(`Machine.States`).
2. `Permutations` 에 `q.block_state('dragonfactory:active') == true` 조건으로 `material_instances` 를 움직이는 타일(`…_on`)로 바꾼 속성을 넣는다. 꺼진 상태는 지금 텍스처 그대로.
3. 서버는 기계가 일을 시작/멈출 때만 그 상태로 `SetBlock` 한다(블록 갱신 패킷 하나). 매 틱 바꾸지 말고, 입력이 없거나 출력이 꽉 차 멈춘 상태가 이어질 때만 끈다.
   - 손 체는 이미 같은 방식(`dragonfactory:look` 상태 + `sievePermutations`)으로 모양을 바꾸고 있다. 체가 아닌 기계도 같은 틀에 상태 하나를 더하면 된다.
   - `Machine` 의 상태가 바뀌면 블록 해시가 달라지므로 `EncodeBlock` 과 저장·불러오기가 그 상태를 함께 다뤄야 한다(체 구현 참고).

## 제약

- **모든 블록이 같은 박자로 움직인다.** flipbook 은 텍스처 단위라 블록마다 시작 시점을 정할 수 없다. 켜지는 순간 0프레임부터 시작하지 않는다.
- 진행도처럼 서버 값에 맞춘 그림은 안 된다(그건 지금처럼 상태별 텍스처/모델로).
- 텍스처만 움직인다. 부품이 실제로 도는 모습이 필요하면 블록 대신 엔티티(애니메이션 모델)를 겹쳐 띄워야 해서 훨씬 무겁다.
- 아이템 아이콘(인벤토리)에서는 flipbook 이 버전에 따라 동작하지 않는다. 아이콘은 정지 이미지로 둔다.

## 구현

- 대상(`animatedMachines`): 전기 화로·분쇄기·광석 세척기·합금로·몬스터 시뮬레이터·전기 도가니·자동 제작기·정수 추출기(정면 `<이름>_front_on`)와 자동 자원 채굴기·수확기·연소 발전기·용암 발전기·아이템 흡수기·자동 벌목기(옆면 `<이름>_side_on`, 정육면체의 "*" 재질).
- 팩: 띠 PNG(16×64, 4프레임)와 `terrain_texture.json`·`flipbook_textures.json`(프레임당 4틱, 섞지 않음) 항목. 전기 화로의 톱니바퀴 띠는 따로 그렸고 나머지는 `go run ./tools/animtex assets/resourcepack/dragonfactory`가 기계 자신의 텍스처에서 만든다(강조 색 띠가 흐르고, 발전기·도가니는 깜빡임). 임시 그림이므로 교체본이 오면 PNG만 바꾼다. `pack_test.go`의 `TestFlipbooksMatchTheAtlas`가 띠 규격과 타일 이름을 확인한다.
- 코드(`animate.go`): 블록 상태 `dragonfactory:active`(`off`/`on`). `on`일 때 위 재질로 바뀐다(`activePermutations`). 서버는 `Tick` 끝의 `showActive`가 판정해 바뀔 때만 `SetBlock`한다. 일했다는 표시는 `markWorked`(`tickWork`와 발전기)가 하고, 마지막 작업 뒤 40틱(`activeHold`)은 켜진 채로 둔다(깜빡임 방지).
- 블록 해시는 켜진 모습이 따로이고(`Machine.active`), 저장은 하지 않는다(다음 틱에 다시 판정).
- 강화되는 기계는 단계 모양과 한 상태(`dragonfactory:tier`)에 합쳐 담는다(`tierlook.go`, 작동 모습 값은 단계×2+1). 단계별 띠(`_on_t<단계>`)와 평면 텍스처는 `tools/tierblocks`가 만든다. `animtex`는 flipbook 목록을 새로 쓰므로 `animtex`를 먼저, `tierblocks`를 나중에 돌린다.
- 새 기계를 더하려면 `animatedMachines`에 종류를 더하고 띠(`tools/animtex`의 `specs`)를 만든다.

커스텀 블록 `alpha_test` 재질에서 flipbook 이 움직이지 않거나 상태 전환이 늦으면 `blend_frames`·재질 렌더 방식을 바꿔 본다.

## 참고

- [Bedrock Wiki: Flipbook Textures](https://wiki.bedrock.dev/blocks/flipbook-textures)
- [Animated textures: the strip, the .mcmeta, and Bedrock](https://resourcepackcreator.com/guide/animated-textures-mcmeta)
- [Microsoft: Converting texture packs](https://learn.microsoft.com/en-us/minecraft/creator/documents/convertingtexturepacks)
