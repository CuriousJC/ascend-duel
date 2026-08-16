package screens

import (
	"image"
	"image/color"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
)

// How the hand is arranged, and the three buttons that choose it.
//
// **The hand's order is not a rule and never has been.** Cross-category order is regrouped
// away by `combat.ResolutionOrder`, a hand is counted rather than read in sequence, and
// defends are a set — so nothing here can change what a round does. This is entirely about
// being able to read eight overlapping cards: cost tells you what you can afford, type tells
// you what the round is made of, element tells you what a mix is worth.
//
// **The sort re-applies on every refill**, not only when a button is pressed, so a hand dealt
// at the end of a round arrives already arranged and a newly drawn card lands where it belongs
// instead of on the right-hand end. A drag still moves a card and still survives until the next
// refill, at which point the sort reclaims the row.
//
// **The mode survives Init**, unlike everything else on this screen. It is a reading preference
// rather than a fact about a duel, and having it snap back to cost at the start of every fight
// would make it something the player re-presses rather than something they set.

// handSort is which arrangement the hand is in. Cost is the zero value because it is the
// default, so a scene that never touches this field is already sorted the way a fresh one is.
type handSort int

const (
	sortByCost handSort = iota
	sortByType
	sortByElement
)

// The column of sort buttons: square, stacked, and sitting to the right of the cards.
//
// **44 is the mute button's footprint** — the other square, iconic control in the game — and
// the two are the same size for the same reason: a single character has no width to speak of,
// so the button is sized to be hit rather than to hold a label.
//
// The column is pinned to the right edge of the band rather than hung off the row's own right
// edge. The row is centred, so both of its edges move when the hand grows or shrinks, and a
// control that slid sideways as cards were drawn would be worse than one standing still.
const (
	sortButtonSize = 44
	sortButtonGap  = 8

	// **Half again the default button label** *(2026-08-16)*. The strip's buttons carry a word
	// and are sized around it; these carry one character on a square, so the size that fits
	// `Discard` leaves a symbol swimming in the middle of the face. The character *is* the
	// control here, so it takes the room.
	sortButtonTextSize = 30
	// How much room the column takes out of the band the cards are laid out in: the buttons
	// themselves plus the gap between them and the nearest card.
	sortColumnGap     = 12
	sortColumnReserve = sortButtonSize + sortColumnGap
)

// sortButtonSpecs is the column, top to bottom, with the symbol each button carries.
//
// **One character each, because the button is 44 pixels square** and the input vocabulary has
// no tooltip yet — long press is the planned way to explain one and is not built. `$` is cost,
// `T` is type, `E` is element; none of them collides with the S/D/C/P a card's own corner mark
// uses, which is the one place in the game a single letter already means something.
var sortButtonSpecs = []struct {
	mode  handSort
	label string
}{
	{sortByCost, "$"},
	{sortByType, "T"},
	{sortByElement, "E"},
}

// sortColumnRect is where the three buttons stand: against the band's right edge, centred on
// the cards rather than on the screen.
//
// It takes the card row's own top and height, so the column stays centred on the cards if the
// row moves — which it has done three times, and each time everything measured off it followed
// and everything measured off a percentage did not.
func sortColumnRect(gs *state.GlobalState) image.Rectangle {
	n := len(sortButtonSpecs)
	height := n*sortButtonSize + (n-1)*sortButtonGap

	left := gs.PctX(handBandRightPct) - sortButtonSize
	top := gs.PctY(handTopPct) + cardHeight/2 - height/2
	return image.Rect(left, top, left+sortButtonSize, top+height)
}

// sortButtonCentre is the centre of the i'th button in the column, which is what
// models.Button stores.
func sortButtonCentre(gs *state.GlobalState, i int) image.Point {
	col := sortColumnRect(gs)
	return image.Pt(
		col.Min.X+sortButtonSize/2,
		col.Min.Y+i*(sortButtonSize+sortButtonGap)+sortButtonSize/2,
	)
}

// setSort switches the arrangement, rearranges the hand and sends every card that moved
// sliding to its new place.
//
// Pressing the active mode again re-sorts rather than doing nothing, which is what makes it
// the way to undo a drag: the button the player is looking at is already the one describing
// the order they want back.
//
// **The cards have already moved by the time the slides exist**, exactly as they have for a
// discard or a deal — see spendSelected. The hand is in its new order the instant sortHand
// returns and every slide is a ghost of a card that is already where it is going.
func (s *CombatScene) setSort(mode handSort) {
	s.sortMode = mode

	n := len(s.hand)
	for to, from := range s.sortHand() {
		if from == to {
			continue
		}
		s.addSlide(handSlide{
			travel:    newTravel(0, slideTicks),
			card:      s.hand[to].actionCard,
			selected:  s.hand[to].selected,
			fromIndex: from, fromCount: n,
			toIndex: to, toCount: n,
		})
	}

	trace.Logf("input", "hand sorted by %v -> %s", mode, handLabel(s.hand))
}

