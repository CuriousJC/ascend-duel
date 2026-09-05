package screens

// How a dealt hand is arranged, and the block of three tabs that chooses it.
//
// **This is the half that belongs to no screen** *(2026-09-05)*. It was all one file on the
// combat screen, which was right while that screen was the only place a hand was laid out. The
// worm screen deals one too — eight cards off the run deck, to point a worm at — and a row of
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
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// handSort is which arrangement a hand is in. Cost is the zero value because it is the
// default, so a state that has never been touched is already sorted the way a fresh one is.
type handSort int

const (
	sortByCost handSort = iota
	sortByType
	sortByElement
)

// handSortOf is the mode the whole game is reading hands in.
//
// **A function rather than a field read, because the state may be nil**: OpeningCards builds a
// bare CombatScene with no global state at all, to answer what a seed deals. Cost is what that
// caller wants and it is also the zero value, so there is one answer rather than a special case.
func handSortOf(gs *state.GlobalState) handSort {
	if gs == nil {
		return sortByCost
	}
	return handSort(gs.HandSort)
}

// setHandSort records the choice. The screen that took the click is what then rearranges its own
// row — this stores the preference and nothing else.
func setHandSort(gs *state.GlobalState, mode handSort) {
	if gs != nil {
		gs.HandSort = int(mode)
	}
}

// sortColumnGap is the clear air between the last card of the widest row and the block beside it.
//
// **It belongs to the cards rather than to the block**: on the combat screen it is what
// cardBandWidth takes off the hand, and on the worm screen it is what the offer row's own right
// edge is measured from. One figure, so the two rows sit the same distance from their tabs.
const sortColumnGap = 12

// sortButtonSpecs is the block, top to bottom, with the label each tab carries.
//
// **The labels are bare nouns** *(2026-09-04, owner's call)*. They were `$`, `T` and `E`, then
// `Sort: Cost` and its two siblings — and the prefix went as soon as the three were one block,
// because a block of three tabs is self-evidently one control and the word was then written three
// times to say what the group is. `Form` is the axis the middle one actually sorts on; it was
// called Type when the label was one letter.
//
// **They carry no tooltip**, and with the labels spelled out they no longer want one — see the
// tooltip entry in TODO.md, which names the figures written straight onto the table as the gap.
var sortButtonSpecs = []struct {
	mode  handSort
	label string
}{
	{sortByCost, "Cost"},
	{sortByType, "Form"},
	{sortByElement, "Element"},
}

// sortButtonColor is a muted slate, quieter than either button on the combat screen's strip.
//
// **Deliberately not crimson or the Discard yellow**: those two commit a round and these three only
// rearrange one — which is no longer the same as saying they change nothing, since 2026-08-26. They
// are still the quieter control of the two kinds. The base is light enough to leave the latched
// state somewhere to go — the active mode is drawn *darker* than the two beside it, so the
// bright end of the ramp stays with hover and press.
var sortButtonColor = color.RGBA{R: 110, G: 125, B: 155, A: 255}

// sortTabs is the block of three as a widget: three models.Button, one latched, drawn touching so
// the group reads as one control with three tabs rather than as three controls that happen to
// agree about which of them is lit.
//
// **It owns no mode and no row.** A scene hands it where its rungs go and what to do when one is
// pressed, and it latches whichever mode is current — which is what lets the same block stand
// beside a queue on one screen and beside a row of deck cards on another without either screen
// learning about the other's list.
type sortTabs struct {
	buttons []*models.Button

	// seat is the i'th rung's rectangle. **A function rather than a stored rectangle**, because
	// both screens derive it from a row that is itself derived from the screen size.
	seat func(gs *state.GlobalState, i int) image.Rectangle
}

