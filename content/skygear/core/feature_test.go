package core

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unsafe"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/customblock"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/skygear/common/clientdef"
	"github.com/presentkim/skygear/common/sghook"
	"github.com/presentkim/skygear/platform/account"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type fakeType struct{ n string }

func (fakeType) Open(*world.Tx, *world.EntityHandle, *world.EntityData) world.Entity { return nil }
func (f fakeType) EncodeEntity() string                                              { return "core_test:" + f.n }
func (fakeType) BBox(world.Entity) cube.BBox                                         { return cube.BBox{} }
func (fakeType) DecodeNBT(map[string]any, *world.EntityData)                         {}
func (fakeType) EncodeNBT(*world.EntityData) map[string]any                          { return nil }

func TestInstallRunsSetupUnderName(t *testing.T) {
	ResetHooks()
	h := NewHook[int]("T")
	Install(
		&Feature{Name: "a", Setup: func() { h.On(10, "a.x", func(int) {}) }},
		&Feature{Name: "b", Setup: func() { h.On(20, "b.x", func(int) {}) }},
	)
	want := []Subscription{{Hook: "T", Prio: 10, Name: "a.x", Feature: "a"}, {Hook: "T", Prio: 20, Name: "b.x", Feature: "b"}}
	if got := Subscriptions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if installing != "" {
		t.Fatal("installing left set")
	}
	defer mustPanic(t, "b.x", "a")
	ResetHooks()
	Install(&Feature{Name: "a", Setup: func() { h.On(10, "b.x", func(int) {}) }})
}

func TestInstallTwicePanics(t *testing.T) {
	ResetHooks()
	Install(&Feature{Name: "a"})
	defer mustPanic(t, "twice")
	Install(&Feature{Name: "b"})
}

