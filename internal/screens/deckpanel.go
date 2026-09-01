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
// the arrangement — four rows, one per colour, sorted so that a card does not move
// when it is spent, it only dims.
//
// **The extraction was the point, not a side effect.** TODO.md carried this as an entry whose own
// wording was "do not copy it, which is the failure this entry exists to prevent": three copies
// of a grid this size would be three places to fix a row that overflows.
//
// What did *not* move is the combat screen's piles or its draw-pile-as-a-control — those are
// facts about a fight. See combat_deck.go.

import (
	"image"
	"image/color"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

	// run is where a card's *original* lives, looked up by ID.
	//
	// **It is what the alterations toggle is made of** *(2026-08-24)*. A card in the hand or the
	// discard has been through a draw and carries only the colour a flip ring made it; it does not
	// remember what it was, deliberately, because a rule reading what a card used to be is an
	// ordering the owner ruled out. The original is not gone — it is on the card the run owns, and
	// the ID is the way back to it. See combat.Card.ID.
	//
	// **Nil is honest and common.** A panel with no run behind it has no alterations to show and no
	// originals to show them against; every card is drawn as it is and the toggle latches over a
	// deck that does not move.
	run *session.Session

	// inFight says whether some of these cards have been played. **It is what FULL/PLAYED is
	// gated on**: a screen between fights has one pile, so both states of that button would be the
	// same picture and it is not drawn at all.
	inFight bool
}

// ownedContents is the panel between fights: the run's whole deck, none of it spent, priced by
// the rings the player is wearing.
//
// **It reads `Session.Deck` rather than `FightDeck`**, which is the difference between what you
// own and what a flip ring would deal you next fight. This panel answers the first question — it
// is what a worm is about to edit and what a ring is being bought against.
func ownedContents(gs *state.GlobalState) deckContents {
	d := deckContents{run: gs.Run}
	if gs.Run != nil {
		d.draw = gs.Run.Deck()
		if f := buildFighter(gs); f != nil {
			d.holder = f.Duelist
		}
	}
	return d
}

// ownedOf is a card as the run owns it: what it was before a demoting or flipping ring got to it.
//
// **The card itself when there is no original to find**, which covers a panel with no run behind
// it, a card with no identity, and a card whose original a worm has since eaten. Drawing the card
// actually in hand is the honest answer to all three.
func (d deckContents) ownedOf(c combat.Card) combat.Card {
	if d.run == nil {
		return c
	}
	owned, ok := d.run.CardByID(c.ID)
	if !ok {
		return c
	}
	return owned
}

// faceOf is the card the panel draws, which is a question about the view rather than about the
// card: the one the run owns, or the one the rings will hand over.
//
// **Both faces are computed from the owned card**, never from the card in the pile. That is what
// makes the two views agree for a card wherever it happens to be sitting: an ice card in the
// discard and the lightning card it came from are one card, so the panel must not draw them as one
// thing on the draw row and another in the hand.
func (d deckContents) faceOf(c combat.Card, unaltered bool) combat.Card {
	owned := d.ownedOf(c)
	if unaltered || d.run == nil {
		return owned
	}
	return d.run.AlteredAs(owned)
}

// deckToggle is the deck panel behind a button: the shared modal chrome, plus the one thing this
// panel needs that the chrome does not know — which cards to show.
//
// **It is a thin wrapper as of 2026-08-24**, when the chrome it used to own moved to modal.go for
// the hands panel to share. What is left is the pairing of a toggle with a `deckContents`, so a
// scene still says `s.deck.update(gs, ownedContents(gs))` and no call site changed.
//
// **The button is not a draw pile.** The combat screen opens this panel by clicking the stack of
// card backs, which is honest there because a draw pile exists. Between fights there is none, and
// a pile drawn where there is nothing to draw from would be a picture claiming a rule.
type deckToggle struct {
	modalToggle

	// view is how this panel is being read — the two toggles along its bottom edge. **It lives on
	// the toggle rather than on the scene**, so a reading preference survives the panel being
	// closed and reopened without every screen that puts one up having to hold the same two fields.
	view deckView
}

