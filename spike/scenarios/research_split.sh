#!/bin/bash
# The research screen as a split view (research.go, ui.SplitMenu): with the
# operator's test commands every research is shared out at random between
# the two tabs and 300 test copies make a long list (the screen is not
# changed to test it). The grid is the first 40 buttons; the bot picks a
# slot, goes to the done tab, turns the page and closes.
timeout 120 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=연구 분배 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgdebug research random 40' 'expect:임의로 완료' \
  'cmd:/sgdebug research load 300' 'expect:시험 복사본 300개' \
  'cmd:/sgdebug research' 'form:연구"=5' 'form:연구"=7' 'form:연구"=null' \
  'cmd:/sgdebug research load 0' 'expect:시험 복사본 0개' \
  'cmd:/sgadmin research reset' 'cmd:/sgdebug status' 'expect:researched=0' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-140
[ $h -eq 0 ]
