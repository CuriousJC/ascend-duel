package screens

// **What the reward screen says about itself in a crash report.**
//
// See ui.Reporter for the contract and combat_report.go for the rule both files are written under:
// **every value below is a plain field already on the struct, a length, or a count over one**,
// because the screen being asked has just panicked and a derivation is a second crash inside the
// first.

// Report is this screen's account of itself.
//
// **The question it exists to answer is which of the screen's clocks was where.** A visit runs the
// payout narration, the essence offer, the dealt row an essence is aimed at and the stage that ends
// the visit side by side, and a fault here is nearly always one of them reading a thing another has
// not built yet or has already taken away.
func (s *PostBattleScene) Report() map[string]any {
	taken := 0
	for _, p := range s.prizes {
		if p.taken {
			taken++
		}
	}
	return map[string]any{
		// The offer, and how far through it the player is.
		"stage":     int(s.stage),
		"prizes":    len(s.prizes),
		"taken":     taken,
		"chosen":    s.chosen,
		"picksLeft": s.picksLeft,
		"held":      s.held,
		"skipping":  s.skipping,
		"pending":   s.pendingWhat,
		"landing":   len(s.lands),

		// The row an essence is aimed at.
		"offer":    len(s.offer),
		"selected": len(s.selected),

		// The payout narration: which line it is on out of how many.
		"proseLine":   s.prose.line,
		"proseLines":  len(s.prose.lines),
		"vitaeFlying": s.prose.flying,

		"fight": s.visit.fight,
	}
}
