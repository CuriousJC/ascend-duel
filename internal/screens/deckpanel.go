package screens

// **The deck panel: every card you own, and the three screens that put it up.**
//
// It was the combat screen's own overlay until 2026-08-22 — drawn and hit-tested off
// `CombatScene`'s three piles, so no other scene could show it. The reward screen and the shop
// both want it, for the same reason and behind the same kind of button: a player choosing a worm
// or a ring is making a decision about a deck, and being unable to look at that deck while
// deciding is the one thing this panel exists to fix.
//
// **So the panel takes a `deckContents` rather than a scene.** Which cards are in which pile,
// what the counts line says and how the dialog is closed are all the caller's; what is here is
// the arrangement — five rows, one per colour plus the plans, sorted so that a card does not move
// when it is spent, it only dims.
//
// **The extraction was the point, not a side effect.** TODO.md carried this as an entry whose own
// wording was "do not copy it, which is the failure this entry exists to prevent": three copies
// of a grid this size would be three places to fix a row that overflows.
//
// What did *not* move is the combat screen's piles or its draw-pile-as-a-control — those are
// facts about a fight. See combat_deck.go.

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// deckContents is what a panel is asked to show: the cards still to draw, the cards that are
// spoken for, and who is holding them.
//
// **The split is "can I still draw this", not "which pile is it in".** The hand and the discard
// are different piles and the same fact from this panel's point of view, which is what lets the
// grid stay still while a round empties it. A screen between fights has no piles at all and puts
// everything in `draw`.
//
// **`holder` prices the faces.** A card's picture is a function of the card *and who is holding
// it* — a slash under Keen says a bigger figure — so the panel has to be told, rather than
// drawing every card as it would look on nobody's hand.
type deckContents struct {
	draw   []combat.Card
	spent  []combat.Card
	holder combat.Duelist

	// counts is the line under the title, and hint is the sentence at the bottom saying how to
	// get out. **Both are the caller's words**, because both are facts about the screen the panel
	// is standing on: a fight has three piles to report and a control that is a draw pile, a shop
	// has a deck size and a button marked D.
	counts string
	hint   string
}

// ownedContents is the panel between fights: the run's whole deck, none of it spent, priced by
// the rings the player is wearing.
//
// **It reads `Session.Deck` rather than `FightDeck`**, which is the difference between what you
// own and what a flip ring would deal you next fight. This panel answers the first question — it
// is what a worm is about to edit and what a ring is being bought against.
func ownedContents(gs *state.GlobalState) deckContents {
	d := deckContents{hint: "D again to close"}
	if gs.Run != nil {
		d.draw = gs.Run.Deck()
		if f := buildFighter(gs); f != nil {
			d.holder = f.Duelist
		}
	}
	d.counts = fmt.Sprintf("%d cards owned", len(d.draw))
	return d
}

// The D button: a square in the bottom-right corner, on every screen that is not a fight.
//
// **It is the Log button's shape, deliberately** — 44 pixels, one character, the slate a control
// that changes nothing wears, and the same rule that the button opening a dialog is the button
// closing it. There is no Escape key and no right click, so a modal has to make its exit the
// brightest thing on screen or it is a trap.
//
// **It is not a draw pile.** The combat screen opens this panel by clicking the stack of card
// backs, which is honest there because a draw pile exists. Between fights there is none, and a
// pile drawn where there is nothing to draw from would be a picture claiming a rule.
const (
	deckToggleLabel = "D"

	// The corner it stands in, matching the mute button's inset on the other side so the two
	// bottom corners share a line. See internal/game/chrome.go.
	deckToggleInset = 10
)

// deckToggle is the button, whether the panel is up, and the tooltip the panel needs.
//
// **A struct rather than three fields on each scene**, because the three go together and the
// failure of letting them drift apart is silent: a scene that forgets `gs.ModalOpen` leaves the
// frame's mute button live on top of a dialog whose whole design is that one control is lit.
type deckToggle struct {
	open   bool
	button *models.Button
	tip    models.Tooltip
}

