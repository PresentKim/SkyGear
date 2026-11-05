package storagenet

// Disk is a capacity-limited Storage.
type Disk struct {
	capacity int64
	stored   int64
	items    map[Key]int64
}

// NewDisk returns an empty disk holding at most capacity items in total.
func NewDisk(capacity int64) *Disk {
	if capacity < 0 {
		capacity = 0
	}
	return &Disk{capacity: capacity, items: make(map[Key]int64)}
}

// Capacity returns the maximum number of items the disk holds.
func (d *Disk) Capacity() int64 { return d.capacity }

// Stored returns the total number of items on the disk.
func (d *Disk) Stored() int64 { return d.stored }

// Contents returns a copy of the disk contents.
func (d *Disk) Contents() map[Key]int64 {
	out := make(map[Key]int64, len(d.items))
	for k, v := range d.items {
		out[k] = v
	}
	return out
}

// Load replaces the disk contents with m, dropping non-positive counts and
// truncating anything beyond the capacity.
func (d *Disk) Load(m map[Key]int64) {
	d.items = make(map[Key]int64, len(m))
	d.stored = 0
	for k, v := range m {
		if v <= 0 {
			continue
		}
		if room := d.capacity - d.stored; v > room {
			v = room
		}
		if v <= 0 {
			continue
		}
		d.items[k] = v
		d.stored += v
	}
}

// Insert implements Storage.
func (d *Disk) Insert(k Key, n int64, sim bool) int64 {
	if n <= 0 {
		return 0
	}
	if room := d.capacity - d.stored; n > room {
		n = room
	}
	if n <= 0 {
		return 0
	}
	if !sim {
		d.items[k] += n
		d.stored += n
	}
	return n
}

// Extract implements Storage.
func (d *Disk) Extract(k Key, n int64, sim bool) int64 {
	if n <= 0 {
		return 0
	}
	have := d.items[k]
	if n > have {
		n = have
	}
	if n <= 0 {
		return 0
	}
	if !sim {
		if n == have {
			delete(d.items, k)
		} else {
			d.items[k] = have - n
		}
		d.stored -= n
	}
	return n
}

// Count implements Storage.
func (d *Disk) Count(k Key) int64 { return d.items[k] }

// Each implements Storage.
func (d *Disk) Each(fn func(Key, int64)) {
	for k, v := range d.items {
		fn(k, v)
	}
}
