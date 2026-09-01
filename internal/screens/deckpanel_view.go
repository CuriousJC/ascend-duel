package screens

// **How the deck panel is being read**: which face each card is shown in, and which half of the
// deck the numbers are about. Two buttons along the bottom of the panel, and the three tallies they
// govern.
//
// It is a separate file from deckpanel.go because that one answers "where does every card go" and
// this one answers "which deck am I looking at". The grid's arithmetic is hard enough to read
// without a tally block in the middle of it.
//
// **Neither toggle can change anything.** This is a reading preference over a picture of a deck —
// the same standing as the hand's sort column, and for the same reason it was safe to build without
// asking what it does to the engine. `ResolveRound` never sees it, the piles are never touched, and
// closing the panel leaves the fight exactly where it was.

import (
	"fmt"
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The two buttons. **Words rather than letters**, unlike the corner squares and the sort column:
// they stand in the middle of an otherwise empty band at the foot of a full-screen panel, so there
// is nothing to be short for, and neither state is guessable from an initial.
const (
	deckViewButtonWidth  = 168
	deckViewButtonHeight = 40
	deckViewButtonText   = 18
	deckViewButtonGap    = 16

	// deckViewButtonBottom is how far the button line's centre sits up from the panel's bottom
	// edge. It clears modalBodyBottom with the button's own half-height inside it.
	deckViewButtonBottom = 42

	// The labels. **Each names the state the panel is in, not the state pressing would move to.**
	// A latched button already says "this is on", so a label naming the other side would have the
	// two contradicting each other.
	deckViewAlteredLabel   = "ALTERATIONS"
	deckViewUnalteredLabel = "AS OWNED"
	deckViewFullLabel      = "FULL"
	deckViewPlayedLabel    = "PLAYED"
)

// deckView is how a panel is being read, and it belongs to whoever puts the panel up.
//
// **A reading preference is not a fact about a fight**, exactly like the hand's `sortMode`: it
// survives the panel being closed and reopened, because a player who has chosen a view should not
// have to re-choose it every look.
type deckView struct {
	// unaltered inverts the default. **Alterations are on unless this is set** — a deck with a flip
	// ring in it is dealt in colours the owned list does not have, and a panel showing the list is
	// showing a deck the player will never draw.
	unaltered bool

	// played picks which half of the deck the tallies count and which half is drawn lit. **False is
	// the whole deck**, which is the question between fights and the commoner one in a fight.
	played bool

	alterations *models.Button
	half        *models.Button
}

// update runs the two buttons. It is called only while the panel is up, so neither can be pressed
// through a screen that is not showing them.
func (v *deckView) update(gs *state.GlobalState, d deckContents) {
	v.build()
	v.layout(gs)

	systems.UpdateButton(gs, v.alterations)
	if d.inFight {
		systems.UpdateButton(gs, v.half)
	}
}

// build makes the two buttons on first use. **Built once and kept**, like every other widget on
// these screens: a button rebuilt each frame would lose the press it was in the middle of.
func (v *deckView) build() {
	if v.alterations != nil {
		return
	}
	v.alterations = models.NewButton(deckViewButtonWidth, deckViewButtonHeight, "",
		func() { v.unaltered = !v.unaltered })
	v.alterations.BaseColor = sortButtonColor
	v.alterations.TextSize = deckViewButtonText

	v.half = models.NewButton(deckViewButtonWidth, deckViewButtonHeight, "",
		func() { v.played = !v.played })
	v.half.BaseColor = sortButtonColor
	v.half.TextSize = deckViewButtonText
}

// layout puts the buttons where they are drawn and hit-tested, and writes what each of them
// currently says. **One function, called from update and from draw**, so the rectangle a press is
// measured against is the rectangle the label was drawn in.
func (v *deckView) layout(gs *state.GlobalState) {
	r := modalPanelRect(gs)
	y := r.Max.Y - deckViewButtonBottom

	// **Both buttons are placed as though both are there**, whether or not the second is drawn. A
	// pair that recentred itself when one of them went away would move the alterations button
	// sideways between a fight and a shop, and it is the same control in both.
	span := 2*deckViewButtonWidth + deckViewButtonGap
	left := r.Min.X + r.Dx()/2 - span/2

	v.alterations.ScreenX = left + deckViewButtonWidth/2
	v.alterations.ScreenY = y
	v.alterations.Text = deckViewAlteredLabel
	v.alterations.Latched = !v.unaltered
	if v.unaltered {
		v.alterations.Text = deckViewUnalteredLabel
	}

	v.half.ScreenX = left + deckViewButtonWidth + deckViewButtonGap + deckViewButtonWidth/2
	v.half.ScreenY = y
	v.half.Text = deckViewFullLabel
	v.half.Latched = v.played
	if v.played {
		v.half.Text = deckViewPlayedLabel
	}
}

// draw puts the two buttons on the panel. The tallies are drawn by drawDeckPanel, which owns the
// band they sit in.
//
// **The alterations button is live and latching even when it does nothing** *(owner's call,
// 2026-08-24)*. Most runs wear no altering ring, and a button that vanished when it had nothing to
// do would be a control the player never learned existed — and would leave them unable to confirm
// that a ring they have just bought is doing anything. It latches, the deck under it does not move,
// and that is the honest answer.
func (v *deckView) draw(gs *state.GlobalState, screen *ebiten.Image, d deckContents) {
	v.build()
	v.layout(gs)

	systems.DrawButton(gs, screen, v.alterations)

	// **No FULL/PLAYED between fights**, because nothing has been played: there is one pile and
	// both states of the button would be the same picture. See deckContents.inFight.
	if d.inFight {
		systems.DrawButton(gs, screen, v.half)
	}
}

// The tally band under the grid: three ways of counting the same cards, side by side.
//
// **Three at once rather than a fourth toggle** *(owner's call, 2026-08-24)*. The question a deck
// panel is opened to answer is a comparison — how much slash have I got, how much of it is cheap,
// how much of the deck is fire — and a player made to cycle between the three has to hold two of
// the answers in their head while looking at the third.
const (
	// tallyTop is how far under the grid the band starts.
	tallyTop = 14

	tallyTitleSize = 15
	tallyTextSize  = 17
	tallyRowHeight = 28

	// tallyHeadDrop is where a block's first row of figures starts, measured from the block's own
	// top. It clears two lines: the block's heading, and the AP headings the middle block writes
	// under its own. Both are written at tallyTitleSize, so this is two of those plus air.
	tallyHeadDrop = 40

	// tallyColumnHead is where the middle block writes its AP headings — between the block's
	// heading and its first row of figures.
	tallyColumnHead = 24
	tallyMarkSize   = 26

	// tallyCostColumn is how wide one AP column is in the middle block. Wide enough for a two-digit
	// count under a `4 AP` heading, which a recoloured deck can reach.
	tallyCostColumn = 52

	// tallyFigureGap is the air between a row's mark and the figure beside it.
	tallyFigureGap = 12
)

// tallyInk is the band's own text colour: the panel is dark, so this is the near-white the title
// and the counts line are already written in.
var tallyInk = color.RGBA{R: 236, G: 236, B: 240, A: 255}

// tallyFadedInk is a heading, and a zero. **A count of none is written rather than left blank**,
// because a blank cell reads as a column that does not exist — and "I own no 4 AP crush" is exactly
// the kind of thing a player opens this panel to find out.
var tallyFadedInk = color.RGBA{R: 130, G: 130, B: 144, A: 255}

// maxTallyCost is how far up the cost axis the tally will look. **A ceiling on the walk, not a cap
// on a card**: a card declares its own cost and a stack of worms could in principle push one past
// this, at which point it would be counted in the form and element blocks and missing from the
// middle one. Ten is far enough above the four the game ships that reaching it means something else
// is wrong.
const maxTallyCost = 10

// deckTally is the three counts, taken over whichever cards the view is showing.
//
// **It counts the faces on screen, not the cards underneath.** With alterations on, a lightning
// card dealt as ice is counted as ice — the tally has to agree with the grid above it or one of the
// two is lying, and the grid is the one the player is looking at.
type deckTally struct {
	byForm map[combat.Form]int

	// byElement is keyed on the *drawing* package's element, which is what the grid's rows are
	// keyed on. Counting the rules' element instead would need an inverse of `artFor` written for
	// this one block, and an element the drawing package cannot show would then be counted in a row
	// that is not on the panel.
	byElement map[cards.Element]int

	// byFormCost is the cross-tab: how many of each form at each AP cost. **Cost is asked of the
	// card as its holder pays it**, so a discount ring moves a card between columns here exactly as
	// it moves the ticks on the card's own face.
	byFormCost map[combat.Form]map[int]int

	// costs is every cost that appears, in ascending order, so the middle block draws a column per
	// cost the deck actually holds rather than a fixed 0..4 that goes stale when a worm invents a
	// rung.
	costs []int

	total int
}

// tallyOf counts the lit cards of a laid-out grid.
//
// **It walks the grid rather than the piles**, which is what keeps it honest: the grid is where the
// alterations toggle has already chosen a face and the FULL/PLAYED toggle has already chosen which
// cards are lit, so a tally taken from anywhere else would be a second answer to both questions.
func tallyOf(slots []pileSlot, holder combat.Duelist) deckTally {
	t := deckTally{
		byForm:     map[combat.Form]int{},
		byElement:  map[cards.Element]int{},
		byFormCost: map[combat.Form]map[int]int{},
	}

	seen := map[int]bool{}
	for _, s := range slots {
		if !s.lit {
			continue
		}
		f := s.card.Form()
		cost := holder.CardCost(s.card)

		t.byForm[f]++
		t.byElement[artFor(s.card.Element)]++
		if t.byFormCost[f] == nil {
			t.byFormCost[f] = map[int]int{}
		}
		t.byFormCost[f][cost]++
		seen[cost] = true
		t.total++
	}

	// A sorted slice rather than the map, because Go randomises map order and a block whose columns
	// swapped places between frames would be unreadable.
	for cost := 0; cost <= maxTallyCost; cost++ {
		if seen[cost] {
			t.costs = append(t.costs, cost)
		}
	}
	return t
}

// drawTallies writes the three blocks across the band under the grid.
//
// **Three fixed columns rather than three measured ones.** The blocks change width as a deck's cost
// spread changes, and a band that re-packed itself would move the element list sideways every time
// a worm invented a rung.
func drawTallies(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle, top int,
	t deckTally) {

	left := r.Min.X + deckRowMargin

	drawFormTally(gs, screen, left, top, t)
	drawCostTally(gs, screen, left+r.Dx()/4, top, t)
	drawElementTally(gs, screen, r.Min.X+r.Dx()*2/3, top, t)
}

// drawFormTally is the first block: how much of each form, whatever it costs and whatever colour it
// is.
func drawFormTally(gs *state.GlobalState, screen *ebiten.Image, left, top int, t deckTally) {
	tallyTitle(gs, screen, left, top, fmt.Sprintf("BY FORM  (%d cards)", t.total))

	for i, f := range tallyForms() {
		y := top + tallyHeadDrop + i*tallyRowHeight
		drawFormMark(screen, f, left, y)
		tallyFigure(gs, screen, left+tallyMarkSize+tallyFigureGap, y, t.byForm[f])
	}
}

// drawCostTally is the middle block, and the one the other two cannot say: how much of each form at
// each price. **A form's row reads as a curve** — four cheap stabs and one dear one is a different
// deck from the reverse, and the hand ladder is built out of which.
func drawCostTally(gs *state.GlobalState, screen *ebiten.Image, left, top int, t deckTally) {
	tallyTitle(gs, screen, left, top, "BY FORM AND AP")

	// The header names each column's cost. Written over the figures rather than beside the block,
	// because a column of numbers with no heading is a column nobody can read.
	headLeft := left + tallyMarkSize + tallyFigureGap
	for i, cost := range t.costs {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(headLeft+i*tallyCostColumn+tallyCostColumn/2),
			float64(top+tallyColumnHead))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(tallyFadedInk)
		text.Draw(screen, fmt.Sprintf("%d AP", cost),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tallyTitleSize}, op)
	}

	for i, f := range tallyForms() {
		y := top + tallyHeadDrop + i*tallyRowHeight
		drawFormMark(screen, f, left, y)
		for j, cost := range t.costs {
			tallyFigureAt(gs, screen, headLeft+j*tallyCostColumn+tallyCostColumn/2, y,
				t.byFormCost[f][cost], text.AlignCenter)
		}
	}
}