// init wires the button. **The button survives a re-entry and the state does not** — a scene's
// Init runs again on every visit, and arriving at a shop with the deck already open would be a
// dialog nobody asked for.
func (t *deckToggle) init() {
	if t.button == nil {
		t.button = models.NewButton(logButtonSize, logButtonSize, deckToggleLabel,
			func() { t.open = !t.open })
		t.button.BaseColor = sortButtonColor
		t.button.TextSize = logButtonTextSize
	}
	t.open = false
	t.tip = models.Tooltip{DwellTicks: tipDwell}
}

// update runs the button and, while the panel is up, the tooltip over its cards.
//
// **It returns whether the panel is covering the screen**, which is the caller's cue to stop
// running everything else: the scene's own rows are still where they were, and a click reaching
// one through a dialog would be a ring bought while reading a deck.
func (t *deckToggle) update(gs *state.GlobalState, d deckContents) bool {
	// **The frame the panel is closed on is still a covered frame.** The press that closes it is
	// the same press the scene's rows would see, so a scene told the panel is down on that frame
	// takes a click the player spent on the exit — the failure `modalUp` prevents on the combat
	// screen by being read before anything else.
	was := t.open

	t.button.ScreenX = gs.PctX(100) - deckToggleInset - logButtonSize/2
	t.button.ScreenY = gs.PctY(100) - deckToggleInset - logButtonSize/2
	t.button.Latched = t.open
	systems.UpdateButton(gs, t.button)

	// **The tooltip is pointed only while the panel is up.** UpdateTooltip releases it by itself
	// on any frame nothing was pointed at, so a closed panel needs no clearing of its own.
	if t.open {
		gs.ModalOpen = true
		hoverDeckPanel(gs, image.Pt(gs.MouseX, gs.MouseY), d, &t.tip)
	}
	systems.UpdateTooltip(gs, &t.tip)
	return was || t.open
}

// draw puts the panel up if it is open, and the button on top of it either way.
func (t *deckToggle) draw(gs *state.GlobalState, screen *ebiten.Image, d deckContents) {
	if t.open {
		drawDeckPanel(gs, screen, d)
	}
	systems.DrawButton(gs, screen, t.button)
	if t.open {
		systems.DrawTooltip(gs, screen, &t.tip)
	}
}

// hoverDeckPanel explains one card in the panel: the same arithmetic the hand gets, for a card
// you cannot play from here. **Which is the point** — the panel is where a deck is read, and
// "what would this be worth" is the question a deck is read to answer.
func hoverDeckPanel(gs *state.GlobalState, at image.Point, d deckContents, tip *models.Tooltip) {
	left := float32(gs.PctX(deckPanelLeftPct))
	width := float32(gs.PctX(deckPanelRightPct)) - left
	top := float32(gs.PctY(deckPanelTopPct))

	for _, slot := range d.grid(left+width/2, top+deckGridTop).slots {
		if !at.In(slot.at) {
			continue
		}
		title, lines := cardTip(slot.card, heldBy(d.holder, slot.card))
		tip.Point(slot.at, title, lines)
		return
	}
}

