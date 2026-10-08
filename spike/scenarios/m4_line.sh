#!/bin/bash
# M4: an operator builds the sample hands-free line (auto miner on the core,
# crusher, sell box) and the line sells items
# without anyone touching it.
timeout 120 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=라인 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin status' 'expectre:sold=0$' \
  'cmd:/dfadmin line' 'expect:생산 라인을 만들었습니다' \
  'cmd:/dfadmin give crusher 1' 'expect:분쇄기 x1' \
  'wait:30' \
  'cmd:/dfadmin status' 'expectre:sold=[1-9][0-9]*$' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
