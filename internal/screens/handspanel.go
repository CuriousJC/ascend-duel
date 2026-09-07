package screens

// **The hands panel: every rung of the hand ladder, drawn as the cards that build it.**
//
// The multipliers are the whole of what building a hand buys, and until this panel landed
// *(2026-08-24)* they were readable in three places, none of which is the game: `data/hands.json`,
// `go run ./tools/handsheet`, and the fight log after a hand had already fired. A player deciding
// which cards to queue was being asked to remember eighteen numbers.
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
// **A merged rung is drawn once per axis it reads** *(owner's call, 2026-09-07)*. The Pair is
// `"match": "any"` — it fires on whichever of concept, form and element the turn happens to satisfy
// — and the panel drew it as the concept reading alone, which is a picture of a rung that does not
// exist. It is now three examples side by side, captioned CARD / FORM / ELEMENTAL in the wording the
// unmerged rungs on the same page use, with a hairline between them: six tokens at the row's own
// pitch would read as one hand of six, which is the one thing this row must not say.
//
// **A rung is one block, ruled off from the next** *(owner's call, 2026-08-24)*, and the block is
// what holds a figure to the rung it belongs to: three columns of names, cards and figures
// otherwise read as three columns of *lists*, with the eye pairing each figure with whatever is
// nearest it. The rule under each rung is what makes that safe, and it is why the multiplier can
// stand at the column's own edge.
//
// **The multiplier is right-justified and the plays count sits beside the last card** *(owner's
// call, 2026-09-07)*, which is the two of them the other way round from the day before. The
// multiplier is the figure the panel exists to put in front of a player, so it gets the column of
// its own that a justified edge makes — eighteen rungs of it in a line is a ladder that can be read
// down. The plays count is a fact about the run rather than about the rung, so it takes the seat
// beside the cards.
//
// **Every rung including the merged one**: the Pair put its figure on the title line for as long
// as its three examples and a `HANDS PLAYED:` label would not both fit the band, and shortening the
// label to `PLAYED:` bought the room back. One placement is worth more than the argument for the
// exception was — a rung whose figure is somewhere else is a rung a reader has to look twice at.
//
// **Both of a rung's counters are drawn on every rung, and they sit apart** *(owner's call,
// 2026-09-07)*. The **level is a parenthetical on the title** — `PAIR (LVL 1)` — because it is
// part of what this rung *is* on this run, the same fact the multiplier is already coloured for.
// The **plays count is centred down the card band**, beside the last card — `PLAYED: 0` — because
// it is a fact about the run rather than about the rung. Neither is omitted at zero: a blank says
// the panel has no opinion, which is how a player comes to think a level is something only some
// rungs have.
//
// **The titles are shouted** *(owner's call, 2026-09-07)*, so nothing on the panel is lower case.
// It costs no width — kubasta is monospaced, and a capital and a lower-case letter take the same
// advance — so this is purely what the panel looks like. The one thing left in lower case is the
// `x` on a multiplier, which is `multiplierText` and is shared with the card faces and the
// tooltips; changing it here alone would make the panel disagree with them.
//
// **Ordered by multiplier, cheapest-paying first, across all three axes at once.** That is the
// comparison the multipliers are making — an Elemental Three of a Kind at 145 says it is worth
// about what a Form Three of a Kind is — and it is exactly the comparison `hands.json`'s
// axis-by-axis layout hides. tools/handsheet orders its page the same way for the same reason.
//
// The panel's chrome is the shared one; see modal.go.

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/session"
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

	// plays is how many times this run has formed each rung, by hand key - see
	// session/play.go. **Handed in rather than read off the run**, because this panel is drawn
	// from two places and one of them is mid-fight, where the scene already holds the run.
	//
	// **The level is not here**, because the holder already carries it: a level is a stone, and
	// `combat.Duelist.HandStoneCount` is the same number the ladder itself is read through. Two
	// sources for one figure is how a panel comes to disagree with the multiplier beside it.
	plays map[string]int
}

