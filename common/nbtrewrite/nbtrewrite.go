// Package nbtrewrite passes the strings of saved data through a function:
// the tag names and string values of little endian NBT (with the values of
// Dragonfly item stacks) and the block palettes of LevelDB sub chunks.
// Everything else is copied byte for byte.
package nbtrewrite

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// ErrShort is the error of data that ends inside a tag.
var ErrShort = errors.New("nbt: unexpected end of data")

// NBT returns b, one or more root tags appended to each other, with every
// tag name and string value passed through rep.
func NBT(b []byte, rep func(string) string) ([]byte, error) {
	return NBTKeys(b, rep, rep)
}

// NBTKeys is NBT with the value keys of Dragonfly item stacks passed through
// key instead of rep: tag names and string values (also the string values of
// item stacks) go through rep.
func NBTKeys(b []byte, rep func(string) string, key func(string) string) ([]byte, error) {
	return NBTStacks(b, rep, key, nil)
}

// NBTStacks is NBTKeys with plain, if set, naming the item stacks (compounds
// whose "Name" string, before rep, plain accepts) that become plain: of the
// stack's tags only the name (through rep) and the tags that are not the
// stack's own (Count, Slot and the like) stay, with a Damage of 0; its "tag"
// (custom name, lore, damage, values, enchantments), "Damage" and "Block"
// are dropped. Stacks nested in other stacks are handled as well.
func NBTStacks(b []byte, rep func(string) string, key func(string) string, plain func(name string) bool) ([]byte, error) {
	r := &rewriter{in: b, rep: rep, key: key, plain: plain, out: make([]byte, 0, len(b))}
	for r.pos < len(r.in) {
		if err := r.root(); err != nil {
			return nil, err
		}
	}
	return r.out, nil
}

// roots rewrites n root tags at the start of b and returns the rewritten
// tags and the bytes after them.
func roots(b []byte, n int, rep func(string) string, block func(string) (string, map[string]any, bool)) (out, rest []byte, err error) {
	r := &rewriter{in: b, rep: rep, key: rep}
	for range n {
		start, outStart := r.pos, len(r.out)
		if err := r.root(); err != nil {
			return nil, nil, err
		}
		if block == nil {
			continue
		}
		if e, ok := replaceEntry(r.in[start:r.pos], block); ok {
			r.out = append(r.out[:outStart], e...)
		}
	}
	return r.out, b[r.pos:], nil
}

// replaceEntry returns the palette entry in with its name and states from
// block, if block names it.
func replaceEntry(in []byte, block func(string) (string, map[string]any, bool)) ([]byte, bool) {
	var m map[string]any
	if err := nbt.UnmarshalEncoding(in, &m, nbt.LittleEndian); err != nil {
		return nil, false
	}
	name, _ := m["name"].(string)
	nn, states, ok := block(name)
	if !ok {
		return nil, false
	}
	if states == nil {
		states = map[string]any{}
	}
	m["name"], m["states"] = nn, states
	e, err := nbt.MarshalEncoding(m, nbt.LittleEndian)
	if err != nil {
		return nil, false
	}
	return e, true
}

// IsSubChunkKey reports whether k is the key of a sub chunk: x, z (and a
// dimension) followed by '/' and the sub chunk y.
func IsSubChunkKey(k []byte) bool {
	return (len(k) == 10 || len(k) == 14) && k[len(k)-2] == '/'
}

// SubChunk rewrites the NBT block palettes of a disk sub chunk (versions 1,
// 8 and 9) and copies the indices as they are.
func SubChunk(b []byte, rep func(string) string) ([]byte, error) {
	return SubChunkBlocks(b, rep, nil)
}

