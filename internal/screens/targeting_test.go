package screens

import "testing"

// TestAConsumableTakingNoCardIgnoresTheSelection. A Hoard is live whatever is picked for the round,
// and a chimera with nothing to copy is live under no selection at all.
func TestAConsumableTakingNoCardIgnoresTheSelection(t *testing.T) {
	none := consumableTarget{}
	for _, ids := range [][]int{nil, {4}, {4, 7, 9}} {
		if !none.satisfiedBy(ids) {
			t.Errorf("a no-card consumable refused a selection of %d", len(ids))
		}
	}
	dead := consumableTarget{needs: -1}
	for _, ids := range [][]int{nil, {4}} {
		if dead.satisfiedBy(ids) {
			t.Errorf("a consumable that can never be used was live under a selection of %d", len(ids))
		}
	}
}
