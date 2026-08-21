// Package actions holds button callbacks that act on the game as a whole — changing
// screen, quitting. They take GlobalState and mutate it; they never draw.
//
// Callbacks that only touch one screen's own state do not belong here. Those are
// methods on the scene that owns the state, wired up when the scene builds its own
// widgets — see CombatScene.startRound.
package actions
