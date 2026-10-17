package dragonfactory

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/presentkim/dragonfactory/internal/i18n"
)

func hasLine(lines []string, part string) bool {
	for _, l := range lines {
		if strings.Contains(l, part) {
			return true
		}
	}
	return false
}

// TestCodexSourcesAndUses: the codex reads where an item comes from and what it
// is for from the tables of the game.
func TestCodexSourcesAndUses(t *testing.T) {
	s := testSession(t)
	lang := i18n.Default
	cases := []struct {
		it            item.Stack
		sources, uses []string
	}{
		{item.NewStack(OreChunk{Ore: OreIron}, 1), []string{"체"}, []string{"연구", "분쇄기"}},
		{item.NewStack(Ingot{Metal: MetalSteel}, 1), []string{"합금로"}, []string{"연구", "장비"}},
		{item.NewStack(Essence{Mob: MobZombie}, 1), []string{"체", "웨이브"}, []string{"몬스터 시뮬레이터"}},
		{item.NewStack(item.Bone{}, 1), []string{"몬스터 시뮬레이터"}, []string{"장비"}},
		{item.NewStack(item.Coal{}, 1), []string{"체"}, []string{"연료"}},
		{item.NewStack(block.Cobblestone{}, 1), []string{"분쇄기"}, []string{"연구", "분쇄기", "화로"}},
		{item.NewStack(MetalDust{Metal: MetalTin}, 1), []string{"광석 세척기"}, []string{"화로"}},
	}
	for _, c := range cases {
		src, use := s.codexSources(lang, c.it), s.codexUses(lang, c.it)
		for _, want := range c.sources {
			if !hasLine(src, want) {
				t.Errorf("%s: no source with %q in %q", itemName(c.it), want, src)
			}
		}
		for _, want := range c.uses {
			if !hasLine(use, want) {
				t.Errorf("%s: no use with %q in %q", itemName(c.it), want, use)
			}
		}
	}
}

// TestCodexPagesAreWritten: the sieve and gear pages have lines for every
// table entry.
func TestCodexPagesAreWritten(t *testing.T) {
	s := testSession(t)
	lang := i18n.Default
	if lines := s.codexProcessLines(lang); len(lines) < len(alloyRecipes)+4 {
		t.Errorf("process page: %d lines", len(lines))
	}
	if lines := s.codexMobLines(lang); len(lines) < len(mobKinds)+len(waveLevels) {
		t.Errorf("mob page: %d lines", len(lines))
	}
	if lines := codexGearLines(lang); !hasLine(lines, "날카로움") || !hasLine(lines, "청동") {
		t.Errorf("gear page: %q", lines)
	}
	for _, in := range codexSieveInputs() {
		for _, tier := range meshTiers {
			if len(sieveTables[tier.tier][itemKey(in)]) == 0 {
				t.Errorf("no drops of %s in the %s table", itemKey(in), tier.tier)
			}
		}
	}
}

// TestCodexKnowsTheLab: the codex lists where an essence is made, what the
// ingredients are for and the recipes with the process tables.
func TestCodexKnowsTheLab(t *testing.T) {
	s := testSession(t)
	essence := item.NewStack(ResearchEssence{EssenceHunting}, 1)
	if src := strings.Join(s.codexSources(i18n.Default, essence), "\n"); !strings.Contains(src, "연구소") || !strings.Contains(src, "화약 ×2") {
		t.Errorf("sources %q", src)
	}
	if uses := strings.Join(s.codexUses(i18n.Default, item.NewStack(item.SpiderEye{}, 1)), "\n"); !strings.Contains(uses, "사냥의 정수") {
		t.Errorf("uses %q", uses)
	}
	if lines := strings.Join(s.codexProcessLines(i18n.Default), "\n"); !strings.Contains(lines, "마법의 정수 ×1") {
		t.Errorf("process %q", lines)
	}
}

// TestCodexSearchAndLinks: the index holds the items the codex can explain,
// search finds them by name (initials too), and an item links to what it is
// made from and made into.
func TestCodexSearchAndLinks(t *testing.T) {
	testSession(t)
	registerCreativeItems()
	steel := Ingot{Metal: MetalSteel}
	steelName := itemName(item.NewStack(steel, 1))
	found := false
	for _, e := range codexFind(steelName) {
		found = found || itemKey(e.stack.Item()) == itemKey(steel)
	}
	if !found {
		t.Fatalf("searching %q does not find the steel ingot (%d entries in the index)", steelName, len(codexIndex()))
	}
	if n := len(codexFind("")); n < 50 {
		t.Errorf("an empty search lists %d items, want the whole index", n)
	}
	if n := len(codexFind("ㅎㅎㅎㅎㅎㅎㅎㅎㅎㅎㅎ")); n != 0 {
		t.Errorf("a nonsense search found %d", n)
	}
	from, into := codexLinks(item.NewStack(steel, 1))
	has := func(list []codexLink, w world.Item) bool {
		for _, l := range list {
			if itemKey(l.stack.Item()) == itemKey(w) {
				return true
			}
		}
		return false
	}
	if !has(from, item.IronIngot{}) || !has(from, item.Coal{}) {
		t.Errorf("steel is made from iron and coal, links: %v", from)
	}
	for _, l := range from {
		if itemKey(l.stack.Item()) == itemKey(item.Coal{}) && l.count != 4 {
			t.Errorf("steel takes 4 coal, the link says %d", l.count)
		}
	}
	if has(from, steel) || has(into, steel) {
		t.Error("an item links to itself")
	}
	if _, i := codexLinks(item.NewStack(block.Cobblestone{}, 1)); !has(i, block.Gravel{}) {
		t.Errorf("cobblestone is crushed into gravel, links: %v", i)
	}
}
