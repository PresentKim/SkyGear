#!/bin/bash
# Energy and item conduits (generator.go, conduit.go, bundle.go): the generator and conduit researches
# open, their recipes and creative entries reach the client, an operator gets
# a generator and conduits (the 64 connection states of three tiers register
# without server errors).
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=에너지 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research furnace' 'cmd:/sgadmin research iron' 'cmd:/sgadmin research generator' 'cmd:/sgadmin research energy_conduit' \
  'cmd:/sgadmin research chest' 'cmd:/sgadmin research hopper' 'cmd:/sgadmin research item_conduit' \
  'cmd:/sgadmin give generator 1' 'expect:연소 발전기 x1' \
  'cmd:/sgadmin give energy_conduit_1 4' 'expect:기본 에너지 도관 x4' \
  'cmd:/sgadmin give item_conduit_1 4' 'expect:기본 아이템 도관 x4' \
  'cmd:/sgadmin give fluid_conduit_1 4' 'expect:기본 유체 도관 x4' \
  'cmd:/sgdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE skygear:generator" host.log && grep -q "^RECIPE skygear:conduit_energy_1" host.log && grep -q "^RECIPE skygear:conduit_item_1" host.log && grep -q "^CREATIVE skygear:conduit_item_2" host.log && grep -q "^CREATIVE skygear:generator" host.log && grep -q "^CREATIVE skygear:conduit_energy_3" host.log && grep -q "^CREATIVE skygear:conduit_fluid_2" host.log
