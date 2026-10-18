#!/bin/bash
# Sky start (docs/design/02-dragonfactory.md §2): a new save, /묘목 gives a
# sapling and refuses a second one within 10 minutes; /sapling is the same.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임"=시작 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/묘목' 'expect:참나무 묘목을 받았습니다' \
  'cmd:/sapling' 'expect:10분에 한 번' \
  'cmd:/메뉴발급' 'expect:이미 메뉴 아이템' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
