package screens

// **How the deck panel is being read**: which face each card is shown in, which half of the deck
// the figures are about, and which cards the panel is being asked to point at. All of it is one
// column of buttons down the panel's left edge.
//
// It is a separate file from deckpanel.go because that one answers "where does every card go" and
// this one answers "which deck am I looking at". The grid's arithmetic is hard enough to read
// without a column of controls in the middle of it.
//
// # The band became a column *(owner's call, 2026-09-11)*
//
// There were three blocks of figures under the grid — by form, by form and AP, by element — and two
// buttons centred beneath them. The figures answered a question and then stopped: "ten crush" does
// not say how much of it is cheap, and a player who wanted the follow-up had to find the cards by
// eye in a grid of sixty. **Every figure is a button now**, pressing one marks the cards it counted,
// and the whole set stands in one column with the two view toggles at the top of it — so everything
// that changes what the panel is showing is in one place and reads as one kind of control.
//
// What it cost is the element name in the gutter beside each grid row. The column's element buttons
// are in the rows' own order and carry the same word in the same colour, so the key moved rather
// than went; what the grid gave up is a label *level* with its row.
//
// **Nothing in here can change anything.** This is a reading preference over a picture of a deck —
// the same standing as the hand's sort column, and for the same reason it was safe to build without
// asking what it does to the engine. `ResolveRound` never sees it, the piles are never touched, and
// closing the panel leaves the fight exactly where it was.