// SubChunkBlocks is SubChunk with a hook for whole palette entries: an entry
// (a little endian compound {name, states, version}) whose name block
// accepts is replaced by the name and states block returns (its version and
// any other tag kept, rep not applied to it); other entries go through rep.
// The index words are never touched. A nil block is SubChunk.
func SubChunkBlocks(b []byte, rep func(string) string, block func(name string) (string, map[string]any, bool)) ([]byte, error) {
	if len(b) < 1 {
		return nil, ErrShort
	}
	out := []byte{b[0]}
	rest := b[1:]
	storages := 1
	switch b[0] {
	case 1:
	case 8, 9:
		head := 1
		if b[0] == 9 {
			head = 2
		}
		if len(rest) < head {
			return nil, ErrShort
		}
		storages = int(rest[0])
		out = append(out, rest[:head]...)
		rest = rest[head:]
	default:
		return nil, fmt.Errorf("sub chunk version %d", b[0])
	}
	for range storages {
		if len(rest) < 1 {
			return nil, ErrShort
		}
		bits := int(rest[0] >> 1)
		out = append(out, rest[0])
		rest = rest[1:]
		if bits == 0x7f {
			continue
		}
		words := 0
		if bits != 0 {
			words = 4096 / (32 / bits)
			if bits == 3 || bits == 5 || bits == 6 {
				words++
			}
		}
		if len(rest) < words*4 {
			return nil, ErrShort
		}
		out = append(out, rest[:words*4]...)
		rest = rest[words*4:]
		count := 1
		if bits != 0 {
			if len(rest) < 4 {
				return nil, ErrShort
			}
			count = int(int32(binary.LittleEndian.Uint32(rest)))
			out = append(out, rest[:4]...)
			rest = rest[4:]
		}
		palette, after, err := roots(rest, count, rep, block)
		if err != nil {
			return nil, err
		}
		out = append(out, palette...)
		rest = after
	}
	return append(out, rest...), nil
}

type rewriter struct {
	in  []byte
	pos int
	out []byte
	rep func(string) string
	key func(string) string
	// plain, if set, says which item names make a stack plain.
	plain func(string) bool
}

// stackTags are the tags of an item stack that a plain stack does not keep.
var stackTags = map[string]bool{"tag": true, "Damage": true, "Block": true}

func (r *rewriter) take(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.in) {
		return nil, ErrShort
	}
	b := r.in[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *rewriter) copyN(n int) error {
	b, err := r.take(n)
	if err != nil {
		return err
	}
	r.out = append(r.out, b...)
	return nil
}

func (r *rewriter) root() error {
	t, err := r.take(1)
	if err != nil {
		return err
	}
	r.out = append(r.out, t[0])
	if t[0] == 0 {
		return nil
	}
	name, err := r.str()
	if err != nil {
		return err
	}
	return r.payload(t[0], name, 0)
}

// str rewrites a string and returns it as it was.
func (r *rewriter) str() (string, error) {
	l, err := r.take(2)
	if err != nil {
		return "", err
	}
	s, err := r.take(int(binary.LittleEndian.Uint16(l)))
	if err != nil {
		return "", err
	}
	ns := r.rep(string(s))
	if len(ns) > 0xffff {
		return "", fmt.Errorf("nbt: string of %d bytes", len(ns))
	}
	r.out = binary.LittleEndian.AppendUint16(r.out, uint16(len(ns)))
	r.out = append(r.out, ns...)
	return string(s), nil
}

func (r *rewriter) length() (int, error) {
	l, err := r.take(4)
	if err != nil {
		return 0, err
	}
	r.out = append(r.out, l...)
	n := int(int32(binary.LittleEndian.Uint32(l)))
	if n < 0 {
		return 0, fmt.Errorf("nbt: negative length %d", n)
	}
	return n, nil
}

