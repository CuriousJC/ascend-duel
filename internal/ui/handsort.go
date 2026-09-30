package ui

// How a dealt hand is arranged, and the block of three tabs that chooses it.
//
// **This is the half that belongs to no screen** *(2026-09-05)*. It was all one file on the
// combat screen, which was right while that screen was the only place a hand was laid out. The
// essence screen deals one too — eight cards off the run deck, to point an essence at — and a row of
// eight overlapping cards is exactly as hard to read there as it is in a duel.
//
// What is here: the modes, the comparison each one makes, and the tab block as a widget. What
// stays with a screen is what it does with a sorted list — see combat_sort.go, where the hand is
// a queue that has to be resynced and every card that moved is sent sliding.
//
// **The mode is `state.GlobalState.HandSort`, not a field on either scene** *(owner's call,
// 2026-09-05)*. One preference over every screen that deals a hand; see that field for why.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// HandSort is which arrangement a hand is in. Cost is the zero value because it is the
// default, so a state that has never been touched is already sorted the way a fresh one is.
type HandSort int

const (
	SortByCost HandSort = iota
	SortByForm
	SortByElement
)

// HandSortOf is the mode the whole game is reading hands in.
//
// **A function rather than a field read, because the state may be nil**: OpeningCards builds a
// bare CombatScene with no global state at all, to answer what a seed deals. Cost is what that
// caller wants and it is also the zero value, so there is one answer rather than a special case.
func HandSortOf(gs *state.GlobalState) HandSort {
	if gs == nil {
		return SortByCost
	}
	return HandSort(gs.HandSort)
}

// SetHandSort records the choice. The screen that took the click is what then rearranges its own
// row — this stores the preference and nothing else.
func SetHandSort(gs *state.GlobalState, mode HandSort) {
	if gs != nil {
		gs.HandSort = int(mode)
	}
}

// SortColumnGap is the clear air between the last card of the widest row and the block beside it.
//
// **It belongs to the cards rather than to the block**: on the combat screen it is what
// cardBandWidth takes off the hand, and on the essence screen it is what the offer row's own right
// edge is measured from. One figure, so the two rows sit the same distance from their tabs.
const SortColumnGap = 12

// SortButtonSpecs is the block, top to bottom, with the label each tab carries.
//
// **The labels are bare nouns** *(2026-09-04, owner's call)*. They were `$`, `T` and `E`, then
// `Sort: Cost` and its two siblings — and the prefix went as soon as the three were one block,
// because a block of three tabs is self-evidently one control and the word was then written three
// times to say what the group is. `Form` is the axis the middle one sorts on: stab, slash, crush,
// then defend.
//
// **They are set in capitals so the button draws them in the figure lettering** — the set is upper
// case, and a label with a lower-case letter in it falls back to the font, which is how these three
// came to be the only buttons on the screen not in the game's own lettering.
//
// **They carry no tooltip**, and with the labels spelled out they no longer want one — see the
// tooltip entry in TODO.md, which names the figures written straight onto the table as the gap.
var SortButtonSpecs = []struct {
	Mode  HandSort
	Label string
}{
	{SortByCost, "COST"},
	{SortByForm, "FORM"},
	{SortByElement, "ELEMENT"},
}

// SortButtonColor is ButtonGray, quieter than either button on the combat screen's strip.
//
// **Deliberately not the red or the Discard yellow**: those two commit a round and these three only
// rearrange one — which is no longer the same as saying they change nothing, since 2026-08-26. They
// are still the quieter control of the two kinds. The active mode is drawn *darker* than the two
// beside it, so the bright end of the ramp stays with hover and press.
var SortButtonColor = ButtonGray

// SortTabs is the block of three as a widget: three models.Button, one latched, drawn touching so
// the group reads as one control with three tabs rather than as three controls that happen to
// agree about which of them is lit.
//
// **It owns no mode and no row.** A scene hands it where its rungs go and what to do when one is
// pressed, and it latches whichever mode is current — which is what lets the same block stand
// beside a queue on one screen and beside a row of deck cards on another without either screen
// learning about the other's list.
type SortTabs struct {
	buttons []*models.Button

	// seat is the i'th rung's rectangle. **A function rather than a stored rectangle**, because
	// both screens derive it from a row that is itself derived from the screen size.
	seat func(gs *state.GlobalState, i int) image.Rectangle
}

// NewSortTabs builds the block, wiring each tab to the mode it selects.
//
// pick is called with the chosen mode. **It fires even when the pressed tab is already latched**,
// which is what makes it the way to undo a drag: the button the player is looking at is already
// the one describing the order they want back.
func NewSortTabs(seat func(gs *state.GlobalState, i int) image.Rectangle, pick func(HandSort)) *SortTabs {
	t := &SortTabs{seat: seat}
	for _, spec := range SortButtonSpecs {
		mode := spec.Mode // captured per button, not per loop
		b := models.NewButton(ControlColumnWidth(), ControlButtonHeight, spec.Label,
			func() { pick(mode) })
		b.BaseColor = SortButtonColor
		b.Bevelled = true
		b.TextSize = ControlButtonText
		t.buttons = append(t.buttons, b)
	}
	return t
}

