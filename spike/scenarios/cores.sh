#!/bin/bash
# Core bases: at Lv.2 an operator switches the shop to the core tab, buys a
# tier 1 core and the free tier 1 core, then cobblestone (a core base) from
# the material tab, and opens the core book from the core menu. Shop option
# indexes: the tab's products, then the five tabs, then back.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=코어 상점' 'expect:입장했습니다' \
  'cmd:/dfadmin level 2' 'expect:레벨을 2' \
  'cmd:/dfadmin money 1000' 'expect:소지금' \
  'cmd:/menu' 'form:드래곤팩토리"=1' 'form:상점 · 구매"=10' 'form:상점 · 구매"=1' 'expect:[1티어] 코어 을(를) 구매했습니다' \
  'form:상점 · 구매"=0' 'expect:무료) 을(를) 구매했습니다' 'form:상점 · 구매"=6' 'form:상점 · 구매"=1' 'expect:조약돌 x16 을(를) 구매했습니다' 'form:상점 · 구매"=null' \
  'cmd:/dfadmin status' 'expectre:money=660 ' \
  'cmd:/menu' 'form:드래곤팩토리"=4' 'form:"코어 관리"=0' 'form:"코어 도감"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
