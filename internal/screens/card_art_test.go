package screens

import (
	"github.com/curiousjc/ascend-duel/internal/session"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// **This is the first test in internal/screens, and it is a deliberate narrow exception
// to "the combat package is the one that can be tested without a window".**
//
// The rule is really about windowlessness, and these two assertions happen to be
// windowless: they compare constants and walk a switch statement. Nothing here creates an
// ebiten.Image, calls RunGame, or touches a GlobalState, so no graphics driver is ever
// initialised and the test runs on a headless CI box.
//
// They earn the exception because both guard a duplication that the compiler cannot see.
// If either becomes awkward, delete it — do not start reaching for a window to keep it
// alive, and do not read this as licence to test the rest of the screen.

func TestCardFootprintMatchesTheRenderer(t *testing.T) {
	// cardWidth and cardHeight lay out the hand — the pitch, the band, the drop
	// indicator, every hit rectangle. cards.Hand draws the card that sits in those
	// rectangles. They are two copies of one number because one is a const and the other
	// a var field, and nothing but this test stops them drifting.
	//
	// Drift would not crash anything. It would put the cards a few pixels out of their
	// own hit boxes, which reads as "clicking the edge of a card sometimes does nothing"
	// and is miserable to track down.
	if cardWidth != cards.Hand.Width {
		t.Errorf("cardWidth is %d but cards.Hand.Width is %d — the hand would lay out cards at the wrong pitch",
			cardWidth, cards.Hand.Width)
	}
	if cardHeight != cards.Hand.Height {
		t.Errorf("cardHeight is %d but cards.Hand.Height is %d — hit rectangles would not match the art",
			cardHeight, cards.Hand.Height)
	}
}

func TestEveryElementHasItsOwnArt(t *testing.T) {
	// combat.Element and cards.Element are separate enums on purpose: the rules say what an
	// element *does* and the drawing package says what colour it is, and neither wants the
	// other's vocabulary. The cost is a hand-written switch, which the compiler cannot check
	// for completeness.
	//
	// A missing case falls through to Basic, so the failure mode is two elements sharing
	// a border colour — a fire card that looks plain. Distinctness is what is asserted.
	//
	// It walks combat.AllElements rather than a list written out here, so an element
	// appended to the rules fails this test until it has been given a colour.
	seen := map[cards.Element]combat.Element{}
	for _, e := range combat.AllElements {
		got := artFor(e)
		if prev, dup := seen[got]; dup {
			t.Errorf("%s and %s both map to cards.%v — one of them is missing from the switch in artFor()",
				prev, e, got)
		}
		seen[got] = e
	}

	if len(seen) != len(cards.Elements()) {
		t.Errorf("the screen maps %d distinct elements but internal/cards knows %d",
			len(seen), len(cards.Elements()))
	}
}

func TestElementNamesAgreeAcrossThePackages(t *testing.T) {
	// The two enums also carry names, and the deck reads element names out of
	// cards.json. If the drawing package spells one differently, a sheet labelled
	// "lightning" could be showing the colour the game calls something else.
	for _, e := range combat.AllElements {
		if got, want := artFor(e).String(), e.String(); got != want {
			t.Errorf("screen calls it %q, internal/cards calls it %q", want, got)
		}
	}
}

func TestEveryFormHasItsOwnMark(t *testing.T) {
	// The same hand-written-switch hazard as the elements, one type over. A form
	// falling through to FormNone draws no mark at all, which on a card whose
	// category *word* has been deleted means the card says nothing about what it is.
	seen := map[cards.Form]combat.Form{}
	for _, f := range combat.Forms() {
		got := form(f)
		if got == cards.FormNone {
			t.Errorf("%v maps to FormNone — it would draw no mark and the card would not say its form", f)
		}
		if prev, dup := seen[got]; dup {
			t.Errorf("%v and %v both map to cards.%v", prev, f, got)
		}
		seen[got] = f

		if got.String() != f.String() {
			t.Errorf("rules call it %q, internal/cards calls it %q", f.String(), got.String())
		}
	}

	// **FormNone has to survive the crossing too.** The opponent's cards carry it, and one that
	// mapped onto a real form would draw a mark claiming membership of a deck the player
	// cannot build hands against.
	if got := form(combat.FormNone); got != cards.FormNone {
		t.Errorf("the rules' FormNone maps to cards.%v, want FormNone", got)
	}
}

// plainText is a card's face text in nobody's hands: no rings, no strength. **The catalogue tests
// below are about the wording, not about a pairing**, so they take the figure the concept declares
// and the ring cases get their own test.
func plainText(a combat.ConceptID) string {
	text, _ := cardEffect(combat.Plain(a), held{})
	return text
}

func TestEveryConceptHasEffectText(t *testing.T) {
	// A card with no text draws a name, a cost, a corner mark and nothing that says what it does.
	//
	// **It walks the whole registry, not the player's twelve** *(2026-08-16)*. Every enemy carries
	// its own cards and the table lays an enemy's queue out as cards, so a verb the generator does
	// not cover is four hundred blank faces rather than one.
	for _, a := range combat.AllConcepts() {
		if plainText(a) == "" {
			t.Errorf("%v has no effect text — its card would say nothing about what it does",
				combat.ConceptOf(a).Key)
		}
	}
}

func TestEveryCardTextFitsItsBand(t *testing.T) {
	// **The wording is here and the band is in internal/cards, so neither package can check
	// this alone.** Render draws every line it wraps to rather than clamping, so an overlong
	// string runs off the bottom of the card — this is what fails first.
	//
	// It needs the real font because wrapping is measured, which is also why it is worth
	// having: "Negate 1 attack, deal 0.5x damage back" fits in three lines or four depending on
	// a comma, and nobody can tell by looking at the string.
	ttf := assets.LoadFontData()["kubasta"]
	if len(ttf) == 0 {
		t.Fatal("no kubasta font data embedded")
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		t.Fatal(err)
	}

	st := cards.Hand
	width := st.Width - st.TextColumnLeft - st.TextInset

	for _, a := range combat.AllConcepts() {
		lines, err := cards.WrapText(f, st.TextSize, plainText(a), width)
		if err != nil {
			t.Fatalf("%v: %v", a, err)
		}
		if len(lines) > st.TextLines() {
			t.Errorf("%v's text wraps to %d lines and the band holds %d: %q",
				combat.ConceptOf(a).Key, len(lines), st.TextLines(), plainText(a))
		}
	}
}

func TestNoEffectTextWordIsWiderThanItsColumn(t *testing.T) {
	// **Wrapping breaks on spaces only**, so a single word wider than the column is not
	// wrapped, it overruns — silently, and only on the one card that has it. The column is
	// ~100px at 18pt, which is around a dozen characters, so this is a real constraint on the
	// wording rather than a theoretical one.
	ttf := assets.LoadFontData()["kubasta"]
	if len(ttf) == 0 {
		t.Fatal("no kubasta font data embedded")
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		t.Fatal(err)
	}

	st := cards.Hand
	width := st.Width - st.TextColumnLeft - st.TextInset

	for _, a := range combat.AllConcepts() {
		for _, word := range strings.Fields(plainText(a)) {
			w, err := cards.TextWidth(f, st.TextSize, word)
			if err != nil {
				t.Fatalf("%v: %v", a, err)
			}
			if w > width {
				t.Errorf("%v: %q is %dpx, wider than the %dpx column — it will run off the card",
					a, word, w, width)
			}
		}
	}
}

func TestDeckPitchMatchesTheCard(t *testing.T) {
	// The overlay lays cards out at deckStackPitch and internal/cards sizes its layout
	// against the strip that leaves visible. The two live in different packages — screens
	// imports cards and never the reverse — so nothing but this stops them drifting.
	//
	// Drift is silent and ugly: tighten the pitch for a longer row and the name simply
	// stops being visible, with no error anywhere.
	if deckStackPitch > cards.Mini.Width {
		t.Errorf("pitch %d exceeds the card width %d, so the row would have gaps in it",
			deckStackPitch, cards.Mini.Width)
	}

	// The internal resolution, which Layout fixes. Written here rather than imported
	// because game imports screens and not the reverse; if it ever changes, this test is
	// the thing that should be updated to match.
	const screenW, screenH = 1280, 960
	pctX := func(p int) int { return screenW * p / 100 }
	pctY := func(p int) int { return screenH * p / 100 }

	// The comfortable pitch has to fit the row the shipping deck actually deals, which is the
	// case the constant is sized for. A row past that tightens rather than overflowing — see
	// rowPitchFor — so what this pins is that the *normal* deck is never tightened.
	longest := 0
	for _, n := range deckRowCounts() {
		if n > longest {
			longest = n
		}
	}
	row := rowWidth(longest, deckStackPitch) + deckRowLabelWidth
	if panel := pctX(deckPanelRightPct) - pctX(deckPanelLeftPct); row > panel-deckRowMargin {
		t.Errorf("the deck's longest row is %dpx wide against a %dpx panel", row, panel)
	}

	// And every row has to fit between the legend above and the closing hint below.
	// Derived from the panel constants rather than written down, because the last time it
	// was a hardcoded number it went stale the moment deckGridTop moved.
	top := pctY(deckPanelTopPct) + deckGridTop
	bottom := pctY(deckPanelBottomPct) - deckHintUp
	rows := deckRowCount*(cards.Mini.Height+deckRowGap) - deckRowGap
	if budget := bottom - top; rows > budget {
		t.Errorf("%d rows of %d is %dpx tall against a %dpx budget (y=%d..%d)",
			deckRowCount, cards.Mini.Height, rows, budget, top, bottom)
	}

	// The grid must also start below the legend it sits under, which is what the six-row
	// squeeze most easily breaks.
	if deckGridTop <= deckLegendTop {
		t.Errorf("the grid starts at y=%d, at or above the legend at y=%d", deckGridTop, deckLegendTop)
	}
}

func TestEveryCardLandsInExactlyOneDeckRow(t *testing.T) {
	// **The panel's whole claim is that it shows the deck**, so a card with nowhere to go is the
	// panel quietly lying. There is no cap to exceed any more — a busy row tightens instead — so
	// what is left to check is that every card lands somewhere and no row is empty.
	counts := deckRowCounts()
	for _, c := range session.StartingDeck() {
		if row := deckRowFor(c); row < 0 || row >= deckRowCount {
			t.Fatalf("%v maps to row %d, which does not exist", c, row)
		}
	}

	for row, n := range counts {
		if name, _ := deckRowLabel(row); n == 0 {
			t.Errorf("the %q row is empty", name)
		}
	}

	// And a plan sits in its colour's row like everything else *(2026-08-23)*. It used to be
	// checked into a row of its own, which was right while every plan was basic; now a fire
	// Prepare belongs under "fire", and the failure this guards against is a plan quietly routed
	// somewhere on the strength of its category.
	for _, c := range session.StartingDeck() {
		if c.Category() != combat.CategoryPlan {
			continue
		}
		if got, want := deckRowFor(c), deckRowFor(combat.Of(combat.Strike, c.Element)); got != want {
			t.Errorf("%v sits in row %d and an attack of the same colour sits in row %d", c, got, want)
		}
	}
}

func TestTheCardHoldsAsManyEffectsAsThereAreStatuses(t *testing.T) {
	// `cards.MaxEffects` is a layout number in a package that cannot see the rules, and the rules
	// decide how many statuses one duelist can carry at once: one of each in the file, since a
	// status does not stack. This is the join.
	//
	// **It is a check against `statuses.json` now** *(2026-08-17)*, where it used to be a check
	// against the element count — the two were the same number only because a status *was* an
	// element. A fifth status would silently drop a badge off the enemy card: the row would draw
	// four of five and look like a rendering glitch rather than a missing status. Failing here is
	// what makes authoring one a layout change too, exactly as MaxStatLines does for a fourth stat
	// row. The row has space for six at the current pitch, so the fix is a number, not a redesign.
	if cards.MaxEffects < combat.StatusCount() {
		t.Errorf("a card shows %d status badges against %d statuses in the file — %d would be dropped",
			cards.MaxEffects, combat.StatusCount(), combat.StatusCount()-cards.MaxEffects)
	}
}

func TestEveryStatusHasABadge(t *testing.T) {
	// A status with no artwork falls back to the default badge, which is a shape nobody has
	// learned — fine as a backstop, wrong as the thing a shipped status draws. This walks the
	// catalogue the rules can actually put on a duelist and asks each for a picture of its own.
	for _, id := range combat.AllStatuses() {
		key, ok := statusBadges[combat.StatusOf(id).Key]
		if !ok {
			t.Errorf("%s has no status badge and would draw the default", combat.StatusOf(id).Key)
			continue
		}
		if _, ok := assets.LoadImageData()[key]; !ok {
			t.Errorf("%s's badge is %q, which is not an embedded image", combat.StatusOf(id).Key, key)
		}
	}
}

func TestEveryRingDrawsSomething(t *testing.T) {
	// A ring's face is either its own picture or the default one, and both are assets keys that
	// nothing resolves until a card is drawn — so a typo in `rings.json` is a pink border around
	// an empty face, on a screen nobody reaches until they have played to a shop. `ArtKey` is
	// what closes the empty case; this closes the misspelled one.
	//
	// **It does not fail a ring for having no art of its own.** Most of the catalogue has none
	// and is meant to draw the default until somebody paints one — see tools/ringsheet, which
	// says how many that is.
	records := data.LoadRings()
	for _, key := range data.RingOrder(records) {
		art := records[key].ArtKey()
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", key, art)
		}
	}
}

