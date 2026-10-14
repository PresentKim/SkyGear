#!/bin/bash
# The client's recipe book lists a recipe only when its result is the
# creative inventory item, block state included. The bot (BOT_DUMP_RECIPES)
# prints a "MISMATCH" line for every recipe result whose block runtime ID is
# not the one of its creative entry (vanilla recipe data named another chest
# state, so the chest and others were missing from the crafting list).
export BOT_DUMP_RECIPES=1
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=레시피 책 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin research chest' 'cmd:/dfadmin research compost' 'cmd:/dfadmin research free' 'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT|MISMATCH" host.log
[ $h -eq 0 ] && ! grep -q "^MISMATCH" host.log
