#!/bin/bash
# Server load on the sidebar: the operator sees the TPS lines on the game
# sidebar in an instance and the "서버 상태" board in the lobby.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=부하 섬' 'expect:입장했습니다' \
  'expect:부하' 'cmd:/lobby' 'expect:서버 상태' 'expect:TPS' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