// Deck overlay geometry. The panel is nearly the whole screen and stops above the button
// band, so the Deck button that closes it stays outside the panel as well as on top of it.
//
// The panel holds **every card you own**, in five rows, and nothing in it moves as cards shift
// between piles — a played card dims where it stands rather than leaving.
const (
	deckPanelLeftPct  = 4
	deckPanelRightPct = 96
	deckPanelTopPct   = 4
	// 92 rather than 86: at 86 the panel stopped short of the hand, so the tops of the cards
	// and the whole AP line sat below it, dimmed by the scrim but still visibly outside the
	// dialog. It still ends above the button band, which is what keeps the Deck button that
	// closes it outside the panel as well as drawn on top of it.
	deckPanelBottomPct = 92

	// **The grid became five overlapping rows, one per element, on 2026-08-09.**
	//
	// It was an 8x3 grid of half-size cards, which held 24 of the up-to-52 cards that can
	// sit outside the hand — so "+N more not shown" fired on every single look. That line
	// was written when the deck was 30 and could not fire, deliberately, so that growing
	// the deck would produce a visible shortfall rather than a panel that quietly lied.
	// It did its job and then kept firing.
	//
	// A half-size card (cards.Mini) overlapped to show only its left half needs 45 pixels
	// of width instead of 146. Twelve concepts per element is 585 pixels a row, five rows
	// is 684 tall, and **the whole deck now fits** — the overflow line is still there and
	// can no longer fire.
	//
	// Half rather than a third: a third-size card was 59 pixels wide and could carry
	// neither a glyph nor text, so a row was a line of coloured slivers. At 90 the
	// 22-pixel category glyph fits, and the visible left 45 pixels are exactly the glyph
	// and the cost dashes under it. A row now says what phase each card resolves in and
	// what it costs. What it still cannot say is which *concept* each card is.
	//
	// **Five rows of 132 is a tight fit and the gap is what absorbs it.** The panel gives
	// about 691 pixels between the legend and the closing hint; five rows plus four
	// 8-pixel gaps is 692, so deckGridTop moved up to 120 to buy the clearance back. A
	// sixth element would not fit and would need a different arrangement, not a smaller
	// gap.
	deckRowGap = 6

	// deckStackPitch is how far apart the cards in a row sit. **It is a constant sized for
	// a full row, never derived from how many cards are actually in one** — the overlay's
	// governing idea is that a card does not move when it is discarded, it only dims, and
	// a pitch that adapted to the count would shuffle the whole row on every draw.
	//
	// Twelve concepts at 75 is 906 pixels plus the 104-pixel label gutter, inside the
	// panel's 1177 with margin. At Mini's full 81 the cards would not overlap at all, so the
	// six pixels of overlap are what buys the row its slack — the stacking is load-bearing
	// arithmetic, not a look.
	//
	// It was Mini.Width/2, which showed 45 pixels of each card and left no room for a
	// name. Widening it to width-less-six is what let the name go on at all.
	//
	// **75 since 2026-08-11**, down from 84 with the card itself — the overlap is what is
	// held constant, not the pitch, so a pitch left behind at 84 would have opened gaps
	// between cards in an 81-pixel row. TestDeckRowFitsThePanel is what catches that.
	deckStackPitch = 75

	// deckMaxPerRow caps a row so it cannot run off the panel. Twelve is what the longest row
	// holds — the plans, at 3 concepts x 4 copies — and what deckStackPitch is sized against.
	// The four colour rows hold nine each.
	//
	// **The cap is what gives the overflow line something to report.** Without it a row
	// simply drew every card it had and ran off the edge, and the "+N more not shown"
	// message below could never fire because nothing was ever not shown. A thirteenth
	// concept should produce a visible, honest shortfall rather than a card halfway off
	// the panel.
	deckMaxPerRow = 12

	// deckRowLabelWidth is the gutter the element name sits in, to the left of each row.
	// The cards no longer carry any text, so without this a row would be an anonymous
	// line of coloured slivers.
	deckRowLabelWidth = 104

	// Offsets down from the panel's top edge.
	//
	// **Everything under the title came up by eight on 2026-08-15**, for a grid that briefly had
	// six rows. It is back to five — four colours and the plans — and the clearance is simply
	// spare now. TestDeckPitchMatchesTheCard is what holds the arithmetic either way.
	deckTitleTop  = 40
	deckCountsTop = 70
	deckLegendTop = 92
	deckGridTop   = 112
	deckHintUp    = 22 // hint's distance up from the panel's bottom edge
)

