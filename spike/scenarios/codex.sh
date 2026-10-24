#!/bin/bash
# The codex (codex.go, codextabs.go): /도감 opens the search tab (the whole
# index, then 44 slots, the page buttons, the search button, five tabs and the
# back button), the sieve tab leads from a block to a mesh to its drops, a
# search finds an item by its initials and opens its page, and a held item
# opens its own page (where it comes from, what it is for, what it takes).
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=도감 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/도감' 'form:도감"=47' 'form:"도감 · 체 드롭표"=0' 'form:"도감 · 자갈"=0' 'form:자갈 · 실 메쉬"=null' \
  'cmd:/도감' 'form:도감"=45' 'input:"도감 · 검색"=ㅊ ㅈㄱ' 'form:도감"=0' 'form:강철 주괴"=null' \
  'cmd:/dfadmin item iron_ingot 1' 'cmd:/도감' 'form:철 주괴"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-160
[ $h -eq 0 ]
