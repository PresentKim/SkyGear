#!/bin/bash
# WAILA panel: an operator looks straight down at the island floor; the
# server sends the panel title (prefix _waila:A:) naming the block.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=보는 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin look 0 90' 'expect:_waila:A:' 'expect:보는 블록' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