import (
	"fmt"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The column. **One width for every button in it**, because they are one set of controls doing one
// job and a column of three widths would read as three unrelated widgets stacked up.
const (
	// deckColumnInset is the air between the panel's left edge and the column.
	deckColumnInset = 26

	// deckColumnWidth is the column itself. Wide enough for a form's mark, its name and a two-digit
	// figure on one line without any of the three crowding the others.
	deckColumnWidth = 264

	// deckColumnGutter is the air between the column and the first card of a row. It replaces
	// deckRowLabelWidth, which was the same idea for the labels this column took over.
	deckColumnGutter = 26

	deckColumnButtonHeight = 34
	deckColumnButtonPitch  = 38

	// deckColumnBlockGap is the air between two blocks of buttons, and deckColumnHeadDrop is how
	// far under a block's heading its first button sits.
	deckColumnBlockGap = 16
	deckColumnHeadDrop = 24

	deckColumnHeadSize = 15
	deckColumnTextSize = 20
	deckColumnMarkSize = 24

	// deckColumnPad is the air inside a button, at both ends of its line.
	deckColumnPad = 12

	// The view toggles' own label size. **Smaller than the 36 the centred pair used**: those had
	// the width of the panel to stand in and these have the column's, and ALTERATIONS is eleven
	// characters.
	deckToggleTextSize = 22

	// The labels. **Each names the state the panel is in, not the state pressing would move to.**
	// A latched button already says "this is on", so a label naming the other side would have the
	// two contradicting each other.
	deckViewAlteredLabel   = "ALTERATIONS"
	deckViewUnalteredLabel = "AS OWNED"
	deckViewFullLabel      = "FULL"
	deckViewPlayedLabel    = "PLAYED"
	deckViewClearLabel     = "SHOW ALL"
)

// deckView is how a panel is being read, and it belongs to whoever puts the panel up.
//
// **A reading preference is not a fact about a fight**, exactly like the hand's `sortMode`: it
// survives the panel being closed and reopened, because a player who has chosen a view should not
// have to re-choose it every look.
type deckView struct {
	// unaltered inverts the default. **Alterations are on unless this is set** — a deck with a flip
	// relic in it is dealt in colours the owned list does not have, and a panel showing the list is
	// showing a deck the player will never draw.
	unaltered bool

	// played picks which half of the deck the figures count and which half is drawn lit. **False is
	// the whole deck**, which is the question between fights and the commoner one in a fight.
	played bool

	// filter is which cards the panel is pointing at. See deckfilter.go.
	filter deckFilter

	alterations *models.Button
	half        *models.Button
	clear       *models.Button

	// The three blocks. **Keyed maps rather than slices**, because the cost block's members are a
	// function of the deck rather than of this file — a worm that invents a price has to get a
	// button without an edit here.
	forms    map[combat.Form]*models.Button
	costs    map[int]*models.Button
	elements map[cards.Element]*models.Button

	// counts is what the column last measured, so update and draw write the same figures. **Held
	// rather than recomputed in draw**: the two run on the same frame and a second walk of the grid
	// is a second answer waiting to disagree with the first.
	counts deckCounts
}

// update runs the column. It is called only while the panel is up, so nothing in it can be pressed
// through a screen that is not showing it.
func (v *deckView) update(gs *state.GlobalState, d deckContents) {
	v.refresh(gs, d)

	systems.UpdateButton(gs, v.alterations)
	if d.inFight {
		systems.UpdateButton(gs, v.half)
	}
	systems.UpdateButton(gs, v.clear)

	for _, f := range deckFilterForms() {
		systems.UpdateButton(gs, v.forms[f])
	}
	for _, cost := range v.counts.costs {
		systems.UpdateButton(gs, v.costs[cost])
	}
	for _, e := range deckRowElements() {
		systems.UpdateButton(gs, v.elements[e])
	}
}

// refresh is the one pass that measures the deck and places every control, called from update and
// from draw. **One function, so the rectangle a press is measured against is the rectangle the
// label was drawn in** — the rule every row of controls in this game follows.
func (v *deckView) refresh(gs *state.GlobalState, d deckContents) {
	v.build()
	centreX, width, top := deckGridRegion(gs)
	v.counts = countsOf(d.grid(*v, centreX, width, top).slots, d.holder, v.filter)
	v.layout(gs, d)
}

// build makes the buttons on first use. **Built once and kept**, like every other widget on these
// screens: a button rebuilt each frame would lose the press it was in the middle of.
//
// **The cost buttons are built for every price the walk can reach, not for the ones the deck
// holds.** Which prices exist changes as a run alters cards, and a button built the frame a price
// first appeared would be a button that cannot be pressed on the frame it appears. Only the ones a
// lit card actually carries are laid out and drawn — see deckCounts.costs.
func (v *deckView) build() {
	if v.alterations != nil {
		return
	}

	v.alterations = v.columnButton(func() { v.unaltered = !v.unaltered })
	v.alterations.TextSize = deckToggleTextSize

	v.half = v.columnButton(func() { v.played = !v.played })
	v.half.TextSize = deckToggleTextSize

	v.clear = v.columnButton(func() { v.filter.clear() })
	v.clear.TextSize = deckToggleTextSize
	v.clear.Text = deckViewClearLabel

	v.forms = map[combat.Form]*models.Button{}
	for _, f := range deckFilterForms() {
		form := f
		v.forms[f] = v.columnButton(func() { v.filter.toggleForm(form) })
	}

	v.costs = map[int]*models.Button{}
	for cost := 0; cost <= maxFilterCost; cost++ {
		at := cost
		v.costs[cost] = v.columnButton(func() { v.filter.toggleCost(at) })
	}

	v.elements = map[cards.Element]*models.Button{}
	for _, e := range deckRowElements() {
		element := e
		v.elements[e] = v.columnButton(func() { v.filter.toggleElement(element) })
	}
}

// columnButton is one button of the column: the shared size and colour, and no label of its own.
//
// **The filter buttons carry no `Text`**, because what they say is a mark, a word and a figure at
// three different alignments and `models.Button` centres one string. They are drawn over instead —
// see drawColumnRow. The three toggles at the top do use `Text`, since a centred word is exactly
// what they are.
func (v *deckView) columnButton(onClick func()) *models.Button {
	b := models.NewButton(deckColumnWidth, deckColumnButtonHeight, "", onClick)
	b.BaseColor = sortButtonColor
	b.TextSize = deckColumnTextSize
	return b
}

// deckColumnLeft is where the column starts, and deckColumnRight where it ends. **The grid measures
// off the same pair**, so the two cannot overlap by arithmetic drifting apart — see
// deckGridRegion.
func deckColumnLeft(gs *state.GlobalState) int {
	return modalPanelRect(gs).Min.X + deckColumnInset
}

func deckColumnRight(gs *state.GlobalState) int {
	return deckColumnLeft(gs) + deckColumnWidth
}

// layout places every control and writes what the three toggles currently say.
func (v *deckView) layout(gs *state.GlobalState, d deckContents) {
	r := modalPanelRect(gs)
	centreX := deckColumnLeft(gs) + deckColumnWidth/2
	y := r.Min.Y + modalBareBodyTop + deckColumnButtonHeight/2

	place := func(b *models.Button) {
		b.ScreenX, b.ScreenY = centreX, y
		y += deckColumnButtonPitch
	}

	v.alterations.Text = deckViewAlteredLabel
	v.alterations.Latched = !v.unaltered
	if v.unaltered {
		v.alterations.Text = deckViewUnalteredLabel
	}
	place(v.alterations)

	// **The FULL/PLAYED button is skipped rather than placed and hidden between fights.** The pair
	// used to be centred and had to hold its partner's space so the alterations button did not move
	// sideways between a fight and a shop; in a column the blocks below would move instead, and a
	// block of filters that sat 38 pixels lower in a shop than in a duel is the same complaint one
	// axis over. What is constant here is the *order*, not the offsets.
	if d.inFight {
		v.half.Text = deckViewFullLabel
		v.half.Latched = v.played
		if v.played {
			v.half.Text = deckViewPlayedLabel
		}
		place(v.half)
	}

	// **SHOW ALL goes dead when there is nothing to show all of**, rather than being taken away.
	// A control that vanishes when it has nothing to do is one the player never learns is there —
	// the same argument the alterations button already wins on one line up.
	enable(v.clear, !v.filter.empty())
	place(v.clear)

	y += deckColumnBlockGap
	y = v.layoutBlock(centreX, y, func(place func(*models.Button)) {
		for _, f := range deckFilterForms() {
			b := v.forms[f]
			b.Latched = v.filter.onForm(f)
			place(b)
		}
	})

	y += deckColumnBlockGap
	y = v.layoutBlock(centreX, y, func(place func(*models.Button)) {
		for _, cost := range v.counts.costs {
			b := v.costs[cost]
			b.Latched = v.filter.onCost(cost)
			place(b)
		}
	})

	y += deckColumnBlockGap
	v.layoutBlock(centreX, y, func(place func(*models.Button)) {
		for _, e := range deckRowElements() {
			b := v.elements[e]
			b.Latched = v.filter.onElement(e)
			place(b)
		}
	})
}

// layoutBlock places one block's buttons under the space its heading takes, and reports where the
// next block may start.
func (v *deckView) layoutBlock(centreX, top int, rows func(place func(*models.Button))) int {
	y := top + deckColumnHeadDrop + deckColumnButtonHeight/2
	rows(func(b *models.Button) {
		b.ScreenX, b.ScreenY = centreX, y
		y += deckColumnButtonPitch
	})
	return y - deckColumnButtonHeight/2
}

// enable puts a button back in play, or takes it out. **Both directions**, because UpdateButton
// returns early on a disabled button and would never clear the state it was left in.
func enable(b *models.Button, on bool) {
	switch {
	case !on:
		b.State = models.ButtonStateDisabled
	case b.State == models.ButtonStateDisabled:
		b.State = models.ButtonStateNormal
	}
}

// draw puts the column on the panel.
func (v *deckView) draw(gs *state.GlobalState, screen *ebiten.Image, d deckContents) {
	v.refresh(gs, d)

	systems.DrawButton(gs, screen, v.alterations)

	// **No FULL/PLAYED between fights**, because nothing has been played: there is one pile and
	// both states of the button would be the same picture. See deckContents.inFight.
	if d.inFight {
		systems.DrawButton(gs, screen, v.half)
	}
	systems.DrawButton(gs, screen, v.clear)

	// The headings are drawn against the first button of each block rather than from a running
	// total, so a block that has moved — the cost block does, whenever a price appears or goes —
	// takes its own words with it.
	v.drawBlockHead(gs, screen, v.forms[deckFilterForms()[0]],
		fmt.Sprintf("BY FORM  (%d cards)", v.counts.total))
	for _, f := range deckFilterForms() {
		v.drawColumnRow(gs, screen, v.forms[f], f, f.String(), deckColumnInk, v.counts.byForm[f])
	}

	if len(v.counts.costs) > 0 {
		v.drawBlockHead(gs, screen, v.costs[v.counts.costs[0]], "BY AP")
	}
	for _, cost := range v.counts.costs {
		v.drawColumnRow(gs, screen, v.costs[cost], combat.FormNone, fmt.Sprintf("%d AP", cost),
			deckColumnInk, v.counts.byCost[cost])
	}

	v.drawBlockHead(gs, screen, v.elements[deckRowElements()[0]], "BY ELEMENT")
	for _, e := range deckRowElements() {
		name, ink := deckRowLabel(e)
		v.drawColumnRow(gs, screen, v.elements[e], combat.FormNone, name, ink, v.counts.byElement[e])
	}
}

// drawBlockHead writes a block's heading in the air above its first button.
func (v *deckView) drawBlockHead(gs *state.GlobalState, screen *ebiten.Image, first *models.Button,
	s string) {

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(first.ScreenX-deckColumnWidth/2),
		float64(first.ScreenY-deckColumnButtonHeight/2-deckColumnHeadDrop))
	op.ColorScale.ScaleWithColor(deckColumnFadedInk)
	text.Draw(screen, s, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckColumnHeadSize}, op)
}

