package waila

import (
	"slices"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

// described is a block with a panel of its own.
type described struct{ block.Stone }

func (described) Waila(v View, info Info) Info {
	info.Lines = append(info.Lines, "own:"+v.Lang)
	return info
}

// withAdapters runs f with only the given adapters registered.
func withAdapters(t *testing.T, as []Adapter, f func()) {
	t.Helper()
	saved := adapters
	t.Cleanup(func() { adapters = saved })
	adapters = nil
	for _, a := range as {
		RegisterAdapter(a)
	}
	f()
}

func appendLine(line string) Adapter {
	return func(_ View, _ world.Block, info Info) Info {
		info.Lines = append(info.Lines, line)
		return info
	}
}

// TestDescribeProviderThenAdapters: a block's own Waila runs first, then the
// adapters in the order they were registered.
func TestDescribeProviderThenAdapters(t *testing.T) {
	withAdapters(t, []Adapter{appendLine("a1"), appendLine("a2")}, func() {
		got := Describe(View{Lang: "ko"}, described{}, Info{Name: "n"})
		if want := []string{"own:ko", "a1", "a2"}; !slices.Equal(got.Lines, want) || got.Name != "n" {
			t.Errorf("lines %q name %q, want %q and n", got.Lines, got.Name, want)
		}
	})
}

// TestDescribePlainBlock: a block that is no Provider gets only the adapters,
// and with none its panel is unchanged.
func TestDescribePlainBlock(t *testing.T) {
	withAdapters(t, nil, func() {
		got := Describe(View{}, block.Stone{}, Info{Name: "n", Lines: []string{"x"}})
		if got.Name != "n" || !slices.Equal(got.Lines, []string{"x"}) {
			t.Errorf("panel %+v changed", got)
		}
	})
	withAdapters(t, []Adapter{appendLine("a1")}, func() {
		if got := Describe(View{}, block.Stone{}, Info{}); !slices.Equal(got.Lines, []string{"a1"}) {
			t.Errorf("lines %q, want the adapter's", got.Lines)
		}
	})
}

// TestDescribeAllocatesNothing: asking a block and the adapters allocates
// nothing by itself.
func TestDescribeAllocatesNothing(t *testing.T) {
	keep := func(_ View, _ world.Block, info Info) Info { return info }
	withAdapters(t, []Adapter{keep, keep}, func() {
		var b world.Block = block.Stone{}
		info := Info{Name: "n"}
		if n := testing.AllocsPerRun(200, func() { info = Describe(View{}, b, info) }); n != 0 {
			t.Errorf("Describe allocates %v times, want 0", n)
		}
	})
}
