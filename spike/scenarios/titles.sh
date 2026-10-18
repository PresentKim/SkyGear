#!/bin/bash
# Counter achievements (docs/design/01-platform.md §5): counting an action
# grants an achievement and its title, the title screen lists them by group
# and a title can be put on.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=3' 'input:"새 게임"=칭호 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/묘목' 'expect:묘목 받기 첫 경험' \
  'cmd:/title' 'form:"칭호 고르기"=1' 'form:성장"=0' 'expect:칭호를 바꿨습니다' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