// runPlays is a run's tally, or nil for no run. **A helper rather than the map inline**, so the
// two callers cannot come to two answers about what no run means.
func runPlays(run *session.Session) map[string]int {
	if run == nil {
		return nil
	}
	return run.PlayCounts()
}

// handsRow is one rung: its name and what forming it pays, and the hand this deck illustrates it
// with.
type handsRow struct {
	name string
	mult string

	// sets is the hands this deck illustrates the rung with, one per axis the rung may be read on.
	// **Almost every rung has exactly one** and this is a slice of one; the Pair is the exception
	// and carries three *(owner's call, 2026-09-07)*.
	//
	// **A merged rung drawn as one example was the panel claiming it counted one thing.** The Pair
	// is `"match": "any"` and fires on whichever of concept, form and element the turn satisfies —
	// and the panel drew it as the concept reading alone, which is a picture of a rung that does
	// not exist. Three sets, captioned with the axis each is read on, is the rule in the same
	// pictures the rest of the ladder is drawn in.
	sets [][]combat.Card

	// axes is what each set is read on, parallel to sets. **Empty for a rung with one axis**,
	// because its name already says which — "Elemental Two Pair" needs no caption reading
	// ELEMENTAL under it.
	axes []combat.Axis

	// plays is how many times this run has formed the rung, and level is how many stones stand on
	// it *(owner's call, 2026-09-05)*. **Two counters and not one**, because they say different
	// things: a level is something the player bought and it moves the multiplier beside it, and a
	// play is something the player did and moves nothing at all.
	//
	// **Level 1 is a rung as shipped** *(owner's call, 2026-09-07)*, so `level` is the stone count
	// plus one and never zero. It read the stones directly until then and drew nothing at zero, on
	// the argument that eighteen rungs saying "LVL 0" is noise — which was true of a *blank* rung
	// and false of a ladder: a level a player can raise is one they have to be able to see the
	// bottom of, and "LVL 0" for the rung as bought reads as a rung with nothing on it rather than
	// as the first step. The plays count is drawn at zero for the same reason.
	plays int
	level int

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
		sets, axes := handsExamples(c, h)
		out = append(out, handsRow{
			name:   h.Name,
			mult:   multiplierText(h.Multiplier),
			sets:   sets,
			axes:   axes,
			plays:  c.plays[h.Key],
			level:  c.holder.HandStoneCount(h.Key) + 1,
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
// **A rung read on more than one axis is illustrated once per axis** *(owner's call, 2026-09-07)*,
// and the axes come back beside the sets so the drawing can caption them. `Hand.On` is what narrows
// the rung to one reading, and it is the matcher's own function — so each set really is a hand that
// rung would score, rather than three examples this panel invented.
func handsExamples(c handsContents, h combat.Hand) ([][]combat.Card, []combat.Axis) {
	axes := h.Axes
	if len(axes) < 2 {
		hand, _ := decks.Example(c.deck, h)
		return [][]combat.Card{hand}, nil
	}

	sets := make([][]combat.Card, 0, len(axes))
	for _, a := range axes {
		hand, _ := decks.Example(c.deck, h.On(a))
		sets = append(sets, hand)
	}
	return sets, append([]combat.Axis(nil), axes...)
}

// handsAxisWord is what a caption under a set reads. **The catalogue's own wording, not the
// matcher's** — a player has "Card Two Pair" and "Elemental Two Pair" on the same page, so the
// merged rung's captions have to be the words those rungs use rather than `Axis.String`'s
// `concept` and `element`.
func handsAxisWord(a combat.Axis) string {
	switch a {
	case combat.AxisForm:
		return "FORM"
	case combat.AxisElement:
		return "ELEMENTAL"
	default:
		return "CARD"
	}
}

// The panel's own geometry: three columns of rungs, a name over a row of cards.
const (
	// **Three columns as of 2026-08-24**, up from two. A rung was three lines of words and is now
	// a name over a row of cards, which is wider and shorter — so the ladder wants columns rather
	// than depth. Eighteen rungs over three is six in the deepest, against a budget that holds
	// seven: TestTheColumnsHoldTheWholeLadder is what fails when a rung is added past that, and
	// the headroom is the point — the panel was rebuilt this way to leave room for hands not yet
	// written.
	handsColumnCount = 3

	handsColumnGap = 28

	// A name line, a row of tokens under it, and a rule under that. See handsCardsTop.
	handsNameLine = 0

	// handsRowGap is the air between the bottom of a rung's cards and the hairline that closes it,
	// and again between that hairline and the next rung's name.
	handsRowGap = 6

	// handsMultGap is the air between the last card and the multiplier. **Beside the cards, not
	// out at the column's edge** *(owner's call, 2026-08-24)*: the figure belongs to the hand it
	// prices, and distance is what says so.
	handsMultGap = 12

	// handsNameSize is the rung's own title, and **it is the dial the whole row is built off**:
	// handsCardsTop is derived from it, so moving this moves the block that holds the cards rather
	// than letting the title grow into them.
	handsNameSize = 28

	// handsMultSize is what forming the rung pays, and it is **the one figure written larger than
	// the rung's own title**. It has been a shade over the title since the panel was built (20
	// against 17) and it scales with it: the name says which rung and the multiplier says what it
	// is worth, which is the thing the panel exists to put in front of a player mid-hand.
	handsMultSize = 32

	// handsTallySize is the HANDS PLAYED line, **written at the title's own size** *(owner's call,
	// 2026-09-07)*. It was 13 and then 16, on the argument that a tally is an annotation; it is
	// drawn on every rung now and it is the figure a player opens this panel to check, so it is
	// read at the size of the thing it belongs to.
	handsTallySize = handsNameSize

	// handsSetGap is the air between one axis's example and the next, on a rung read on more than
	// one, and it is sized to hold the word that goes in it. **Wider than the two pixels between
	// cards of one set**: six tokens at the row's own pitch read as one hand of six, which is the
	// one thing this row must not say.
	handsSetGap = 28

	// handsOrSize is the "OR" between two of a merged rung's examples. **A word rather than the
	// hairline it replaces** *(owner's call, 2026-09-07)*: a rule says the sets are separate and
	// leaves what separates them to be worked out, where the word says the rung takes whichever of
	// them the turn happens to make. It is written under the title's size on purpose — it is the
	// one thing in the band that is not a card, and at the title's size it would be read first.
	handsOrSize = 20

	// handsAxisSize is the caption naming which axis a set is read on, and handsAxisBand is what
	// the line costs the rung in depth — the type plus the air under it.
	handsAxisSize = 20
	handsAxisBand = handsAxisSize + 3
)

// handsCardsTop is where a rung's tokens start, measured from its own top. **Derived from the
// title rather than written down** *(2026-09-07)*, for the reason handsRuleDrop is derived from the
// token: a constant here has to be kept in step with a type size by hand, and the failure is a
// title drawn through the cards under it.
var handsCardsTop = handsNameSize + 4

// handsRuleDrop is where the hairline separating one rung from the next sits, measured from the
// row's top. **It is what makes a rung one block rather than three lists** - a name, a row of cards
// and a figure in three columns otherwise read down the page instead of across, and a reader pairs
// each figure with whichever thing is nearest, which at a column's edge is the rung below.
//
// **It is derived from the token rather than written down** *(2026-09-05)*. It was `handsCardsTop +
// 62` against a token that is 70 tall, so every rung's cards overhung its own rule and landed on
// the name of the rung beneath - which is what the panel actually looked like. A measurement that
// has to be kept in step with `cards.Token` by hand is one that will not be.
var handsRuleDrop = handsCardsTop + cards.Token.Height + handsRowGap

// handsRowHeight is the pitch from one rung to the next: its whole block, and the air under the
// rule before the next name. Derived for the reason handsRuleDrop is.
var handsRowHeight = handsRuleDrop + handsRowGap + handsNameSize

// The three measurements above are the plain rung's. A rung carrying axis captions is that much
// taller, and these are the same three read against a row rather than assumed.
//
// **The column stacks by accumulated depth rather than by a fixed pitch** *(2026-09-07)*, which is
// what a row of two heights costs. A pitch is cheaper and was right while every rung was the same
// block; multiplying an index by it once one rung is taller draws the ladder through itself.
func handsCardsTopFor(row handsRow) int { return handsCardsTop + handsCaptionBand(row) }
func handsRuleDropFor(row handsRow) int { return handsRuleDrop + handsCaptionBand(row) }
func handsRowDepth(row handsRow) int    { return handsRowHeight + handsCaptionBand(row) }

// handsCaptionBand is the depth a rung's axis captions cost it, or zero for a rung that names its
// own axis.
func handsCaptionBand(row handsRow) int {
	if len(row.axes) < 2 {
		return 0
	}
	return handsAxisBand
}

// handsColumnDepth is how tall a column of rungs stands: every block, less the air the last one
// leaves under its own rule.
func handsColumnDepth(column []handsRow) int {
	if len(column) == 0 {
		return 0
	}
	tall := 0
	for _, row := range column {
		tall += handsRowDepth(row)
	}
	last := column[len(column)-1]
	return tall - (handsRowDepth(last) - handsRuleDropFor(last))
}

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

	// The tally is written in the ground's quietest ink. **No hue of its own** - the wheel is full,
	// and the two counters are told apart by their words rather than by a colour a player would
	// have to learn. See CLAUDE.md on hue belonging to the elements.
	handsTallyInk = color.RGBA{R: 150, G: 146, B: 141, A: 255}
	handsMultInk  = color.RGBA{R: 240, G: 198, B: 108, A: 255}

	// The rule is barely there on purpose: it groups a rung, and a line loud enough to be read as
	// a border would make eighteen boxes out of a ladder.
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
const handsPanelTitle = "DUELIST HANDS"

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

// handsSetsWidth is the whole band of examples: every set, and the air between them.
func handsSetsWidth(sets [][]combat.Card) int {
	span := 0
	for i, set := range sets {
		if i > 0 {
			span += handsSetGap
		}
		span += handsCardsWidth(len(set))
	}
	return span
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
		top := body.Min.Y
		for _, row := range column {
			drawHandRow(gs, screen, c, row, left, top, colWidth)
			top += handsRowDepth(row)
		}
	}
}

// drawHandRow writes one rung: its name, the cards this deck builds it from, the multiplier
// beside them, and the rule that closes the block.
func drawHandRow(gs *state.GlobalState, screen *ebiten.Image, c handsContents, row handsRow,
	left, top, width int) {

	face := func(size float64) *text.GoTextFace {
		return &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}
	}
	write := func(x, y int, size float64, ink color.RGBA, s string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, s, face(size), op)
	}
	widthOf := func(s string, size float64) int {
		adv, _ := text.Measure(s, face(size), 0)
		return int(adv)
	}

	title := handsTitleText(row)
	write(left, top+handsNameLine, handsNameSize, handsNameInk, title)

	multInk := handsMultInk
	if row.raised {
		// **A raised rung is written in `boostInk`, which is the ring pink.** It is reused rather
		// than given a hue of its own: on a card it means "a ring moved this figure", and here it
		// means "a stone did" — the shared reading is *something you bought moved this number*,
		// which is the thing a player needs to see. A second pink-ish hue for the second source
		// would be two colours a player has to tell apart to learn the same fact.
		multInk = boostInk
	}

	cardsTop := top + handsCardsTopFor(row)

	// **The cards are drawn as themselves**, through the same spec every other screen builds, so
	// a token cannot say something the card in the hand does not. `enabled` is true and nothing is
	// selected: this is a catalogue, and a dimmed card here would mean "unaffordable" against a
	// budget no round has yet set.
	x := left
	for i, set := range row.sets {
		if i > 0 {
			// **"OR" in the air between two sets**, centred both ways in the gap. It replaced a
			// hairline: a rule says the sets are separate and leaves what separates them to be
			// worked out, where the word says the rung takes whichever of them the turn makes.
			write(x-handsSetGap/2-widthOf(handsOrWord, handsOrSize)/2,
				cardsTop+handsMultDrop(handsOrSize), handsOrSize, handsTallyInk, handsOrWord)
		}
		if i < len(row.axes) {
			write(x, cardsTop-handsAxisBand, handsAxisSize, handsTallyInk,
				handsAxisWord(row.axes[i]))
		}
		for j, card := range set {
			at := image.Pt(x+j*handsCardPitch, cardsTop)
			drawCard(gs, screen, at, cards.Token, card, heldBy(c.holder, card), true, false)
		}
		x += handsCardsWidth(len(set)) + handsSetGap
	}

	// **The plays count takes the seat beside the last card and the multiplier is justified to the
	// column's edge** *(owner's call, 2026-09-07)*. Both are centred down the token band, which is
	// what stops either reading as a caption on whichever card it happens to sit next to; the
	// multiplier gets the justified edge because eighteen of them in a line is a ladder that can be
	// read down, which is the whole of what this panel is for.
	tally := handsTallyText(row)
	write(left+handsSetsWidth(row.sets)+handsMultGap, cardsTop+handsMultDrop(handsTallySize),
		handsTallySize, handsTallyInk, tally)
	write(left+width-widthOf(row.mult, handsMultSize),
		cardsTop+handsMultDrop(handsMultSize), handsMultSize, multInk, row.mult)

	rule := float32(top + handsRuleDropFor(row))
	vector.StrokeLine(screen, float32(left), rule, float32(left+width), rule, 1, handsRuleInk, false)
}

