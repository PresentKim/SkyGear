#!/bin/bash
# M5: core upgrades. An operator reaches Lv.4 with core points, opens the
# core management from the menu and buys tier 2 for the earth core.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=코어 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin level 4' 'expect:레벨을 4' \
  'cmd:/dfadmin points 5' 'expect:코어 강화 포인트: 5' \
  'cmd:/menu' 'form:드래곤팩토리"=4' 'form:"코어 관리"=0' 'form:대지 코어 강화"=0' 'expect:티어 2가 되었습니다' 'form:대지 코어 강화"=null' \
  'cmd:/dfadmin status' 'expectre:points=2 tier=2 ' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
