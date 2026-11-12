package core

import "testing"

// TestNewFeatureNeedsOnlyTheList: a feature defined entirely in this file, with
// one subscriber on one real hook, is installed through Install (the call the
// root's feature list makes) and receives its event. Nothing else is touched:
// no handler, no dispatch table, no hook declaration. The subscription is
// recorded under the feature's name, which is what the root's guard tests
// check for every real subscriber.
func TestNewFeatureNeedsOnlyTheList(t *testing.T) {
	ResetHooks()
	defer ResetHooks()
	var got []PlayerEvent
	fake := &Feature{Name: "fake", Setup: func() {
		Jump.On(Normal, "fake.jump", func(e PlayerEvent) { got = append(got, e) })
	}}
	Install(fake)

	Jump.Emit(PlayerEvent{})
	if len(got) != 1 {
		t.Fatalf("the fake feature received %d events, want 1", len(got))
	}
	subs := Subscriptions()
	if len(subs) != 1 || subs[0] != (Subscription{Hook: "Jump", Name: "fake.jump", Feature: "fake", Tier: Normal}) {
		t.Fatalf("subscriptions %v", subs)
	}
	if fs := Features(); len(fs) != 1 || fs[0] != fake {
		t.Fatalf("features %v", fs)
	}
}
