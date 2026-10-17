package ui

// **A tonic as a card, and what it says it does.** The shop sells one, the Compendium shelves every
// one, and the deck panel shows the ones a run has drunk — three screens, one face and one sentence,
// so a tonic reads the same wherever it is met.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// TonicSpec is a tonic drawn as a card, on a potion's terms: basic, the picture as the whole face,
// and what it does in the tooltip.
func TonicSpec(gs *state.GlobalState, t session.Tonic, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    t.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: ArtFor(combat.Basic),
		Art:     Artwork(gs, t.Art),
		Enabled: enabled,
	}
}

// TonicEffect is what a tonic changes, and that it lasts the whole run. **No price**: the shop adds
// its own, and a tonic already drunk has none left to pay.
func TonicEffect(t session.Tonic) []string {
	var what []string
	switch t.Effect {
	case session.TonicDiscount:
		what = []string{fmt.Sprintf("everything in the shop costs %d%% less", t.Amount), "and at least 1 vitae less"}
	case session.TonicCostCut:
		what = []string{fmt.Sprintf("your cards cost %d AP less", t.Amount), "but never less than 1 AP"}
	case session.TonicPressure:
		what = []string{"every fight is 1 round shorter", fmt.Sprintf("and every vitae you earn is x%d", t.Amount)}
	case session.TonicHandSize:
		what = []string{fmt.Sprintf("%d more card in every hand", t.Amount)}
	case session.TonicDiscards:
		what = []string{fmt.Sprintf("%d more discard every round", t.Amount)}
	case session.TonicRelicSlots:
		what = []string{fmt.Sprintf("wear %d more relic", t.Amount)}
	}
	return append(what, "for the rest of the run")
}
