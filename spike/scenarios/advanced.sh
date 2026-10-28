#!/bin/bash
# Progression expansion: the metals,
# alloy smelter, energy cell, simulator, fluid machines and washer
# register without server errors, an operator can be given them, and their
# recipes and creative entries reach the client.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임 이름 설정"=고급 섬' 'form:"세이브 선택"=3' 'expect:입장했습니다' \
  'cmd:/sgadmin research free' \
  'cmd:/sgadmin give alloy_smelter 1' 'expect:합금로 x1' \
  'cmd:/sgadmin give energy_cell 1' 'expect:에너지 셀 x1' \
  'cmd:/sgadmin give mob_simulator 1' 'expect:몬스터 시뮬레이터 x1' \
  'cmd:/sgadmin give fluid_tank 1' 'expect:유체 탱크 x1' \
  'cmd:/sgadmin give crucible 1' 'expect:전기 도가니 x1' \
  'cmd:/sgadmin give lava_generator 1' 'expect:용암 발전기 x1' \
  'cmd:/sgadmin give ore_washer 1' 'expect:광석 세척기 x1' \
  'cmd:/sgadmin give dimension_gate 1' 'expect:차원 게이트 고정 앵커 x1' \
  'cmd:/sgadmin give tinker_table 1' 'expect:장비 제작대 x1' \
  'cmd:/sgadmin give tool_pickaxe_steel_bone 1' 'expect:강철 곡괭이' \
  'cmd:/sgadmin give storage_controller 1' 'expect:스토리지 컨트롤러 x1' \
  'cmd:/sgadmin give seed_flower_red 2' 'expect:x2' \
  'cmd:/sgadmin give petal_red 2' 'expect:x2' \
  'cmd:/sgadmin give electric_furnace 1' 'expect:전기 화로 x1' \
  'cmd:/sgadmin give item_vacuum 1' 'expect:아이템 수집기 x1' \
  'cmd:/sgadmin give tree_feller 1' 'expect:자동 벌목기 x1' \
  'cmd:/sgadmin give auto_crafter 1' 'expect:자동 제작기 x1' \
  'cmd:/sgadmin give ingot_bronze 4' 'expect:청동 주괴 x4' \
  'cmd:/sgadmin give metal_dust_tin 4' 'expect:주석 가루 x4' \
  'cmd:/sgadmin give essence_blaze 2' 'expect:블레이즈 에센스 x2' \
  'cmd:/sgdebug wave 3' 'expect:wave 3 started' 'wait:4' \
  'cmd:/sgdebug status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE skygear:alloy_smelter" host.log && grep -q "^RECIPE skygear:mob_simulator" host.log && grep -q "^RECIPE skygear:conduit_item_2" host.log && grep -q "^CREATIVE skygear:ore_washer" host.log && grep -q "^CREATIVE skygear:ingot_steel" host.log
