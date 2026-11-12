package netstore

import (
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/content/skygear/machine"
)

// DiskCaps are the item counts the four disk tiers hold (1k, 4k, 16k, 64k).
var DiskCaps = [4]int64{1024, 4096, 16384, 65536}

// diskNames are the id suffixes of the disk tiers.
var diskNames = [4]string{"1k", "4k", "16k", "64k"}

// diskIDKey is the stack value that holds the id of a disk's contents in the
// save document.
const diskIDKey = "sg_disk_id"

// Disk is a storage disk: it holds DiskCaps[Tier] items, by count.
type Disk struct{ Tier int }

func (d Disk) EncodeItem() (string, int16) {
	return "skygear:storage_disk_" + diskNames[max(0, min(d.Tier, len(diskNames)-1))], 0
}

func (d Disk) Name() string {
	return i18n.T(i18n.Default, "name.NetDisk"+diskNames[max(0, min(d.Tier, len(diskNames)-1))])
}

func (Disk) Category() category.Category { return category.Items() }

func (Disk) MaxCount() int { return 1 }

// Upgrade is an upgrade of an importer or exporter, as Refined Storage 2's
// (AbstractModInitializer.registerUpgradeMappings at d33a173017): up to
// four speed upgrades, each taking 2 ticks off the 9 between operations,
// and one stack upgrade, which moves a stack an operation instead of one.
type Upgrade struct{ Stack bool }

func (u Upgrade) EncodeItem() (string, int16) {
	if u.Stack {
		return "skygear:stack_upgrade", 0
	}
	return "skygear:speed_upgrade", 0
}

func (u Upgrade) Name() string {
	if u.Stack {
		return i18n.T(i18n.Default, "name.NetStackUpgrade")
	}
	return i18n.T(i18n.Default, "name.NetSpeedUpgrade")
}

func (Upgrade) Category() category.Category { return category.Items() }

const (
	maxSpeedUpgrades = 4
	maxStackUpgrades = 1
)

// upgradesOf counts the speed and stack upgrades in an importer's or
// exporter's slots.
func upgradesOf(st *machine.State) (speed, stack int) {
	if st == nil {
		return 0, 0
	}
	for _, s := range st.Def.Input {
		it, _ := st.Inv().Item(s)
		if u, ok := it.Item().(Upgrade); ok {
			if u.Stack {
				stack++
			} else {
				speed++
			}
		}
	}
	return min(speed, maxSpeedUpgrades), min(stack, maxStackUpgrades)
}

// upgradeFits reports whether one more upgrade like it goes into an importer
// or exporter holding st.
func upgradeFits(st *machine.State, it item.Stack) bool {
	u, ok := it.Item().(Upgrade)
	if !ok {
		return false
	}
	speed, stack := upgradesOf(st)
	if u.Stack {
		return stack < maxStackUpgrades
	}
	return speed < maxSpeedUpgrades
}

// Pattern is a crafting pattern, empty until the pattern encoder writes a
// recipe on it (pattern.go).
type Pattern struct{}

func (Pattern) EncodeItem() (string, int16) { return "skygear:pattern", 0 }

func (Pattern) Name() string { return i18n.T(i18n.Default, "name.NetPattern") }

func (Pattern) Category() category.Category { return category.Items() }

func (Pattern) MaxCount() int { return 1 }

// DiskID returns the contents id a disk stack carries, "" for a new disk or
// anything that is no disk.
func DiskID(st item.Stack) string {
	if _, ok := st.Item().(Disk); !ok {
		return ""
	}
	v, _ := st.Value(diskIDKey)
	id, _ := v.(string)
	return id
}

// WithDiskID returns the disk stack with its contents id set.
func WithDiskID(st item.Stack, id string) item.Stack { return st.WithValue(diskIDKey, id) }

func init() {
	for t := range DiskCaps {
		world.RegisterItem(Disk{Tier: t})
	}
	world.RegisterItem(Pattern{})
	world.RegisterItem(Upgrade{})
	world.RegisterItem(Upgrade{Stack: true})
}

// acceptsUpgrade, acceptsDisk and acceptsPattern say what the slots of an
// importer or exporter, of a disk drive and of an autocrafter take.
func acceptsUpgrade(_ int, st item.Stack) bool {
	_, ok := st.Item().(Upgrade)
	return ok
}

func acceptsDisk(_ int, st item.Stack) bool {
	_, ok := st.Item().(Disk)
	return ok
}

func acceptsPattern(_ int, st item.Stack) bool {
	_, ok := st.Item().(Pattern)
	return ok
}