// init wires the button into the corner it stands in.
func (t *deckToggle) init() {
	t.modalToggle.init(deckToggleLabel, logButtonSize, logButtonSize, logButtonTextSize,
		cornerSlot(0))
}

// update runs the button and the tooltip over the panel's cards, and reports whether the screen
// is covered.
func (t *deckToggle) update(gs *state.GlobalState, d deckContents) bool {
	return t.modalToggle.update(gs, func(at image.Point, tip *models.Tooltip) {
		// **The view's buttons run before the tooltip is pointed**, and only while the panel is up,
		// which is what this closure already guarantees.
		t.view.update(gs, d)
		hoverDeckPanel(gs, at, t.view, d, tip)
	})
}

// draw puts the panel up if it is open, and the button on top of it either way.
func (t *deckToggle) draw(gs *state.GlobalState, screen *ebiten.Image, d deckContents) {
	t.modalToggle.draw(gs, screen, func() { drawDeckPanel(gs, screen, &t.view, d) })
}

// hoverDeckPanel explains one card in the panel: the same arithmetic the hand gets, for a card
// you cannot play from here. **Which is the point** — the panel is where a deck is read, and
// "what would this be worth" is the question a deck is read to answer.
func hoverDeckPanel(gs *state.GlobalState, at image.Point, v deckView, d deckContents,
	tip *models.Tooltip) {

	left := float32(gs.PctX(modalPanelLeftPct))
	width := float32(gs.PctX(modalPanelRightPct)) - left
	top := float32(gs.PctY(modalPanelTopPct))

	for _, slot := range d.grid(v, left+width/2, width, top+modalBareBodyTop).slots {
		if !at.In(slot.at) {
			continue
		}
		title, lines := cardTip(slot.card, heldBy(d.holder, slot.card))
		tip.Point(slot.at, title, lines)
		return
	}
}

