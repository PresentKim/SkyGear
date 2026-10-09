#!/bin/bash
# Base-block cores: at Lv.2 an operator buys a tier 1 core, the free tier 1
# core and a cobblestone base from the shop's core category, then opens the
# core book from the core menu.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=코어 상점' 'expect:입장했습니다' \
  'cmd:/dfadmin level 2' 'expect:레벨을 2' \
  'cmd:/dfadmin money 1000' 'expect:소지금' \
  'cmd:/menu' 'form:드래곤팩토리"=1' 'form:상점 · 구매"=2' 'form:상점 · 코어"=1' 'expect:[1티어] 코어 을(를) 구매했습니다' \
  'form:상점 · 코어"=0' 'expect:무료) 을(를) 구매했습니다' 'form:상점 · 코어"=5' 'expect:받침용 조약돌 을(를) 구매했습니다' 'form:상점 · 코어"=null' \
  'cmd:/dfadmin status' 'expectre:money=690 ' \
  'cmd:/menu' 'form:드래곤팩토리"=4' 'form:"코어 관리"=0' 'form:"코어 도감"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
