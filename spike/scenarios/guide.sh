#!/bin/bash
# The server guide: the first join shows it,
# a topic page goes back to it and 시작하기 goes on to the saves. /guide
# opens it again: in a save 시작하기 only closes it, in the lobby it does not
# open the saves.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=2' 'form:"서버 안내 · 함께 하기"=0' \
  'form:"서버 안내"=5' 'form:"서버 안내 · 명령어와 규칙"=0' \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=안내 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/guide' 'form:서버 안내"=1' 'form:서버 안내 · 세이브와 섬"=0' 'form:서버 안내"=0' 'cmd:/menu' 'form:스카이기어"=null' \
  'cmd:/lobby' 'expect:로비로 이동합니다' \
  'cmd:/guide' 'form:"서버 안내"=1' 'form:"서버 안내 · 세이브와 섬"=0' 'form:"서버 안내"=0' \
  'cmd:/saves' 'form:"세이브 선택"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-200
[ $h -eq 0 ]
