#!/bin/bash
# Advancements (advancement.go): having a crafting table completes "조합대"
# alone (checked every second), not the ones before it; the advancement tree
# opens from the menu, a node opens its detail; an operator completes one
# with the ones before it by /dfadmin advance.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=발전 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin item crafting_table 1' 'expect:발전과제 달성' \
  'cmd:/dfdebug status' 'expect:advanced=1' \
  'cmd:/menu' 'form:드래곤팩토리"=1' 'form:발전과제"=0' 'form:발전과제 · =null' \
  'cmd:/dfadmin advance research_table' 'expect:달성했습니다' \
  'cmd:/dfdebug status' 'expect:advanced=4' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
