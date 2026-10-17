#!/bin/bash
# Research (research.go): recipes are locked but the base ones. Completing
# the stick research sends a short recipe list, /dfadmin research free
# opens every recipe and locks them again; then the chest research is paid
# with planks from the research form (menu → 연구 → 상자 → 연구 진행). The bot
# counts a recipe list as "recipes=<n>".
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=연구 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin research stick' 'expectre:^recipes=[0-9]{1,2}$' \
  'cmd:/dfadmin research free' 'expectre:^recipes=[0-9]{3,}$' \
  'cmd:/dfadmin research free' 'expectre:^recipes=[0-9]{1,2}$' \
  'cmd:/dfadmin item oak_planks 64' 'expect:x64' \
  'cmd:/menu' 'form:드래곤팩토리"=0' 'form:연구"=2' 'form:연구"=40' 'expect:연구 완료' 'form:연구"=null' \
  'cmd:/dfadmin status' 'expect:researched=2' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
