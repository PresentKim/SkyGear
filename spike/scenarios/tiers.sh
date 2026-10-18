#!/bin/bash
# Machine tiers (docs/design/13-machine-tiers.md): the alloys, circuits and
# kits and the configuration copier register without server errors, an operator can be given them, their
# recipes and creative entries reach the client.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임"=강화 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin research free' \
  'cmd:/dfadmin give alloy_infused 2' 'expect:인퓨즈 합금 x2' \
  'cmd:/dfadmin give circuit_2 1' 'expect:제어 회로 II x1' \
  'cmd:/dfadmin give kit_1 2' 'expect:고급 키트 x2' \
  'cmd:/dfadmin give kit_3 1' 'expect:궁극 키트 x1' \
  'cmd:/dfadmin give crusher 1' 'expect:분쇄기 x1' \
  'cmd:/dfadmin give config_copier 1' 'expect:설정 복사기 x1' \
  'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:kit_1" host.log && grep -q "^RECIPE dragonfactory:circuit_4" host.log && grep -q "^RECIPE dragonfactory:config_copier" host.log && grep -q "^CREATIVE dragonfactory:kit_3" host.log && grep -q "^CREATIVE dragonfactory:alloy_atomic" host.log
