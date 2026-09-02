package screens

// **The hands panel: every rung of the hand ladder, drawn as the cards that build it.**
//
// The multipliers are the whole of what building a hand buys, and until this panel landed
// *(2026-08-24)* they were readable in three places, none of which is the game: `data/hands.json`,
// `go run ./tools/handsheet`, and the fight log after a hand had already fired. A player deciding
// which cards to queue was being asked to remember nineteen numbers.
//
// **A rung is drawn as the cards that build it** *(2026-08-24, owner's call)*. It was two lines of
// words until then — a grammar line saying what the rung wants in the axis's own words, and an
// example line naming the cheapest real cards that form it. Both are gone, replaced by the cards
// themselves at `cards.Token` size, because a hand is counted on exactly three things a card
// already says in pictures: its **element**, its **form mark** and its **cost ticks**. Four
// crimson slash marks in a row *is* the rule, and it is the same reading the player does across
// their own hand rather than a second vocabulary to learn.
//
// **The example varies everything the rung does not count**, which is what makes the pictures
// carry the rule *(owner's call, 2026-08-24)*. A form pair is drawn as a 1 AP stab beside a 3 AP
// stab and an elemental pair as a fire stab beside a fire slash, so the row says what a rung lets
// you get away with rather than showing two identical cards — which is the picture cheapest-set
// produced for a form pair, a card pair and an elemental pair alike. See `decks.Example`.
//
// **The example is the deck's own answer**, not an invented one: cards this run actually holds.
//
// **A rung is one block, ruled off from the next** *(owner's call, 2026-08-24)*, and the
// multiplier stands right beside the last card rather than out at the column's edge. Three
// columns of names, cards and figures otherwise read as three columns of *lists* — the eye pairs
// a figure with whatever is nearest it, and at the far edge the nearest thing is the rung below.
//
// **Ordered by multiplier, cheapest-paying first, across all three axes at once.** That is the
// comparison the multipliers are making — an Elemental Three of a Kind at 145 says it is worth
// about what a Form Three of a Kind is — and it is exactly the comparison `hands.json`'s
// axis-by-axis layout hides. tools/handsheet orders its page the same way for the same reason.
//
// The panel's chrome is the shared one; see modal.go.

