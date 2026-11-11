package hooktest

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Canon returns a text of v that depends only on its content: maps by
// sorted key, slices in order, struct fields by name (unexported ones
// too), pointers and interfaces followed, strings quoted, byte slices in
// hex and other scalars as type(value). Equal content gives equal text
// whatever the map order.
func Canon(v any) string { return CanonWith(v, nil) }

// FieldText is asked for every struct field CanonWith is about to write:
// owner is the struct's type, name the field's name and v its value. It
// returns the text to write in place of the value, or false to write the
// value as Canon does.
type FieldText func(owner reflect.Type, name string, v reflect.Value) (string, bool)

// CanonWith is Canon, but writes a struct field as field says where field
// says so (nil: nowhere). It is for a text that names what a number stands
// for, so the text does not change when only the numbering does.
func CanonWith(v any, field FieldText) string {
	var b strings.Builder
	canon(&b, reflect.ValueOf(v), 0, map[uintptr]bool{}, field)
	return b.String()
}

func canon(b *strings.Builder, v reflect.Value, depth int, seen map[uintptr]bool, field FieldText) {
	if !v.IsValid() {
		b.WriteString("nil")
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		p := v.Pointer()
		if seen[p] {
			b.WriteString("cycle")
			return
		}
		seen[p] = true
		canon(b, v.Elem(), depth, seen, field)
		delete(seen, p)
	case reflect.Interface:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		canon(b, v.Elem(), depth, seen, field)
	case reflect.Map:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		type kv struct {
			key string
			val reflect.Value
		}
		kvs := make([]kv, 0, v.Len())
		for it := v.MapRange(); it.Next(); {
			var kb strings.Builder
			canon(&kb, it.Key(), 0, seen, field)
			kvs = append(kvs, kv{keyText(it.Key(), kb.String()), it.Value()})
		}
		sort.Slice(kvs, func(i, j int) bool { return kvs[i].key < kvs[j].key })
		open(b, '{', len(kvs), depth)
		for _, e := range kvs {
			indent(b, depth+1)
			b.WriteString(e.key + ": ")
			canon(b, e.val, depth+1, seen, field)
			b.WriteByte('\n')
		}
		closing(b, '}', len(kvs), depth)
	case reflect.Struct:
		t := v.Type()
		names := make([]string, 0, t.NumField())
		idx := map[string]int{}
		for i := range t.NumField() {
			names = append(names, t.Field(i).Name)
			idx[t.Field(i).Name] = i
		}
		sort.Strings(names)
		open(b, '{', len(names), depth)
		for _, n := range names {
			indent(b, depth+1)
			b.WriteString(n + ": ")
			if text, ok := fieldText(field, t, n, v.Field(idx[n])); ok {
				b.WriteString(text)
			} else {
				canon(b, v.Field(idx[n]), depth+1, seen, field)
			}
			b.WriteByte('\n')
		}
		closing(b, '}', len(names), depth)
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			b.WriteString("nil")
			return
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			buf := make([]byte, v.Len())
			for i := range buf {
				buf[i] = byte(v.Index(i).Uint())
			}
			b.WriteString("bytes(" + hex.EncodeToString(buf) + ")")
			return
		}
		open(b, '[', v.Len(), depth)
		for i := range v.Len() {
			indent(b, depth+1)
			canon(b, v.Index(i), depth+1, seen, field)
			b.WriteByte('\n')
		}
		closing(b, ']', v.Len(), depth)
	case reflect.String:
		b.WriteString(strconv.Quote(v.String()))
	case reflect.Bool:
		fmt.Fprintf(b, "bool(%v)", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(b, "%s(%d)", v.Kind(), v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		fmt.Fprintf(b, "%s(%d)", v.Kind(), v.Uint())
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(b, "%s(%s)", v.Kind(), strconv.FormatFloat(v.Float(), 'g', -1, v.Type().Bits()))
	case reflect.Complex64, reflect.Complex128:
		fmt.Fprintf(b, "%s(%v)", v.Kind(), v.Complex())
	default: // func, chan, unsafe pointer: no content to compare
		b.WriteString(v.Kind().String())
	}
}

// fieldText is field's text for a struct field, if field is set and has one.
func fieldText(field FieldText, owner reflect.Type, name string, v reflect.Value) (string, bool) {
	if field == nil {
		return "", false
	}
	return field(owner, name, v)
}

// keyText is how a map key is written: strings bare, others as canon.
func keyText(k reflect.Value, text string) string {
	for k.Kind() == reflect.Interface && !k.IsNil() {
		k = k.Elem()
	}
	if k.Kind() == reflect.String {
		return k.String()
	}
	return strings.ReplaceAll(text, "\n", " ")
}

func open(b *strings.Builder, c byte, n, depth int) {
	b.WriteByte(c)
	if n > 0 {
		b.WriteByte('\n')
	}
}

func closing(b *strings.Builder, c byte, n, depth int) {
	if n > 0 {
		indent(b, depth)
	}
	b.WriteByte(c)
}

func indent(b *strings.Builder, depth int) { b.WriteString(strings.Repeat("  ", depth)) }
