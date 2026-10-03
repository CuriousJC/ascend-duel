package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// A screen backdrop painted against a layout is a picture of where things stood when it was
// generated. `data` cannot import the layout to check it, so these hold each anchored record against
// the function that lays its screen out: a layout that moves fails here, which is the moment to
// update the record and regenerate the picture.

// anchorRects is every anchor on one record, by name, as rectangles.
func anchorRects(t *testing.T, backdrop string) map[string]image.Rectangle {
	t.Helper()
	rec, ok := data.LoadScreenArt()[backdrop]
	if !ok {
		t.Fatalf("screen_art.json has no %s", backdrop)
	}
	out := map[string]image.Rectangle{}
	for _, a := range rec.Anchors {
		out[a.Name] = image.Rect(a.X, a.Y, a.X+a.W, a.Y+a.H)
	}
	return out
}

func checkAnchors(t *testing.T, backdrop string, want map[string]image.Rectangle) {
	t.Helper()
	got := anchorRects(t, backdrop)
	if len(got) != len(want) {
		t.Errorf("%s has %d anchors, the layout has %d", backdrop, len(got), len(want))
	}
	for name, r := range want {
		if got[name] != r {
			t.Errorf("%s anchor %q is %v, the layout puts it at %v", backdrop, name, got[name], r)
		}
	}
}

func TestTheEssenceBackdropStandsWhereTheEssencesAre(t *testing.T) {
	prizes, _ := essenceRowSeats(testState(), 2)
	checkAnchors(t, "screen-reward", map[string]image.Rectangle{
		"left essence":  prizes[0],
		"right essence": prizes[1],
	})
}

func TestThePortalBackdropStandsWhereTheGatesAre(t *testing.T) {
	gs := testState()
	var s PortalScene
	s.Init(gs)
	want := map[string]image.Rectangle{}
	for i, side := range []string{"left", "right"} {
		want[side+" portal"] = portalGateRect(gs, i)
		want[side+" swirl"] = systems.SwirlBounds(s.take[i])
	}
	checkAnchors(t, "screen-portal", want)
}
