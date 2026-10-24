#!/bin/bash
# Machine window preview (/dfdebug ui, docs/tech/admin.md): the picker form
# lists the machine windows (a DDUI screen the bot cannot read) and a kind opens its window with sample items
# without placing the machine.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=창 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfdebug ui crusher' 'chest:분쇄기' 'close' \
  'cmd:/dfdebug ui alloy_smelter' 'chest:합금로' 'close' \
  'cmd:/dfdebug ui auto_crafter' 'chest:자동 제작기' 'close' \
  'cmd:/dfdebug ui magic_sieve' 'chest:마법 체' 'close' \
  'cmd:/dfdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|CHEST|chest" host.log
exit $h
