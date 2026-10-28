#!/bin/bash
# Machine tiers: the alloys, circuits, kits and the configuration copier
# register without server errors, an operator can be given them, and their
# recipes and creative entries reach the client.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=강화 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research free' \
  'cmd:/sgadmin give alloy_infused 2' 'expect:인퓨즈 합금 x2' \
  'cmd:/sgadmin give circuit_2 1' 'expect:제어 회로 II x1' \
  'cmd:/sgadmin give kit_1 2' 'expect:고급 키트 x2' \
  'cmd:/sgadmin give kit_3 1' 'expect:궁극 키트 x1' \
  'cmd:/sgadmin give crusher 1' 'expect:분쇄기 x1' \
  'cmd:/sgadmin give config_copier 1' 'expect:설정 복사기 x1' \
  'cmd:/sgdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE skygear:kit_1" host.log && grep -q "^RECIPE skygear:circuit_4" host.log && grep -q "^RECIPE skygear:config_copier" host.log && grep -q "^CREATIVE skygear:kit_3" host.log && grep -q "^CREATIVE skygear:alloy_atomic" host.log
