#!/bin/bash
# Gift fairy: an operator calls the fairy, uses (right-clicks) it and gets
# a gift as it flies away.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임 이름 설정"=요정 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfdebug fairy' 'touch:minecraft:allay' 'expect:요정이 선물을 주고 날아갑니다' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT|>" host.log
[ $h -eq 0 ]