// drawColumnRow draws one filter button and the three things on it: a form's mark where there is
// one, the name, and the figure.
//
// **The button goes down first and everything else over it**, because the face is a cached image
// the widget owns and a label baked into it could only be one alignment. See columnButton.
func (v *deckView) drawColumnRow(gs *state.GlobalState, screen *ebiten.Image, b *models.Button,
	f combat.Form, name string, ink color.RGBA, n int) {

	systems.DrawButton(gs, screen, b)

	left := b.ScreenX - deckColumnWidth/2 + deckColumnPad
	right := b.ScreenX + deckColumnWidth/2 - deckColumnPad

	if f != combat.FormNone {
		drawFormMark(screen, f, left, b.ScreenY-deckColumnMarkSize/2)
		left += deckColumnMarkSize + 10
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(left), float64(b.ScreenY))
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, name,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckColumnTextSize}, op)

	// **A count of none is written and faded, never left blank** — a blank cell reads as a row that
	// does not exist, and "I own no 3 AP crush" is exactly the kind of thing this panel is opened to
	// find out.
	figure := deckColumnInk
	if n == 0 {
		figure = deckColumnFadedInk
	}
	fop := &text.DrawOptions{}
	fop.GeoM.Translate(float64(right), float64(b.ScreenY))
	fop.PrimaryAlign = text.AlignEnd
	fop.SecondaryAlign = text.AlignCenter
	fop.ColorScale.ScaleWithColor(figure)
	text.Draw(screen, fmt.Sprintf("%d", n),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckColumnTextSize}, fop)
}

