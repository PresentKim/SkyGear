#!/bin/bash
# Recipe locking (research spike): the operator locks the island's recipes,
# the client gets a short recipe list at once, unlocking sticks and opening
# all send it again.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=레시피 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin recipes lock' 'expectre:^recipes=[0-9]$' \
  'cmd:/dfadmin recipes unlock stick' 'expectre:^recipes=1?[0-9]$' \
  'cmd:/dfadmin recipes all' 'expectre:^recipes=[0-9]{3,}$' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