// drawDeckPanel covers the screen with the deck.
//
// The cards are drawn as themselves at half size rather than as a table of counts. A count could
// say "six Strikes"; it could not say which of them are fire and which are plain, and with
// elements on the cards that is most of what the player wants to know. The whole deck fits on the
// panel at once, so it is one look.
//
// Everything in one grid, with the spoken-for cards dimmed. The discard belongs here because
// those cards are coming back — a reshuffle folds the pile in, so "what is left" honestly
// means both piles — and merging them is what lets a card stay in place and simply dim
// when it is spent. See drawPileGrid.
func drawDeckPanel(gs *state.GlobalState, screen *ebiten.Image, d deckContents) {
	// A scrim over everything, so the panel reads as covering the screen rather than
	// floating on it, and so the cards underneath look as inert as they now are.
	bounds := screen.Bounds()
	vector.DrawFilledRect(screen, 0, 0,
		float32(bounds.Dx()), float32(bounds.Dy()),
		color.RGBA{A: 190}, false)

	left, top := float32(gs.PctX(deckPanelLeftPct)), float32(gs.PctY(deckPanelTopPct))
	width := float32(gs.PctX(deckPanelRightPct)) - left
	height := float32(gs.PctY(deckPanelBottomPct)) - top

	vector.DrawFilledRect(screen, left, top, width, height,
		color.RGBA{R: 30, G: 30, B: 38, A: 255}, false)
	vector.StrokeRect(screen, left, top, width, height, 2, apBarColor, false)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 28}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}

	title := &text.DrawOptions{}
	title.GeoM.Translate(float64(left+width/2), float64(top+deckTitleTop))
	title.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "Your deck", heading, title)

	// Hyphens, not em dashes. The kubasta font has no U+2014 and draws a missing-glyph box
	// for it — the middle dot is in the font, the dash is not.
	line := func(y float32, s string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(left+width/2), float64(top+y))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, s, small, op)
	}
	// **The counts line is the caller's**, because what there is to count depends on the screen:
	// a fight has three piles and a total nobody holds in their head at 48 cards, and a screen
	// between fights has one number.
	line(deckCountsTop, d.counts)

	// **The legend is only written when something is dimmed.** Between fights nothing is, and a
	// sentence explaining a state no card on the panel is in would be the panel describing a
	// screen it is not standing on.
	if len(d.spent) > 0 {
		line(deckLegendTop, "dimmed cards are in your hand or the discard - the rest are still to draw")
	}

	drawPileGrid(gs, screen, left+width/2, top+deckGridTop, d)

	hint := &text.DrawOptions{}
	hint.GeoM.Translate(float64(left+width/2), float64(top+height-deckHintUp))
	hint.PrimaryAlign = text.AlignCenter
	text.Draw(screen, d.hint, small, hint)
}

// **Attacks, then defends, then prepares; within each, cheapest first.** The rows are
// already one element each, so this is the order along a row — and it is what turns a
// row from a list into a shape: the same three runs in the same places in every row,
// so a gap is a card you have spent rather than a card you never had.
//
// Availability is the last key rather than the first, so a card's position does not
// depend on which pile it is in. That is the whole governing idea of the panel — a
// card does not move when it is played, it only dims — and sorting by pile would undo
// it. It only breaks ties between genuinely identical cards, which are interchangeable.
//
// Pulled out of drawPileGrid so it can be tested: the drawing needs a window and this does
// not, and the order along a row is a user-visible decision worth pinning.
func sortPileEntries(entries []pileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if ra, rb := formRank(a.card.Form()), formRank(b.card.Form()); ra != rb {
			return ra < rb
		}
		if ca, cb := a.card.Cost(), b.card.Cost(); ca != cb {
			return ca < cb
		}
		if a.card.Concept != b.card.Concept {
			return a.card.Concept < b.card.Concept
		}
		if a.card.Element != b.card.Element {
			return a.card.Element < b.card.Element
		}
		return a.available && !b.available
	})
}

// formRank is the order forms run in along a deck row: stab, slash, crush, then the plans.
//
// **It sorts on form rather than category** *(2026-08-15)*. Category has two values now, so
// sorting by it would put nine cards in one undifferentiated block — and the thing a player is
// looking for in this panel is how much of a form they still hold, because that is what a pair
// is counted on.
//
// A function rather than the enum's own order, because that order is grouped for the *rules* —
// it is what an expanded hand ID is derived from. Reading it here would tie how the deck panel
// is arranged to how hands are numbered, so that changing one silently changed the other.
func formRank(f combat.Form) int {
	switch f {
	case combat.FormStab:
		return 0
	case combat.FormSlash:
		return 1
	case combat.FormCrush:
		return 2
	case combat.FormPlan:
		return 3
	default:
		return 4
	}
}

