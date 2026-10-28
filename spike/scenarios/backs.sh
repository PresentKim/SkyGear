#!/bin/bash
# The back buttons of the sub screens: inside a save the friends screen goes
# back to the game menu and the member screen to island management, the
# advancement tree goes back to the game menu and a node to its tab, and
# closing the codex search input shows the search again; in the lobby the
# guide goes back to the main menu (whose first button opens the saves) and
# cancelling a save's deletion shows the save list again.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=뒤로 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/friends' 'form:"친구"=6' 'form:스카이기어"=null' \
  'cmd:/members' 'form:"참여자 관리"=2' 'form:"섬 관리"=null' \
  'cmd:/menu' 'form:스카이기어"=1' 'form:발전과제"=0' 'form:발전과제 · =2' 'form:발전과제"=231' 'form:스카이기어"=null' \
  'cmd:/codex' 'form:도감"=45' 'input:"도감 · 검색"=null' 'form:도감"=null' \
  'cmd:/lobby' 'expect:로비로 이동합니다' \
  'cmd:/guide' 'form:"서버 안내"=6' 'form:=0' \
  'form:"세이브 선택"=7' 'form:"세이브 삭제"=1' 'form:"세이브 선택"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-600
[ $h -eq 0 ]
