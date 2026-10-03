package state

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// TestEveryScreenArtNamesAScreen holds data/screen_art.json against the screens that exist. `data`
// cannot import this package, so a misspelled screen would otherwise be a picture nothing ever
// stands behind.
func TestEveryScreenArtNamesAScreen(t *testing.T) {
	names := map[string]bool{}
	for s := Title; s.String() != "Unknown"; s++ {
		names[s.String()] = true
	}
	art := data.LoadScreenArt()
	for _, key := range data.ScreenArtOrder(art) {
		for _, screen := range art[key].Screens {
			if !names[screen] {
				t.Errorf("screen_art.json: %s names %q, which is no screen", key, screen)
			}
		}
	}
}
