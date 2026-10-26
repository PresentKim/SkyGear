#!/bin/bash
# The game's own recipes reach the client: after the hammer research the
# recipe list holds a recipe whose result is dragonfactory:hammer and is in
# the creative inventory (the bot logs "RECIPE <name>" and "CREATIVE <name>").
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=망치 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin research hammer' 'cmd:/dfdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT|RECIPE|CREATIVE" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:hammer" host.log && grep -q "^CREATIVE dragonfactory:hammer" host.log