// handsOrWord is what stands between two of a merged rung's examples.
const handsOrWord = "OR"

// handsTitleText is the rung's name and the level this run has it at. **The level is a
// parenthetical on the title** *(owner's call, 2026-09-07)*, because it says what this rung *is*
// here rather than what has happened to it — the same thing the multiplier beside the cards is
// already written in the ring pink to say.
//
// **Shouted, and it is free.** kubasta is monospaced, so the capitals measure exactly what the
// catalogue's own casing did; nothing about the layout depends on this and the whole of what it
// buys is a panel with nothing lower case on it.
func handsTitleText(row handsRow) string {
	return fmt.Sprintf("%s (LVL %d)", strings.ToUpper(row.name), row.level)
}

// handsTallyText is how often this run has formed the rung, and **every rung carries one**
// *(owner's call, 2026-09-07)*.
//
// It was drawn only where there was something to say, on the argument that eighteen rungs reading
// "PLAYED 0" is noise. What that missed is that the figure is defined for every rung all the time,
// and a blank says something else: that the panel has no opinion. Drawn everywhere, it is also a
// column the eye can run down.
//
// **`PLAYED:` rather than `HANDS PLAYED:`** *(owner's call, 2026-09-07)*. The longer label was
// what pushed the Pair's multiplier off the card band and onto its title line, and the word it
// spends 61 pixels on is one the panel's own heading has already said. The colon is what keeps it
// reading as a count rather than as a label on the card beside it.
func handsTallyText(row handsRow) string {
	return fmt.Sprintf("PLAYED: %d", row.plays)
}

// handsTallyGap is the air between the rung's name and its tally, and handsTallyDrop is what sits
// the smaller type on the name's baseline rather than on its top edge.
const (
	handsTallyGap  = 10
	handsTallyDrop = handsNameSize - handsTallySize
)

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

// initInColumn wires it as a rung of the combat screen's control column: the column's width, the
// column's button height, and the same label size every other rung uses.
//
// **A second constructor rather than a width argument on the first**, because the two are
// different placements rather than one placement in two sizes — the corner form is what a screen
// with no hand on it gets, and it stands beside the deck button there.
func (t *handsToggle) initInColumn(place func(gs *state.GlobalState) image.Point) {
	t.modalToggle.init(handsToggleLabel, ControlButtonWidth, ControlButtonHeight,
		ControlButtonText, place)
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
		c.plays = runPlays(gs.Run)
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
