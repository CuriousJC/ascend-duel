package screens

// How many cards one essence takes, wherever it is being spent.
//
// **One number, three screens.** An essence is aimed at the deck from the reward screen's offer,
// from a vial in the shop and out of the satchel mid-fight, and each of them has to ask for the
// same number of cards or the relic that moves it would be a relic that works in some rooms. The
// run owns the figure — see `session.Session.EssenceTargets` and `combat.MomentEssenceSpent`; what
// is here is the one thing a *screen* knows that the run does not, which is how many cards it is
// able to offer.

import "github.com/curiousjc/ascend-duel/internal/state"

// essenceTargetCount is how many cards an essence spent on this screen takes, given how many are
// standing in front of the player.
//
// **Clamped to what is on offer, and never below one.** A run wearing two Cloud Necklaces wants
// four cards, and a deck thinned to three would otherwise leave every essence in the game
// unspendable — a consumable that can never be clicked is worse than one that does less than it
// promised. The clamp is here rather than in the run because the *row* is a fact about a screen.
func essenceTargetCount(gs *state.GlobalState, available int) int {
	if gs.Run == nil {
		return 1
	}
	n := gs.Run.EssenceTargets()
	if n > available {
		n = available
	}
	if n < 1 {
		n = 1
	}
	return n
}

// runEssenceTargets is the run's own reach, with nothing to clamp it against.
//
// **For a pane that cannot know where the essence will be spent.** The consumables pane is drawn on
// the between-fights screens too, and an essence carried past one is aimed at a hand that has not
// been dealt — so the honest thing to say is what the essence will do rather than a number derived
// from a row that is not the row it will land in.
func runEssenceTargets(gs *state.GlobalState) int {
	if gs.Run == nil {
		return 1
	}
	return gs.Run.EssenceTargets()
}

// essenceReach is the number a tooltip should say: how many cards this click would actually change.
//
// **The reach is a ceiling, so what an essence does depends on what is selected** *(owner's call,
// 2026-09-19)*. A player wearing a Cloud Necklace who has picked one card is about to change one
// card, and a tooltip promising two would be describing a click they are not making. With nothing
// picked there is no click to describe, so it says how far the essence can go.
//
// This is the one place that decision is made, so the reward screen, the shop's vial and the
// consumables pane cannot come to three different answers.
func essenceReach(selected, ceiling int) int {
	if selected > 0 && selected <= ceiling {
		return selected
	}
	return ceiling
}
