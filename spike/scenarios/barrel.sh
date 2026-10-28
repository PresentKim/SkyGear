#!/bin/bash
# Composter and barrel (composter.go, barrel.go): both researches open, the
# barrel recipe and creative entry reach the client, an operator gets a
# barrel (the block with its seven stages registers without server errors);
# the stone barrel (lava) too, and witch water with the crushed blocks
# (witchwater.go, sgitem/crushedblock.go).
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=배럴 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research compost' 'cmd:/sgadmin research barrel' \
  'cmd:/sgadmin give barrel 1' 'expect:참나무 배럴 x1' \
  'cmd:/sgadmin research furnace' 'cmd:/sgadmin research stone_barrel' \
  'cmd:/sgadmin give stone_barrel 1' 'expect:돌 배럴 x1' \
  'cmd:/sgadmin give witch_water_bucket 1' 'expect:마녀의 물 양동이 x1' \
  'cmd:/sgadmin give crushed_netherrack 1' 'expect:분쇄된 네더랙 x1' \
  'cmd:/sgdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT|RECIPE|CREATIVE" host.log | grep -v "^RECIPE skygear:\(drawer\|hammer\|sieve\|crusher\|auto\|harvester\|research\)" 
[ $h -eq 0 ] && grep -q "^RECIPE skygear:barrel_oak" host.log && grep -q "^CREATIVE skygear:barrel_oak" host.log && grep -q "^RECIPE skygear:barrel_stone" host.log && grep -q "^CREATIVE skygear:barrel_stone" host.log && grep -q "^CREATIVE skygear:crushed_blackstone" host.log && grep -q "^CREATIVE skygear:mycelium_spores" host.log
