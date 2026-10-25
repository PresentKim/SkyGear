#!/bin/bash
# The help window (docs/design/10-user-flow.md): /도움말 opens it on the first
# tab, a topic's next button goes on through the tab and back returns to it,
# and the window opens again on the tab looked at last (/dfhelp too).
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=도움말 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/도움말' 'form:도움말"=0' 'form:도움말 · 시작하기"=0' 'form:도움말 · 목표와 흐름"=2' \
  'form:도움말"=6' 'form:도움말"=null' \
  'cmd:/dfhelp' 'form:도움말"=0' 'form:도움말 · 손작업 자원"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-220
[ $h -eq 0 ]
