// Package wintext writes the texts of machine window items in each client's
// language.
package wintext

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/chestui"
	"github.com/presentkim/skygear/common/i18n"
	_ "github.com/presentkim/skygear/internal/locales" // this server's texts
	"github.com/presentkim/skygear/internal/skygear/naming"
)

// The items of machine windows (progress, energy, info, buttons, the recipe
// slot) are in the machine's inventory, which everyone who opens the window
// shares. They carry their name and lore as wtexts (catalog keys with their
// arguments) in an item value; chestui.Localize writes them in each client's
// language into the window packets sent to it, and the item's own name and
// lore are the default language's.

// Text is a text made of parts, written out in a language with in.
type Text []Part

// Part is a part of a wtext: plain text, a catalog text with its arguments,
// or an item's name.
type Part struct {
	S    string `json:"s,omitempty"`
	Key  string `json:"k,omitempty"`
	Args []Arg  `json:"a,omitempty"`
	// Item and Meta name an item (EncodeItem) whose name the part is.
	Item string `json:"i,omitempty"`
	Meta int16  `json:"m,omitempty"`
}

// Arg is an argument of a catalog text: a number, a string or a wtext.
type Arg struct {
	I *int64   `json:"i,omitempty"`
	F *float64 `json:"f,omitempty"`
	S *string  `json:"s,omitempty"`
	T Text     `json:"t,omitempty"`
}

// Raw is a part of plain text.
func Raw(s string) Part { return Part{S: s} }

// Key is a part of the catalog text of key formatted with args (int,
// float64, string, wtext or wpart).
func Key(key string, args ...any) Part {
	p := Part{Key: key}
	for _, a := range args {
		var w Arg
		switch a := a.(type) {
		case int:
			n := int64(a)
			w.I = &n
		case int64:
			w.I = &a
		case float64:
			w.F = &a
		case string:
			w.S = &a
		case Text:
			w.T = a
		case Part:
			w.T = Text{a}
		default:
			panic(fmt.Sprintf("wkey %s: argument of type %T", key, a))
		}
		p.Args = append(p.Args, w)
	}
	return p
}

// Item is a part that is the name of an item.
func Item(st item.Stack) Part {
	name, meta := st.Item().EncodeItem()
	return Part{Item: name, Meta: meta}
}

// in writes the text out in lang.
func (t Text) in(lang string) string {
	var b strings.Builder
	for _, p := range t {
		switch {
		case p.Key != "":
			args := make([]any, len(p.Args))
			for i, a := range p.Args {
				switch {
				case a.I != nil:
					args[i] = *a.I
				case a.F != nil:
					args[i] = *a.F
				case a.S != nil:
					args[i] = *a.S
				default:
					args[i] = a.T.in(lang)
				}
			}
			b.WriteString(i18n.T(lang, p.Key, args...))
		case p.Item != "":
			if it, ok := world.ItemByName(p.Item, p.Meta); ok {
				b.WriteString(naming.ItemName(lang, item.NewStack(it, 1)))
			} else {
				b.WriteString(p.Item)
			}
		default:
			b.WriteString(p.S)
		}
	}
	return b.String()
}

// ValueKey is the item value that holds the texts of a window item.
const ValueKey = "sg_window_text"

// Texts are the name and lore of a window item.
type Texts struct {
	Name Text   `json:"n"`
	Lore []Text `json:"l,omitempty"`
}

// With gives a window item its name and lore: written in the
// default language on the item, and kept as texts for chestui.Localize.
func With(st item.Stack, name Text, lore ...Text) item.Stack {
	raw, err := json.Marshal(Texts{Name: name, Lore: lore})
	if err != nil {
		panic(err)
	}
	st = st.WithCustomName(name.in(i18n.Default)).WithValue(ValueKey, string(raw))
	if len(lore) == 0 {
		return st
	}
	lines := make([]string, len(lore))
	for i, l := range lore {
		lines[i] = l.in(i18n.Default)
	}
	return st.WithLore(lines...)
}

// Localize is chestui.Localize: the texts of a window item in lang.
func Localize(lang string, st item.Stack) (string, []string, bool) {
	v, ok := st.Value(ValueKey)
	if !ok {
		return "", nil, false
	}
	raw, ok := v.(string)
	if !ok {
		return "", nil, false
	}
	var t Texts
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return "", nil, false
	}
	lore := make([]string, len(t.Lore))
	for i, l := range t.Lore {
		lore[i] = l.in(lang)
	}
	return t.Name.in(lang), lore, true
}

func init() { chestui.Localize = Localize }