// Place puts each tab on its rung. Called whenever the screen may have changed size — which for
// both callers is Init, the same Place every other widget on either screen is placed.
func (t *SortTabs) Place(gs *state.GlobalState) {
	for i, b := range t.buttons {
		r := t.seat(gs, i)
		b.ScreenX, b.ScreenY = r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2
	}
}

// Rect is what the three of them occupy together: **one block, with no air in it**
// *(2026-09-04, owner's call)*.
//
// Derived from the tabs rather than written down, so anything measured against the block follows
// if the row moves.
func (t *SortTabs) Rect(gs *state.GlobalState) image.Rectangle {
	first := t.seat(gs, 0)
	last := t.seat(gs, len(SortButtonSpecs)-1)
	return image.Rect(first.Min.X, first.Min.Y, first.Max.X, last.Max.Y)
}

// Update runs the block and latches whichever mode is active. live is the screen's own answer to
// "may the row be rearranged right now"; a dead tab still latches, so the block keeps saying what
// order the cards are in even where it cannot be changed.
func (t *SortTabs) Update(gs *state.GlobalState, live bool) {
	mode := HandSortOf(gs)
	for i, b := range t.buttons {
		b.Latched = SortButtonSpecs[i].Mode == mode
		SetEnabled(b, live)
		systems.UpdateButton(gs, b)
	}
}

func (t *SortTabs) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	for _, b := range t.buttons {
		systems.DrawButton(gs, screen, b)
	}
}

// HandLess is the comparison each mode makes. Every one of them falls through to the same
// secondary chain, so the modes differ only in what they put first.
//
// **worn is the holder's relics, and cost is read through them** — `combat.CostWith`, the figure
// the card's face prints and the AP bar charges. A card's own `Cost` is its printed cost before any
// discount, so a row sorted on it puts a card showing 2 AP after one showing 3. Nil worn is a card
// nobody is holding, which is its printed cost.
func HandLess(mode HandSort, a, b combat.Card, worn []combat.WornRelic) bool {
	switch mode {
	case SortByForm:
		if ra, rb := formRank(a.Form()), formRank(b.Form()); ra != rb {
			return ra < rb
		}
	case SortByElement:
		if ra, rb := elementSortRank(a), elementSortRank(b); ra != rb {
			return ra < rb
		}
	}
	return costChainLess(a, b, worn)
}

// elementSortRank is where a card stands in the element sort: its element's rank, and **every
// wildcard after all of them**. A wildcard counts as every element, so no one color's run of cards
// is where it belongs; filed under the element it carries, it sits among the fires while its corner
// mark says nothing about fire.
func elementSortRank(c combat.Card) int {
	if c.Wild(combat.AxisElement) {
		return len(combat.AllElements) + 1
	}
	return ElementRank(c.Element)
}

// costChainLess is the default order, and the tail every other mode ends with: cheapest first,
// then the deck overlay's own keys.
//
// **It is the overlay's chain with cost promoted to the front**, deliberately rather than
// coincidentally. The panel arranges the whole deck form-first because what a player looks
// for there is how much of a form they still hold; a hand is looked at to find what can be
// afforded, so cost leads. Everything under that is the same order in both places, so scanning
// a row of cards means the same thing wherever the row is.
func costChainLess(a, b combat.Card, worn []combat.WornRelic) bool {
	// **The run's flat cut is left out**, and the order is still the face's: a cut never reorders
	// two cards, it can only tie them at its floor, and a tie then falls to the price before it.
	if ca, cb := combat.CostWith(worn, 0, a), combat.CostWith(worn, 0, b); ca != cb {
		return ca < cb
	}
	if ra, rb := formRank(a.Form()), formRank(b.Form()); ra != rb {
		return ra < rb
	}
	if a.Concept != b.Concept {
		return a.Concept < b.Concept
	}
	return ElementRank(a.Element) < ElementRank(b.Element)
}

// ElementRank is the order the element sort runs in: fire, ice, lightning, earth, arcane, then the
// colorless cards.
//
// **Basic is last and the enum has it first**, which is the whole reason this is written out.
// `combat.Basic` is the zero value because a card that names no element is a plain card — a
// rules decision — but on screen the colorless cards are the plans, and the player is reading
// the five colors to see what a mix is worth. So the run of colors leads and the drab tail
// follows, which also puts the plans at the same end of the row as the form sort does.
func ElementRank(e combat.Element) int {
	switch e {
	case combat.Fire:
		return 0
	case combat.Ice:
		return 1
	case combat.Lightning:
		return 2
	case combat.Earth:
		return 3
	case combat.Arcane:
		return 4
	case combat.Basic:
		return 5
	default:
		return 6
	}
}

func (m HandSort) String() string {
	switch m {
	case SortByCost:
		return "cost"
	case SortByForm:
		return "form"
	case SortByElement:
		return "element"
	default:
		return "?"
	}
}
