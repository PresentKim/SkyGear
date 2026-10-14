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
	"air": "없음", "water": "물", "lava": "용암", "coal_block": "석탄 블록", "iron_block": "철 블록", "copper_block": "구리 블록", "gold_block": "금 블록",
	"pumpkin": "호박", "melon_block": "수박 블록", "hay_block": "건초 더미", "dried_kelp_block": "말린 켈프 블록", "mud": "진흙", "kelp": "켈프", "dried_kelp": "말린 켈프", "brick": "벽돌",
	"wheat_seeds": "밀 씨앗", "wheat": "밀", "carrot": "당근", "potato": "감자", "beetroot": "비트", "beetroot_seeds": "비트 씨앗", "melon_slice": "수박 조각", "pumpkin_seeds": "호박 씨앗", "melon_seeds": "수박 씨앗", "farmland": "경작지", "oak_sapling": "참나무 묘목", "sapling": "묘목",
	"chest": "상자", "furnace": "화로", "composter": "퇴비통", "hopper": "호퍼", "torch": "횃불", "barrel": "통",
	"water_bucket": "물 양동이", "bucket": "양동이", "paper": "종이",
	"wooden_pickaxe": "나무 곡괭이", "stone_pickaxe": "돌 곡괭이", "iron_pickaxe": "철 곡괭이",
	"wooden_axe": "나무 도끼", "stone_hoe": "돌 괭이", "wooden_hoe": "나무 괭이", "stone_axe": "돌 도끼", "wooden_shovel": "나무 삽", "stone_shovel": "돌 삽",
	"stick": "막대기", "wooden_sword": "나무 검", "stone_sword": "돌 검", "iron_axe": "철 도끼", "iron_shovel": "철 삽", "iron_hoe": "철 괭이", "iron_sword": "철 검", "shears": "가위",
	"oak_slab": "참나무 반 블록", "oak_stairs": "참나무 계단", "oak_fence": "참나무 울타리", "fence_gate": "참나무 울타리 문", "wooden_door": "참나무 문", "trapdoor": "참나무 다락문", "ladder": "사다리", "bowl": "그릇",
	"cobblestone_slab": "조약돌 반 블록", "stone_stairs": "조약돌 계단", "cobblestone_wall": "조약돌 담장", "normal_stone_slab": "돌 반 블록", "normal_stone_stairs": "돌 계단", "stone_bricks": "석재 벽돌",
	"andesite": "안산암", "diorite": "섬록암", "granite": "화강암", "string": "실", "bone_meal": "뼛가루", "redstone": "레드스톤 가루", "gunpowder": "화약",
	"glass_pane": "유리판", "glass_bottle": "유리병", "bread": "빵", "brick_block": "벽돌 블록", "pumpkin_pie": "호박 파이",
}
