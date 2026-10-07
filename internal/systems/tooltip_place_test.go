package systems

import (
	"image"
	"testing"
)

// A left tooltip stands beside its anchor, centered on it, and flips to the right when the screen
// has no room on the left.
func TestALeftTooltipStandsBesideItsAnchor(t *testing.T) {
	anchor := image.Rect(1500, 600, 1568, 668)
	at := tipLeftIn(1920, 1080, anchor, 200, 80)
	if at.X+200 != anchor.Min.X-tipGap {
		t.Errorf("the panel ends at x=%d, want %d left of the anchor at %d", at.X+200, tipGap, anchor.Min.X)
	}
	if mid := at.Y + 40; mid != (anchor.Min.Y+anchor.Max.Y)/2 {
		t.Errorf("the panel is centered at y=%d, want the anchor's %d", mid, (anchor.Min.Y+anchor.Max.Y)/2)
	}

	edge := image.Rect(20, 600, 88, 668)
	if at := tipLeftIn(1920, 1080, edge, 200, 80); at.X != edge.Max.X+tipGap {
		t.Errorf("against the left edge the panel starts at x=%d, want %d right of the anchor",
			at.X, edge.Max.X+tipGap)
	}
}
