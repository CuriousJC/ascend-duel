package screens

// **The combos panel: every rung of the hand ladder, written as a sum.**
//
// The multipliers are the whole of what building a hand buys, and until this panel landed
// *(2026-08-24)* they were readable in three places, none of which is the game: `data/hands.json`,
// `go run ./tools/handsheet`, and the fight log after a hand had already fired. A player deciding
// which cards to queue was being asked to remember nineteen numbers.
//
// **Two lines per rung, and they answer different questions.** The grammar line is the rule — what
// the rung wants, in the axis's own words, true whatever deck you are holding. The example line is
// this run's deck actually building it: the cheapest real cards that form the rung, their damage in
// your hands, and what the multiplier makes of the total. The rule alone leaves "so what would that
// be worth" unanswered; the example alone reads as a fact about three particular cards.
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

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// comboContents is what the panel is asked to show: the deck the examples are built out of, and
// who is holding it.
//
// **The holder prices the example**, exactly as it prices a card's face — a slash under Keen is
// worth a bigger figure — so the sum a player reads here is the sum they would actually deal.
// **A zero holder is honest rather than broken**: with no DMG known the example names its cards
// and drops the arithmetic, the same choice the card tooltip makes between fights.
type comboContents struct {
	deck   []combat.Card
	holder combat.Duelist
}

// comboRow is one rung, laid out as words.
type comboRow struct {
	name    string
	grammar string
	mult    string
	example string
}

// comboRows is the whole ladder, in the order it is drawn.
//
// **Sorted by multiplier, then by axis, then by ID.** The axis tie-break is the matcher's own —
// narrowest first, see combat.Axis — so two rungs paying the same read in the order the rules
// would pick between them, and the ID is the last key so the order cannot depend on how the
// catalogue happens to be filed.
func comboRows(c comboContents) []comboRow {
	hands := combat.Hands()
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

	out := make([]comboRow, 0, len(hands))
	for _, h := range hands {
		out = append(out, comboRow{
			name:    h.Name,
			grammar: comboGrammar(h),
			mult:    multiplierText(h.Multiplier),
			example: comboExample(c, h),
		})
	}
	return out
}

// comboGrammar is the rung as a sum of slots: one term per card the hand is formed from, named by
// what the hand counts on.
//
// **A group letter only appears when there is more than one group**, because that is the only time
// it says anything: a Three of a Kind wants three of the same thing and lettering them A, A, A is
// noise, where a Full House's `A A A B B` is the whole rule.
func comboGrammar(h combat.Hand) string {
	word := comboAxisWord(h.Match)
	var terms []string
	for g, want := range h.Groups {
		label := word
		if len(h.Groups) > 1 {
			label += " " + string(rune('A'+g))
		}
		for i := 0; i < want; i++ {
			terms = append(terms, label+" DMG")
		}
	}
	return strings.Join(terms, " + ")
}

// comboAxisWord is what an axis is called on this panel: the thing the cards have to share.
//
// **The axis's own String() is not used**, because those are the names `hands.json` writes —
// "concept" is the rules' word for a card, and a player has never seen it. The lookup is exhaustive
// over the enum rather than a map, so a fourth axis fails to compile here instead of drawing a
// blank term.
func comboAxisWord(a combat.Axis) string {
	switch a {
	case combat.AxisForm:
		return "FORM"
	case combat.AxisElement:
		return "ELEMENT"
	default:
		return "CARD"
	}
}

// comboExample is the cheapest real hand this deck can form for the rung, and what it comes to.
//
// **It is the deck's answer, not an invented one.** A rung the deck cannot build says so — which
// is a fact about the deck worth seeing rather than a gap to paper over, and it is how a run that
// has recoloured itself into one element discovers which rungs have gone out of reach.
func comboExample(c comboContents, h combat.Hand) string {
	cards, cost, ok := decks.CheapestExample(c.deck, h)
	if !ok {
		return "your deck cannot build this"
	}

	var terms []string
	total := 0
	for _, card := range cards {
		d := c.holder.CardDamage(card)
		total += d
		if c.holder.DMG > 0 {
			terms = append(terms, fmt.Sprintf("%s %d", card.Label(), d))
			continue
		}
		terms = append(terms, card.Label())
	}

	line := strings.Join(terms, " + ")
	// **Without a DMG figure the arithmetic is dropped rather than worked out against zero.**
	// A run's stats belong to a fight, so a shop has no strength to price a card at — the same
	// case the card tooltip answers by stating the multipliers and stopping.
	if c.holder.DMG > 0 {
		line += fmt.Sprintf(" = %d", total*h.Multiplier/100)
	}
	return fmt.Sprintf("%s  for %d AP", line, cost)
}

// The panel's own geometry: two columns of rungs, three lines each.
const (
	// **Two columns because nineteen rungs do not fit in one.** The panel gives about 711 pixels
	// between the heading and the closing hint and a rung needs three lines, so one column would
	// hold ten of nineteen and the ladder would be half a ladder. A third column would leave each
	// too narrow for the example line — TestEveryComboLineFitsItsColumn is what fails rather than
	// the panel quietly running its text off the edge.
	comboColumnCount = 2

	comboColumnGap = 40
	// **Nineteen rungs over two columns is ten in the deeper one**, and ten rows have to fit the
	// panel's ~711 pixels between the heading and the hint. TestTheColumnsHoldTheWholeLadder is
	// what fails when a rung is added rather than the last one being drawn under the closing line.
	comboRowHeight = 70

	// The three lines of a rung, as offsets down from the row's top.
	comboNameLine    = 0
	comboGrammarLine = 26
	comboExampleLine = 48

	comboNameSize    = 17
	comboGrammarSize = 14
	comboExampleSize = 13
)

