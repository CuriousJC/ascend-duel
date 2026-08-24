package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// shippingHands is the panel as a run actually meets it: the starting deck, in the hands of a
// duelist with the fighter's own DMG, so the example lines carry the arithmetic rather than the
// between-fights short form.
func shippingHands() handsContents {
	return handsContents{
		deck:   session.StartingDeck(),
		holder: combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60},
	}
}

// The panel's own footprint at the internal resolution, which Layout fixes. Written here rather
// than imported because game imports screens and not the reverse.
func handsTestBody() (left, top, right, bottom int) {
	const screenW, screenH = 1280, 960
	pctX := func(p int) int { return screenW * p / 100 }
	pctY := func(p int) int { return screenH * p / 100 }
	r := handsBodyRect(image.Rect(
		pctX(modalPanelLeftPct), pctY(modalPanelTopPct),
		pctX(modalPanelRightPct), pctY(modalPanelBottomPct)))
	return r.Min.X, r.Min.Y, r.Max.X, r.Max.Y
}

// **The ladder is a ladder, and reading down it walks up.** The order is the panel's whole claim
// about which hands are worth aiming at, so a rung out of place is the panel arguing the opposite
// of what it says.
func TestTheLadderReadsCheapestPayingFirst(t *testing.T) {
	rows := handsRows(shippingHands())
	if len(rows) != len(combat.Hands()) {
		t.Fatalf("the panel draws %d rungs against a catalogue of %d",
			len(rows), len(combat.Hands()))
	}

	hands := combat.Hands()
	byName := map[string]combat.Hand{}
	for _, h := range hands {
		byName[h.Name] = h
	}

	last := 0
	for _, row := range rows {
		h, ok := byName[row.name]
		if !ok {
			t.Fatalf("the panel drew a rung named %q, which is in no catalogue", row.name)
		}
		if h.Multiplier < last {
			t.Errorf("%s pays %d, under the %d of the rung above it", row.name, h.Multiplier, last)
		}
		last = h.Multiplier
	}
}

// **The panel reads the same whether or not a duelist is holding the deck.** It stopped saying
// anything a strength could change when the AP-and-damage line went on 2026-08-24 — what is left
// is the rung, the cards and the multiplier, all three facts about the deck rather than about a
// fight. This is what fails if a figure that needs a DMG comes back onto the row, since the shop
// and the reward screen have no strength to work one out against.
func TestThePanelSaysNothingAStrengthCouldChange(t *testing.T) {
	c := handsContents{deck: session.StartingDeck()}
	with := handsRows(shippingHands())
	without := handsRows(c)
	if len(with) != len(without) {
		t.Fatalf("%d rungs with a holder against %d without", len(with), len(without))
	}
	for i := range with {
		if len(with[i].cards) != len(without[i].cards) {
			t.Errorf("%s is illustrated with %d cards in a fight and %d out of one",
				with[i].name, len(with[i].cards), len(without[i].cards))
		}
		if with[i].mult != without[i].mult {
			t.Errorf("%s pays %q in a fight and %q out of one",
				with[i].name, with[i].mult, without[i].mult)
		}
	}
}

// **Every rung has to fit the column it is drawn in**, and this is what fails instead of the
// panel running its name off the edge or its cards into the column beside it. Text is measured
// against the real font at the real size, because the column budget is pixels and a character
// count is a guess.
func TestEveryHandRowFitsItsColumn(t *testing.T) {
	fonts := assets.LoadFonts()
	src := fonts["kubasta"]
	if src == nil {
		t.Fatal("no kubasta font to measure with")
	}

	left, _, right, _ := handsTestBody()
	body := image.Rect(left, 0, right, 0)
	width := handsColumnWidth(body, handsColumnCount)

	for _, row := range handsRows(shippingHands()) {
		adv, _ := text.Measure(row.name, &text.GoTextFace{Source: src, Size: handsNameSize}, 0)
		if int(adv) > width {
			t.Errorf("%s: the name is %dpx against a %dpx column", row.name, int(adv), width)
		}

		// The cards start at the column's left edge and the multiplier stands beside the last of
		// them, so what has to fit is the row and the figure with their gap.
		mult, _ := text.Measure(row.mult, &text.GoTextFace{Source: src, Size: handsMultSize}, 0)
		span := handsCardsWidth(len(row.cards)) + handsMultGap + int(mult)
		if span > width {
			t.Errorf("%s: %d cards and %q come to %dpx against a %dpx column",
				row.name, len(row.cards), row.mult, span, width)
		}
	}
}

// **Every rung is drawn, and none of them runs off the bottom.** The panel never hides a rung, for
// the reason the deck panel never hides a card: a catalogue with entries missing is worse than no
// catalogue, because nothing says which ones went.
func TestTheColumnsHoldTheWholeLadder(t *testing.T) {
	rows := handsRows(shippingHands())
	columns := handsColumns(rows, handsColumnCount)

	drawn := 0
	deepest := 0
	for _, column := range columns {
		drawn += len(column)
		if len(column) > deepest {
			deepest = len(column)
		}
	}
	if drawn != len(rows) {
		t.Errorf("%d of %d rungs are drawn", drawn, len(rows))
	}
	if len(columns) > handsColumnCount {
		t.Errorf("the ladder wants %d columns against the %d the panel draws",
			len(columns), handsColumnCount)
	}

	_, top, _, bottom := handsTestBody()
	if tall := deepest*handsRowHeight - (handsRowHeight - handsRuleDrop); tall > bottom-top {
		t.Errorf("the deepest column is %dpx against a %dpx budget (y=%d..%d)",
			tall, bottom-top, top, bottom)
	}
}

// **The columns are filled down and then across.** Snaking across two columns would interleave the
// cheap rungs with the dear ones, which is the one thing the multiplier order exists to prevent.
func TestTheColumnsAreFilledDownwards(t *testing.T) {
	rows := handsRows(shippingHands())
	columns := handsColumns(rows, handsColumnCount)
	if len(columns) < 2 {
		t.Skip("one column, so there is nothing to interleave")
	}
	if columns[0][0].name != rows[0].name {
		t.Errorf("the first column starts with %q rather than the cheapest rung %q",
			columns[0][0].name, rows[0].name)
	}
	if last := columns[0][len(columns[0])-1]; columns[1][0].name == last.name {
		t.Errorf("the second column repeats %q", last.name)
	}
}
