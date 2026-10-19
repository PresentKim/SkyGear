#!/bin/bash
# Load test: an operator builds factory modules on the island with
# /dfadmin factory (core line, sieve line, farm with harvester each) and the
# load line (turned on with /dfadmin tps) is shown after it runs a while. FACTORY sets
# the module count (default 4).
n=${FACTORY:-4}
timeout 180 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임 이름 설정"=공장 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin tps' 'expect:서버 부하 표시를 켰습니다' 'wait:3' 'show:TPS' \
  "cmd:/dfadmin factory $n" 'expect:공장 모듈' \
  'wait:30' 'show:TPS' 'wait:5' 'show:TPS' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
