package nsmigrate

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"sort"
)

// rewriteNBT walks little endian NBT, one or more root tags appended to each
// other, and returns it with every tag name and string value passed through
// rep. Everything else is copied byte for byte.
func rewriteNBT(b []byte, rep func(string) string) ([]byte, error) {
	r := &nbtRewriter{in: b, rep: rep, out: make([]byte, 0, len(b))}
	for r.pos < len(r.in) {
		if err := r.root(); err != nil {
			return nil, err
		}
	}
	return r.out, nil
}

// rewriteNBTRoots rewrites n root tags at the start of b and returns the
// rewritten tags and the bytes after them.
func rewriteNBTRoots(b []byte, n int, rep func(string) string) (out, rest []byte, err error) {
	r := &nbtRewriter{in: b, rep: rep}
	for i := 0; i < n; i++ {
		if err := r.root(); err != nil {
			return nil, nil, err
		}
	}
	return r.out, b[r.pos:], nil
}

var errShort = errors.New("nbt: unexpected end of data")

type nbtRewriter struct {
	in  []byte
	pos int
	out []byte
	rep func(string) string
}

func (r *nbtRewriter) take(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.in) {
		return nil, errShort
	}
	b := r.in[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *nbtRewriter) copyN(n int) error {
	b, err := r.take(n)
	if err != nil {
		return err
	}
	r.out = append(r.out, b...)
	return nil
}

func (r *nbtRewriter) root() error {
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
func (r *nbtRewriter) str() (string, error) {
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

func (r *nbtRewriter) length() (int, error) {
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

func (r *nbtRewriter) payload(t byte, name string, depth int) error {
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
		for i := 0; i < n; i++ {
			if err := r.payload(et[0], "", depth+1); err != nil {
				return err
			}
		}
		return nil
	case 10:
		for {
			ct, err := r.take(1)
			if err != nil {
				return err
			}
			r.out = append(r.out, ct[0])
			if ct[0] == 0 {
				return nil
			}
			name, err := r.str()
			if err != nil {
				return err
			}
			if err := r.payload(ct[0], name, depth+1); err != nil {
				return err
			}
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

// mapValue is Dragonfly's item stack value (item/nbt.go): the values are
// saved gob encoded, sorted by key, in dragonflyData, a byte array or a list
// of bytes. Both are a length and the bytes.
type mapValue struct {
	K string
	V any
}

// stackValues rewrites the keys and string values of dragonflyData. Values gob cannot decode here leave the array as it is.
func (r *nbtRewriter) stackValues() error {
	l, err := r.take(4)
	if err != nil {
		return err
	}
	n := int(int32(binary.LittleEndian.Uint32(l)))
	data, err := r.take(n)
	if err != nil {
		return err
	}
	if nd, ok := rewriteStackValues(data, r.rep); ok {
		data = nd
	}
	r.out = binary.LittleEndian.AppendUint32(r.out, uint32(len(data)))
	r.out = append(r.out, data...)
	return nil
}

func rewriteStackValues(data []byte, rep func(string) string) ([]byte, bool) {
	if !hasOld(data) && !hasDFKey(data) {
		return nil, false
	}
	var values []mapValue
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&values); err != nil {
		return nil, false
	}
	for i, v := range values {
		values[i].K = replaceKey(v.K)
		if s, ok := v.V.(string); ok {
			values[i].V = rep(s)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].K < values[j].K })
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(values); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}
