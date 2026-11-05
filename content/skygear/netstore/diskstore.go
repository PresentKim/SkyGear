package netstore

import (
	"reflect"
	"strconv"

	"github.com/df-mc/dragonfly/server/item"

	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
)

// diskState is the storage network's part of a save: the contents of every
// disk by disk id, and the counter that makes new ids.
type diskState struct {
	Disks map[string]map[storagenet.Key]int64 `json:"net_disks,omitempty"`
	Next  int                                 `json:"net_disk_next,omitempty"`
}

var (
	diskSaved = core.NewSaved[diskState](nil)
	diskPart  = core.NewPart[DiskStore](nil)
)

// DiskStore holds the open disks of a session and writes their contents to
// the save.
type DiskStore struct {
	s      *core.Session
	open   map[string]*storagenet.Disk
	protos map[storagenet.Key]item.Stack
}

// Disks is the disk store of s.
func Disks(s *core.Session) *DiskStore {
	d := diskPart.Of(s)
	if d.s == nil {
		d.s = s
		d.open = map[string]*storagenet.Disk{}
		d.protos = map[storagenet.Key]item.Stack{}
	}
	return d
}

// Open is the disk with the given id, read from the save on first use. Later
// calls return the same disk.
func (d *DiskStore) Open(id string, capacity int64) *storagenet.Disk {
	if disk, ok := d.open[id]; ok {
		return disk
	}
	disk := storagenet.NewDisk(capacity)
	disk.Load(diskSaved.Of(d.s).Disks[id])
	d.open[id] = disk
	return disk
}

// NewID makes a disk id that no disk has.
func (d *DiskStore) NewID() string {
	st := diskSaved.Of(d.s)
	for {
		st.Next++
		id := "d" + strconv.Itoa(st.Next)
		if _, used := st.Disks[id]; used {
			continue
		}
		if _, used := d.open[id]; used {
			continue
		}
		d.s.MarkDirty()
		return id
	}
}

// Flush copies the contents of the open disks into the save and marks it
// dirty if anything changed.
func (d *DiskStore) Flush() {
	st := diskSaved.Of(d.s)
	for id, disk := range d.open {
		c := disk.Contents()
		if old, ok := st.Disks[id]; ok && reflect.DeepEqual(old, c) || !ok && len(c) == 0 {
			continue
		}
		if st.Disks == nil {
			st.Disks = map[string]map[storagenet.Key]int64{}
		}
		if len(c) == 0 {
			delete(st.Disks, id)
		} else {
			st.Disks[id] = c
		}
		d.s.MarkDirty()
	}
}
