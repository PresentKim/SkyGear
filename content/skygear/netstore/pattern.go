package netstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/recipe"
	"github.com/presentkim/skygear/common/i18n"
	"github.com/presentkim/skygear/common/storagenet"
	"github.com/presentkim/skygear/content/skygear/machinerecipe"
	"github.com/presentkim/skygear/content/skygear/naming"
)

// A pattern item carries its recipe in stack values: a crafting pattern the
// id of a crafting table recipe (machinerecipe.Crafter.ID), a processing
// pattern its input and output samples as "count:key;count:key" lists.
const (
	patKindKey   = "sg_pat_kind"
	patRecipeKey = "sg_pat_recipe"
	patInKey     = "sg_pat_in"
	patOutKey    = "sg_pat_out"

	kindCrafting   = "craft"
	kindProcessing = "proc"
)

// The most a processing pattern holds (spec §6).
const (
	MaxProcessingIn  = 9
	MaxProcessingOut = 3
)

// keyOf makes the key of a sample; a variable so tests can swap it.
var keyOf = KeyOf

// EncodeCrafting writes the crafting table recipe crafterID on a pattern. A
// stack that is no pattern, or a recipe that does not exist, is returned
// unchanged.
func EncodeCrafting(st item.Stack, crafterID string) item.Stack {
	cr := machinerecipe.CrafterByID(crafterID)
	if _, ok := st.Item().(Pattern); !ok || cr == nil {
		return st
	}
	lang := i18n.Default
	return st.WithValue(patKindKey, kindCrafting).WithValue(patRecipeKey, crafterID).
		WithLore("§7" + fmt.Sprintf(i18n.T(lang, "netstore.pattern.crafting"), sampleName(lang, cr.Out)))
}

// EncodeProcessing writes a processing pattern: what goes into the machine
// and what comes out. Empty samples are skipped and the lists are cut at
// MaxProcessingIn and MaxProcessingOut; a pattern left with no input or no
// output is returned unchanged.
func EncodeProcessing(st item.Stack, in, out []item.Stack) item.Stack {
	if _, ok := st.Item().(Pattern); !ok {
		return st
	}
	in, out = samples(in, MaxProcessingIn), samples(out, MaxProcessingOut)
	if len(in) == 0 || len(out) == 0 {
		return st
	}
	lang := i18n.Default
	return st.WithValue(patKindKey, kindProcessing).WithValue(patInKey, joinSamples(in)).WithValue(patOutKey, joinSamples(out)).
		WithLore("§7" + fmt.Sprintf(i18n.T(lang, "netstore.pattern.processing"), listNames(lang, in), listNames(lang, out)))
}

// samples drops the empty stacks and keeps the first limit.
func samples(l []item.Stack, limit int) []item.Stack {
	var out []item.Stack
	for _, s := range l {
		if !s.Empty() && len(out) < limit {
			out = append(out, s)
		}
	}
	return out
}

func joinSamples(l []item.Stack) string {
	parts := make([]string, len(l))
	for i, s := range l {
		parts[i] = fmt.Sprintf("%d:%s", s.Count(), keyOf(s))
	}
	return strings.Join(parts, ";")
}

func sampleName(lang string, s item.Stack) string {
	return fmt.Sprintf("%s ×%d", naming.ItemName(lang, s), s.Count())
}

func listNames(lang string, l []item.Stack) string {
	names := make([]string, len(l))
	for i, s := range l {
		names[i] = sampleName(lang, s)
	}
	return strings.Join(names, ", ")
}

// parseSamples reads what joinSamples wrote.
func parseSamples(v string) ([]storagenet.Stack, bool) {
	var out []storagenet.Stack
	for _, part := range strings.Split(v, ";") {
		n, key, ok := strings.Cut(part, ":")
		count, err := strconv.ParseInt(n, 10, 64)
		if !ok || err != nil || count <= 0 || key == "" {
			return nil, false
		}
		out = append(out, storagenet.Stack{Key: storagenet.Key(key), Count: count})
	}
	return out, len(out) > 0
}

func stringValue(st item.Stack, key string) string {
	v, _ := st.Value(key)
	s, _ := v.(string)
	return s
}

// PatternOf reads the pattern a stack carries: false for a blank pattern, an
// item that is no pattern or a recipe that no longer exists. keysFor gives
// the keys of the items that fill a recipe slot that is a tag or takes any
// variant of an item; a slot of one exact item needs no help.
func PatternOf(st item.Stack, keysFor func(recipe.Item) []storagenet.Key) (storagenet.Pattern, bool) {
	if _, ok := st.Item().(Pattern); !ok {
		return storagenet.Pattern{}, false
	}
	switch stringValue(st, patKindKey) {
	case kindCrafting:
		id := stringValue(st, patRecipeKey)
		cr := machinerecipe.CrafterByID(id)
		if cr == nil {
			return storagenet.Pattern{}, false
		}
		p := storagenet.Pattern{ID: patternID(kindCrafting, id)}
		for _, req := range cr.Reqs {
			var opts []storagenet.Key
			if s, ok := req.Need.(item.Stack); ok && !hasVariants(s) {
				opts = []storagenet.Key{keyOf(s)}
			} else if keysFor != nil {
				opts = keysFor(req.Need)
			}
			if len(opts) == 0 {
				return storagenet.Pattern{}, false
			}
			p.Inputs = append(p.Inputs, storagenet.Ingredient{Count: int64(req.Count), Options: opts})
		}
		p.Outputs = []storagenet.Stack{{Key: keyOf(cr.Out), Count: int64(cr.Out.Count())}}
		return p, true
	case kindProcessing:
		in, ok1 := parseSamples(stringValue(st, patInKey))
		out, ok2 := parseSamples(stringValue(st, patOutKey))
		if !ok1 || !ok2 {
			return storagenet.Pattern{}, false
		}
		p := storagenet.Pattern{ID: patternID(kindProcessing, stringValue(st, patInKey)+">"+stringValue(st, patOutKey)), Processing: true, Outputs: out}
		for _, s := range in {
			p.Inputs = append(p.Inputs, storagenet.Ingredient{Count: s.Count, Options: []storagenet.Key{s.Key}})
		}
		return p, true
	}
	return storagenet.Pattern{}, false
}

func hasVariants(s item.Stack) bool {
	_, ok := s.Value("variants")
	return ok
}

// patternID is a hash of the encoded contents: the same recipe always has
// the same id, whatever pattern item it is on.
func patternID(kind, contents string) string {
	sum := sha256.Sum256([]byte(kind + "|" + contents))
	return hex.EncodeToString(sum[:8])
}
