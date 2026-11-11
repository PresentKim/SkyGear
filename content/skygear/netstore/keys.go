package netstore

import (
	"encoding/base64"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/nbtcanon"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
)

// KeyOf is the network key of an item: the encoding of the stack with its
// count set to one, so stacks that are Comparable have the same key. An
// empty stack has the empty key.
func KeyOf(st item.Stack) storagenet.Key {
	if st.Empty() {
		return ""
	}
	if k, ok := fastKeyOf(st); ok {
		return k
	}
	return slowKeyOf(st)
}

// fastKeyOf is KeyOf for a plain stack by way of the cache; ok is false for
// any other stack.
func fastKeyOf(st item.Stack) (k storagenet.Key, ok bool) {
	it, plain := plainItem(st)
	if !plain {
		return "", false
	}
	// An item whose comparable type holds an uncomparable value in an
	// interface field panics as a map key: it takes the slow path.
	defer func() {
		if recover() != nil {
			k, ok = "", false
		}
	}()
	if v, hit := plainKeys.Load(plainKey{it}); hit {
		return v.(storagenet.Key), true
	}
	k = slowKeyOf(st)
	if plainCount.Load() < maxPlainKeys {
		if _, loaded := plainKeys.LoadOrStore(plainKey{it}, k); !loaded {
			plainCount.Add(1)
		}
	}
	return k, true
}

// slowKeyOf is KeyOf by way of the NBT of the stack.
func slowKeyOf(st item.Stack) storagenet.Key {
	return storagenet.Key(base64.RawStdEncoding.EncodeToString(nbtcanon.Encode(core.EncodeItem(st))))
}

// plainKey is the cache key of the fast path: the item value, which with no
// stack data decides the whole encoding (name, meta and block states).
type plainKey struct{ it world.Item }

// maxPlainKeys bounds the fast path cache.
const maxPlainKeys = 1 << 14

var (
	plainKeys  sync.Map // plainKey -> storagenet.Key
	plainCount atomic.Int64
	// comparable caches, by item type, whether values of it can be map keys.
	comparableTypes sync.Map // reflect.Type -> bool
)

// plainItem is the item of st and whether the key of st depends on that item
// alone: no custom name, lore, enchantments, values, anvil cost, unbreakable
// flag, damage or item NBT.
func plainItem(st item.Stack) (world.Item, bool) {
	it := st.Item()
	if !core.Plain(st) || st.Unbreakable() {
		return nil, false
	}
	if _, ok := it.(world.NBTer); ok {
		return nil, false
	}
	if st.MaxDurability() != st.Durability() {
		return nil, false
	}
	t := reflect.TypeOf(it)
	c, ok := comparableTypes.Load(t)
	if !ok {
		c, _ = comparableTypes.LoadOrStore(t, t.Comparable())
	}
	return it, c.(bool)
}

// Proto is the stack of count one that k stands for in s (empty for a key
// that cannot be read). Decoded stacks are cached per session.
func Proto(s *core.Session, k storagenet.Key) item.Stack {
	if k == "" {
		return item.Stack{}
	}
	d := Disks(s)
	if st, ok := d.protos[k]; ok {
		return st
	}
	b, err := base64.RawStdEncoding.DecodeString(string(k))
	if err != nil {
		return item.Stack{}
	}
	st := decodeCanon(b)
	d.protos[k] = st
	return st
}

// decodeCanon is the stack nbtcanon wrote.
func decodeCanon(b []byte) item.Stack {
	m, raw, err := nbtcanon.Decode(b)
	if err != nil {
		return item.Stack{}
	}
	if m == nil {
		return core.DecodeItem(raw)
	}
	return item.ReadNBT(m, nil)
}
