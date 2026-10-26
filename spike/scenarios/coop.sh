#!/bin/bash
# Friend request, invite code join, live InvSee (entity chest), the
# game menu forms and kick between a host (Hana) and a guest (Gyu). /invite
# opens the friends screen with the invite code in its first row; a request
# is accepted on the requests tab (the friend's row, then accept), which
# tells the one who asked. The guest waits out its login toast first: a
# player gets one toast per ui.ToastGap and a held one gives way to the next,
# so "request sent" would be replaced by the host's quick "friends now".
# /members lists the members with the chosen one's actions on the right
# (inventory, go to, bring, kick); after a kick the list shows the guest gone.
rm -f "$WORK/code.txt"
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=하나 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/invite' "capture:\"친구\"=$WORK/code.txt" \
  'expect:친구 요청을 보냈습니다' 'cmd:/friends' 'form:"친구"=4' 'form:"친구"=1' 'form:"친구"=null' \
  'expect:참여했습니다' 'wait:1' \
  'cmd:/members' 'form:"참여자 관리"=1' 'form:"참여자 관리"=2' 'chest:Gyu' 'take:35=ok' 'close' \
  'cmd:/dfdebug research' 'form:연구"=null' \
  'cmd:/members' 'form:"참여자 관리"=1' 'form:"참여자 관리"=5' 'form:"추방"=0' 'form:탈퇴·추방됨=null' > host.log 2>&1 &
host=$!
timeout 90 "$BOT" -name Gyu \
  "waitfile:$WORK/code.txt" 'form:"서버 안내"=0' 'form:"세이브 선택"=null' \
  'wait:3' 'cmd:/friends' 'form:"친구"=0' 'input:"친구 추가"=Hana' 'expect:친구 요청을 보냈습니다' 'expect:친구가 되었습니다' \
  "cmd:/join @$WORK/code.txt" 'expect:입장했습니다' 'slot:8=empty' 'expect:추방되어' > guest.log 2>&1
g=$?
wait $host; h=$?
echo "=== host";  grep -E "FORM|OK|FAIL|DONE|CAPTURED" host.log
echo "=== guest"; grep -E "FORM|OK|FAIL|DONE" guest.log
[ $h -eq 0 ] && [ $g -eq 0 ]
