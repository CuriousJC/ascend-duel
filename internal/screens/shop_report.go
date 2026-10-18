package screens

// **What the shop says about itself in a crash report.**
//
// See ui.Reporter for the contract and combat_report.go for the rule both files are written under:
// **every value below is a plain field already on the struct, a length, or a count over one**,
// because the screen being asked has just panicked and a derivation is a second crash inside the
// first.

// Report is this screen's account of itself.
//
// **The shelf's keys are here and nothing else about it is.** They are what a scenario fixture
// would write down to stock the same shelf again; everything else on a relic is in the catalog.
func (s *ShopScene) Report() map[string]any {
	bought := 0
	for _, it := range s.shelf {
		if it.bought {
			bought++
		}
	}
	return map[string]any{
		"shelf":   shelfKeys(s.shelf),
		"bought":  bought,
		"offered": len(s.offered),
		"opened":  len(s.opened),
		"drunk":   len(s.drunk),

		// A reroll is two ticks — armed on one, applied on the next — so a crash between them is
		// the one this pair is for.
		"rerolling": int(s.rerolling),
		"rerollNow": s.rerollNow,
		"leaving":   s.leaving,

		// The visit the scene's state belongs to. A re-entry that matches is picked up rather than
		// dealt again, so a mismatch here against the run's own fight is a shelf from another visit.
		"fight": s.visit.fight,
	}
}
