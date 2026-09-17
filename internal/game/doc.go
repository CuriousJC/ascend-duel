// Package game is the Ebitengine game loop and the frame around every screen.
//
// main builds a Game, loads assets, fonts and data once, and hands control to ebiten.RunGame.
// Three methods then drive everything:
//
//   - Update — the 60 TPS logic tick. Advances counters, reads the mouse, runs the active scene's
//     Init if NewScreen is set, then returns the scene's Update. A non-nil error quits the game;
//     ShouldClose becomes ErrClosing, which main treats as a clean exit.
//   - Draw — per-frame rendering. Returns early while NewScreen is set, so a scene is never drawn
//     before its Init has run.
//   - Layout — returns the fixed 1920x1080 internal resolution whatever the window is resized to,
//     which is what makes every absolute coordinate in the game safe.
//
// # The scene registry
//
// One map from ActiveScreen to Scene, replacing the two parallel switches Update and Draw used to
// carry — those could drift, and a screen added to one and forgotten in the other silently did
// nothing. Adding a screen is one constant in internal/state and one entry here.
//
// NewScreen is the one-shot init flag and it is consumed centrally, here, so no scene has to
// remember the dance.
//
// # The frame
//
// chrome.go draws the two controls that belong to no screen — the settings cog and the ledger —
// plus the achievement toast, which is not a control at all but the game telling the player they
// earned something. All three are deliberately outside the "scenes own their own widgets" rule
// rather than exceptions to it. The alternative was the same button on every scene: as many
// placements to keep in step and as many callbacks into one package.
//
// The bar for joining the frame is high: something true for the whole session, on every screen,
// owned by no scene. A frame is easy to grow by accident, and two of the three pass a test the
// settings button could not have passed alone — the ledger *could not* be a scene, because leaving
// the combat screen and coming back re-runs Init and deals a fresh duel, and the toast can land
// during a duel, on the post-battle screen, or on the transition between them.
//
// The gallery's button is the one thing in the strip that is not chrome by that test. It is
// instrumentation, drawn only while state.DebugAnimations is on, and it is there because the frame
// is the only place a debug page is reachable from every screen.
//
// state.ModalOpen is what it cost. A scene sets it while it has a dialog up and the chrome neither
// updates nor draws, because a modal has to make its exit the brightest thing on screen or it is a
// trap — there is no Escape key and no right click. The frame clears the flag each tick and the
// scene re-asserts it, so leaving a screen with its overlay open cannot hide the chrome for the
// rest of the session.
package game