import (
	"image"
	"image/color"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// handsContents is what the panel is asked to show: the deck the examples are built out of, and
// who is holding it.
//
// **The holder prices the example**, exactly as it prices a card's face — a slash under Keen is
// worth a bigger figure — so the sum a player reads here is the sum they would actually deal.
// **A zero holder is honest rather than broken**: with no DMG known the example names its cards
// and drops the arithmetic, the same choice the card tooltip makes between fights.
type handsContents struct {
	deck   []combat.Card
	holder combat.Duelist
}

// handsRow is one rung: its name and what forming it pays, and the hand this deck illustrates it
// with.
type handsRow struct {
	name  string
	mult  string
	cards []combat.Card

	// raised is whether a stone has moved this rung, which is what the figure is written in the
	// ring pink for. **A colour rather than a second number**: the panel says what a hand pays
	// this run, and the pink is what stops that reading as the catalogue having changed.
	raised bool
}

// handsRows is the whole ladder, in the order it is drawn.
//
// **Sorted by multiplier, then by axis, then by ID.** The axis tie-break is the matcher's own —
// narrowest first, see combat.Axis — so two rungs paying the same read in the order the rules
// would pick between them, and the ID is the last key so the order cannot depend on how the
// catalogue happens to be filed.
func handsRows(c handsContents) []handsRow {
	// **The holder's ladder, not the catalogue's** *(2026-08-27)*. A stone raises one rung for one
	// run, and a panel showing the shipped figure would be the one place in the game where what a
	// hand pays is stated wrongly — the resolver, the preview and this all read
	// `Duelist.HandTable` now, so they cannot come to three answers.
	hands := c.holder.HandTable()
	base := combat.Hands()
	sort.SliceStable(hands, func(i, j int) bool {
		a, b := hands[i], hands[j]
		if a.Multiplier != b.Multiplier {
			return a.Multiplier < b.Multiplier
		}
		if a.Match != b.Match {
			return a.Match < b.Match
		}
		return a.ID < b.ID
	})

	out := make([]handsRow, 0, len(hands))
	for _, h := range hands {
		out = append(out, handsRow{
			name:   h.Name,
			mult:   multiplierText(h.Multiplier),
			cards:  handsExample(c, h),
			raised: h.Multiplier != catalogueMultiplier(base, h.Key),
		})
	}
	return out
}

// handsExample is the hand this deck illustrates the rung with.
//
// **The action-point cost went on 2026-08-24** *(owner's call)*, and with it the damage the
// example would deal. A rung is a multiplier and a shape of hand; what a *particular* five cards
// out of the deck happen to cost is a fact about the illustration rather than about the rung, and
// the cards carry their own ticks for anyone adding it up.
//
// **Every rung is illustrated, reachable or not** *(owner's call, 2026-08-24)*. A hand wanting
// five copies of one card cannot be dealt from the shipping deck, and the panel used to say so in
// words where the cards belong — see decks.Example, which repeats a card rather than coming back
// empty. Whether a rung is reachable is a fact about today's deck; the rung is the ladder.
func handsExample(c handsContents, h combat.Hand) []combat.Card {
	hand, _ := decks.Example(c.deck, h)
	return hand
}

// The panel's own geometry: three columns of rungs, a name over a row of cards.
const (
	// **Three columns as of 2026-08-24**, up from two. A rung was three lines of words and is now
	// a name over a row of cards, which is wider and shorter — so the ladder wants columns rather
	// than depth. Nineteen rungs over three is seven in the deepest, against a budget that holds
	// eight: TestTheColumnsHoldTheWholeLadder is what fails when a rung is added past that, and
	// the headroom is the point — the panel was rebuilt this way to leave room for hands not yet
	// written.
	handsColumnCount = 3

	handsColumnGap = 28

	// A name line, a row of tokens under it, and a rule under that.
	handsRowHeight = 86
	handsNameLine  = 0
	handsCardsTop  = 24

	// handsRuleDrop is where the hairline separating one rung from the next sits, measured from
	// the row's top. **It is what makes a rung one block rather than three lists** — a name, a row
	// of cards and a figure in three columns otherwise read down the page instead of across, and
	// a reader pairs each figure with whichever thing is nearest, which at a column's edge is the
	// rung below.
	handsRuleDrop = handsCardsTop + 62

	// handsMultGap is the air between the last card and the multiplier. **Beside the cards, not
	// out at the column's edge** *(owner's call, 2026-08-24)*: the figure belongs to the hand it
	// prices, and distance is what says so.
	handsMultGap = 12

	handsNameSize = 17
	handsMultSize = 20
)

// handsCardPitch is how far apart the tokens in a row sit: the token plus two pixels of air.
//
// **Read off the style rather than written down**, so a token that changes size takes the row's
// arithmetic with it — the trap TestDeckPitchMatchesTheCard exists to catch on the other panel.
// Two pixels of air rather than the deck panel's overlap: a rung is at most five cards and the
// column has room for them, and overlapping would hide the very ticks the row is drawn to show.
var handsCardPitch = cards.Token.Width + 2

// The panel's ink. It is written on the modal's dark fill, so these run light.
//
// **The multiplier is the one coloured figure**, because it is the one thing on the row that is
// not a description: the name says which rung, the cards say what builds it, and the multiplier
// is what forming it pays.
var (
	handsNameInk = color.RGBA{R: 236, G: 232, B: 226, A: 255}
	handsMultInk = color.RGBA{R: 240, G: 198, B: 108, A: 255}

	// The rule is barely there on purpose: it groups a rung, and a line loud enough to be read as
	// a border would make nineteen boxes out of a ladder.
	handsRuleInk = color.RGBA{R: 92, G: 90, B: 88, A: 255}
)

// handsColumns splits the ladder into the columns it is drawn in, filling the first column top to
// bottom before the second starts.
//
// **Down then across, not across then down.** The rows are in multiplier order and the point of
// that order is that reading down the list walks up the ladder; snaking across the columns would
// interleave the cheap rungs with the dear ones.
func handsColumns(rows []handsRow, columns int) [][]handsRow {
	if columns < 1 {
		columns = 1
	}
	per := (len(rows) + columns - 1) / columns

	out := make([][]handsRow, 0, columns)
	for i := 0; i < len(rows); i += per {
		end := i + per
		if end > len(rows) {
			end = len(rows)
		}
		out = append(out, rows[i:end])
	}
	return out
}

// handsPanelTitle is the whole of the panel's words *(owner's call, 2026-08-24)*.
//
// **It said three things and now says one.** Under the title were a count of the rungs and a line
// explaining the ordering — both true, both describing pictures that were already on the screen
// saying it better. A panel built to show a rule in cards and then captioned in prose is a panel
// that does not trust its own drawings, and the two lines cost it forty pixels of ladder.
const handsPanelTitle = "Duelist Hands"

// handsBodyRect is the room the rungs are laid out in: the panel, less the title above it, and
// inset at the sides by the margin the deck panel's rows keep.
func handsBodyRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+deckRowMargin, r.Min.Y+modalTitleOnlyBodyTop,
		r.Max.X-deckRowMargin, r.Max.Y-modalBodyBottom)
}

// handsColumnWidth is how wide one column is. **One function rather than the arithmetic written
// twice**, because the drawing and the test that holds the text inside it must not be able to
// disagree about the answer.
func handsColumnWidth(body image.Rectangle, columns int) int {
	if columns < 1 {
		columns = 1
	}
	return (body.Dx() - handsColumnGap*(columns-1)) / columns
}