func TestEveryRingNameFitsItsCard(t *testing.T) {
	// A ring card breaks its name a word to a line, and the art starts a fixed distance down —
	// so how many words a ring may be called is a layout fact, and `rings.json` is where it can
	// be broken. A three-word ring draws its last word over its own picture, on a screen nobody
	// reaches until they have played to a shop.
	//
	// **It measures the face name, not the record's**, because the face is what is drawn: the
	// trailing "Ring" is dropped there and keeping it would fail the file for a word it does not
	// print. And it measures ink rather than counting words, since a single word wider than the
	// card is the other way this breaks.
	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		t.Fatal(err)
	}
	st := cards.RingStyle
	_, lineHeight, err := faces.Measure(st.NameSize, "Frozen")
	if err != nil {
		t.Fatal(err)
	}
	room := st.NameLinesAbove(st.ArtTop, lineHeight)
	usable := st.Width - 2*st.BorderWidth - 4

	records := data.LoadRings()
	for _, key := range data.RingOrder(records) {
		words := strings.Fields(records[key].FaceName())
		if len(words) > room {
			t.Errorf("%s draws %d lines of name where the card has room for %d — the rest lands "+
				"on the artwork", key, len(words), room)
		}
		for _, w := range words {
			got, _, err := faces.Measure(st.NameSize, w)
			if err != nil {
				t.Fatal(err)
			}
			if got > usable {
				t.Errorf("%s's %q is %dpx at %gpt, wider than the %dpx a ring card has",
					key, w, got, st.NameSize, usable)
			}
		}
	}
}

