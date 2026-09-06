package screens

// Aiming a consumable: **select the cards, then click the thing you are spending on them.**
//
// **The gesture reversed on 2026-09-06** *(owner's call)*. It used to be arm-then-aim — open a
// dialog, click the parasite, then click the cards it eats — and it is now select-then-apply, on
// the row of cards the player is already looking at. The worm offer follows the same order, so
// there is one way to point a consumable at a card anywhere in the game.
//
// **What it costs, said out loud.** On the combat screen a selected card is also a card queued for
// the round, so one gesture now carries two meanings: these are the cards I am playing, and these
// are the cards this parasite eats. The objection was raised and overruled; what makes it workable
// is that a consumable is only *clickable* when the current selection is exactly what it needs, so
// the player is never asked which of the two meanings a click had. That predicate is this file.
//
// **The order of a selection is the order of the row.** `syncQueue` walks the hand in row order and
// so does this, which means a parasite naming a first and a second target — Clone — reads them left
// to right, and the player reorders by dragging exactly as they reorder the queue. There is no
// separate click order to learn or to draw.

// consumableTarget is what one consumable needs from a selection: how many cards, and whether a
// given set of them is legal.
//
// **The count is the record's and the legality is the run's.** Neither is this file's to decide —
// `session.Parasite.Count` says how many cards, `Session.CanApplyParasite` says whether these ones
// can take it, and a worm answers both through its own offer. What is here is the one rule that
// joins them, so the parasite pane and the worm row cannot come to two different conclusions about
// whether a click should be allowed.
type consumableTarget struct {
	// needs is how many cards must be selected. **Zero is a consumable that takes no target at
	// all** — a rock shower, a purse of vitae — and it is ready whatever is selected, because there
	// is nothing for a selection to be wrong about.
	needs int

	// legal reports whether this exact set of card identities can take it. Nil means any set of the
	// right size will do.
	//
	// **Identities, not positions.** A row is re-laid-out by a sort and re-dealt by a refill; an
	// identity survives both, and it is what the run's own apply takes.
	legal func(ids []int) bool
}

// satisfiedBy reports whether a selection is exactly what this consumable needs.
//
// **Exactly, not at least.** A parasite that ate two cards out of a selection of five would be
// choosing for the player which two, and there is nothing on screen that could say which it picked.
// So a selection of the wrong size leaves the consumable unclickable rather than the click picking
// a subset — the card goes dim, which is the same thing an unaffordable card on the shop shelf
// does.
func (c consumableTarget) satisfiedBy(ids []int) bool {
	if len(ids) != c.needs {
		return false
	}
	if c.needs == 0 || c.legal == nil {
		return true
	}
	return c.legal(ids)
}
