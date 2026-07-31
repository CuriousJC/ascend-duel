package screens

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