func TestEveryStatusTheRulesHoldFitsTheDuelistArray(t *testing.T) {
	// The other half of the same join, one layer down: `Duelist.Statuses` is a fixed array because
	// a duelist has to stay comparable, and registration refuses a record past the end of it. This
	// fails while there is still room, so the file being one short of the wall is visible before
	// somebody hits it.
	if combat.StatusCount() > combat.MaxStatuses {
		t.Fatalf("%d statuses against an array of %d", combat.StatusCount(), combat.MaxStatuses)
	}
}

func TestEveryWormTextFitsItsCard(t *testing.T) {
	// **The gap this closes** *(2026-08-23)*: the two tests above hold the *duelist* cards against
	// their band, and nothing held a worm against WormStyle's. A worm's line is the whole of what
	// the card says, and it was one string away from overrunning — "make one card LIGHTNING" is
	// 135px in a 142px band — with nothing to fail if the next element name were longer.
	//
	// It checks both halves, because they fail differently: too many lines runs off the bottom of
	// the card, and one word wider than the band overruns the side silently, since wrapping breaks
	// on spaces only.
	ttf := assets.LoadFontData()["kubasta"]
	if len(ttf) == 0 {
		t.Fatal("no kubasta font data embedded")
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		t.Fatal(err)
	}

	st := cards.WormStyle
	width := st.Width - st.TextColumnLeft - st.TextInset

	for _, w := range session.Worms() {
		lines, err := cards.WrapText(f, st.TextSize, w.Text, width)
		if err != nil {
			t.Fatalf("%s: %v", w.Record, err)
		}
		if len(lines) > st.TextLines() {
			t.Errorf("%s's text wraps to %d lines and the band holds %d: %q",
				w.Record, len(lines), st.TextLines(), w.Text)
		}
		for _, word := range strings.Fields(w.Text) {
			got, err := cards.TextWidth(f, st.TextSize, word)
			if err != nil {
				t.Fatal(err)
			}
			if got > width {
				t.Errorf("%s's %q is %dpx at %gpt, wider than the %dpx band — it will overrun",
					w.Record, word, got, st.TextSize, width)
			}
		}
	}
}

