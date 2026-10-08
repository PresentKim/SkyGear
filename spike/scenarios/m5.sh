#!/bin/bash
# M5: core upgrades and urgent missions. An operator reaches Lv.4 with core
# points, buys tier 2 for the earth core from the menu, starts an urgent
# mission and opens its delivery from the main mission form.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=코어 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin level 4' 'expect:레벨을 4' \
  'cmd:/dfadmin points 5' 'expect:코어 강화 포인트: 5' \
  'cmd:/menu' 'form:드래곤팩토리"=4' 'form:"코어 관리"=0' 'form:대지 코어 강화"=0' 'expect:티어 2가 되었습니다' 'form:대지 코어 강화"=null' \
  'cmd:/dfadmin status' 'expectre:points=2 tier=2 ' \
  'cmd:/dfadmin urgent' 'expect:[긴급 매입]' \
  'cmd:/menu' 'form:드래곤팩토리"=0' 'form:"메인 미션"=1' 'expect:긴급 매입에 낼 아이템이 없습니다' 'form:"메인 미션"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
