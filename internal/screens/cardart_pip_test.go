package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/cards"
)

// The shield pip is the defend mark in the element that raised it, and every element has one.
//
// **This replaced TestTintedPipKeepsItsBrightness on 2026-09-16.** That test held the multiply that
// turned one near-white mark into five colored ones, and the multiply is gone: there are five
// authored shields plus a neutral one, so a pip is a lookup rather than a recolor. What is worth
// pinning is that the lookup is total — an element with no drawing would put a hole in the badge
// row rather than a pip somebody could see was wrong.
func TestEveryShieldPipHasItsOwnDrawing(t *testing.T) {
	seen := map[string]cards.Element{}
	for _, e := range append(cards.Elements(), cards.Basic) {
		key := shieldPipKey(e)
		if key == "" {
			t.Errorf("%s has no shield pip key", e)
			continue
		}
		if prev, dup := seen[key]; dup && isElemental(e) && isElemental(prev) {
			t.Errorf("%s and %s both draw pip %q", prev, e, key)
		}
		seen[key] = e
	}
}

// isElemental is the five a card is counted on. Basic and Relic share the neutral pip on purpose,
// so the duplicate check above has to know which pairs are allowed to collide.
func isElemental(e cards.Element) bool {
	switch e {
	case cards.Fire, cards.Ice, cards.Lightning, cards.Earth, cards.Arcane:
		return true
	}
	return false
}
