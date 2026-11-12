// Package nbtcanon is the canonical binary form of the NBT of one item, the
// form behind the keys of the storage network (netstore.KeyOf). The NBT
// encoder writes the entries of a compound in map order, so the same item
// encodes to different bytes; keys need one form per item. Both netstore (to
// make keys) and migrate (to rewrite the keys in saves) use this one codec.
package nbtcanon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sort"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// The form is a small tagged binary form of the decoded tree, with compounds
// sorted by key.

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

var errCanon = errors.New("nbtcanon: unsupported nbt value")

// Encode returns the canonical form of the little endian NBT of one item, or
// the NBT itself behind cRaw when it holds a value of a type the form lacks.
func Encode(enc []byte) []byte {
	var m map[string]any
	if err := nbt.NewDecoderWithEncoding(bytes.NewReader(enc), nbt.LittleEndian).Decode(&m); err == nil {
		var b bytes.Buffer
		if putVal(&b, m) == nil {
			return b.Bytes()
		}
	}
	return append([]byte{cRaw}, enc...)
}

// Decode is the NBT tree Encode wrote; raw reports the cRaw form, whose NBT
// bytes it returns instead.
func Decode(b []byte) (m map[string]any, raw []byte, err error) {
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

// NBT is the little endian NBT that the canonical form b stands for. For a
// form of the cRaw kind it is the NBT Encode was given; otherwise it is the
// NBT of the decoded tree (equal in meaning, not always in bytes).
func NBT(b []byte) ([]byte, error) {
	m, raw, err := Decode(b)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return raw, nil
	}
	return nbt.MarshalEncoding(m, nbt.LittleEndian)
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