// Deck overlay geometry. The panel's footprint is the shared modal one — see modal.go; what is
// here is the grid inside it.
//
// The panel holds **every card you own**, one row per element, and nothing in it moves as cards
// shift between piles — a played card dims where it stands rather than leaving. A card arriving
// or leaving the *deck* does move the row it is in, because the row's pitch is a function of how
// many cards are in it; see rowPitchFor.
const (
	// **The grid became five overlapping rows, one per element, on 2026-08-09.**
	//
	// It was an 8x3 grid of half-size cards, which held 24 of the up-to-52 cards that can
	// sit outside the hand — so "+N more not shown" fired on every single look. That line
	// was written when the deck was 30 and could not fire, deliberately, so that growing
	// the deck would produce a visible shortfall rather than a panel that quietly lied.
	// It did its job, then kept firing, and **went altogether on 2026-08-23** when the owner
	// ruled that the panel never hides a card. See rowPitchFor.
	//
	// A half-size card (cards.Mini) overlapped to show only its left half needs 45 pixels
	// of width instead of 146. Twelve concepts per element is 585 pixels a row, four rows
	// is 684 tall, and the whole deck fits with room over.
	//
	// Half rather than a third: a third-size card was 59 pixels wide and could carry
	// neither a mark nor text, so a row was a line of coloured slivers. At 81 the 16-pixel
	// form mark fits, and the visible strip is exactly that mark and the cost ticks under
	// it — so a row says which form each card is, which element it is, and what it costs.
	// What it still cannot say is which *concept* each card is.
	//
	// **Height is the dimension with no give.** The panel gives about 691 pixels between the
	// tally band and the toggles under it — see modalBareBodyTop, which is where the grid starts now
	// that there are no words above it.
	// Width absorbs a busy row by tightening; height cannot, so the fifth row arcane brought on
	// 2026-08-25 was paid for out of four places at once rather than out of this one: the panel
	// grew three percent (modalPanelBottomPct), this gap gave up two pixels, and the tally band
	// under the grid tightened by fourteen. **The card itself could not help** — Mini is Hand
	// halved, and the form mark is pixel art on a 32px canvas, so the only scales that keep a legal
	// mark are a half and a quarter. A *sixth* colour is a redesign of the grid rather than another
	// round of this. TestTheTallyBandFitsBetweenTheGridAndTheButtons is what says so.
	deckRowGap = 4

	// deckStackPitch is how far apart the cards in a row sit **when the row has room for it**.
	//
	// **It stopped being the only pitch on 2026-08-23** *(owner's call)*. It was a constant sized
	// for a full row and deliberately never derived from how many cards were in one, on the
	// grounds that a card should not move when it is discarded — and the price of that was a cap
	// on the row and a "+N more not shown" line, which is the panel declining to show you your own
	// deck. A run that recolours nine cards into one element hits that cap immediately. The
	// owner's call is that the panel never hides a card: a busy row overlaps harder instead. See
	// rowPitchFor.
	//
	// Twelve concepts at 75 is 906 pixels plus the 104-pixel label gutter, inside the panel's 1177
	// with margin. At Mini's full 81 the cards would not overlap at all, so the six pixels of
	// overlap are what buys the row its slack — the stacking is load-bearing arithmetic, not a
	// look.
	//
	// It was Mini.Width/2, which showed 45 pixels of each card and left no room for a name.
	// Widening it to width-less-six is what let the name go on at all.
	//
	// **75 since 2026-08-11**, down from 84 with the card itself — the overlap is what is
	// held constant, not the pitch, so a pitch left behind at 84 would have opened gaps
	// between cards in an 81-pixel row. TestDeckPitchMatchesTheCard is what catches that.
	deckStackPitch = 75

	// deckRowMargin is the air left at both ends of the widest row, so a row that has had to
	// tighten stops short of the panel's edge rather than running up against it.
	deckRowMargin = 24

	// deckRowLabelWidth is the gutter the element name sits in, to the left of each row.
	// The cards no longer carry any text, so without this a row would be an anonymous
	// line of coloured slivers.
	deckRowLabelWidth = 104
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
func drawDeckPanel(gs *state.GlobalState, screen *ebiten.Image, v *deckView, d deckContents) {
	// **No words at the top at all** *(owner's call, 2026-08-24)*. There was a title, a counts line
	// and a legend; all three are gone and the grid starts where they were. The panel is a picture
	// of a deck and the cards say what they are — a title naming what the player has just clicked
	// to open, a total they can see laid out in front of them, and a sentence explaining the
	// dimming were three captions on something that needed none. What is left up here is the X.
	r := drawModalFrame(gs, screen, modalHead{})

	grid := drawPileGrid(gs, screen, *v, float32(r.Min.X+r.Dx()/2), float32(r.Dx()),
		float32(r.Min.Y+modalBareBodyTop), d)

	// The tallies sit under the last row of cards, which is where the grid ends rather than a
	// constant: the rows are one per element and a fifth colour would push the band down with them.
	drawTallies(gs, screen, r, r.Min.Y+modalBareBodyTop+deckRowCount*grid.rowPitch+tallyTop,
		tallyOf(grid.slots, d.holder))

	v.draw(gs, screen, d)
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
		if a.available != b.available {
			return a.available && !b.available
		}
		// **Identity is the last word** *(2026-08-24)*, so that two cards the player cannot tell
		// apart still land in a fixed order. Without it the sort is not total any more — a card now
		// carries an ID and two entries equal on every visible key would be left in whatever order
		// the piles happened to hand them over, so a card could swap places with its twin between
		// one look and the next.
		return a.card.ID < b.card.ID
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
	case combat.FormDefend:
		return 3
	default:
		return 4
	}
}

