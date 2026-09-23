package ui

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene is one screen of the game. Each implementation owns its own state and its
// own widgets, rather than parking them on GlobalState — a screen's working data is
// nobody else's business, and the combat screen in particular is about to grow a
// great deal of it.
//
// GlobalState is still passed in, because a scene legitimately needs what is
// genuinely global: assets, fonts, layout, the mouse, and the ability to ask for a
// screen change.
//
// Init runs once per entry to the screen, driven by GlobalState.NewScreen from
// game.Update. A scene may be entered more than once in a session, so Init must be
// safe to call repeatedly: build expensive things behind a nil check and reset the
// per-visit state unconditionally.
type Scene interface {
	Init(gs *state.GlobalState)
	Update(gs *state.GlobalState) error
	Draw(gs *state.GlobalState, screen *ebiten.Image)
}

// Reporter is a scene that can describe itself for a crash report. **Optional**: a screen that does
// not implement it is left out of the report's scene tier rather than broken by not having one, so
// a new screen owes this nothing.
//
// **The method may read plain fields and nothing else.** The scene being asked to describe itself
// is the scene that has just panicked, so anything derived — a method call, a lookup, an index into
// a slice something else owns — is a second crash inside the first. Every value handed back is an
// int, a string or a bool already sitting on the struct. The call is made under its own recover
// anyway, on the same terms internal/game already reads a run under, but a Report that needs it has
// already lost the tier it was written to provide.
//
// **What belongs in it is whichever of a screen's state machines can disagree with another** — a
// playback cursor against the log it walks, a seat index against the row it points into. What does
// not is the contents of any of them: the journal holds the player's choices and the ledger holds
// the engine's, and a list of cards here would be a third, worse copy of both.
type Reporter interface {
	Report() map[string]any
}