// pileEntry is one card in the overlay and whether it can still be drawn.
type pileEntry struct {
	card      actionCard
	available bool
}

// deckRowElements is the colours the overlay gives a row to, in the fixed order internal/cards
// declares them.
//
// **Basic is not among them as of 2026-08-15**, because no attack card is basic any more — every
// attack ships in one of the four colours, and the only basic cards in the deck are the plans,
// which have their own row. A basic row would draw an empty gutter label over nothing at all.
//
// A function rather than a package-level slice so nothing can append to it, and derived from
// `cards.Elements()` rather than written out, so a fifth colour added to the drawing package
// arrives here without an edit.
func deckRowElements() []cards.Element {
	out := make([]cards.Element, 0, len(cards.Elements()))
	for _, e := range cards.Elements() {
		if e == cards.Basic {
			continue
		}
		out = append(out, e)
	}
	return out
}

// deckRowCount is how many rows the overlay draws: one per colour, then the plans.
//
// **The plans get a row of their own rather than sitting with the basics** *(2026-08-15)*. Every
// plan card is basic and no attack card is, so the alternative was a row holding nothing but the
// plans under a label reading "basic" — which names the colour rather than the thing, on the one
// row where the colour is the least interesting fact about the cards in it.
var deckRowCount = len(deckRowElements()) + 1

// deckPlanRow is the index of that row, and where a card goes when it is a plan. It is
// deliberately not a `cards.Element` — a plan is basic and saying otherwise would put a colour
// on a card that has none.
var deckPlanRow = deckRowCount - 1

// deckRowFor is which row a card belongs to: its colour, or the plan row.
//
// **A basic attack has nowhere to go and lands in the first row**, which is the deck list being
// wrong rather than this being lenient — `data/duelist_cards.json` ships no basic attacks and
// TestEveryCardLandsInExactlyOneDeckRow is what would catch one arriving.
func deckRowFor(c actionCard) int {
	if c.Category() == combat.CategoryPlan {
		return deckPlanRow
	}
	for i, e := range deckRowElements() {
		if e == artFor(c.Element) {
			return i
		}
	}
	return 0
}

// deckRowLabel is what the gutter says beside a row, and the colour it says it in.
func deckRowLabel(row int) (string, color.RGBA) {
	if row == deckPlanRow {
		// Neutral, because the plan row is the one row that is not a colour. It borrows the
		// basic border rather than inventing a fifth hue for a row that means "no element".
		return "plan", cards.BorderOf(cards.Basic)
	}
	e := deckRowElements()[row]
	return e.String(), cards.BorderOf(e)
}

