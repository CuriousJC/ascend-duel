package combat

// **The round limit: every fight is on a clock.**
//
// A duel that has run its rounds out kills whoever was still fighting it — see MECHANICS.md
// §The round limit, where the design is written down. What is here is only the arithmetic: how
// many rounds a duelist gets, and what happens on the last one.
//
// **It is a rule and not a screen's countdown**, which is the whole reason it lives in this
// package. The bar the player watches fill is drawn from `CombatScene.round` and decides nothing;
// this is what actually ends the fight, resolved inside the round like every other outcome and
// handed to the screen as events to replay. A clock the presentation owned would be a clock that
// ran at the speed of the animation.

// DefaultRoundLimit is how many rounds a fight gets. **Five, for every fight in the tower** —
// the ordinary rooms and the stairway protectors alike *(owner's call, 2026-09-06)*.
//
// **It is a default and not the rule.** Nothing in this package reads it: a duelist's own
// `RoundLimit` is what the clock is checked against, and this is the number a run starts that
// field at. The indirection is what a relic or a brand that buys the player a sixth round will
// move — see session.Session.RoundLimit, which is where a run's copy lives.
const DefaultRoundLimit = 5

// outOfTime reports that this duelist's clock has run out on the round just finished.
//
// **`>=` rather than `==`**, so a limit lowered mid-run by whatever eventually moves it cannot be
// stepped over by a fight that was already past it.
func outOfTime(d Duelist, round int) bool {
	return d.RoundLimit > 0 && round >= d.RoundLimit && d.Alive()
}

// FightOver reports that a duel between these two is finished, whoever won it.
//
// **The clock only fires on a fight still being fought**, which is the whole of what this is for:
// a duelist who killed their opponent on the final round has *beaten* the clock, and a rule that
// read only their own aliveness would take the win away on the same event that earned it. See
// TestKillingOnTheLastRoundIsAWin, which is the case a player actually plays for.
func FightOver(a, b Duelist) bool { return !a.Alive() || !b.Alive() }

// callTime is the clock killing a duelist who is still standing at the end of their last round.
//
// **Life goes to zero rather than taking a figure.** The clock is not a blow — there is nothing
// to survive with one life left and nothing to block it with — so what it announces is the whole
// of what it took, and `KindDefeated` follows it exactly as it follows the last burn tick.
func callTime(events []Event, side Side, d Duelist, round int) ([]Event, Duelist) {
	if !outOfTime(d, round) {
		return events, d
	}

	took := d.CurrentLife
	d.CurrentLife = 0

	events = append(events, Event{
		Kind:   KindTimeUp,
		Side:   side,
		Target: side,
		Amount: took,
		Life:   0,
		Round:  round,
	})
	events = append(events, Event{
		Kind:   KindDefeated,
		Side:   side,
		Target: side,
		Round:  round,
	})
	return events, d
}