// The panel's ink. It is written on the modal's dark fill, so these run light.
//
// **The multiplier is the one coloured figure**, because it is the one thing on the row that is
// not a description: the name says which rung, the grammar says what it wants, and the multiplier
// is what forming it pays.
var (
	comboNameInk    = color.RGBA{R: 236, G: 232, B: 226, A: 255}
	comboGrammarInk = color.RGBA{R: 186, G: 196, B: 214, A: 255}
	comboExampleInk = color.RGBA{R: 146, G: 142, B: 136, A: 255}
	comboMultInk    = color.RGBA{R: 240, G: 198, B: 108, A: 255}
)

// comboColumns splits the ladder into the columns it is drawn in, filling the first column top to
// bottom before the second starts.
//
// **Down then across, not across then down.** The rows are in multiplier order and the point of
// that order is that reading down the list walks up the ladder; snaking across two columns would
// interleave the cheap rungs with the dear ones.
func comboColumns(rows []comboRow, columns int) [][]comboRow {
	if columns < 1 {
		columns = 1
	}
	per := (len(rows) + columns - 1) / columns

	out := make([][]comboRow, 0, columns)
	for i := 0; i < len(rows); i += per {
		end := i + per
		if end > len(rows) {
			end = len(rows)
		}
		out = append(out, rows[i:end])
	}
	return out
}

// comboBodyRect is the room the rungs are laid out in: the panel, less the heading above and the
// closing hint below, and inset at the sides by the margin the deck panel's rows keep.
func comboBodyRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+deckRowMargin, r.Min.Y+modalBodyTop,
		r.Max.X-deckRowMargin, r.Max.Y-modalBodyBottom)
}

// comboColumnWidth is how wide one column is. **One function rather than the arithmetic written
// twice**, because the drawing and the test that holds the text inside it must not be able to
// disagree about the answer.
func comboColumnWidth(body image.Rectangle, columns int) int {
	if columns < 1 {
		columns = 1
	}
	return (body.Dx() - comboColumnGap*(columns-1)) / columns
}

// drawComboPanel covers the screen with the ladder.
func drawComboPanel(gs *state.GlobalState, screen *ebiten.Image, c comboContents) {
	rows := comboRows(c)

	r := drawModalFrame(gs, screen, modalHead{
		title:  "Combos",
		counts: fmt.Sprintf("%d hands - the multiplier is what forming one pays", len(rows)),
		legend: "every hand your cards can make, cheapest-paying first",
	})

	columns := comboColumns(rows, comboColumnCount)
	body := comboBodyRect(r)
	colWidth := comboColumnWidth(body, len(columns))

	for i, column := range columns {
		left := body.Min.X + i*(colWidth+comboColumnGap)
		for j, row := range column {
			drawComboRow(gs, screen, row, left, body.Min.Y+j*comboRowHeight, colWidth)
		}
	}
}

// drawComboRow writes one rung: its name and multiplier on a line, the rule under it, the deck's
// own example under that.
func drawComboRow(gs *state.GlobalState, screen *ebiten.Image, row comboRow, left, top, width int) {
	write := func(down, size int, ink color.RGBA, s string, right bool) {
		op := &text.DrawOptions{}
		x := left
		if right {
			x = left + width
			op.PrimaryAlign = text.AlignEnd
		}
		op.GeoM.Translate(float64(x), float64(top+down))
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, s,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: float64(size)}, op)
	}

	write(comboNameLine, comboNameSize, comboNameInk, row.name, false)
	write(comboNameLine, comboNameSize, comboMultInk, row.mult, true)
	write(comboGrammarLine, comboGrammarSize, comboGrammarInk, row.grammar, false)
	write(comboExampleLine, comboExampleSize, comboExampleInk, row.example, false)
}

// comboToggle is the combos panel behind a button, the deck panel's counterpart.
//
// **It carries no tooltip.** The panel is already all words — there is nothing on it whose meaning
// is hidden behind a picture, which is what a tooltip is for everywhere else in this game.
type comboToggle struct {
	modalToggle
}

// init wires the button wherever the screen wants it. A screen with a hand on it has somewhere
// better than the corner; see comboButtonPlace on the combat scene.
func (t *comboToggle) init(place func(gs *state.GlobalState) image.Point) {
	t.modalToggle.init(combosToggleLabel, combosButtonWidth, logButtonSize, combosButtonText,
		place)
}

func (t *comboToggle) update(gs *state.GlobalState) bool {
	return t.modalToggle.update(gs, nil)
}

func (t *comboToggle) draw(gs *state.GlobalState, screen *ebiten.Image, c comboContents) {
	t.modalToggle.draw(gs, screen, func() { drawComboPanel(gs, screen, c) })
}

// ownedCombos is the panel between fights: the run's whole deck, priced by the rings worn but by
// no duelist's strength, because a run's stats belong to a fight.
func ownedCombos(gs *state.GlobalState) comboContents {
	c := comboContents{}
	if gs.Run != nil {
		c.deck = gs.Run.Deck()
		if f := buildFighter(gs); f != nil {
			c.holder = f.Duelist
		}
	}
	return c
}
