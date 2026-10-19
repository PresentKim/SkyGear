#!/bin/bash
# The game's own recipes reach the client: after the hammer research the
# recipe list holds a recipe whose result is dragonfactory:hammer (the bot
# resolves result network IDs through the item registry of the join and
# logs them as "RECIPE <name>", and the creative inventory as "CREATIVE <name>": the client lists a recipe only when its result is creative).
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임 이름 설정"=망치 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin research hammer' 'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT|RECIPE|CREATIVE" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:hammer" host.log && grep -q "^CREATIVE dragonfactory:hammer" host.log
