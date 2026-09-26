package data

import "testing"

func TestEveryElementCanBeFoughtSomewhere(t *testing.T) {
	// Not a requirement the loader holds — a floor with no backdrop of its own falls back to the
	// default — but the catalog is authored with one place per element at least, and an element
	// losing its last one should be a decision rather than something nobody noticed.
	catalog := LoadBackgrounds()
	for _, element := range AffinityElements {
		found := false
		for _, b := range catalog {
			found = found || b.Element == element
		}
		if !found {
			t.Errorf("no backdrop is authored for %s", element)
		}
	}
}

func TestAFloorKeepsItsBackdrop(t *testing.T) {
	// The three rooms of a floor each ask; they have to get one answer, and the same run code has
	// to show the same place every time it is played.
	catalog := map[string]BackgroundData{
		"a": {BackgroundRecord: "a", Element: "fire", Art: "a"},
		"b": {BackgroundRecord: "b", Element: "fire", Art: "b"},
		"c": {BackgroundRecord: "c", Element: "fire", Art: "c"},
	}
	for seed := int64(0); seed < 50; seed++ {
		first := BackdropFor(catalog, "fire", seed, 3)
		for range 3 {
			if got := BackdropFor(catalog, "fire", seed, 3); got != first {
				t.Fatalf("seed %d floor 3 drew %q then %q", seed, first, got)
			}
		}
	}
}

func TestABackdropComesFromItsFloorsElement(t *testing.T) {
	catalog := map[string]BackgroundData{
		"hot":  {BackgroundRecord: "hot", Element: "fire", Art: "hot"},
		"cold": {BackgroundRecord: "cold", Element: "ice", Art: "cold"},
	}
	for seed := int64(0); seed < 50; seed++ {
		if got := BackdropFor(catalog, "ice", seed, 1); got != "cold" {
			t.Fatalf("an ice floor drew %q", got)
		}
	}
	if got := BackdropFor(catalog, "earth", 7, 1); got != DefaultBackgroundArt {
		t.Errorf("an element with no backdrop drew %q, want the default", got)
	}
}

func TestEveryPlaceIsSometimesChosen(t *testing.T) {
	// A pick that always landed on the same record would pass the two tests above and make every
	// backdrop after the first one a picture nobody sees.
	catalog := map[string]BackgroundData{
		"a": {BackgroundRecord: "a", Element: "fire", Art: "a"},
		"b": {BackgroundRecord: "b", Element: "fire", Art: "b"},
		"c": {BackgroundRecord: "c", Element: "fire", Art: "c"},
	}
	seen := map[string]bool{}
	for seed := int64(0); seed < 200; seed++ {
		seen[BackdropFor(catalog, "fire", seed, 1)] = true
	}
	if len(seen) != len(catalog) {
		t.Errorf("200 runs drew only %v", seen)
	}
}

func TestAnUndrawnBackdropDrawsTheDefault(t *testing.T) {
	if got := (BackgroundData{}).ArtKey(); got != DefaultBackgroundArt {
		t.Errorf("an empty Art drew %q", got)
	}
}
