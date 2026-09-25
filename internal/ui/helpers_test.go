package ui

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// handEventFor builds a KindHand event by hand, standing in for one the resolver produced.
func handEvent(hand string, amounts []int, multiplier, total int) combat.Event {
	id, ok := combat.HandIDForKey(hand)
	if !ok {
		panic("the catalog has no hand keyed " + hand)
	}

	e := combat.Event{
		Kind:       combat.KindHand,
		Hand:       id,
		Multiplier: multiplier,
		Amount:     total,
	}
	for i, a := range amounts {
		e.HandCards[i] = i
		e.HandAmounts[i] = a
		e.HitAmounts[i] = a * multiplier / 100
		e.HandCardCount++
	}
	return e
}
