// Package decks turns the card lists in `data` into cards the rules understand, and holds an
// opponent's three piles while a duel is played.
//
// **It exists so an enemy deck can be built without a screen** *(2026-08-11)*. `internal/screens`
// links Ebitengine, which on Linux calls `glfw.Init()` from a package `init()` and would need a
// display server — so anything headless that wants the opponent the game actually fights has to be
// able to reach the deck without going through the screen that draws it.
//
// **No Ebitengine here, ever**, for that reason. It sits between `data` and
// `internal/combat`, importing both, which neither of them may do.
//
// The player's deck deliberately stays in `internal/screens`. Its cards are drawn on screen and
// move through a hand that can be reordered, neither of which is true of an enemy's, and pulling
// it down here would mean giving this package a screen's vocabulary to reuse a loop.
//
// **This package is where enemy concepts are registered** *(2026-08-16)*. Card definitions became
// data — see `combat.RegisterConcept` — and `data` may not import the rules, so something between
// the two has to hand one to the other. This is the package built for exactly that edge, and it is
// the reason every enemy deck in the game is built here rather than in the screen that draws it.
//
// # The enemy's hand does not persist between rounds
//
// The player's does, because Discard exists. An enemy has no such lever: a planner only takes what
// it can spend, so everything else would accumulate until the hand locked up — which it did.
package decks