// drawElementTally is the third block: how much of each colour.
//
// **It says in a number what the grid says in a row length.** The rows are already one element
// each, so a long row is a lot of fire — but "is that eleven or thirteen" is not a question a row of
// overlapping cards answers, and it is the question a player counting toward an elemental hand is
// actually asking.
func drawElementTally(gs *state.GlobalState, screen *ebiten.Image, left, top int, t deckTally) {
	tallyTitle(gs, screen, left, top, "BY ELEMENT")

	for i, e := range deckRowElements() {
		y := top + tallyHeadDrop + i*tallyRowHeight

		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(left), float64(y+tallyMarkSize/2))
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(cards.BorderOf(e))
		text.Draw(screen, e.String(),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tallyTextSize}, op)

		tallyFigure(gs, screen, left+96, y, t.byElement[e])
	}
}

// tallyForms is the order the two form blocks run in: the deck panel's own, so a form is in the same
// place in the tally as it is along a grid row.
func tallyForms() []combat.Form {
	return []combat.Form{combat.FormStab, combat.FormSlash, combat.FormCrush, combat.FormDefend}
}

// drawFormMark puts a form's own drawing at the head of a row, **untinted**. The card's corner tints
// the same art by the card's element, which is what makes an element legible on a card; a tally row
// is about the form, and a coloured mark there would be claiming one.
func drawFormMark(screen *ebiten.Image, f combat.Form, left, top int) {
	kind, ok := form(f).Glyph()
	if !ok {
		return
	}

	img := systems.Glyph(kind, systems.PaletteWhite)
	size := img.Bounds().Dx()
	if size == 0 {
		return
	}

	scale := float64(tallyMarkSize) / float64(size)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(left), float64(top))
	screen.DrawImage(img, op)
}

// tallyTitle writes one block's heading.
func tallyTitle(gs *state.GlobalState, screen *ebiten.Image, left, top int, s string) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(left), float64(top))
	op.ColorScale.ScaleWithColor(tallyFadedInk)
	text.Draw(screen, s, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tallyTitleSize}, op)
}

// tallyFigure writes one count, left-aligned against a mark.
func tallyFigure(gs *state.GlobalState, screen *ebiten.Image, left, top, n int) {
	tallyFigureAt(gs, screen, left, top, n, text.AlignStart)
}

// tallyFigureAt writes one count. **A zero is written and faded, never omitted** — see
// tallyFadedInk.
func tallyFigureAt(gs *state.GlobalState, screen *ebiten.Image, x, top, n int, align text.Align) {
	ink := tallyInk
	if n == 0 {
		ink = tallyFadedInk
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(top+tallyMarkSize/2))
	op.PrimaryAlign = align
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, fmt.Sprintf("%d", n),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: tallyTextSize}, op)
}
