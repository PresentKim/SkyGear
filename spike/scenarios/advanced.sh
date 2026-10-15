#!/bin/bash
# Progression expansion (docs/design/07-progression-expansion.md): the metals,
# alloy smelter, energy cell, simulator, composter, fluid machines and washer
# register without server errors, an operator can be given them, and their
# recipes and creative entries reach the client.
timeout 90 "$BOT" -name Hana \
  'form:"서버 안내"=0' 'form:"세이브 선택"=0' 'input:"새 게임"=고급 섬' 'expect:입장했습니다' \
  'cmd:/dfadmin research free' \
  'cmd:/dfadmin give alloy_smelter 1' 'expect:합금로 x1' \
  'cmd:/dfadmin give energy_cell 1' 'expect:에너지 셀 x1' \
  'cmd:/dfadmin give mob_simulator 1' 'expect:몬스터 시뮬레이터 x1' \
  'cmd:/dfadmin give auto_composter 1' 'expect:자동 퇴비통 x1' \
  'cmd:/dfadmin give fluid_tank 1' 'expect:유체 탱크 x1' \
  'cmd:/dfadmin give crucible 1' 'expect:전기 도가니 x1' \
  'cmd:/dfadmin give lava_generator 1' 'expect:용암 발전기 x1' \
  'cmd:/dfadmin give ore_washer 1' 'expect:광석 세척기 x1' \
  'cmd:/dfadmin give dimension_gate 1' 'expect:차원 게이트 x1' \
  'cmd:/dfadmin give tinker_table 1' 'expect:장비 제작대 x1' \
  'cmd:/dfadmin give tool_pickaxe_steel_bone 1' 'expect:강철 곡괭이' \
  'cmd:/dfadmin give storage_controller 1' 'expect:스토리지 컨트롤러 x1' \
  'cmd:/dfadmin give seed_flower_red 2' 'expect:x2' \
  'cmd:/dfadmin give petal_red 2' 'expect:x2' \
  'cmd:/dfadmin give electric_furnace 1' 'expect:전기 화로 x1' \
  'cmd:/dfadmin give item_vacuum 1' 'expect:아이템 흡수기 x1' \
  'cmd:/dfadmin give tree_feller 1' 'expect:자동 벌목기 x1' \
  'cmd:/dfadmin give auto_crafter 1' 'expect:자동 제작기 x1' \
  'cmd:/dfadmin give ingot_bronze 4' 'expect:청동 주괴 x4' \
  'cmd:/dfadmin give metal_dust_tin 4' 'expect:주석 가루 x4' \
  'cmd:/dfadmin give essence_blaze 2' 'expect:블레이즈 에센스 x2' \
  'cmd:/dfadmin wave 3' 'expect:wave 3 started' 'wait:4' \
  'cmd:/dfadmin status' 'expect:researched=' > host.log 2>&1
h=$?
echo "=== host"; grep -E "FORM|OK|FAIL|DONE|OUTPUT" host.log
[ $h -eq 0 ] && grep -q "^RECIPE dragonfactory:alloy_smelter" host.log && grep -q "^RECIPE dragonfactory:mob_simulator" host.log && grep -q "^RECIPE dragonfactory:conduit_item_2" host.log && grep -q "^CREATIVE dragonfactory:ore_washer" host.log && grep -q "^CREATIVE dragonfactory:ingot_steel" host.log
