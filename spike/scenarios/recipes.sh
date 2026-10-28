#!/bin/bash
# Research (research.go): recipes are locked but the base ones. Completing
# the stick research sends a short recipe list, /sgadmin research free
# opens every recipe and locks them again; then the chest research is paid
# with planks from the research form (/sgdebug research → 상자 → 연구 진행). The bot
# counts a recipe list as "recipes=<n>".
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=연구 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research stick' 'expectre:^recipes=[0-9]{1,2}$' \
  'cmd:/sgadmin research free' 'expectre:^recipes=[0-9]{3,}$' \
  'cmd:/sgadmin research free' 'expectre:^recipes=[0-9]{1,2}$' \
  'cmd:/sgadmin item oak_planks 64' 'expect:x64' \
  'cmd:/sgdebug research' 'form:연구"=2' 'form:연구"=55' 'expect:연구 완료' 'form:연구"=null' \
  'cmd:/sgdebug status' 'expect:researched=2' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