// handsCardsWidth is how much of a column a rung's own cards take.
func handsCardsWidth(n int) int {
	if n <= 0 {
		return 0
	}
	return (n-1)*handsCardPitch + cards.Token.Width
}

// drawHandPanel covers the screen with the ladder.
func drawHandPanel(gs *state.GlobalState, screen *ebiten.Image, c handsContents) {
	rows := handsRows(c)

	r := drawModalFrame(gs, screen, modalHead{title: handsPanelTitle})

	columns := handsColumns(rows, handsColumnCount)
	body := handsBodyRect(r)
	colWidth := handsColumnWidth(body, len(columns))

	for i, column := range columns {
		left := body.Min.X + i*(colWidth+handsColumnGap)
		for j, row := range column {
			drawHandRow(gs, screen, c, row, left, body.Min.Y+j*handsRowHeight, colWidth)
		}
	}
}

// drawHandRow writes one rung: its name, the cards this deck builds it from, the multiplier
// beside them, and the rule that closes the block.
func drawHandRow(gs *state.GlobalState, screen *ebiten.Image, c handsContents, row handsRow,
	left, top, width int) {

	write := func(x, y int, size float64, ink color.RGBA, s string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, s,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}, op)
	}

	write(left, top+handsNameLine, handsNameSize, handsNameInk, row.name)

	// **The cards are drawn as themselves**, through the same spec every other screen builds, so
	// a token cannot say something the card in the hand does not. `enabled` is true and nothing is
	// selected: this is a catalogue, and a dimmed card here would mean "unaffordable" against a
	// budget no round has yet set.
	for i, card := range row.cards {
		at := image.Pt(left+i*handsCardPitch, top+handsCardsTop)
		drawCard(gs, screen, at, cards.Token, card, heldBy(c.holder, card), true, false)
	}
	// Beside the last card and centred against the band, so the figure reads as belonging to the
	// hand rather than to the column.
	// **A raised rung is written in `boostInk`, which is the ring pink.** It is reused rather than
	// given a hue of its own: on a card it means "a ring moved this figure", and here it means "a
	// stone did" — the shared reading is *something you bought moved this number*, which is the
	// thing a player needs to see. A second pink-ish hue for the second source would be two
	// colours a player has to tell apart to learn the same fact.
	multInk := handsMultInk
	if row.raised {
		multInk = boostInk
	}
	write(left+handsCardsWidth(len(row.cards))+handsMultGap,
		top+handsCardsTop+handsMultDrop(handsMultSize), handsMultSize, multInk, row.mult)

	vector.StrokeLine(screen, float32(left), float32(top+handsRuleDrop),
		float32(left+width), float32(top+handsRuleDrop), 1, handsRuleInk, false)
}

// handsMultDrop centres a line of type down the token band. **Against the cards, not against the
// row**: the name sits above the band, so a figure centred on the whole row would ride high of
// the thing it is pricing.
func handsMultDrop(size float64) int {
	return (cards.Token.Height - int(size)) / 2
}

// handsToggle is the hands panel behind a button, the deck panel's counterpart.
//
// **It carries no tooltip.** The panel is already all words — there is nothing on it whose meaning
// is hidden behind a picture, which is what a tooltip is for everywhere else in this game.
type handsToggle struct {
	modalToggle
}

// init wires the button wherever the screen wants it. A screen with a hand on it has somewhere
// better than the corner; see handsButtonPlace on the combat scene.
func (t *handsToggle) init(place func(gs *state.GlobalState) image.Point) {
	t.modalToggle.init(handsToggleLabel, handsButtonWidth, pileSlotSize, handsButtonText,
		place)
}

func (t *handsToggle) update(gs *state.GlobalState) bool {
	return t.modalToggle.update(gs, nil)
}

func (t *handsToggle) draw(gs *state.GlobalState, screen *ebiten.Image, c handsContents) {
	t.modalToggle.draw(gs, screen, func() { drawHandPanel(gs, screen, c) })
}

// ownedHands is the panel between fights: the run's whole deck, priced by the rings worn but by
// no duelist's strength, because a run's stats belong to a fight.
func ownedHands(gs *state.GlobalState) handsContents {
	c := handsContents{}
	if gs.Run != nil {
		c.deck = gs.Run.Deck()
		if f := buildFighter(gs); f != nil {
			c.holder = f.Duelist
		}
	}
	return c
}

// catalogueMultiplier is what one rung pays as shipped, by key. It is the figure a raised rung is
// compared against, and it is looked up rather than remembered so that the comparison cannot drift
// from what `hands.json` actually says.
func catalogueMultiplier(base []combat.Hand, key string) int {
	for _, h := range base {
		if h.Key == key {
			return h.Multiplier
		}
	}
	return 0
}