// TestInstallRunsChecksAfterEverySetup: a feature's Check runs when the Setup
// of every feature has run, in list order, so it sees what a feature listed
// after it provided.
func TestInstallRunsChecksAfterEverySetup(t *testing.T) {
	ResetHooks()
	var got []string
	Install(
		&Feature{Name: "a", Setup: func() { got = append(got, "setup a") }, Check: func() error { got = append(got, "check a"); return nil }},
		&Feature{Name: "b", Setup: func() { got = append(got, "setup b") }},
		&Feature{Name: "c", Check: func() error { got = append(got, "check c"); return nil }},
	)
	if want := []string{"setup a", "setup b", "check a", "check c"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// TestInstallCheckErrorPanics: Install panics with the failing feature's name
// and the error of its Check.
func TestInstallCheckErrorPanics(t *testing.T) {
	ResetHooks()
	defer mustPanic(t, `"a"`, "kind x has no behaviour")
	Install(&Feature{Name: "a", Check: func() error { return errors.New("kind x has no behaviour") }})
}

func TestDuplicateFeatureNamePanics(t *testing.T) {
	ResetHooks()
	ran := 0
	func() {
		defer mustPanic(t, `"a"`, "twice")
		Install(&Feature{Name: "a", Setup: func() { ran++ }}, &Feature{Name: "a"})
	}()
	if ran != 0 || len(Features()) != 0 {
		t.Fatal("a rejected list was partly installed")
	}
	func() {
		defer mustPanic(t, "no name")
		Install(&Feature{})
	}()
}

func TestEntityTypesInListOrder(t *testing.T) {
	ResetHooks()
	x, y, z := fakeType{"x"}, fakeType{"y"}, fakeType{"z"}
	Install(&Feature{Name: "a", Entities: []world.EntityType{x}}, &Feature{Name: "m"}, &Feature{Name: "b", Entities: []world.EntityType{y, z}})
	if got := EntityTypes(); !reflect.DeepEqual(got, []world.EntityType{x, y, z}) {
		t.Fatalf("got %v", got)
	}
	if n := len(Features()); n != 3 {
		t.Fatalf("%d features", n)
	}
}

func TestSettingsRegisteredInListOrder(t *testing.T) {
	ResetHooks()
	before := len(account.Settings())
	Install(
		&Feature{Name: "a", Settings: []account.Setting{{Key: "core_test.a1"}, {Key: "core_test.a2"}}},
		&Feature{Name: "b", Settings: []account.Setting{{Key: "core_test.b1"}}},
	)
	var got []string
	for _, s := range account.Settings()[before:] {
		got = append(got, s.Key)
	}
	if !slices.Equal(got, []string{"core_test.a1", "core_test.a2", "core_test.b1"}) {
		t.Fatalf("got %v", got)
	}
}

// setField sets the unexported field name of srv, as Listen builds it.
func setField[T any](srv *server.Server, name string, v T) {
	f := reflect.ValueOf(srv).Elem().FieldByName(name)
	*(*T)(unsafe.Pointer(f.UnsafeAddr())) = v
}

func entries(srv *server.Server) []protocol.ItemEntry {
	f := reflect.ValueOf(srv).Elem().FieldByName("customItems")
	return *(*[]protocol.ItemEntry)(unsafe.Pointer(f.UnsafeAddr()))
}

// TestItemEditsComposeInListOrder: clientdef.ItemEntry first, then each
// feature's ItemEntry in list order.
func TestItemEditsComposeInListOrder(t *testing.T) {
	ResetHooks()
	t.Cleanup(func() { sghook.EditCustomItems(nil, nil); sghook.EditCustomBlocks(nil, nil) })
	app := func(s string) func(protocol.ItemEntry) protocol.ItemEntry {
		return func(e protocol.ItemEntry) protocol.ItemEntry { e.Name += s; return e }
	}
	Install(&Feature{Name: "a", ItemEntry: app("-a")}, &Feature{Name: "n"}, &Feature{Name: "b", ItemEntry: app("-b")})
	srv := &server.Server{}
	EditEntries(srv)
	setField(srv, "customItems", []protocol.ItemEntry{{Name: "x"}})
	if err := CheckEntries(); err != nil {
		t.Fatal(err)
	}
	if got := entries(srv)[0].Name; got != "x-a-b" {
		t.Fatalf("got %q, want x-a-b", got)
	}
}

type badBlock struct{}

func (badBlock) EncodeBlock() (string, map[string]any) { return "core_test:badbone", nil }
func (badBlock) Hash() (uint64, uint64)                { return 0xc0de0000, 1 }
func (badBlock) Model() world.BlockModel               { return model.Empty{} }
func (badBlock) Properties() customblock.Properties    { return customblock.Properties{} }
func (badBlock) ClientExtras() (clientdef.Extras, map[string]clientdef.Extras) {
	return clientdef.Extras{BoneVisibility: map[string]string{"core": "false"}}, nil
}

func init() { world.RegisterBlock(badBlock{}) }

// TestCheckEntries: a block declaration clientdef.BlockEntry rejects (bones
// without geometry) comes back as an error naming the block. The block is
// registered in this test binary only and no entry of it is built unless the
// test builds one, so the other tests are unaffected.
func TestCheckEntries(t *testing.T) {
	ResetHooks()
	t.Cleanup(func() { sghook.EditCustomItems(nil, nil); sghook.EditCustomBlocks(nil, nil) })
	Install()
	entry := func(name string) protocol.BlockEntry {
		return protocol.BlockEntry{Name: name, Properties: map[string]any{
			"components": map[string]any{"minecraft:friction": map[string]any{"value": float32(0.5)}},
		}}
	}
	srv := &server.Server{}
	EditEntries(srv)
	setField(srv, "customBlocks", []protocol.BlockEntry{entry("minecraft:data_driven"), entry("core_test:badbone")})
	err := CheckEntries()
	if err == nil || !strings.Contains(err.Error(), "core_test:badbone") {
		t.Fatalf("err %v", err)
	}

	ResetHooks()
	sghook.EditCustomItems(nil, nil)
	sghook.EditCustomBlocks(nil, nil)
	Install()
	srv = &server.Server{}
	EditEntries(srv)
	setField(srv, "customBlocks", []protocol.BlockEntry{entry("minecraft:data_driven")})
	if err := CheckEntries(); err != nil {
		t.Fatal(err)
	}
}
