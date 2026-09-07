package screens

import (
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
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
	const screenW, screenH = state.ScreenWidth, state.ScreenHeight
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
		if len(with[i].sets) != len(without[i].sets) {
			t.Errorf("%s is illustrated with %d sets in a fight and %d out of one",
				with[i].name, len(with[i].sets), len(without[i].sets))
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
		title := handsTitleText(row)
		adv, _ := text.Measure(title, &text.GoTextFace{Source: src, Size: handsNameSize}, 0)
		if int(adv) > width {
			t.Errorf("%s: the title line is %dpx against a %dpx column", row.name, int(adv), width)
		}

		// The cards start at the column's left edge, the multiplier stands beside the last of them
		// and the plays count is right-aligned at the column's right edge, so what has to fit
		// across the band is all three and the air between them.
		mult, _ := text.Measure(row.mult, &text.GoTextFace{Source: src, Size: handsMultSize}, 0)
		tally, _ := text.Measure(handsTallyText(row),
			&text.GoTextFace{Source: src, Size: handsTallySize}, 0)
		span := handsSetsWidth(row.sets) + handsMultGap + int(mult) + handsMultGap + int(tally)
		if span > width {
			t.Errorf("%s: %d sets, %q and %q come to %dpx against a %dpx column",
				row.name, len(row.sets), row.mult, handsTallyText(row), span, width)
		}
	}
}

// **A rung read on more than one axis is illustrated on every one of them.** The Pair is
// `"match": "any"` and fires on whichever of concept, form and element the turn satisfies; a single
// example is a picture of a rung that counts one thing, which is exactly the reading the merge was
// made to remove.
func TestAMergedRungIsDrawnOnEveryAxisItReads(t *testing.T) {
	byName := map[string]combat.Hand{}
	for _, h := range combat.Hands() {
		byName[h.Name] = h
	}

	merged := 0
	for _, row := range handsRows(shippingHands()) {
		h := byName[row.name]
		want := len(h.Axes)
		if want < 2 {
			want = 1
		} else {
			merged++
		}
		if len(row.sets) != want {
			t.Errorf("%s is read on %d axes and drawn with %d examples",
				row.name, want, len(row.sets))
		}
		for i, set := range row.sets {
			if len(set) != h.Cards() {
				t.Errorf("%s: example %d holds %d cards against a rung of %d",
					row.name, i, len(set), h.Cards())
			}
		}
		if want > 1 && len(row.axes) != want {
			t.Errorf("%s carries %d captions for %d examples", row.name, len(row.axes), want)
		}
		if want == 1 && len(row.axes) != 0 {
			t.Errorf("%s names its own axis and should carry no caption, got %v",
				row.name, row.axes)
		}
	}
	if merged == 0 {
		t.Fatal("no rung in the catalogue is read on more than one axis, so nothing was checked")
	}
}