// sortHand rearranges the hand into the current mode.
//
// **Stable**, so two genuinely identical cards keep the order they were in — one of them may
// be selected and therefore lifted out of the row, and a card jumping sideways because its twin
// was dealt is a movement with no cause on screen.
//
// **It resyncs the queue, and that is not housekeeping.** The list is the authority on the
// queue's order as well as its membership, and `handIndexForQueue` is the inverse of that one
// walk — so a hand rearranged under a stale `fighterActions` would leave the combo preview
// bracketing whichever cards happen to sit at the old positions. Nothing about the *round*
// changes: the queue holds the same cards, and order is not something the engine reads.
//
// **It returns the permutation it applied** — for each new position, the index that card came
// from — because a card sliding to its new place has to know where it set off from, and two
// identical cards cannot be told apart afterwards by looking at them. Sorting a slice of
// indices and rebuilding the hand from it is what makes that answer available at all; sorting
// the cards in place throws it away.
func (s *CombatScene) sortHand() []int {
	mode := s.sortMode

	order := make([]int, len(s.hand))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return handLess(mode, s.hand[order[i]].actionCard, s.hand[order[j]].actionCard)
	})

	sorted := make([]paletteCard, len(s.hand))
	for to, from := range order {
		sorted[to] = s.hand[from]
	}
	s.hand = sorted

	s.syncQueue()
	return order
}

// handLess is the comparison each mode makes. Every one of them falls through to the same
// secondary chain, so the modes differ only in what they put first.
func handLess(mode handSort, a, b actionCard) bool {
	switch mode {
	case sortByType:
		if ra, rb := categoryRank(a.Action.Category()), categoryRank(b.Action.Category()); ra != rb {
			return ra < rb
		}
	case sortByElement:
		if ra, rb := elementRank(a.Element), elementRank(b.Element); ra != rb {
			return ra < rb
		}
	}
	return costChainLess(a, b)
}

// costChainLess is the default order, and the tail every other mode ends with: cheapest first,
// then the deck overlay's own keys.
//
// **It is the overlay's chain with cost promoted to the front**, deliberately rather than
// coincidentally. The panel arranges the whole deck family-first because what a player looks
// for there is how much of a family they still hold; a hand is looked at to find what can be
// afforded, so cost leads. Everything under that is the same order in both places, so scanning
// a row of cards means the same thing wherever the row is.
func costChainLess(a, b actionCard) bool {
	if ca, cb := a.Action.Cost(), b.Action.Cost(); ca != cb {
		return ca < cb
	}
	if ra, rb := familyRank(a.Action.Family()), familyRank(b.Action.Family()); ra != rb {
		return ra < rb
	}
	if a.Action != b.Action {
		return a.Action < b.Action
	}
	return elementRank(a.Element) < elementRank(b.Element)
}

// categoryRank is the order the type sort runs in: everything that attacks, then the plans.
//
// A function rather than the enum's own order, for the reason familyRank is one — the enum is
// grouped for the rules, and reading it here would tie how the hand is arranged to a rules
// decision that has no reason to keep agreeing with it.
func categoryRank(c combat.Category) int {
	switch c {
	case combat.CategoryAttack:
		return 0
	case combat.CategoryPlan:
		return 1
	default:
		return 2
	}
}

// elementRank is the order the element sort runs in: fire, ice, lightning, earth, then the
// colourless cards.
//
// **Basic is last and the enum has it first**, which is the whole reason this is written out.
// `combat.Basic` is the zero value because a card that names no element is a plain card — a
// rules decision — but on screen the colourless cards are the plans, and the player is reading
// the four colours to see what a mix is worth. So the run of colours leads and the drab tail
// follows, which also puts the plans at the same end of the row as the type sort does.
func elementRank(e combat.Element) int {
	switch e {
	case combat.Fire:
		return 0
	case combat.Ice:
		return 1
	case combat.Lightning:
		return 2
	case combat.Earth:
		return 3
	case combat.Basic:
		return 4
	default:
		return 5
	}
}

// updateSortButtons runs the column and latches whichever mode is active.
//
// **All three go dead outside planning**, and that is a rule rather than tidiness: a card that
// has resolved is drawn from the hand slot it flew out of — see resolvedCard.handIndex — so
// rearranging the hand mid-round would light the wrong card on the table. The deck overlay
// takes them out for the reason it takes out everything else: it is a dialog.
func (s *CombatScene) updateSortButtons(gs *state.GlobalState) {
	live := s.planning() && !s.showDeck
	for i, b := range s.sortButtons {
		b.Latched = sortButtonSpecs[i].mode == s.sortMode
		setEnabled(b, live)
		systems.UpdateButton(gs, b)
	}
}

// drawSortButtons draws the column.
func (s *CombatScene) drawSortButtons(gs *state.GlobalState, screen *ebiten.Image) {
	for _, b := range s.sortButtons {
		systems.DrawButton(gs, screen, b)
	}
}

// buildSortButtons builds the column, wiring each button to the mode it selects. A method on
// the scene rather than a free function because the callback has to reach the scene's own
// state, which is the same reason every other widget on this screen is built here.
func (s *CombatScene) buildSortButtons() {
	s.sortButtons = make([]*models.Button, 0, len(sortButtonSpecs))
	for _, spec := range sortButtonSpecs {
		mode := spec.mode // captured per button, not per loop
		b := models.NewButton(sortButtonSize, sortButtonSize, spec.label,
			func() { s.setSort(mode) })
		b.BaseColor = sortButtonColor
		b.TextSize = sortButtonTextSize
		s.sortButtons = append(s.sortButtons, b)
	}
}

// sortButtonColor is a muted slate, quieter than either button on the strip below.
//
// **Deliberately not crimson or the Discard yellow**: those two commit a round, and these
// three cannot change what a round does at all. The base is light enough to leave the latched
// state somewhere to go — the active mode is drawn *darker* than the two beside it, so the
// bright end of the ramp stays with hover and press.
var sortButtonColor = color.RGBA{R: 110, G: 125, B: 155, A: 255}

func (m handSort) String() string {
	switch m {
	case sortByCost:
		return "cost"
	case sortByType:
		return "type"
	case sortByElement:
		return "element"
	default:
		return "?"
	}
}
