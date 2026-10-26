#!/bin/bash
# The codex (codex.go, codextabs.go): /도감 opens the search tab (the whole
# index, then 44 slots, the page buttons, the search button, five tabs and the
# back button), the sieve tab shows a block's drops under every mesh (another
# block in its top row, a mesh to the mesh's own page), a search finds an item
# by its initials and opens its page (the index then starts with the recently
# viewed items, so the search button moves down), and a held item opens its own
# page (where it comes from, what it is for, what it takes).
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=도감 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/도감' 'form:도감"=47' 'form:"도감 · 체 드롭표 · 자갈"=1' 'form:"도감 · 체 드롭표 · 모래"=22' 'form:"도감 · 실 메쉬"=null' \
  'cmd:/도감' 'form:도감"=78' 'input:"도감 · 검색"=ㅊ ㅈㄱ' 'form:도감"=0' 'form:강철 주괴"=null' \
  'cmd:/dfadmin item iron_ingot 1' 'cmd:/도감' 'form:철 주괴"=null' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log | cut -c1-160
[ $h -eq 0 ]
