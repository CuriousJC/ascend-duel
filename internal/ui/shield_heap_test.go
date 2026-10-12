package ui

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
)

// Every element has a shield drawing of its own, and the lookup is total — an element with no
// drawing would put a hole in a heap rather than a shield somebody could see was wrong.
func TestEveryShieldHasItsOwnDrawing(t *testing.T) {
	seen := map[string]cards.Element{}
	for _, e := range append(cards.Elements(), cards.Basic) {
		key := ShieldKey(e)
		if key == "" {
			t.Errorf("%s has no shield key", e)
			continue
		}
		if prev, dup := seen[key]; dup && isElemental(e) && isElemental(prev) {
			t.Errorf("%s and %s both draw shield %q", prev, e, key)
		}
		seen[key] = e
	}
}

// isElemental is the five a card is counted on. Basic and Relic share the neutral shield on purpose,
// so the duplicate check above has to know which pairs are allowed to collide.
func isElemental(e cards.Element) bool {
	switch e {
	case cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane:
		return true
	}
	return false
}
