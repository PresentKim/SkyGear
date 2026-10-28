#!/bin/bash
# The game's hoppers (hopper/hopper.go): the wood hopper and the turn recipes reach
# the client after their researches, the blocks register without server
# errors and an operator can get all three.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=호퍼 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research wood_hopper' 'cmd:/sgadmin research hopper' \
  'cmd:/sgadmin give wood_hopper 1' 'expect:나무 호퍼 x1' \
  'cmd:/sgadmin give wood_hopper_reverse 1' 'expect:역방향 나무 호퍼 x1' \
  'cmd:/sgadmin give iron_hopper_reverse 1' 'expect:역방향 철 호퍼 x1' \
  'cmd:/sgdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE skygear:wood_hopper$" host.log && grep -q "^RECIPE skygear:wood_hopper_reverse$" host.log && grep -q "^RECIPE skygear:iron_hopper_reverse$" host.log && grep -q "^CREATIVE skygear:wood_hopper$" host.log
