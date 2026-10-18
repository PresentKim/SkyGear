#!/bin/bash
# The save screen as a split view (platform showSaves, ui.ListSplitMenu): the
# slots on the left and the actions of the chosen save on the right. With no
# save the first empty slot is chosen and its only action starts a new game;
# with saves the first one is chosen, a row press chooses another, and an
# empty slot's 생성하기 opens the name form.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임"=저장 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/lobby' 'expect:로비로 이동합니다' \
  'cmd:/saves' 'form:"세이브 선택"=null' \
  'cmd:/saves' 'form:"세이브 선택"=6' 'input:"이름 변경"=null' \
  'cmd:/saves' 'form:"세이브 선택"=3' 'input:"새 게임"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-160
[ $h -eq 0 ]
