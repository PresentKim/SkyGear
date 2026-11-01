package ui

import (
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

// Screens that show something off rather than list it: big cards to pick
// one of, a result that comes in piece by piece, and an equipment screen laid
// out like the vanilla inventory.

// PickCard is a card of PickCards: an icon, a title, a text, a small tag
// over the title and a rarity ("common", "rare" or "epic") that colours it.
type PickCard struct {
	Icon, Title, Text, Tag, Rarity string
	Run                            func(p *player.Player, tx *world.Tx)
}

type dPickCard struct {
	ID     string `json:"id"`
	Icon   string `json:"icon"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Tag    string `json:"tag"`
	Rarity string `json:"rarity"`
}

// pickData is the data of the pick screen.
type pickData struct {
	Chrome dChrome     `json:"chrome"`
	Body   string      `json:"body"`
	Cards  []dPickCard `json:"cards"`
	Items  []dItem     `json:"items"`
}

// PickCards shows big cards side by side to pick one of, the body over them
// and the options under them.
func PickCards(p *player.Player, title, body string, cards []PickCard, opts ...Option) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(title, opts)
	d := pickData{Chrome: chrome, Body: body, Cards: []dPickCard{}, Items: l.itemsOf(rest)}
	for _, c := range cards {
		d.Cards = append(d.Cards, dPickCard{ID: l.id(c.Run), Icon: oreIcon(c.Icon), Title: c.Title, Text: c.Text, Tag: c.Tag, Rarity: c.Rarity})
	}
	showScreen(p, l, "pick", d)
}

// ResultLine is a line of a result: a label and its value.
type ResultLine struct{ Label, Value string }

// ResultScreen is what Result shows: a headline (a win in gold, else grey),
// lines of figures, three stars (New marks the ones just earned), the
// rewards as slots and the button that closes it.
type ResultScreen struct {
	Title, Headline string
	Win             bool
	Lines           []ResultLine
	Stars, New      [3]bool
	Rewards         []ItemSlot
	OK              Option
}

type dResultLine struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type dStar struct {
	On  bool `json:"on"`
	New bool `json:"new"`
}

// resultData is the data of the result screen.
type resultData struct {
	Chrome   dChrome       `json:"chrome"`
	Headline string        `json:"headline"`
	Win      bool          `json:"win"`
	Lines    []dResultLine `json:"lines"`
	Stars    []dStar       `json:"stars"`
	Rewards  []dSlot       `json:"rewards"`
	OK       dButton       `json:"ok"`
}

// Result shows the result of a game: the headline, the stars and the
// rewards come in one after another.
func Result(p *player.Player, r ResultScreen) {
	l := newOreLayout()
	chrome, _ := l.chromeOf(r.Title, nil)
	d := resultData{Chrome: chrome, Headline: r.Headline, Win: r.Win, Lines: []dResultLine{}, Stars: []dStar{}, Rewards: l.slotsOf(r.Rewards), OK: l.buttonOf(r.OK)}
	for _, ln := range r.Lines {
		d.Lines = append(d.Lines, dResultLine{ln.Label, ln.Value})
	}
	for i := range r.Stars {
		d.Stars = append(d.Stars, dStar{On: r.Stars[i], New: r.New[i]})
	}
	showScreen(p, l, "result", d)
}

// EquipSlot is a slot of an equipment screen: what is in it (an empty Item
// shows Empty, the slot's silhouette) and its name.
type EquipSlot struct {
	Item  ItemSlot
	Empty string
}

// EquipGroup is a column of equipment slots under a heading.
type EquipGroup struct {
	Heading string
	Slots   []EquipSlot
}

// EquipScreen is what Equip shows: the groups of equipment slots side by
// side, a note, the bag (what the player carries) under them and the
// options.
type EquipScreen struct {
	Title, Note string
	Groups      []EquipGroup
	BagHeading  string
	Bag         []ItemSlot
	Options     []Option
}

type dEquipSlot struct {
	dSlot
	Empty string `json:"empty"`
}

type dEquipGroup struct {
	Heading string       `json:"heading"`
	Slots   []dEquipSlot `json:"slots"`
}

// equipData is the data of the equip screen.
type equipData struct {
	Chrome     dChrome       `json:"chrome"`
	Note       string        `json:"note"`
	Groups     []dEquipGroup `json:"groups"`
	BagHeading string        `json:"bagHeading"`
	Bag        []dSlot       `json:"bag"`
	Items      []dItem       `json:"items"`
}

// Equip shows equipment slots laid out like the vanilla inventory, with the
// player's bag under them.
func Equip(p *player.Player, e EquipScreen) {
	l := newOreLayout()
	chrome, rest := l.chromeOf(e.Title, e.Options)
	d := equipData{Chrome: chrome, Note: e.Note, Groups: []dEquipGroup{}, BagHeading: e.BagHeading, Bag: l.slotsOf(e.Bag), Items: l.itemsOf(rest)}
	for _, g := range e.Groups {
		dg := dEquipGroup{Heading: g.Heading, Slots: []dEquipSlot{}}
		for _, s := range g.Slots {
			dg.Slots = append(dg.Slots, dEquipSlot{dSlot: l.slotOf(s.Item), Empty: oreIcon(s.Empty)})
		}
		d.Groups = append(d.Groups, dg)
	}
	showScreen(p, l, "equip", d)
}
