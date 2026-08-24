package screens

import (
	"image"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// shippingCombos is the panel as a run actually meets it: the starting deck, in the hands of a
// duelist with the fighter's own DMG, so the example lines carry the arithmetic rather than the
// between-fights short form.
func shippingCombos() comboContents {
	return comboContents{
		deck:   session.StartingDeck(),
		holder: combat.Duelist{DMG: 10, Actions: 6, MaxLife: 60, CurrentLife: 60},
	}
}

// The panel's own footprint at the internal resolution, which Layout fixes. Written here rather
// than imported because game imports screens and not the reverse.
func comboTestBody() (left, top, right, bottom int) {
	const screenW, screenH = 1280, 960
	pctX := func(p int) int { return screenW * p / 100 }
	pctY := func(p int) int { return screenH * p / 100 }
	r := comboBodyRect(image.Rect(
		pctX(modalPanelLeftPct), pctY(modalPanelTopPct),
		pctX(modalPanelRightPct), pctY(modalPanelBottomPct)))
	return r.Min.X, r.Min.Y, r.Max.X, r.Max.Y
}

// **The ladder is a ladder, and reading down it walks up.** The order is the panel's whole claim
// about which hands are worth aiming at, so a rung out of place is the panel arguing the opposite
// of what it says.
func TestTheLadderReadsCheapestPayingFirst(t *testing.T) {
	rows := comboRows(shippingCombos())
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

// **One term per card, never per group.** The grammar line is the rule the player is reading, and
// a Full House written as two terms would say it wants two cards.
func TestEveryRungHasATermPerCard(t *testing.T) {
	for _, h := range combat.Hands() {
		terms := strings.Count(comboGrammar(h), "+") + 1
		if terms != h.Cards() {
			t.Errorf("%s is written with %d terms for a %d-card hand: %q",
				h.Name, terms, h.Cards(), comboGrammar(h))
		}
	}
}

// **A group letter appears exactly when there is more than one group**, because that is the only
// time it says anything — and it is the whole rule for the hands that have two.
func TestOnlyAMultiGroupRungIsLettered(t *testing.T) {
	for _, h := range combat.Hands() {
		lettered := strings.Contains(comboGrammar(h), " A ")
		if want := len(h.Groups) > 1; lettered != want {
			t.Errorf("%s has %d groups and is lettered=%v: %q",
				h.Name, len(h.Groups), lettered, comboGrammar(h))
		}
	}
}

// **The shipping deck builds every rung but one**, and the one it cannot is a fact about the deck
// rather than about this panel: no attack concept ships more than its four elemental copies, so a
// Card Five of a Kind cannot be dealt at all. CLAUDE.md records the same arithmetic against
// tools/seeds.
//
// What this pins is that nothing *else* has quietly gone out of reach — a shipping deck that could
// not form half the ladder would be a ladder mostly made of apologies.
func TestTheShippingDeckBuildsEveryReachableRung(t *testing.T) {
	unreachable := map[string]bool{"Card Five of a Kind": true}

	for _, row := range comboRows(shippingCombos()) {
		cannot := strings.Contains(row.example, "cannot build")
		if cannot && !unreachable[row.name] {
			t.Errorf("the starting deck cannot build %s", row.name)
		}
		if !cannot && unreachable[row.name] {
			t.Errorf("%s is buildable now, so this test knows something untrue", row.name)
		}
	}
}

// **A holder with no DMG states the cards and stops.** A run's stats belong to a fight, so the
// shop and the reward screen have no strength to price a card at, and a sum worked out against a
// zero would be the panel inventing a figure.
func TestWithNoStrengthTheExampleDropsTheArithmetic(t *testing.T) {
	c := comboContents{deck: session.StartingDeck()}
	for _, row := range comboRows(c) {
		if strings.Contains(row.example, "=") {
			t.Errorf("%s prints arithmetic with no DMG known: %q", row.name, row.example)
		}
	}
	// The cheapest set is still named, and what it costs is still true without a duelist. A rung
	// the deck cannot build says so instead, which is the one line with no cost in it.
	for _, row := range comboRows(c) {
		if strings.Contains(row.example, "cannot build") {
			continue
		}
		if !strings.Contains(row.example, "AP") {
			t.Errorf("%s says nothing about what its example costs: %q", row.name, row.example)
		}
	}
}

// **Every line has to fit the column it is written in**, and this is what fails instead of the
// panel running its text off the edge. Measured against the real font at the real size, because
// the column budget is pixels and a character count is a guess.
func TestEveryComboLineFitsItsColumn(t *testing.T) {
	fonts := assets.LoadFonts()
	src := fonts["kubasta"]
	if src == nil {
		t.Fatal("no kubasta font to measure with")
	}

	left, _, right, _ := comboTestBody()
	body := image.Rect(left, 0, right, 0)
	width := comboColumnWidth(body, comboColumnCount)

	rows := comboRows(shippingCombos())
	widest := func(size float64, pick func(comboRow) string) (string, float64) {
		var worst string
		var w float64
		for _, row := range rows {
			adv, _ := text.Measure(pick(row), &text.GoTextFace{Source: src, Size: size}, 0)
			if adv > w {
				worst, w = pick(row), adv
			}
		}
		return worst, w
	}

	// The name and the multiplier share a line, from opposite ends, so they are measured together.
	if s, w := widest(comboNameSize, func(r comboRow) string { return r.name + "  " + r.mult }); int(w) > width {
		t.Errorf("the widest name line is %dpx against a %dpx column: %q", int(w), width, s)
	}
	if s, w := widest(comboGrammarSize, func(r comboRow) string { return r.grammar }); int(w) > width {
		t.Errorf("the widest grammar line is %dpx against a %dpx column: %q", int(w), width, s)
	}
	if s, w := widest(comboExampleSize, func(r comboRow) string { return r.example }); int(w) > width {
		t.Errorf("the widest example line is %dpx against a %dpx column: %q", int(w), width, s)
	}
}

// **Every rung is drawn, and none of them runs off the bottom.** The panel never hides a rung, for
// the reason the deck panel never hides a card: a catalogue with entries missing is worse than no
// catalogue, because nothing says which ones went.
func TestTheColumnsHoldTheWholeLadder(t *testing.T) {
	rows := comboRows(shippingCombos())
	columns := comboColumns(rows, comboColumnCount)

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
	if len(columns) > comboColumnCount {
		t.Errorf("the ladder wants %d columns against the %d the panel draws",
			len(columns), comboColumnCount)
	}

	_, top, _, bottom := comboTestBody()
	if tall := deepest*comboRowHeight - (comboRowHeight - comboExampleLine); tall > bottom-top {
		t.Errorf("the deepest column is %dpx against a %dpx budget (y=%d..%d)",
			tall, bottom-top, top, bottom)
	}
}

// **The columns are filled down and then across.** Snaking across two columns would interleave the
// cheap rungs with the dear ones, which is the one thing the multiplier order exists to prevent.
func TestTheColumnsAreFilledDownwards(t *testing.T) {
	rows := comboRows(shippingCombos())
	columns := comboColumns(rows, comboColumnCount)
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