func TestTheElementalWormsAllBreakInTheSamePlace(t *testing.T) {
	// **Why the authored break exists**, pinned so it cannot be quietly undone by deleting a `\n`
	// from worms.json. The four recolouring worms differ only in the element they name, and the
	// names differ in width — FIRE sits comfortably on the line where LIGHTNING all but fills it —
	// so left to the measurer the four read as four layouts of one card.
	ttf := assets.LoadFontData()["kubasta"]
	if len(ttf) == 0 {
		t.Fatal("no kubasta font data embedded")
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		t.Fatal(err)
	}

	st := cards.WormStyle
	width := st.Width - st.TextColumnLeft - st.TextInset

	want := 0
	for _, w := range session.Worms() {
		if w.Target != session.TargetElement {
			continue
		}
		lines, err := cards.WrapText(f, st.TextSize, w.Text, width)
		if err != nil {
			t.Fatalf("%s: %v", w.Record, err)
		}
		if want == 0 {
			want = len(lines)
		}
		if len(lines) != want {
			t.Errorf("%s draws on %d lines where another elemental worm draws on %d: %q",
				w.Record, len(lines), want, w.Text)
		}
		if len(lines) < 2 {
			t.Errorf("%s draws on one line — the authored break in worms.json has been lost: %q",
				w.Record, w.Text)
		}
	}
	if want == 0 {
		t.Error("no worm targets an element — this test is checking nothing")
	}
}

