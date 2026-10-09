# 블록 애니메이션 텍스처 (조사)

목표: 기계가 동작 중(활성)일 때만 움직이는 텍스처를 보여 준다. 아직 구현하지 않았고 실기 확인 전이다.

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
- `terrain_texture.json`·`flipbook_textures.json` 은 팩끼리 **병합**된다. Dragonfly 가 만드는 팩의 타일을 우리 팩(`assets/resourcepack`)의 flipbook 이 가리켜도 된다.

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

## 구현 방안

텍스처는 리소스팩에서 관리하므로(addon.md §6) 전부 팩 파일로 한다. Dragonfly 패치는 필요 없다.

1. 띠 이미지 `assets/resourcepack/dragonfactory/textures/blocks/harvester_side_on.png`(16×(16×N)).
2. `textures/terrain_texture.json` 에 `"dragonfactory_harvester_side_on": {"textures": "textures/blocks/harvester_side_on"}`.
3. `textures/flipbook_textures.json` 에 위 예시 항목(`flipbook_texture` 는 `textures/blocks/harvester_side_on`, `atlas_tile` 은 `dragonfactory_harvester_side_on`).
4. Go 는 활성 상태의 permutation 재질 이름만 `dragonfactory_harvester_side_on` 으로 적는다(`pack_test.go` 가 키가 있는지 확인).

실기에서 먼저 확인할 것: (1) 커스텀 블록 `alpha_test` 재질에서 flipbook 이 움직이는지, (2) 상태 전환 시 텍스처가 바로 바뀌는지. 작은 기계 하나(자동 수확기 옆면)로 시험한 뒤 다른 기계로 넓힌다.

## 참고

- [Bedrock Wiki: Flipbook Textures](https://wiki.bedrock.dev/blocks/flipbook-textures)
- [Animated textures: the strip, the .mcmeta, and Bedrock](https://resourcepackcreator.com/guide/animated-textures-mcmeta)
- [Microsoft: Converting texture packs](https://learn.microsoft.com/en-us/minecraft/creator/documents/convertingtexturepacks)