// deckColumnInk is the column's own text colour: the panel is dark, so this is a near-white.
var deckColumnInk = color.RGBA{R: 236, G: 236, B: 240, A: 255}

// deckColumnFadedInk is a heading, and a zero.
var deckColumnFadedInk = color.RGBA{R: 130, G: 130, B: 144, A: 255}

// maxFilterCost is how far up the cost axis the column will look. **A ceiling on the walk, not a
// cap on a card**: a card declares its own cost and a stack of worms could in principle push one
// past this, at which point it would be filterable by form and element and missing from the AP
// block. Ten is far enough above the four the game ships that reaching it means something else is
// wrong.
const maxFilterCost = 10

// deckFilterForms is the order the form block runs in: the deck panel's own, so a form is in the
// same place in the column as it is along a grid row.
func deckFilterForms() []combat.Form {
	return []combat.Form{combat.FormStab, combat.FormSlash, combat.FormCrush, combat.FormDefend}
}

// drawFormMark puts a form's own drawing at the head of a button, **untinted**. The card's corner
// tints the same art by the card's element, which is what makes an element legible on a card; this
// row is about the form, and a coloured mark here would be claiming one.
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

	scale := float64(deckColumnMarkSize) / float64(size)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(left), float64(top))
	screen.DrawImage(img, op)
}

// deckColumnBottom is where the column's last button ends, given how many toggles are standing and
// how many prices the deck holds. **The same walk layout does**, at the fixed internal resolution,
// so the test that says the column fits the panel measures what is actually drawn.
func deckColumnBottom(toggles, costs int) int {
	y := state.ScreenHeight*modalPanelTopPct/100 + modalBareBodyTop + toggles*deckColumnButtonPitch
	for _, n := range []int{len(deckFilterForms()), costs, deckRowCount} {
		y += deckColumnBlockGap + deckColumnHeadDrop + n*deckColumnButtonPitch
	}
	return y - (deckColumnButtonPitch - deckColumnButtonHeight)
}
