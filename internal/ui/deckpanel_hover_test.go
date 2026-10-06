package ui

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The deck panel explains the card the cursor is on: every card's visible strip, hovered, names that
// card. **The hover reads the grid the drawing lays out**, and a hover measuring a grid of its own —
// wider, or starting further left — names a card some way off from the one under the cursor.
func TestTheDeckPanelExplainsTheCardUnderTheCursor(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: 1920, ScreenHeight: 1080}

	var deck []combat.Card
	concepts := []combat.ConceptID{combat.Bash, combat.Brace, combat.Block}
	for _, e := range []combat.Element{combat.Ice, combat.Lightning, combat.Earth} {
		for i := 0; i < 12; i++ {
			deck = append(deck, combat.Card{Concept: concepts[i%len(concepts)], Element: e})
		}
	}
	d := DeckContents{Draw: deck}
	var v DeckView

	centerX, width, top := deckGridRegion(gs)
	for i, slot := range d.grid(v, centerX, width, top).slots {
		at := image.Pt(slot.at.Min.X+1, (slot.at.Min.Y+slot.at.Max.Y)/2)
		var tip models.Tooltip
		HoverDeckPanel(gs, at, v, d, &tip)
		want, _ := CardTip(slot.card, HeldBy(d.Holder, slot.card))
		if got := tip.Title.Text(); got != TipLine(want).Text() || tip.Anchor != slot.at {
			t.Fatalf("slot %d at %v explained %q at %v, want %q at %v",
				i, at, got, tip.Anchor, want, slot.at)
		}
	}
}