// deckRowCounts is how many cards of the shipping deck land in each row of the overlay. Shared by
// the two tests above rather than counted twice: they ask different questions of the same tally.
func deckRowCounts() []int {
	counts := make([]int, deckRowCount)
	for _, c := range session.StartingDeck() {
		if row := deckRowFor(c); row >= 0 && row < deckRowCount {
			counts[row]++
		}
	}
	return counts
}

// TestTheDeckPanelHidesNothing is the 2026-08-23 rule in a test: however many cards land in one
// row, the panel draws all of them and every one of them stays on the panel.
//
// **The failure it exists for is silent.** The old cap dropped the overflow and wrote a line
// under the grid saying so, which is at least honest; a pitch that clamps wrongly instead draws a
// card off the right-hand edge of the panel, where nothing reports it and nothing is visible.
func TestTheDeckPanelHidesNothing(t *testing.T) {
	const screenW = 1280
	pctX := func(p int) int { return screenW * p / 100 }
	width := pctX(deckPanelRightPct) - pctX(deckPanelLeftPct)
	room := width - deckRowLabelWidth - deckRowMargin

	// Well past anything a run can produce: 48 cards is the whole starting deck, and a flip ring
	// recolouring every one of them into a single element is the worst case the panel has.
	for n := 1; n <= 64; n++ {
		pitch := rowPitchFor(n, room)
		if pitch < 1 {
			t.Fatalf("%d cards got a pitch of %d, which would stack them on one spot", n, pitch)
		}
		if pitch > deckStackPitch {
			t.Errorf("%d cards got a pitch of %d, wider than the %d ceiling — a short row must be laid out as it always was",
				n, pitch, deckStackPitch)
		}
		if w := rowWidth(n, pitch); w > room {
			t.Errorf("%d cards at pitch %d is %dpx against %dpx of room — the row runs off the panel",
				n, pitch, w, room)
		}
	}
}

