#!/bin/bash
# The codex (codex.go): /도감 opens the front page, the sieve table pages
# lead from a block to a mesh to its drops, and a held item opens its own
# page (where it comes from, what it is for). The search card finds an item
# by its initials and opens the page of the first match.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=도감 섬' 'expect:입장했습니다' \
  'cmd:/도감' 'form:도감"=1' 'form:"도감 · 체 드롭표"=0' 'form:"도감 · 자갈"=0' 'form:자갈 · 실 메쉬"=null' \
  'cmd:/도감' 'form:도감"=0' 'input:"도감 · 검색"=ㅊ ㅈㄱ' 'form:검색"=0' 'form:철 주괴"=null' \
  'cmd:/dfadmin item iron_ingot 1' 'cmd:/도감' 'form:철 주괴"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