func (r *rewriter) payload(t byte, name string, depth int) error {
	if depth > 512 {
		return errors.New("nbt: nested too deep")
	}
	switch t {
	case 1:
		return r.copyN(1)
	case 2:
		return r.copyN(2)
	case 3, 5:
		return r.copyN(4)
	case 4, 6:
		return r.copyN(8)
	case 7:
		if name == "dragonflyData" {
			return r.stackValues()
		}
		n, err := r.length()
		if err != nil {
			return err
		}
		return r.copyN(n)
	case 8:
		_, err := r.str()
		return err
	case 9:
		et, err := r.take(1)
		if err != nil {
			return err
		}
		r.out = append(r.out, et[0])
		if name == "dragonflyData" && et[0] == 1 {
			return r.stackValues()
		}
		n, err := r.length()
		if err != nil {
			return err
		}
		for range n {
			if err := r.payload(et[0], "", depth+1); err != nil {
				return err
			}
		}
		return nil
	case 10:
		start := len(r.out)
		var kids []tagSpan
		stack := false
		for {
			begin := len(r.out) - start
			ct, err := r.take(1)
			if err != nil {
				return err
			}
			r.out = append(r.out, ct[0])
			if ct[0] == 0 {
				if stack {
					r.makePlain(start, kids)
				}
				return nil
			}
			cn, err := r.str()
			if err != nil {
				return err
			}
			if r.plain != nil && ct[0] == 8 && cn == "Name" && len(r.in)-r.pos >= 2 {
				n := int(binary.LittleEndian.Uint16(r.in[r.pos:]))
				if len(r.in)-r.pos >= 2+n {
					stack = r.plain(string(r.in[r.pos+2 : r.pos+2+n]))
				}
			}
			if err := r.payload(ct[0], cn, depth+1); err != nil {
				return err
			}
			kids = append(kids, tagSpan{cn, begin, len(r.out) - start})
		}
	case 11:
		n, err := r.length()
		if err != nil {
			return err
		}
		return r.copyN(n * 4)
	case 12:
		n, err := r.length()
		if err != nil {
			return err
		}
		return r.copyN(n * 8)
	case 0:
		return nil
	}
	return fmt.Errorf("nbt: unknown tag type %d", t)
}

// makePlain rewrites the compound written at r.out[start:], whose tags are
// kids and whose end tag is written, as a plain stack.
func (r *rewriter) makePlain(start int, kids []tagSpan) {
	body := slices.Clone(r.out[start:])
	r.out = r.out[:start]
	for _, k := range kids {
		if !stackTags[k.name] {
			r.out = append(r.out, body[k.begin:k.end]...)
		}
	}
	r.out = append(r.out, 2) // Damage, a short of 0
	r.out = binary.LittleEndian.AppendUint16(r.out, uint16(len("Damage")))
	r.out = append(r.out, "Damage"...)
	r.out = append(r.out, 0, 0, 0) // the value, the end tag
}

// tagSpan is a tag of a compound: its name and where its bytes are in the
// output, relative to the start of the compound.
type tagSpan struct {
	name       string
	begin, end int
}

// mapValue is Dragonfly's item stack value (item/nbt.go): the values are
// saved gob encoded, sorted by key, in dragonflyData, a byte array or a list
// of bytes. Both are a length and the bytes.
type mapValue struct {
	K string
	V any
}

// stackValues rewrites the keys and string values of dragonflyData. Values
// gob cannot decode here leave the array as it is.
func (r *rewriter) stackValues() error {
	l, err := r.take(4)
	if err != nil {
		return err
	}
	data, err := r.take(int(int32(binary.LittleEndian.Uint32(l))))
	if err != nil {
		return err
	}
	if nd, ok := rewriteStackValues(data, r.rep, r.key); ok {
		data = nd
	}
	r.out = binary.LittleEndian.AppendUint32(r.out, uint32(len(data)))
	r.out = append(r.out, data...)
	return nil
}

func rewriteStackValues(data []byte, rep, key func(string) string) ([]byte, bool) {
	var values []mapValue
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&values); err != nil {
		return nil, false
	}
	changed := false
	for i, v := range values {
		if k := key(v.K); k != v.K {
			values[i].K, changed = k, true
		}
		if s, ok := v.V.(string); ok {
			if ns := rep(s); ns != s {
				values[i].V, changed = ns, true
			}
		}
	}
	if !changed {
		return nil, false
	}
	sort.Slice(values, func(i, j int) bool { return values[i].K < values[j].K })
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(values); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}
