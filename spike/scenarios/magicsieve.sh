#!/bin/bash
# The magic sieve and the new mobs (docs/design/15-magic-sieve.md): the machine,
# its recipe and the four new essences register without server errors, an
# operator can be given them, and the production statistics screen opens.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=마법 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/dfadmin research free' \
  'cmd:/dfadmin give magic_sieve 1' 'expect:마법 체 x1' \
  'cmd:/dfadmin give essence_drowned 2' 'expect:익사자 에센스 x2' \
  'cmd:/dfadmin give essence_witch 2' 'expect:마녀 에센스 x2' \
  'cmd:/dfadmin give essence_piglin 2' 'expect:피글린 에센스 x2' \
  'cmd:/dfadmin give essence_wither_skeleton 2' 'expect:위더 스켈레톤 에센스 x2' \
  'cmd:/통계' 'form:통계=0' \
  'cmd:/dfdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:magic_sieve" host.log && grep -q "^CREATIVE dragonfactory:magic_sieve" host.log && grep -q "^CREATIVE dragonfactory:essence_piglin" host.log
