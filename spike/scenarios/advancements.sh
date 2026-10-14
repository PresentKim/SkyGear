#!/bin/bash
# Advancements (advancement.go): having a crafting table completes "조합대"
# and the ones before it (checked every second) with a message naming the
# next one; the advancement screen opens from the menu; an operator
# completes one with /dfadmin advance.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=발전 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin item crafting_table 1' 'expect:발전과제 달성' \
  'cmd:/dfadmin status' 'expect:advanced=3' \
  'cmd:/menu' 'form:드래곤팩토리"=1' 'form:"발전과제"=0' 'form:"발전과제 · 나무 캐기"=0' 'form:"발전과제"=null' \
  'cmd:/dfadmin advance research_table' 'expect:다음: ' \
  'cmd:/dfadmin status' 'expect:advanced=4' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
