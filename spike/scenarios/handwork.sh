#!/bin/bash
# Hand resources (handwork.go): the new items exist and the "돌 다지기"
# research opens the pebble recipes (a few more recipes after it). Using a
# stick on dirt and stripping logs are covered by unit tests.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=손작업 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin item dragonfactory:pebble_stone 4' 'expect:x4' \
  'cmd:/dfadmin item dragonfactory:bark_fiber 4' 'expect:x4' \
  'cmd:/dfadmin item dragonfactory:dust 1' 'expect:x1' \
  'cmd:/dfadmin research stick' 'expectre:^recipes=[0-9]{1,2}$' \
  'cmd:/dfadmin research pebbles' 'expect:돌 다지기' \
  'cmd:/dfdebug status' 'expect:researched=2' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
