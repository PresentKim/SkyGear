#!/bin/bash
# M4: an operator gets machines and a drawer with /dfadmin give (they are
# researched in play), and enters and leaves island expansion mode (the map
# view).
timeout 120 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=라인 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin give crusher 1' 'expect:분쇄기 x1' 'cmd:/dfadmin give drawer 2' 'expect:서랍 x2' \
  'cmd:/df' 'form:드래곤팩토리"=null' 'cmd:/menu' 'form:드래곤팩토리"=4' 'form:"섬 관리"=2' 'expect:섬 확장 모드' 'cmd:/menu' 'form:"섬 확장"=1' 'expect:섬 확장 모드를 끝냈습니다' \
  'cmd:/dfdebug status' 'expect:chunks=1' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
