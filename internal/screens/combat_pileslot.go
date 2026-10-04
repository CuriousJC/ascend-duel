package screens

// modalUp reports whether any of this screen's dialogs is covering it.
//
// **One predicate rather than the conditions spelled out at every call site**, because every
// control on this screen has to go dead for all of them and the failure is silent: a button left
// live under a dialog is a round edited through a panel the player is only reading.
func (s *CombatScene) modalUp() bool {
	return s.showDeck || s.hands.IsOpen()
}
