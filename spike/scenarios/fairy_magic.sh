#!/bin/bash
# Fairy magic: the "요정 마법" research opens with fairy dust, and the
# operator casts rain, snow, clear and the fixed suns (fairyspell.go). The
# board's form and the growth and machine speeds are covered by unit tests.
timeout 60 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=마법 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin item skygear:fairy_dust 2' 'expect:x2' \
  'cmd:/sgadmin research fairy_magic' 'expect:요정 마법' \
  'cmd:/sgdebug weather rain' 'expect:날씨: 비' \
  'cmd:/sgdebug weather snow' 'expect:날씨: 눈' \
  'cmd:/sgdebug weather clear' 'expect:날씨: 화창' \
  'cmd:/sgdebug sun night' 'expect:해: 밤 고정' \
  'cmd:/sgdebug sun flow' 'expect:해: 흐름' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ]
