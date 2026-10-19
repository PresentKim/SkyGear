#!/bin/bash
# Composter and barrel (composter.go, barrel.go): both researches open, the
# barrel recipe and creative entry reach the client, an operator gets a
# barrel (the block with its seven stages registers without server errors);
# the stone barrel (lava) too.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임 이름 설정"=배럴 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin research compost' 'cmd:/dfadmin research barrel' \
  'cmd:/dfadmin give barrel 1' 'expect:참나무 배럴 x1' \
  'cmd:/dfadmin research furnace' 'cmd:/dfadmin research stone_barrel' \
  'cmd:/dfadmin give stone_barrel 1' 'expect:돌 배럴 x1' \
  'cmd:/dfdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT|RECIPE|CREATIVE" host.log | grep -v "^RECIPE dragonfactory:\(drawer\|hammer\|sieve\|crusher\|auto\|harvester\|research\)" 
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:barrel_oak" host.log && grep -q "^CREATIVE dragonfactory:barrel_oak" host.log && grep -q "^RECIPE dragonfactory:barrel_stone" host.log && grep -q "^CREATIVE dragonfactory:barrel_stone" host.log