// **A merged rung's examples really are that rung, read one axis at a time.** They come out of
// `decks.Example` through `Hand.On`, so each set is a hand the matcher would score — this is what
// fails if the panel ever starts inventing an illustration of its own.
func TestEachOfAMergedRungsExamplesMatchesOnItsOwnAxis(t *testing.T) {
	for _, row := range handsRows(shippingHands()) {
		for i, axis := range row.axes {
			seen := map[int]bool{}
			for _, card := range row.sets[i] {
				v, ok := combat.MatchValue(card, axis)
				if !ok {
					t.Errorf("%s: the %s example holds a card carrying no value on that axis",
						row.name, axis)
					continue
				}
				seen[v] = true
			}
			if len(seen) != 1 {
				t.Errorf("%s: the %s example spreads over %d values, so it is not that pair",
					row.name, axis, len(seen))
			}
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
	// **Measured rather than multiplied out.** A rung carrying axis captions is taller than the
	// rest, so a pitch times a count is the wrong arithmetic: it would report a column that fits
	// while the panel drew it through its own bottom edge.
	for i, column := range columns {
		if tall := handsColumnDepth(column); tall > bottom-top {
			t.Errorf("column %d is %d rungs and %dpx against a %dpx budget (y=%d..%d)",
				i, len(column), tall, bottom-top, top, bottom)
		}
	}
	if deepest == 0 {
		t.Error("no column holds a rung")
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

// **A rung's cards have to fit inside the rung.** The panel's geometry was three constants written
// out by hand against a token that has since grown to 70px tall, so every row's cards overhung its
// own hairline and landed on the name of the rung below it — nineteen collisions, none of which any
// test could see. The measurements are derived off `cards.Token` now, and this is what fails if one
// is written down again.
func TestARungsCardsFitBetweenItsNameAndItsRule(t *testing.T) {
	for _, row := range handsRows(shippingHands()) {
		cardsTop, ruleDrop := handsCardsTopFor(row), handsRuleDropFor(row)
		if cardsTop < handsNameSize {
			t.Errorf("%s: the cards start at %dpx and the name is %dpx tall",
				row.name, cardsTop, handsNameSize)
		}
		if len(row.axes) > 1 && cardsTop-handsAxisBand < handsNameSize {
			t.Errorf("%s: the axis captions land on the name", row.name)
		}
		if bottom := cardsTop + cards.Token.Height; bottom > ruleDrop {
			t.Errorf("%s: the cards run to %dpx and the rule closing the rung is at %d",
				row.name, bottom, ruleDrop)
		}
		// The next rung's name is drawn at its own top, so the depth has to clear the rule by at
		// least the air the block leaves under it.
		if gap := handsRowDepth(row) - ruleDrop; gap < handsRowGap {
			t.Errorf("%s: %dpx between its rule and the next rung's name, want at least %d",
				row.name, gap, handsRowGap)
		}
	}
}

// **The tally says only what is true.** A rung nobody has built and nobody has raised carries no
// annotation at all: eighteen rungs each reading "PLAYED 0" is noise around the two or three the
// player is actually working on.
func TestTheTallyIsDrawnOnEveryRung(t *testing.T) {
	for _, tc := range []struct {
		plays int
		want  string
	}{
		{0, "PLAYED: 0"},
		{3, "PLAYED: 3"},
		{147, "PLAYED: 147"},
	} {
		if got := handsTallyText(handsRow{plays: tc.plays}); got != tc.want {
			t.Errorf("%d plays reads %q, want %q", tc.plays, got, tc.want)
		}
	}
}

// **The level is a parenthetical on the rung's own title**, so the title line says what this rung
// is on this run rather than leaving the level to be found in an annotation somewhere else. **And
// the title is shouted**, which is what keeps the panel free of lower case; it costs no width,
// kubasta being monospaced.
func TestTheLevelIsPartOfTheTitle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		level int
		want  string
	}{
		{"Pair", 1, "PAIR (LVL 1)"},
		{"Card Full House", 3, "CARD FULL HOUSE (LVL 3)"},
	} {
		got := handsTitleText(handsRow{name: tc.name, level: tc.level})
		if got != tc.want {
			t.Errorf("%s at level %d reads %q, want %q", tc.name, tc.level, got, tc.want)
		}
	}
}

// **A rung as shipped is level one, not level zero.** The stone count is what the rules read and it
// starts at nothing; the ladder a player looks at starts at one, because a level they can raise has
// to have a bottom step they can see.
func TestAnUnraisedRungIsLevelOne(t *testing.T) {
	for _, row := range handsRows(shippingHands()) {
		if row.level != 1 {
			t.Errorf("%s reads LVL %d on a run holding no stones", row.name, row.level)
		}
		if row.raised {
			t.Errorf("%s reads as raised on a run holding no stones", row.name)
		}
	}
}

// **The tally fits its column beside the name it annotates.** It is drawn after the name on the
// same line, so the pair is what has to fit rather than either alone.
func TestEveryRungsNameAndTallyFitTheColumn(t *testing.T) {
	fonts := assets.LoadFonts()
	src := fonts["kubasta"]
	if src == nil {
		t.Fatal("no kubasta font to measure with")
	}

	left, _, right, _ := handsTestBody()
	width := handsColumnWidth(image.Rect(left, 0, right, 0), handsColumnCount)

	// A deliberately loud tally: three figures of plays and two of level is more than a run will
	// reach, and it is the width the layout has to survive.
	// A deliberately loud pair of figures: three digits of plays and two of level is more than a
	// run will reach, and it is the width the layout has to survive.
	tally, _ := text.Measure(handsTallyText(handsRow{plays: 999}),
		&text.GoTextFace{Source: src, Size: handsTallySize}, 0)
	for _, row := range handsRows(shippingHands()) {
		title, _ := text.Measure(handsTitleText(handsRow{name: row.name, level: 99}),
			&text.GoTextFace{Source: src, Size: handsNameSize}, 0)
		if int(title) > width {
			t.Errorf("%s at a two-figure level is %dpx against a %dpx column",
				row.name, int(title), width)
		}
		mult, _ := text.Measure(row.mult, &text.GoTextFace{Source: src, Size: handsMultSize}, 0)
		span := handsSetsWidth(row.sets) + handsMultGap + int(mult) + handsMultGap + int(tally)
		if span > width {
			t.Errorf("%s: its examples and a three-figure tally come to %dpx against a %dpx column",
				row.name, span, width)
		}
	}
}

// **An axis caption has to fit over the set it names, and so does the word between two sets.** The
// captions carry the whole of the merged rung's claim, and one running into the set beside it would
// say the wrong pair counts on that axis; the OR has only the gap to stand in.
func TestEveryAxisCaptionFitsItsExample(t *testing.T) {
	fonts := assets.LoadFonts()
	src := fonts["kubasta"]
	if src == nil {
		t.Fatal("no kubasta font to measure with")
	}

	for _, row := range handsRows(shippingHands()) {
		for i, axis := range row.axes {
			word := handsAxisWord(axis)
			adv, _ := text.Measure(word, &text.GoTextFace{Source: src, Size: handsAxisSize}, 0)
			room := handsCardsWidth(len(row.sets[i])) + handsSetGap
			if int(adv) > room {
				t.Errorf("%s: %q is %dpx over a %dpx example", row.name, word, int(adv), room)
			}
		}
		if len(row.sets) < 2 {
			continue
		}
		or, _ := text.Measure(handsOrWord, &text.GoTextFace{Source: src, Size: handsOrSize}, 0)
		if int(or) > handsSetGap {
			t.Errorf("%s: %q is %dpx in a %dpx gap between examples",
				row.name, handsOrWord, int(or), handsSetGap)
		}
	}
}