// drawPileGrid lays **every card you own** into rows by element, plus a row of plans,
// centred on centerX.
//
// It used to show only what was outside the hand, under the heading "What is left". That
// made the panel change *shape* as a round went on: eight cards vanished at the start of
// every round and came back at the end of it, so the rows shortened and lengthened and
// nothing sat still. The point of sorting rather than showing pile order was always that
// **a card does not move when it is played, it only dims** — and leaving the hand out
// broke exactly that.
//
// So all sixty are here, always, and the dimming carries the state instead: full strength
// means still in the draw pile, washed out means in your hand or already discarded. The
// rows are now a fixed twelve long, which is what the layout was sized for anyway.
//
// Sorted, never in pile order. The draw pile is shuffled, and drawing it in order would
// hand the player their next few cards and make the shuffle pointless. This is a picture
// of what you own, not of the sequence it will arrive in.
// pileGrid is where every card in the panel goes, and where each row is named.
//
// **One layout, read by the drawing and by the cursor** *(2026-08-21)*. It was inline in
// drawPileGrid until the tooltip needed to know which card is under the pointer, and a second set
// of arithmetic saying where a card is drawn is exactly the bug the one-rectangle rule prevents
// everywhere else on this screen.
func (d deckContents) grid(centerX, top float32) pileGridLayout {
	entries := make([]pileEntry, 0, len(d.draw)+len(d.spent))
	for _, c := range d.draw {
		entries = append(entries, pileEntry{c, true})
	}
	// The hand dims the same way the discard does. They are different piles but the same
	// fact from this panel's point of view — this card is not one you can still draw.
	for _, c := range d.spent {
		entries = append(entries, pileEntry{c, false})
	}

	sortPileEntries(entries)

	// One row per element in the fixed order internal/cards declares, then the plans. A slice
	// indexed by row rather than a map: Go randomises map iteration, and a panel whose rows
	// swapped places between looks would be unreadable.
	rows := make([][]pileEntry, deckRowCount)
	for _, e := range entries {
		rows[deckRowFor(e.card)] = append(rows[deckRowFor(e.card)], e)
	}

	out := pileGridLayout{rowPitch: cards.Mini.Height + deckRowGap}
	pitch := deckStackPitch

	// Widest row sets the left edge, so the rows share one origin and the columns line up
	// down the panel rather than each row centring on its own count.
	widest := 0
	for _, group := range rows {
		n := min(len(group), deckMaxPerRow)
		if w := (n-1)*pitch + cards.Mini.Width; w > widest {
			widest = w
		}
	}
	cardsLeft := int(centerX) - (deckRowLabelWidth+widest)/2 + deckRowLabelWidth

	shown := 0
	for i, group := range rows {
		rowTop := int(top) + i*out.rowPitch
		out.labels = append(out.labels, pileRowLabel{
			row: i,
			at:  image.Pt(cardsLeft-12, rowTop+cards.Mini.Height/2),
		})

		for j, e := range group {
			if j >= deckMaxPerRow {
				break
			}
			at := image.Pt(cardsLeft+j*pitch, rowTop)
			out.slots = append(out.slots, pileSlot{
				pileEntry: e,
				at:        image.Rect(at.X, at.Y, at.X+cards.Mini.Width, at.Y+cards.Mini.Height),
			})
		}
		shown += min(len(group), deckMaxPerRow)
	}
	out.hidden = len(entries) - shown
	return out
}

// pileGridLayout is the panel's geometry: where each card sits, where each row is named, how tall a
// row is, and how many cards did not fit.
type pileGridLayout struct {
	slots    []pileSlot
	labels   []pileRowLabel
	rowPitch int
	hidden   int
}

// pileSlot is one card and the rectangle it occupies. **The same rectangle is drawn in and
// hit-tested against**, which is the rule every row of cards in this game follows.
type pileSlot struct {
	pileEntry
	at image.Rectangle
}

// pileRowLabel is one row's name in the gutter, at the point it is drawn from.
type pileRowLabel struct {
	row int
	at  image.Point
}

func drawPileGrid(gs *state.GlobalState, screen *ebiten.Image, centerX, top float32,
	d deckContents) {

	grid := d.grid(centerX, top)

	for _, l := range grid.labels {
		// A mini card says its own concept but nothing about which row it is in, so this is what
		// names the row.
		name, ink := deckRowLabel(l.row)
		labelOp := &text.DrawOptions{}
		labelOp.GeoM.Translate(float64(l.at.X), float64(l.at.Y))
		labelOp.PrimaryAlign = text.AlignEnd
		labelOp.SecondaryAlign = text.AlignCenter
		labelOp.ColorScale.ScaleWithColor(ink)
		text.Draw(screen, name,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, labelOp)
	}

	// **Left to right, so each card is covered on its *right* edge by the next one.** This was
	// backwards and the screenshot showed it: drawing right to left puts card 0 on top of card 1,
	// and card 1's left edge is exactly where its glyph and dashes are — so every row rendered as
	// one complete card followed by eleven blank slivers.
	for _, slot := range grid.slots {
		// available carries "can be drawn", not "can be afforded". Never selected: this is an
		// inventory, not a choice, and dimming by the round's remaining AP would say something
		// about a budget that has nothing to do with a pile you cannot play from.
		drawCard(gs, screen, slot.at.Min, cards.Mini, slot.card,
			heldBy(d.holder, slot.card), slot.available, false)
	}

	if grid.hidden > 0 {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(centerX), float64(int(top)+deckRowCount*grid.rowPitch))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, fmt.Sprintf("+%d more not shown", grid.hidden),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, op)
	}
}
