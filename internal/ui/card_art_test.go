package ui

import (
	"bytes"
	"image"
	"strconv"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// **This is the first test in internal/screens, and it is a deliberate narrow exception
// to "the combat package is the one that can be tested without a window".**
//
// The rule is really about windowlessness, and these two assertions happen to be
// windowless: they compare constants and walk a switch statement. Nothing here creates an
// ebiten.Image, calls RunGame, or touches a GlobalState, so no graphics driver is ever
// initialized and the test runs on a headless CI box.
//
// They earn the exception because both guard a duplication that the compiler cannot see.
// If either becomes awkward, delete it — do not start reaching for a window to keep it
// alive, and do not read this as license to test the rest of the screen.

func TestEveryElementHasItsOwnArt(t *testing.T) {
	// combat.Element and cards.Element are separate enums on purpose: the rules say what an
	// element *does* and the drawing package says what color it is, and neither wants the
	// other's vocabulary. The cost is a hand-written switch, which the compiler cannot check
	// for completeness.
	//
	// A missing case falls through to Basic, so the failure mode is two elements sharing
	// a border color — a fire card that looks plain. Distinctness is what is asserted.
	//
	// It walks combat.AllElements rather than a list written out here, so an element
	// appended to the rules fails this test until it has been given a color.
	seen := map[cards.Element]combat.Element{}
	for _, e := range combat.AllElements {
		got := ArtFor(e)
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
	// cards.json. If the drawing package spells one differently, a sheet labeled
	// "lightning" could be showing the color the game calls something else.
	for _, e := range combat.AllElements {
		if got, want := ArtFor(e).String(), e.String(); got != want {
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

// plainText is the type on a plain card's face, which is nothing.
//
// **A face says what a card does in pictures**: the form is the corner mark, the element is its
// color and the cost ticks under it, and the multiplier is the badge in the bottom-left. The only
// words a face is ever set with are an upgrade's, and those are held to the band by
// TestEveryUpgradedCardTextFitsItsBand. What the two tests below hold is the floor — that a card
// carrying no upgrade asks the band for nothing.
func plainText(a combat.ConceptID) string {
	return riderText(combat.Plain(a))
}

func TestEveryConceptSaysWhatItDoes(t *testing.T) {
	// **A card has to say what it does somewhere on its face**, and what it says it with is a
	// picture: the multiplier is a drawn badge and a defense stacks one shield per shield it
	// raises. A concept that is neither an attack with a multiplier nor a defense raising shields
	// is a card the player cannot read.
	//
	// **It walks the whole registry, not the player's nineteen** *(2026-08-16)*. Every enemy
	// carries its own cards and the table lays an enemy's queue out as cards, so a verb the
	// generator does not cover is four hundred blank faces rather than one.
	for _, a := range combat.AllConcepts() {
		card := combat.Plain(a)
		switch {
		case cardBadge(card) != "":
		case cardShields(card) > 0:
		default:
			t.Errorf("%v says nothing on its face — no damage badge, no shields and no text",
				combat.ConceptOf(a).Key)
		}
	}
}

func TestAnAttacksMultiplierIsOnItsFace(t *testing.T) {
	// The badge is the only place an attack's multiplier is written now, so a verb that stopped
	// producing one would take the figure off the card silently.
	for _, a := range combat.AllConcepts() {
		card := combat.Plain(a)
		if combat.ConceptOf(a).Verb != combat.VerbAttack {
			continue
		}
		if cardBadge(card) == "" || cardBadgePct(card) == 0 {
			t.Errorf("%v is an attack with no multiplier on its badge", combat.ConceptOf(a).Key)
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
	const screenH = state.ScreenHeight
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
	row := rowWidth(longest, deckStackPitch)
	// The room a row actually has is what the filter column left it — see deckGridRegion.
	if room := deckGridRoom(); row > room {
		t.Errorf("the deck's longest row is %dpx wide against %dpx of room", row, room)
	}

	// And every row has to fit inside the panel. Derived from the panel constants rather than
	// written down, because the last time it was a hardcoded number it went stale the moment the
	// body's top edge moved.
	top := pctY(ModalPanelTopPct) + modalBareBodyTop
	bottom := pctY(ModalPanelBottomPct) - modalBodyBottom
	rows := deckRowCount*(cards.Mini.Height+deckRowGap) - deckRowGap
	if budget := bottom - top; rows > budget {
		t.Errorf("%d rows of %d is %dpx tall against a %dpx budget (y=%d..%d)",
			deckRowCount, cards.Mini.Height, rows, budget, top, bottom)
	}

	// **The grid starts below the close button**, which is the only thing left at the top of the
	// panel now that the title, the counts line and the legend are gone. A tightened row reaches
	// almost the panel's right edge, so a grid starting any higher would run a row under the one
	// control that closes the dialog.
	if modalBareBodyTop < ModalCloseInset+ModalCloseSize {
		t.Errorf("the grid starts at y=%d and the close button ends at y=%d",
			modalBareBodyTop, ModalCloseInset+ModalCloseSize)
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
		if name, _ := deckRowLabel(deckRowElements()[row]); n == 0 {
			t.Errorf("the %q row is empty", name)
		}
	}

	// And a plan sits in its color's row like everything else *(2026-08-23)*. It used to be
	// checked into a row of its own, which was right while every plan was basic; now a fire
	// Prepare belongs under "fire", and the failure this guards against is a plan quietly routed
	// somewhere on the strength of its category.
	for _, c := range session.StartingDeck() {
		if c.Category() != combat.CategoryDefend {
			continue
		}
		if got, want := deckRowFor(c), deckRowFor(combat.Of(combat.Bash, c.Element)); got != want {
			t.Errorf("%v sits in row %d and an attack of the same color sits in row %d", c, got, want)
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
	// catalog the rules can actually put on a duelist and asks each for a picture of its own.
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

func TestEveryRelicDrawsSomething(t *testing.T) {
	// A relic's face is either its own picture or the default one, and both are assets keys that
	// nothing resolves until a card is drawn — so a typo in `relics.json` is a pink border around
	// an empty face, on a screen nobody reaches until they have played to a shop. `ArtKey` is
	// what closes the empty case; this closes the misspelled one.
	//
	// **It does not fail a relic for having no art of its own.** Most of the catalog has none
	// and is meant to draw the default until somebody paints one — see tools/relicsheet, which
	// says how many that is.
	records := data.LoadRelics()
	for _, key := range data.RelicOrder(records) {
		art := records[key].ArtKey()
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", key, art)
		}
	}
}

func TestEveryPotionDrawsSomething(t *testing.T) {
	// The potion half of TestEveryRelicDrawsSomething. `PotionData.ArtKey` closes the empty case —
	// a bottle nobody has painted draws the catalog default — and this closes the misspelled one,
	// which is otherwise a hole in a card only reached by playing to a shop.
	for _, rec := range data.LoadPotions() {
		art := rec.ArtKey()
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", rec.PotionRecord, art)
		}
	}
}

func TestEverySealedGoodDrawsSomething(t *testing.T) {
	// The sealed goods differ from every other catalog in the empty case: a good with no Art of its
	// own borrows the picture of whatever is inside it rather than falling back to a default of its
	// own — see goodArt, which this walks. What is checked is that every good draws *something*,
	// named or borrowed, so a misspelled key is a failure here rather than a hole on the shelf.
	for _, good := range session.Goods() {
		if good.Art == "" {
			continue
		}
		if _, ok := assets.LoadImageData()[good.Art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", good.Record, good.Art)
		}
	}
}

func TestEveryStoneDrawsSomething(t *testing.T) {
	// The stone half of TestEveryRelicDrawsSomething, and **it no longer has an empty case**
	// *(2026-09-16)*. A stone with no Art fell back to a generated boulder until the silhouette
	// generator was deleted; `StoneData.ArtKey` answers the relics' default face now, exactly as
	// the potions' does, so every record resolves to a key and every key has to name a file.
	//
	// What this closes is the misspelled one — a stone naming a file that is in no embed draws a
	// hole, on a card only reached by buying a bag of rocks.
	records := data.LoadStones()
	for _, key := range data.StoneOrder(records) {
		art := records[key].ArtKey()
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", key, art)
		}
	}
}

// TestEveryEssenceDrawsSomething and TestEveryRuneDrawsSomething are the two catalogs that had no
// coverage at all until 2026-09-16 — found by auditing the families rather than by anything going
// wrong, which is the point: an essence naming a misspelled key draws a blank card on the reward
// screen and nothing fails.
func TestEveryEssenceDrawsSomething(t *testing.T) {
	records := data.LoadEssences()
	for _, key := range data.EssenceOrder(records) {
		art := records[key].ArtKey()
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", key, art)
		}
	}
}

func TestEveryRuneDrawsSomething(t *testing.T) {
	records := data.LoadRunes()
	for _, key := range data.RuneOrder(records) {
		art := records[key].ArtKey()
		if _, ok := assets.LoadImageData()[art]; !ok {
			t.Errorf("%s draws %q, which is not an embedded image", key, art)
		}
	}
}

func TestEveryStoneArtIsTheCardsOwnSize(t *testing.T) {
	// A stone bleeds like a relic, so the same rule applies: the picture is committed at the card's
	// own 200x280 and nothing resamples at draw time. TestEveryBleedingCardArtIsTheCardsOwnSize
	// holds the three catalogs that had art when it was written; this holds the fourth.
	for _, st := range session.Stones() {
		if st.Art == "" {
			continue
		}
		raw := assets.LoadImageData()[st.Art]
		if len(raw) == 0 {
			continue // the test above is what reports this
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("%s: decoding %s: %v", st.Record, st.Art, err)
			continue
		}
		if cfg.Width != cards.EssenceStyle.Width || cfg.Height != cards.EssenceStyle.Height {
			t.Errorf("%s draws %s at %dx%d, want the card's %dx%d",
				st.Record, st.Art, cfg.Width, cfg.Height,
				cards.EssenceStyle.Width, cards.EssenceStyle.Height)
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

func TestEveryEssenceTextFitsItsCard(t *testing.T) {
	// **The gap this closes** *(2026-08-23)*: the two tests above hold the *duelist* cards against
	// their band, and nothing held an essence against EssenceStyle's. An essence's line is the whole of what
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

	st := cards.EssenceStyle
	width := st.Width - st.TextColumnLeft - st.TextInset

	for _, w := range session.Essences() {
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

func TestTheElementalEssencesAllBreakInTheSamePlace(t *testing.T) {
	// **Why the authored break exists**, pinned so it cannot be quietly undone by deleting a `\n`
	// from essences.json. The four recoloring essences differ only in the element they name, and the
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

	st := cards.EssenceStyle
	width := st.Width - st.TextColumnLeft - st.TextInset

	want := 0
	for _, w := range session.Essences() {
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
			t.Errorf("%s draws on %d lines where another elemental essence draws on %d: %q",
				w.Record, len(lines), want, w.Text)
		}
		if len(lines) < 2 {
			t.Errorf("%s draws on one line, so nothing about the set is being checked: %q",
				w.Record, w.Text)
		}
	}
	if want == 0 {
		t.Error("no essence targets an element — this test is checking nothing")
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
	room := deckGridRoom()

	// Well past anything a run can produce: 48 cards is the whole starting deck, and a flip relic
	// recoloring every one of them into a single element is the worst case the panel has.
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
	d := DeckContents{Draw: deck}

	// Every card recolored into one element, which is what a flip relic does and what used to
	// overflow the row cap by a factor of four.
	oneRow := make([]combat.Card, 0, len(deck))
	for _, c := range deck {
		c.Element = combat.Fire
		oneRow = append(oneRow, c)
	}

	for _, tc := range []struct {
		name string
		d    DeckContents
	}{
		{"the shipping deck", d},
		{"every card in one element", DeckContents{Draw: oneRow}},
	} {
		grid := tc.d.grid(DeckView{}, 640, 1177, 120)
		if got, want := len(grid.slots), len(tc.d.Draw); got != want {
			t.Errorf("%s: the panel laid out %d of %d cards", tc.name, got, want)
		}
	}
}

func TestEveryOpponentHasSomethingToDraw(t *testing.T) {
	// **Nearly every record's own picture is still to be generated**, so what this holds is the
	// fallback rather than the pictures: a record whose art key names no file has to land on the
	// placeholder, because a card with a hole in it reads as a bug where a placeholder reads as
	// art nobody has made yet.
	//
	// **The key is the filename stem** — see the //go:embed in assets/embed.go — so renaming a
	// file is what this catches once the pictures exist.
	if _, ok := assets.LoadImageData()[data.DefaultEnemyArt]; !ok {
		t.Fatalf("%s is not embedded, so a record with no picture draws nothing at all",
			data.DefaultEnemyArt)
	}

	motifs := data.LoadMotifs()
	if len(motifs) == 0 {
		t.Fatal("no motifs loaded — this test is checking nothing")
	}
	for _, key := range data.MotifOrder(motifs) {
		for _, r := range motifs[key].Records {
			for _, e := range r.Affinities {
				if _, ok := assets.LoadImageData()[r.ArtKey(e)]; ok {
					continue
				}
				// Not an error: it is the state the whole roster is in. The check that matters is
				// that the key is well formed and the placeholder is there to take it.
				if r.ArtKey(e) == "" {
					t.Errorf("%s dealt as %s names no picture at all", r.Record, e)
				}
			}
		}
	}
}

func TestNoTwoRecordsDrawTheSamePicture(t *testing.T) {
	// Every picture lands in one flat map of images, so two records claiming one key is one
	// lookup with two answers — and the wrong creature drawn, silently and correctly as far as
	// every lookup is concerned.
	motifs := data.LoadMotifs()
	taken := map[string]string{}
	for _, key := range data.MotifOrder(motifs) {
		for _, r := range motifs[key].Records {
			for _, e := range r.Affinities {
				art := r.ArtKey(e)
				if other, clash := taken[art]; clash {
					t.Errorf("%s and %s both draw %q", r.Record, other, art)
				}
				taken[art] = r.Record
			}
		}
	}
}

// TestNoOpponentCardWritesItsOwnName holds the trade EnemyStyle makes: the picture is the card,
// and the name is the tooltip's.
//
// **It replaced a width check.** A name set as one centered line and never wrapped is not a name
// that spills onto a second line, it is a name with a letter clipped off each end — which is what
// half the roster did until the name came off the face. There is nothing left to measure, and the
// thing worth holding is that nobody turns it back on without deciding to.
func TestNoOpponentCardWritesItsOwnName(t *testing.T) {
	if cards.EnemyStyle.ShowName {
		t.Error("the opponent card names itself across its own picture again")
	}

	// The full name is still built, because the tooltip needs it — and it is built in one place so
	// the hover and a review sheet cannot join the two halves differently.
	motifs := data.LoadMotifs()
	for _, key := range data.MotifOrder(motifs) {
		for _, r := range motifs[key].Records {
			if r.Title == "" {
				continue
			}
			if want := r.Name + " " + r.Title; r.FullName() != want {
				t.Errorf("%s: FullName is %q, want %q", r.Record, r.FullName(), want)
			}
		}
	}
}

// **Every rider is on the face, not only in the tooltip** *(2026-09-09)*. A card the player spent a
// rune on carries a wash whatever the rider is, and the wash is what carries across a row of
// eight cards — but it is not what answers "what does that mean". Six of the ten riders said
// nothing at all until this test existed.
//
// **The two metals are the exception and are named here rather than skipped by a rule**
// *(owner's call, 2026-09-09)*. Their wash is a sheen rather than a placeholder tint, and gold and
// silver are what the mechanic is called — so the picture names them and a word would be the same
// fact twice. A third silent rider has to be argued for by editing this list.
func TestEveryRiderKindIsOnTheFace(t *testing.T) {
	silent := map[combat.RiderKind]bool{combat.RiderGolden: true, combat.RiderSilver: true}

	for _, k := range combat.RiderKinds() {
		c := combat.Plain(combat.Bash).SetRider(combat.Rider{Kind: k, Amount: 5})
		switch got := riderText(c); {
		case silent[k] && got != "":
			t.Errorf("rider %s writes %q on the face, and its sheen is what names it", k, got)
		case !silent[k] && got == "":
			t.Errorf("rider %s adds nothing to the card's face", k)
		}
	}
	if got := riderText(combat.Plain(combat.Bash)); got != "" {
		t.Errorf("an unridden card claimed an upgrade on its face: %q", got)
	}
}

// **A metal explains itself in the tooltip: named first, then the odds.** The face says nothing at
// all about a metal — its sheen is what names it — so the panel is the whole explanation, and it
// opens by saying which metal rather than with two rate lines about a card the player has to
// identify from the wash.
func TestAMetalStillExplainsItselfInTheTooltip(t *testing.T) {
	for _, metal := range []struct {
		kind combat.RiderKind
		word string
	}{
		{combat.RiderGolden, carddesc.Gold},
		{combat.RiderSilver, carddesc.Silver},
	} {
		c := combat.Plain(combat.Bash).SetRider(combat.Rider{Kind: metal.kind, Amount: 5})
		lines := carddesc.RiderLines(c, 100)
		if len(lines) < 2 {
			t.Errorf("%s says %d lines, and the name alone is not an explanation", metal.kind, len(lines))
			continue
		}
		if lines[0] != metal.word+" CARD" {
			t.Errorf("%s opens with %q, want %q", metal.kind, lines[0], metal.word+" CARD")
		}
		for _, line := range lines[1:] {
			if !strings.Contains(line, "1 IN 5") || !strings.Contains(line, "ON PLAY") {
				t.Errorf("%s's tooltip line %q is neither the odds nor the moment", metal.kind, line)
			}
		}
	}
}

// **The name is written in the metal's own color**, sampled out of the sheen the card is washed in
// rather than written down anywhere — so a repaint of the ink moves the word with it. Lifted for the
// dark panel; see cards.WashLight.
func TestAMetalsNameIsLitInTheTooltip(t *testing.T) {
	for _, metal := range []struct {
		kind combat.RiderKind
		word string
	}{
		{combat.RiderGolden, carddesc.Gold},
		{combat.RiderSilver, carddesc.Silver},
	} {
		c := combat.Plain(combat.Bash).SetRider(combat.Rider{Kind: metal.kind, Amount: 5})
		spans := TipLine(carddesc.RiderLines(c, 100)[0])

		lit := ""
		for _, span := range spans {
			if span.Ink.A != 0 {
				lit += span.Text
			}
		}
		if lit != metal.word {
			t.Errorf("%s's name line lights %q, want %q: %v", metal.kind, lit, metal.word, spans)
		}
	}
}

// **Every rider says *when*, except the one that has no when.** What the player has to learn off an
// upgraded card is whether it happens as the card is played or while it merely sits in the hand,
// and that is exactly the thing a figure alone cannot say. A wildcard is the deliberate exception:
// it is something the card permanently is.
func TestEveryRiderSaysWhenItHappens(t *testing.T) {
	for _, k := range combat.RiderKinds() {
		// The wildcard has no moment — it is something the card permanently is — and the two metals
		// write nothing on the face at all; see TestEveryRiderKindIsOnTheFace.
		if k == combat.RiderWildElement || k == combat.RiderGolden || k == combat.RiderSilver {
			continue
		}
		c := combat.Plain(combat.Bash).SetRider(combat.Rider{Kind: k, Amount: 5})
		lines := carddesc.FaceLines(c)
		if len(lines) == 0 {
			t.Errorf("rider %s has no face lines", k)
			continue
		}
		switch lines[0] {
		case "ON PLAY", "IN HAND", "SCORING":
		default:
			t.Errorf("rider %s opens with %q, which is not a moment the player can read", k, lines[0])
		}
	}
}

// **An upgraded card's whole face still fits the band**, which is what the plain concepts are held
// to above. A rider adds lines to a card that already carries two, so this is the case that runs
// out of room first — and it runs out silently, by drawing off the bottom edge.
func TestEveryUpgradedCardTextFitsItsBand(t *testing.T) {
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
		for _, k := range combat.RiderKinds() {
			// **A two-digit amount, because the figure is part of the measure.** A rider drawn at 5
			// and shipped at 25 is a card that passed this test and overruns in play.
			c := combat.Plain(a).SetRider(combat.Rider{Kind: k, Amount: 25})
			text := riderText(c)
			lines, err := cards.WrapText(f, st.TextSize, text, width)
			if err != nil {
				t.Fatalf("%v + %v: %v", a, k, err)
			}
			if len(lines) > st.TextLines() {
				t.Errorf("%v carrying %v wraps to %d lines and the band holds %d: %q",
					combat.ConceptOf(a).Key, k, len(lines), st.TextLines(), text)
			}
		}
	}
}

// **No word an upgrade writes is wider than the column.** Wrapping breaks on spaces only, so a long
// word overruns rather than wrapping — and the upgrade vocabulary is where the long words are:
// SHIELDS, ELEMENT, SCORES.
func TestNoUpgradeWordIsWiderThanItsColumn(t *testing.T) {
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

	for _, k := range combat.RiderKinds() {
		c := combat.Plain(combat.Bash).SetRider(combat.Rider{Kind: k, Amount: 25})
		for _, line := range carddesc.FaceLines(c) {
			for _, word := range strings.Fields(line) {
				w, err := cards.TextWidth(f, st.TextSize, word)
				if err != nil {
					t.Fatalf("%v: %v", k, err)
				}
				if w > width {
					t.Errorf("%v writes %q, %d wide in a %d column", k, word, w, width)
				}
			}
		}
	}
}

// **The word carddesc writes and the word cards lights are the same word.** They are two packages
// with no arrow between them — see cards.MetalWords — so nothing but this stops the tooltip writing
// GOLD while the highlight looks for GOLDEN, which fails as a line that is simply never colored.
func TestTheMetalWordsAgree(t *testing.T) {
	written := map[systems.Upgrade]string{
		systems.UpgradeGolden: carddesc.Gold,
		systems.UpgradeSilver: carddesc.Silver,
	}
	if len(cards.MetalWords) != len(written) {
		t.Fatalf("cards knows %d metal words and carddesc writes %d",
			len(cards.MetalWords), len(written))
	}
	for _, w := range cards.MetalWords {
		if want := written[w.Upgrade]; w.Word != want {
			t.Errorf("cards lights %q for %v where carddesc writes %q", w.Word, w.Upgrade, want)
		}
	}
}

func TestEveryBleedingCardArtIsTheCardsOwnSize(t *testing.T) {
	// Full-bleed art covers the card, so a picture bigger than 200x280 is downsampled on the way
	// in and a non-integer reduction softens exactly the hard block edges the art prompt spends
	// most of its words demanding. Authored at the card's own size, nothing resamples at all.
	//
	// **The weight is the other half of it, and it is the half that fails silently.** The
	// generator hands back 1060x1484, which is about 1.1 MB a relic. Committing those would be
	// roughly 155 MB across a 137-ring catalog, against a repo whose CLAUDE.md already counts
	// 4.9 MB of sheets as a cost worth managing. At the card's size it is about 57 KB each.
	//
	// So: keep the generator's output in `.scratch/relic-art`, and commit the 200x280 reduction.
	for _, st := range []cards.Style{cards.RelicStyle, cards.EssenceStyle} {
		if !st.ArtBleed {
			t.Fatal("a style in this list no longer bleeds — the test is checking the wrong thing")
		}
	}

	w, h := cards.RelicStyle.Width, cards.RelicStyle.Height
	for key, raw := range assets.LoadImageData() {
		if !strings.HasSuffix(key, "-ring") && !strings.HasSuffix(key, "-essence") {
			continue
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("%s: %v", key, err)
			continue
		}
		if cfg.Width != w || cfg.Height != h {
			t.Errorf("%s is %dx%d where a full-bleed card is %dx%d — reduce it before committing "+
				"it, and keep the original in .scratch", key, cfg.Width, cfg.Height, w, h)
		}
	}
}

// A card told to count as another form draws that form's card at its own rung — and a defense told
// the same thing draws exactly what it always did.
//
// **The two halves are one decision** *(owner's call, 2026-09-16)*: what a form override changes
// about a picture is the weapon, so it applies where the card does the same thing with a different
// one and stops at the attack/defend line, where the card does something else entirely.
func TestAFormOverrideRepaintsAnAttackAndLeavesADefenseAlone(t *testing.T) {
	// **Found with a flag rather than a sentinel**: `combat.NoConcept` is -1 and a zero Card holds
	// concept 0, which is a real card — so an unset check against it finds the first card in the
	// registry and tests nothing.
	var slash, shield combat.Card
	var foundSlash, foundShield bool
	for _, id := range combat.PlayerConcepts() {
		c := combat.ConceptOf(id)
		if !foundSlash && c.Form == combat.FormSlash {
			slash, foundSlash = combat.Card{Concept: id, Element: combat.Fire}, true
		}
		if !foundShield && c.Verb == combat.VerbShield {
			shield, foundShield = combat.Card{Concept: id, Element: combat.Fire}, true
		}
	}
	if !foundSlash || !foundShield {
		t.Skip("the catalog holds no slash attack or no defense")
	}

	crushed := slash
	crushed.FormOverride = combat.FormCrush

	id, ok := combat.Counterpart(slash.Concept, combat.FormCrush)
	if !ok {
		t.Fatalf("%s has no crush counterpart", slash.Label())
	}
	want := data.CardArtKey(combat.ConceptOf(id).Label, slash.Element.String())
	if got := cardArtRecord(crushed); got != want {
		t.Errorf("%s told to be a crush draws %q, want %q", slash.Label(), got, want)
	}

	blocked := shield
	blocked.FormOverride = combat.FormCrush
	if got, want := cardArtRecord(blocked), cardArtRecord(shield); got != want {
		t.Errorf("%s told to be a crush draws %q, want its own %q", shield.Label(), got, want)
	}
}

// TestTheFaceCacheIsBounded holds the cap that stops a long session growing without limit.
//
// **It files nil faces, which is the negative-cached entry, so the test creates no images and
// needs no graphics context** — the bookkeeping being checked is the map and the insertion order,
// and those are the same whether an entry is a picture or a nil.
//
// The failure it exists to catch is the one that was live until this cap arrived: cardCache is
// keyed on the whole cards.Spec, whose field space has grown far past anything the deck bounds,
// so nothing stopped it accumulating an entry per combination for the life of the process.
func TestTheFaceCacheIsBounded(t *testing.T) {
	cardCache = map[cardKey]*ebiten.Image{}
	cardOrder = nil
	t.Cleanup(func() {
		cardCache = map[cardKey]*ebiten.Image{}
		cardOrder = nil
	})

	for i := 0; i < maxCardCache*3; i++ {
		rememberCard(cardKey{spec: cards.Spec{Name: strconv.Itoa(i)}}, nil)
	}

	if len(cardCache) != maxCardCache {
		t.Errorf("cache holds %d faces, cap is %d", len(cardCache), maxCardCache)
	}
	if len(cardOrder) != maxCardCache {
		t.Errorf("insertion order holds %d keys against %d cached faces", len(cardOrder), len(cardCache))
	}

	// The oldest go first, so the newest cap entries are the ones still there.
	oldest := cardKey{spec: cards.Spec{Name: "0"}}
	if _, ok := cardCache[oldest]; ok {
		t.Error("the first face filed was still cached after three times the cap went through it")
	}
	newest := cardKey{spec: cards.Spec{Name: strconv.Itoa(maxCardCache*3 - 1)}}
	if _, ok := cardCache[newest]; !ok {
		t.Error("the last face filed was evicted, so eviction is not taking the oldest")
	}
}

// **A full-bleed card's art must be opaque** *(bug, 2026-09-18)*.
//
// The picture *is* the face on these styles, so a pixel it leaves transparent is a hole in the
// card. Three rune pictures shipped with alpha — Unmake at 99% of its pixels, Maulstave and
// Loammark the same — and the cards were see-through: the table, and whatever card was drawn
// behind them, showed straight through the face.
//
// **`drawArtBleed` composites rather than replaces now**, so the failure can no longer reach the
// screen as a hole; what a thin picture shows instead is the card's own surface. This is the other
// half of that fix and it is the half that says *which file* — a washed-out card looks like a
// picture somebody drew that way, where a test naming the file does not.
//
// **It walks the catalogs rather than the asset directory**, so it also fails on a record pointing
// at a picture that is not there — and it covers the runes, which the size test above misses
// because its keys carry no suffix to match on.
func TestEveryBleedingCardArtIsOpaque(t *testing.T) {
	pictures := assets.LoadImageData()

	keys := map[string]string{}
	for key, rec := range data.LoadRelics() {
		keys[rec.ArtKey()] = "relic " + key
	}
	for _, w := range session.Essences() {
		keys[w.Art] = "essence " + w.Record
	}
	for _, p := range session.Runes() {
		keys[p.Art] = "rune " + p.Record
	}
	for _, st := range session.Stones() {
		keys[st.Art] = "stone " + st.Record
	}
	for _, g := range session.Goods() {
		keys[g.Art] = "good " + g.Record
	}
	for _, p := range session.Potions() {
		keys[p.Art] = "potion " + p.Record
	}
	if len(keys) < 100 {
		t.Fatalf("the catalogs answered %d pictures, which is too few to be the whole set", len(keys))
	}

	for key, what := range keys {
		if key == "" {
			continue
		}
		raw := pictures[key]
		if len(raw) == 0 {
			t.Errorf("%s names %q, which no asset answers", what, key)
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("%s: decoding %s: %v", what, key, err)
			continue
		}

		b := img.Bounds()
		soft := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
					soft++
				}
			}
		}
		if soft > 0 {
			t.Errorf("%s draws %s with %.1f%% of its pixels not opaque — a bleeding card's art is "+
				"the whole face, so regenerate it against a ground rather than on transparency",
				what, key, 100*float64(soft)/float64(b.Dx()*b.Dy()))
		}
	}
}
