#!/bin/bash
# M3: friend request, invite code join, live InvSee (entity chest), the
# game menu forms and kick between a host (Hana) and a guest (Gyu).
rm -f "$WORK/code.txt"
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임"=하나 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/invite' 'form:"초대하기"=0' "capture:\"초대 코드\"=$WORK/code.txt" \
  'expect:친구 요청을 보냈습니다' 'cmd:/friends' 'form:"친구"=1' 'form:"받은 친구 요청"=0' 'form:"친구 요청"=0' 'expect:친구가 되었습니다' \
  'expect:참여했습니다' 'wait:1' \
  'cmd:/members' 'form:"참여자 관리"=1' 'form:"Gyu"=0' 'chest:Gyu' 'take:35=ok' 'close' \
  'cmd:/menu' 'form:드래곤팩토리"=0' 'form:연구"=null' \
  'cmd:/members' 'form:"참여자 관리"=1' 'form:"Gyu"=1' 'form:"추방"=0' 'expect:추방했습니다' > host.log 2>&1 &
host=$!
timeout 90 "$BOT" -name Gyu \
  "waitfile:$WORK/code.txt" 'form:"서버 안내"=0' 'form:"세이브 선택"=null' \
  'cmd:/friends' 'form:"친구"=0' 'input:"친구 추가"=Hana' 'expect:친구 요청을 보냈습니다' 'expect:수락했습니다' \
  "cmd:/join @$WORK/code.txt" 'expect:입장했습니다' 'slot:8=empty' 'expect:추방되어' > guest.log 2>&1
g=$?
wait $host; h=$?
echo "=== host";  grep -E "FORM|OK|FAIL|DONE|CAPTURED" host.log
echo "=== guest"; grep -E "FORM|OK|FAIL|DONE" guest.log
[ $h -eq 0 ] && [ $g -eq 0 ]
