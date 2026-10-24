#!/bin/bash
# Time-limited items: an operator takes a wooden pickaxe (players start
# bare-handed) and makes it expire in two real-time seconds; the server
# removes it from the hotbar and says so.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=기간제 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin item wooden_pickaxe' 'expect:x1' \
  'cmd:/dfdebug timed 2' 'expect:기간제로 바꿨습니다' \
  'slot:0=empty' 'expect:기간이 끝나 사라졌습니다: §f나무 곡괭이' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
