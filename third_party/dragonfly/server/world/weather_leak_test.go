package world

import "testing"

func TestWeatherChangedForgottenOnDelete(t *testing.T) {
	w := &World{}
	if !weatherChanged(w, true, false) || weatherChanged(w, true, false) {
		t.Fatal("first call must report a change, the repeat must not")
	}
	lastWeather.Delete(w)
	if !weatherChanged(w, true, false) {
		t.Fatal("after Delete the weather must be sent again")
	}
	lastWeather.Delete(w)
}