// TestTheDeckPanelDrawsEveryCardItIsGiven checks the layout itself rather than the arithmetic:
// hand the grid a deck and count the slots that come back.
func TestTheDeckPanelDrawsEveryCardItIsGiven(t *testing.T) {
	deck := session.StartingDeck()
	d := deckContents{draw: deck}

	// Every card recoloured into one element, which is what a flip ring does and what used to
	// overflow the row cap by a factor of four.
	oneRow := make([]combat.Card, 0, len(deck))
	for _, c := range deck {
		c.Element = combat.Fire
		oneRow = append(oneRow, c)
	}

	for _, tc := range []struct {
		name string
		d    deckContents
	}{
		{"the shipping deck", d},
		{"every card in one element", deckContents{draw: oneRow}},
	} {
		grid := tc.d.grid(640, 1177, 120)
		if got, want := len(grid.slots), len(tc.d.draw); got != want {
			t.Errorf("%s: the panel laid out %d of %d cards", tc.name, got, want)
		}
	}
}

func TestEveryBossDrawsItsPortrait(t *testing.T) {
	// The same failure as the ring above, one catalogue over, and worse: a boss has no default
	// picture to fall back on, so a mistyped `Portrait` is a stairway fight against a card with a
	// hole in it — and the earliest one of those is three fights into a run.
	//
	// **The key is the filename stem, unlike almost every other asset** — see the //go:embed in
	// assets/embed.go — so renaming a file is exactly what this catches.
	records := data.LoadBosses()
	if len(records) == 0 {
		t.Fatal("no bosses loaded — this test is checking nothing")
	}
	for _, key := range data.BossOrder(records) {
		art := records[key].Portrait
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", key, art)
		}
	}
}

func TestNoBossPortraitIsAnEnemyPortrait(t *testing.T) {
	// The two families share one flat map of images, so the `-boss` suffix is the whole of what
	// keeps them apart. A boss whose portrait key collided with a creature's would draw that
	// creature, silently and correctly as far as every lookup is concerned.
	enemies := data.LoadEnemies()
	taken := make(map[string]string, len(enemies))
	for _, key := range data.EnemyOrder(enemies) {
		taken[enemies[key].Portrait] = key
	}

	bosses := data.LoadBosses()
	for _, key := range data.BossOrder(bosses) {
		if other, clash := taken[bosses[key].Portrait]; clash {
			t.Errorf("%s and the enemy %s both draw %q", key, other, bosses[key].Portrait)
		}
	}
}

// TestEveryOpponentNameFitsItsCard holds both opponent pools against the width of the card they
// are drawn on.
//
// **`EnemyStyle` sets a name as one centred line and never wraps it** — `NameWordPerLine` is off,
// so `nameLines` hands the whole string back — which means a name too wide is not a name that
// spills onto a second line, it is a name with a letter clipped off each end. That is what
// `Jerry the Toll-Taker` did to half the boss roster until the title moved into its own field on
// 2026-08-24, and it was invisible until `tools/bosssheet` drew all thirty on one page.
//
// It measures the whole string rather than the longest word, because there is nowhere for a
// second word to go.
func TestEveryOpponentNameFitsItsCard(t *testing.T) {
	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		t.Fatal(err)
	}
	st := cards.EnemyStyle
	usable := st.Width - 2*st.BorderWidth - 4

	fits := func(kind, key, name string) {
		t.Helper()
		got, _, err := faces.Measure(st.NameSize, name)
		if err != nil {
			t.Fatal(err)
		}
		if got > usable {
			t.Errorf("%s %s is called %q, %dpx at %gpt against the %dpx the card has — the card "+
				"clips it at both ends rather than wrapping", kind, key, name, got, st.NameSize, usable)
		}
	}

	// **The roster is deliberately not checked yet, and that is an open decision rather than an
	// oversight.** Five creatures are over the line today — Mega Mutant II Torch, Mega Mutant III
	// Torch, Mega Mutant III Fist, Greater Abomination and Bio-Titan Plagueborn, by 4 to 16px —
	// and every one of them is a member of a family whose other members fit. Trimming a subset of
	// "Mega Mutant III Torch" and leaving "Mega Mutant III Gun" alone would make a naming scheme
	// that reads as a mistake, so the fix is an owner's call between renaming the family, dropping
	// NameSize on EnemyStyle, and letting a long name wrap. Widen this loop to the roster when it
	// is made.
	bosses := data.LoadBosses()
	for _, key := range data.BossOrder(bosses) {
		// **The name alone, which is what the card carries.** `FullName` is for the hover that
		// will print the title, and measuring it here would fail the file for a string the card
		// never draws.
		fits("boss", key, bosses[key].Name)
	}
}
