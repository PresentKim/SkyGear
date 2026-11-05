package netstore

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/core"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
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
	return storagenet.Key(base64.RawStdEncoding.EncodeToString(canonical(core.EncodeItem(st))))
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

// decodeCanon is the stack canonical wrote.
func decodeCanon(b []byte) item.Stack {
	m, raw, err := uncanonical(b)
	if err != nil {
		return item.Stack{}
	}
	if m == nil {
		return core.DecodeItem(raw)
	}
	return item.ReadNBT(m, nil)
}

// The NBT encoder writes the entries of a compound in map order, so the same
// item encodes to different bytes. Keys need one form per item: this is a
// small tagged binary form of the decoded tree, with compounds sorted by key.

const (
	cByte byte = iota + 1
	cInt16
	cInt32
	cInt64
	cFloat32
	cFloat64
	cString
	cBytes
	cInt32s
	cInt64s
	cList
	cMap
	// cRaw marks a tree with a type this form lacks: the rest is NBT as is.
	cRaw = 0xff
)

var errCanon = errors.New("netstore: unsupported nbt value")

// canonical returns the canonical form of the NBT of one item, or the NBT
// itself behind cRaw when it holds a value of a type canonical lacks.
func canonical(enc []byte) []byte {
	var m map[string]any
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(enc), nbt.LittleEndian).Decode(&m); err == nil {
		var b bytes.Buffer
		if putVal(&b, m) == nil {
			return b.Bytes()
		}
	}
	return append([]byte{cRaw}, enc...)
}

// uncanonical is the NBT tree canonical wrote; raw reports the cRaw form,
// whose bytes it returns instead.
func uncanonical(b []byte) (m map[string]any, raw []byte, err error) {
	if len(b) > 0 && b[0] == cRaw {
		return nil, b[1:], nil
	}
	r := bytes.NewReader(b)
	v, err := getVal(r)
	if err != nil {
		return nil, nil, err
	}
	m, ok := v.(map[string]any)
	if !ok || r.Len() != 0 {
		return nil, nil, errCanon
	}
	return m, nil, nil
}

func putUvarint(b *bytes.Buffer, n uint64) {
	var t [binary.MaxVarintLen64]byte
	b.Write(t[:binary.PutUvarint(t[:], n)])
}

func putVarint(b *bytes.Buffer, n int64) {
	var t [binary.MaxVarintLen64]byte
	b.Write(t[:binary.PutVarint(t[:], n)])
}

func putStr(b *bytes.Buffer, s string) {
	putUvarint(b, uint64(len(s)))
	b.WriteString(s)
}

func putVal(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case uint8:
		b.WriteByte(cByte)
		b.WriteByte(x)
	case int16:
		b.WriteByte(cInt16)
		putVarint(b, int64(x))
	case int32:
		b.WriteByte(cInt32)
		putVarint(b, int64(x))
	case int64:
		b.WriteByte(cInt64)
		putVarint(b, x)
	case float32:
		b.WriteByte(cFloat32)
		putUvarint(b, uint64(math.Float32bits(x)))
	case float64:
		b.WriteByte(cFloat64)
		putUvarint(b, math.Float64bits(x))
	case string:
		b.WriteByte(cString)
		putStr(b, x)
	case []byte:
		b.WriteByte(cBytes)
		putUvarint(b, uint64(len(x)))
		b.Write(x)
	case []int32:
		b.WriteByte(cInt32s)
		putUvarint(b, uint64(len(x)))
		for _, n := range x {
			putVarint(b, int64(n))
		}
	case []int64:
		b.WriteByte(cInt64s)
		putUvarint(b, uint64(len(x)))
		for _, n := range x {
			putVarint(b, n)
		}
	case []any:
		b.WriteByte(cList)
		putUvarint(b, uint64(len(x)))
		for _, e := range x {
			if err := putVal(b, e); err != nil {
				return err
			}
		}
	case map[string]any:
		b.WriteByte(cMap)
		putUvarint(b, uint64(len(x)))
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			putStr(b, k)
			if err := putVal(b, x[k]); err != nil {
				return err
			}
		}
	default:
		return errCanon
	}
	return nil
}

func getStr(r *bytes.Reader) (string, error) {
	n, err := binary.ReadUvarint(r)
	if err != nil || n > uint64(r.Len()) {
		return "", errCanon
	}
	s := make([]byte, n)
	_, _ = r.Read(s)
	return string(s), nil
}

func getVal(r *bytes.Reader) (any, error) {
	t, err := r.ReadByte()
	if err != nil {
		return nil, errCanon
	}
	switch t {
	case cByte:
		return r.ReadByte()
	case cInt16, cInt32, cInt64:
		n, err := binary.ReadVarint(r)
		if err != nil {
			return nil, errCanon
		}
		switch t {
		case cInt16:
			return int16(n), nil
		case cInt32:
			return int32(n), nil
		}
		return n, nil
	case cFloat32, cFloat64:
		n, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, errCanon
		}
		if t == cFloat32 {
			return math.Float32frombits(uint32(n)), nil
		}
		return math.Float64frombits(n), nil
	case cString:
		return getStr(r)
	case cBytes:
		n, err := binary.ReadUvarint(r)
		if err != nil || n > uint64(r.Len()) {
			return nil, errCanon
		}
		out := make([]byte, n)
		_, _ = r.Read(out)
		return out, nil
	case cInt32s, cInt64s, cList, cMap:
		n, err := binary.ReadUvarint(r)
		if err != nil || n > uint64(r.Len()) {
			return nil, errCanon
		}
		switch t {
		case cInt32s:
			out := make([]int32, n)
			for i := range out {
				v, err := binary.ReadVarint(r)
				if err != nil {
					return nil, errCanon
				}
				out[i] = int32(v)
			}
			return out, nil
		case cInt64s:
			out := make([]int64, n)
			for i := range out {
				v, err := binary.ReadVarint(r)
				if err != nil {
					return nil, errCanon
				}
				out[i] = v
			}
			return out, nil
		case cList:
			out := make([]any, n)
			for i := range out {
				if out[i], err = getVal(r); err != nil {
					return nil, err
				}
			}
			return out, nil
		}
		out := make(map[string]any, n)
		for i := uint64(0); i < n; i++ {
			k, err := getStr(r)
			if err != nil {
				return nil, err
			}
			if out[k], err = getVal(r); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	return nil, errCanon
}