// newSortTabs builds the block, wiring each tab to the mode it selects.
//
// pick is called with the chosen mode. **It fires even when the pressed tab is already latched**,
// which is what makes it the way to undo a drag: the button the player is looking at is already
// the one describing the order they want back.
func newSortTabs(seat func(gs *state.GlobalState, i int) image.Rectangle, pick func(handSort)) *sortTabs {
	t := &sortTabs{seat: seat}
	for _, spec := range sortButtonSpecs {
		mode := spec.mode // captured per button, not per loop
		b := models.NewButton(ControlColumnWidth(), ControlButtonHeight, spec.label,
			func() { pick(mode) })
		b.BaseColor = sortButtonColor
		b.TextSize = ControlButtonText
		t.buttons = append(t.buttons, b)
	}
	return t
}

// place puts each tab on its rung. Called whenever the screen may have changed size — which for
// both callers is Init, the same place every other widget on either screen is placed.
func (t *sortTabs) place(gs *state.GlobalState) {
	for i, b := range t.buttons {
		r := t.seat(gs, i)
		b.ScreenX, b.ScreenY = r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2
	}
}

// rect is what the three of them occupy together: **one block, with no air in it**
// *(2026-09-04, owner's call)*.
//
// Derived from the tabs rather than written down, so anything measured against the block follows
// if the row moves.
func (t *sortTabs) rect(gs *state.GlobalState) image.Rectangle {
	first := t.seat(gs, 0)
	last := t.seat(gs, len(sortButtonSpecs)-1)
	return image.Rect(first.Min.X, first.Min.Y, first.Max.X, last.Max.Y)
}

// update runs the block and latches whichever mode is active. live is the screen's own answer to
// "may the row be rearranged right now"; a dead tab still latches, so the block keeps saying what
// order the cards are in even where it cannot be changed.
func (t *sortTabs) update(gs *state.GlobalState, live bool) {
	mode := handSortOf(gs)
	for i, b := range t.buttons {
		b.Latched = sortButtonSpecs[i].mode == mode
		setEnabled(b, live)
		systems.UpdateButton(gs, b)
	}
}

func (t *sortTabs) draw(gs *state.GlobalState, screen *ebiten.Image) {
	for _, b := range t.buttons {
		systems.DrawButton(gs, screen, b)
	}
}

// handLess is the comparison each mode makes. Every one of them falls through to the same
// secondary chain, so the modes differ only in what they put first.
func handLess(mode handSort, a, b actionCard) bool {
	switch mode {
	case sortByType:
		if ra, rb := categoryRank(a.Category()), categoryRank(b.Category()); ra != rb {
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
// coincidentally. The panel arranges the whole deck form-first because what a player looks
// for there is how much of a form they still hold; a hand is looked at to find what can be
// afforded, so cost leads. Everything under that is the same order in both places, so scanning
// a row of cards means the same thing wherever the row is.
func costChainLess(a, b actionCard) bool {
	if ca, cb := a.Cost(), b.Cost(); ca != cb {
		return ca < cb
	}
	if ra, rb := formRank(a.Form()), formRank(b.Form()); ra != rb {
		return ra < rb
	}
	if a.Concept != b.Concept {
		return a.Concept < b.Concept
	}
	return elementRank(a.Element) < elementRank(b.Element)
}

// categoryRank is the order the type sort runs in: everything that attacks, then the plans.
//
// A function rather than the enum's own order, for the reason formRank is one — the enum is
// grouped for the rules, and reading it here would tie how the hand is arranged to a rules
// decision that has no reason to keep agreeing with it.
func categoryRank(c combat.Category) int {
	switch c {
	case combat.CategoryAttack:
		return 0
	case combat.CategoryDefend:
		return 1
	default:
		return 2
	}
}

// elementRank is the order the element sort runs in: fire, ice, lightning, earth, arcane, then the
// colourless cards.
//
// **Basic is last and the enum has it first**, which is the whole reason this is written out.
// `combat.Basic` is the zero value because a card that names no element is a plain card — a
// rules decision — but on screen the colourless cards are the plans, and the player is reading
// the five colours to see what a mix is worth. So the run of colours leads and the drab tail
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
	case combat.Arcane:
		return 4
	case combat.Basic:
		return 5
	default:
		return 6
	}
}

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
