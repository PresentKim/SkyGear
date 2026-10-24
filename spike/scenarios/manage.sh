#!/bin/bash
# Member management by sneaking and right-clicking a player (members.go
# usePlayer): the host (Hana) sneaks and uses the guest (Gyu) in its save, and
# the member list opens with Gyu chosen; bringing Gyu tells Gyu.
rm -f "$WORK/code.txt" "$WORK/joined"
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=하나 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/invite' "capture:\"친구\"=$WORK/code.txt" \
  'expect:참여했습니다' 'wait:1' 'sneak' 'wait:0.5' 'touchplayer:Gyu' 'form:"참여자 관리"=4' > host.log 2>&1 &
host=$!
timeout 90 "$BOT" -name Gyu \
  "waitfile:$WORK/code.txt" 'form:"서버 안내"=0' 'form:"세이브 선택"=null' \
  "cmd:/join @$WORK/code.txt" 'expect:입장했습니다' 'expect:Hana 님이 불렀습니다' > guest.log 2>&1
g=$?
wait $host; h=$?
echo "=== host";  grep -E "FORM|OK|FAIL|DONE|CAPTURED|touched" host.log
echo "=== guest"; grep -E "FORM|OK|FAIL|DONE" guest.log
[ $h -eq 0 ] && [ $g -eq 0 ] && grep -q "참여자 관리" host.log