// pileEntry is one card in the overlay, in the face the view asked for, and whether it can still be
// drawn.
type pileEntry struct {
	card      actionCard
	available bool

	// lit is whether this card is one of the ones the view is *about* — full strength rather than
	// dimmed, and counted in the tallies under the grid.
	//
	// **It is `available` under FULL and its opposite under PLAYED** *(owner's call, 2026-08-24)*,
	// which is why it is a second field rather than the same one read twice. The panel's governing
	// idea is that a card does not move when it is spent, it only dims — so the FULL/PLAYED toggle
	// inverts which half is dimmed and moves nothing at all. Between fights nothing is spent and
	// everything is lit.
	lit bool
}

// deckRowElements is the colours the overlay gives a row to, in the fixed order internal/cards
// declares them.
//
// **Basic is not among them as of 2026-08-15**, because no attack card is basic any more — every
// attack ships in one of the five colours, and the only basic cards in the deck are the plans,
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

// deckRowCount is how many rows the overlay draws: one per colour, and that is all. **Five since
// 2026-08-25**, and the number is derived rather than written, so arcane arrived here without an
// edit — what it did cost was the card's size. See cards.Mini.
//
// **The defences lost their own row on 2026-08-23**, when they stopped being basic. They had one
// because every defence was colourless and no attack was, so the alternative then was a row
// labelled "basic" holding nothing but defences — naming the colour rather than the thing, on the
// one row where the colour was the least interesting fact about the cards in it. Now a Ward is a
// fire card, and the row that says "fire" is where a player looks for it.
//
// A row therefore holds a colour's whole share of the deck — the nine attacks and the defences —
// which is inside the width the grid was already sized against.
var deckRowCount = len(deckRowElements())

// deckRowFor is which row a card belongs to: its colour, and nothing else decides.
//
// **A basic card has nowhere to go and lands in the first row**, which is the deck list being
// wrong rather than this being lenient — `data/duelist_cards.json` ships no basic card of any kind
// since the plans were coloured, and TestEveryCardLandsInExactlyOneDeckRow is what would catch one
// arriving.
func deckRowFor(c actionCard) int {
	for i, e := range deckRowElements() {
		if e == artFor(c.Element) {
			return i
		}
	}
	return 0
}

// deckRowLabel is what the gutter says beside a row, and the colour it says it in. Every row is a
// colour now, so there is no longer a case for the one that was not.
func deckRowLabel(row int) (string, color.RGBA) {
	e := deckRowElements()[row]
	return e.String(), cards.BorderOf(e)
}

