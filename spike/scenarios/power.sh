#!/bin/bash
# Energy and item conduits (generator.go, conduit.go, itemconduit.go): the generator and conduit researches
# open, their recipes and creative entries reach the client, an operator gets
# a generator and conduits (the 64 connection states of three tiers register
# without server errors).
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=전기 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin research furnace' 'cmd:/dfadmin research iron' 'cmd:/dfadmin research generator' 'cmd:/dfadmin research energy_conduit' \
  'cmd:/dfadmin research chest' 'cmd:/dfadmin research hopper' 'cmd:/dfadmin research item_conduit' \
  'cmd:/dfadmin give generator 1' 'expect:연소 발전기 x1' \
  'cmd:/dfadmin give energy_conduit_1 4' 'expect:철 전기 도관 x4' \
  'cmd:/dfadmin give item_conduit_1 4' 'expect:철 아이템 도관 x4' \
  'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:generator" host.log && grep -q "^RECIPE dragonfactory:energy_conduit_1" host.log && grep -q "^RECIPE dragonfactory:item_conduit_1" host.log && grep -q "^CREATIVE dragonfactory:item_conduit_2" host.log && grep -q "^CREATIVE dragonfactory:generator" host.log && grep -q "^CREATIVE dragonfactory:energy_conduit_3" host.log
