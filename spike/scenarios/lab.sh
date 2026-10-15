#!/bin/bash
# Research essences (docs/design/11-research-vials.md): the laboratory and the
# five essences register without server errors, an operator can be given them,
# and their recipe and creative entries reach the client. The brewing itself
# and the essence costs of the researches are covered by unit tests.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=연구소 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin research free' \
  'cmd:/dfadmin give laboratory 1' 'expect:연구소 x1' \
  'cmd:/dfadmin give research_essence_engineering 4' 'expect:공학의 정수 x4' \
  'cmd:/dfadmin give research_essence_alchemy 2' 'expect:연금술의 정수 x2' \
  'cmd:/dfadmin give research_essence_farming 2' 'expect:농사의 정수 x2' \
  'cmd:/dfadmin give research_essence_hunting 2' 'expect:사냥의 정수 x2' \
  'cmd:/dfadmin give research_essence_magic 2' 'expect:마법의 정수 x2' \
  'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:laboratory" host.log && grep -q "^CREATIVE dragonfactory:laboratory" host.log