// drawPileGrid lays **every card you own** into rows by element, centred on centerX. `formRank`
// puts the plans at the end of their colour's row, after stab, slash and crush.
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
func (d deckContents) grid(v deckView, centerX, width, top float32) pileGridLayout {
	entries := make([]pileEntry, 0, len(d.draw)+len(d.spent))
	for _, c := range d.draw {
		entries = append(entries, d.entry(v, c, true))
	}
	// The hand dims the same way the discard does. They are different piles but the same
	// fact from this panel's point of view — this card is not one you can still draw.
	for _, c := range d.spent {
		entries = append(entries, d.entry(v, c, false))
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
	room := int(width) - deckRowLabelWidth - deckRowMargin

	// Each row gets its own pitch, so a row that has collected extra cards closes up without
	// dragging the quiet rows in with it.
	pitches := make([]int, deckRowCount)
	widest := 0
	for i, group := range rows {
		pitches[i] = rowPitchFor(len(group), room)
		if w := rowWidth(len(group), pitches[i]); w > widest {
			widest = w
		}
	}

	// The widest row sets the left edge and every row starts there, so the labels line up in one
	// gutter and the block sits centred on the panel. **Rows do not each centre on their own
	// count** — that would move a row sideways as cards were added to it, and the panel's whole
	// idea is that a card stays where it is.
	cardsLeft := int(centerX) - (deckRowLabelWidth+widest)/2 + deckRowLabelWidth

	for i, group := range rows {
		rowTop := int(top) + i*out.rowPitch
		out.labels = append(out.labels, pileRowLabel{
			row: i,
			at:  image.Pt(cardsLeft-12, rowTop+cards.Mini.Height/2),
		})

		for j, e := range group {
			at := image.Pt(cardsLeft+j*pitches[i], rowTop)
			out.slots = append(out.slots, pileSlot{
				pileEntry: e,
				at:        image.Rect(at.X, at.Y, at.X+cards.Mini.Width, at.Y+cards.Mini.Height),
			})
		}
	}
	return out
}

// entry is one card turned into a grid entry: the face the alterations toggle asked for, and
// whether the FULL/PLAYED toggle is lighting it.
//
// **Nothing is lit between fights except everything.** `inFight` is false there, so PLAYED cannot
// be reached and this reduces to what the panel has always drawn.
func (d deckContents) entry(v deckView, c combat.Card, available bool) pileEntry {
	lit := available
	if d.inFight && v.played {
		lit = !available
	}
	return pileEntry{card: d.faceOf(c, v.unaltered), available: available, lit: lit}
}

// rowWidth is how much of the panel n cards at this pitch occupy: the last card is drawn whole,
// every earlier one contributes only its pitch.
func rowWidth(n, pitch int) int {
	if n <= 0 {
		return 0
	}
	return (n-1)*pitch + cards.Mini.Width
}

// rowPitchFor is how far apart a row of n cards sits, given the room it has.
//
// **The panel never hides a card** *(owner's call, 2026-08-23)*, so this is where a row that has
// outgrown the comfortable pitch pays for it: the cards overlap harder rather than the extras
// being dropped with a "+N more not shown" line under the grid. That line existed because a
// twelve-card cap could be exceeded, and it fired for real the moment a run recoloured most of
// the deck into one element — at which point the panel was hiding exactly the cards the player
// had gone looking for.
//
// **It tightens and never loosens.** deckStackPitch is the ceiling, so a short row is laid out
// exactly as it always was and only a row that does not fit is touched. What a tightened row
// costs is the card's name, then its cost column: at deckStackPitch a card shows 75 of its 81
// pixels, and the strip narrows from the right as the pitch falls.
//
// **There is no floor.** A pitch of one pixel is unreadable, but it fits — and a floor would put
// the cap back under another name, with a row running off the panel instead of being clamped.
// Sixty cards in one row is about seventeen pixels each, which still shows the form mark that
// now carries the element.
func rowPitchFor(n, room int) int {
	if n <= 1 || rowWidth(n, deckStackPitch) <= room {
		return deckStackPitch
	}
	pitch := (room - cards.Mini.Width) / (n - 1)
	if pitch < 1 {
		pitch = 1
	}
	return pitch
}

// pileGridLayout is the panel's geometry: where each card sits, where each row is named, and how
// tall a row is. **There is no count of what did not fit**, because nothing does not fit — see
// rowPitchFor.
type pileGridLayout struct {
	slots    []pileSlot
	labels   []pileRowLabel
	rowPitch int
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

func drawPileGrid(gs *state.GlobalState, screen *ebiten.Image, v deckView,
	centerX, width, top float32, d deckContents) pileGridLayout {

	grid := d.grid(v, centerX, width, top)

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
		// **`lit` rather than `available`**, because the FULL/PLAYED toggle inverts which half of
		// the deck the panel is about. It still carries "can be drawn" under FULL, which is what it
		// has always meant; never "can be afforded", since dimming by the round's remaining AP
		// would say something about a budget that has nothing to do with a pile you cannot play
		// from.
		drawCard(gs, screen, slot.at.Min, cards.Mini, slot.card,
			heldBy(d.holder, slot.card), slot.lit, false)
	}

	return grid
}
