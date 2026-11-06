package ui

import (
	"github.com/df-mc/dragonfly/server/player"
)

// Stepper is one column of a StepperScreen: an up arrow, its label and
// value, and a down arrow. A nil run leaves its arrow dim.
type Stepper struct {
	Label, Value string
	Up, Down     Option
}

// StepperScreen is a small window of steppers in a row, as the Java Edition
// mods set numbers (Item Collectors' range): a line over them and square
// buttons right of them.
type StepperScreen struct {
	Title, Line string
	Steppers    []Stepper
	Side        []Option
	Options     []Option
}

type dStepper struct {
	Label string     `json:"label"`
	Value string     `json:"value"`
	Up    dBarButton `json:"up"`
	Down  dBarButton `json:"down"`
}

type stepperData struct {
	Chrome   dChrome      `json:"chrome"`
	Line     string       `json:"line"`
	Steppers []dStepper   `json:"steppers"`
	Side     []dBarButton `json:"side"`
}

// ShowSteppers shows p a StepperScreen.
func ShowSteppers(p *player.Player, s StepperScreen) {
	l := newOreLayout()
	l.size = "stepper"
	chrome, _ := l.chromeOf(s.Title, s.Options)
	button := func(o Option) dBarButton {
		return dBarButton{ID: l.id(o.Run), Tip: untoken(o.Label), Icon: oreIcon(o.Icon), Quick: true}
	}
	d := stepperData{Chrome: chrome, Line: s.Line, Steppers: []dStepper{}, Side: []dBarButton{}}
	for _, st := range s.Steppers {
		d.Steppers = append(d.Steppers, dStepper{Label: st.Label, Value: st.Value, Up: button(st.Up), Down: button(st.Down)})
	}
	for _, o := range s.Side {
		d.Side = append(d.Side, button(o))
	}
	showScreen(p, l, "stepper", d)
}
