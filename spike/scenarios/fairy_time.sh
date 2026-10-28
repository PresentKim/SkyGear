#!/bin/bash
# Fairy time: the "요정의 시간" research after the fairy magic opens fairy
# bone meal and the fairy hourglass (fairytime.go). What they do to crops,
# plants, saplings and machines is covered by unit tests.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=시간 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research fairy_magic' 'expect:요정 마법' \
  'cmd:/sgadmin research fairy_time' 'expect:요정의 시간' \
  'cmd:/sgadmin item skygear:fairy_bone_meal 4' 'expect:x4' \
  'cmd:/sgadmin item skygear:fairy_hourglass 1' 'expect:x1' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
