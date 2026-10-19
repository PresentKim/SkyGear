#!/bin/bash
# WAILA panel: an operator looks straight down at the island floor; the
# server sends the panel title (prefix _waila:A:) naming the block by its
# translation key (tile.<name>.name, raw JSON title).
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임 이름 설정"=보는 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin look 0 90' 'expectre:_waila:A:.*tile\.[a-z_.]+\.name' 'expect:보는 블록' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
