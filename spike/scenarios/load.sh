#!/bin/bash
# HUD rows (internal/hud) and the server load: the island HUD has the day
# and the next advancement (quest panel); the operator turns the load line on with
# /dfadmin tps and sees it on the island and in the lobby. (An expect step
# takes a whole board, later boards only carry changed rows, and the first
# board comes before the join message: an advancement sends the quest rows.)
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임"=부하 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin advance research_table' 'expect:유레카' \
  'cmd:/dfadmin tps' 'expect:서버 부하 표시를 켰습니다' 'expect:부하' \
  'cmd:/lobby' 'expect:TPS' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
