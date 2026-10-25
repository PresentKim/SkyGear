#!/bin/bash
# Ore UI inventory probes (/dfdebug oreinput, oreengine, orescreen, oreinv,
# docs/tech/oreui.md): every probe opens (DDUI screens the bot cannot read)
# and the server log stays clean; orescreen reports the screen it opened.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=시험 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfdebug oreinput' \
  'cmd:/dfdebug oreengine' \
  'cmd:/dfdebug orescreen minecraft:inventory_screen true' 'expect:opened minecraft:inventory_screen' \
  'cmd:/dfdebug orescreen close' \
  'cmd:/dfdebug oreinv' \
  'cmd:/dfdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
exit $h
