package dragonfactory

import "github.com/presentkim/dragonfactory/internal/timed"

// Time-limited item messages use the Korean names.
func init() { timed.Name = itemName }

// koreanNames are the Korean names of the vanilla items the game shows in
// its own texts (missions, shop, sidebar), by item name without the
// "minecraft:" prefix. Items without an entry show their identifier.
var koreanNames = map[string]string{
	"cobblestone": "조약돌", "dirt": "흙", "grass_block": "잔디 블록", "stone": "돌", "sand": "모래", "gravel": "자갈",
	"oak_log": "참나무 원목", "birch_log": "자작나무 원목", "spruce_log": "가문비나무 원목", "oak_planks": "참나무 판자",
	"clay": "점토 블록", "clay_ball": "점토", "glass": "유리",
	"coal_ore": "석탄 광석", "iron_ore": "철 광석", "copper_ore": "구리 광석", "gold_ore": "금 광석",
	"coal": "석탄", "charcoal": "숯", "flint": "부싯돌", "iron_nugget": "철 조각", "gold_nugget": "금 조각",
	"raw_iron": "철 원석", "raw_copper": "구리 원석", "raw_gold": "금 원석", "iron_ingot": "철 주괴", "copper_ingot": "구리 주괴", "gold_ingot": "금 주괴",
	"pumpkin": "호박", "melon_block": "수박 블록", "hay_block": "건초 더미", "dried_kelp_block": "말린 켈프 블록", "mud": "진흙",
	"wheat_seeds": "밀 씨앗", "wheat": "밀", "carrot": "당근", "potato": "감자", "beetroot": "비트", "beetroot_seeds": "비트 씨앗", "melon_slice": "수박 조각", "pumpkin_seeds": "호박 씨앗", "melon_seeds": "수박 씨앗", "farmland": "경작지", "oak_sapling": "참나무 묘목", "sapling": "묘목",
	"chest": "상자", "furnace": "화로", "hopper": "호퍼", "torch": "횃불", "barrel": "통",
	"water_bucket": "물 양동이", "bucket": "양동이", "paper": "종이",
	"wooden_pickaxe": "나무 곡괭이", "stone_pickaxe": "돌 곡괭이", "iron_pickaxe": "철 곡괭이",
	"wooden_axe": "나무 도끼", "stone_hoe": "돌 괭이", "wooden_hoe": "나무 괭이", "stone_axe": "돌 도끼", "wooden_shovel": "나무 삽", "stone_shovel": "돌 삽",
}
