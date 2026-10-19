#!/bin/bash
# Machine window preview (/dfadmin ui, docs/tech/admin.md): the picker form
# lists the machine windows and a kind opens its window with sample items
# without placing the machine.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=창 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin ui' 'form:"/dfadmin ui"=0' 'chest:분쇄기' 'close' \
  'cmd:/dfadmin ui alloy_smelter' 'chest:합금로' 'close' \
  'cmd:/dfadmin ui auto_crafter' 'chest:자동 제작기' 'close' \
  'cmd:/dfadmin ui laboratory' 'chest:연구소' 'close' \
  'cmd:/dfadmin ui magic_sieve' 'chest:마법 체' 'close' \
  'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|CHEST|chest" host.log
exit $h
