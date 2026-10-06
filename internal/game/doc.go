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
// One map from ActiveScreen to Scene, rather than parallel switches in Update and Draw, which could
// drift — a screen added to one and forgotten in the other silently does nothing. Adding a screen
// is one constant in internal/state and one entry here.
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
//
// # What the frame holds
//
// The **settings button** is a 44px square in the bottom-left corner of every screen, carrying a
// generated cog, and the **ledger button** sits beside it: the run's account of itself. See
// MECHANICS.md §The ledger, `internal/screens/ledger.go` for the panel and
// `internal/session/ledger.go` for what it holds.
//
//   - **The cog opens the settings screen and nothing else.** There is no mute anywhere, only a level,
//     and what the corner buys is one place for the game speed and the volume to live together.
//   - **While the ledger's panel is up, the active scene is not updated at all.** That freezes pacing
//     and, like every other dialog, cannot change an outcome.
//   - **The toast's queue is `state.EarnedThisSession`**, written wherever an award happens in
//     `internal/screens` and drained one box at a time — a five-element turn earns several achievements
//     together. Like the ledger it takes the frame while it is up.
//   - **The frame stands down on the settings screen itself**, the one screen where the corner would be a
//     door into the room the player is already standing in. `chromeShowing` is the one predicate both
//     halves ask; that screen carries its own Back button.
//   - **Never disabled.** The cog opens a screen, which always works — it is the *volume bar* on that
//     screen that goes dead, and it says why underneath itself rather than merely going gray.
//   - **Square and iconic because the corner is 52 pixels wide** on the combat screen — the hand band
//     starts at x=52 and the action-point figure sits on its left edge, so a labeled button does not fit.
//   - **The cog is `icon-settings`, and it has eight teeth: four on the axes, four on the
//     diagonals.** Four teeth read as a compass rose — at 32 pixels a gear is recognized by the *count*
//     of its teeth before any one of them is legible — and the hole in the middle is what makes it a cog
//     rather than a flower. Both are constraints on any replacement drawing.
//   - **The ledger's panel scrolls with `models.Scrollbar`**, built the way `models.Button` and
//     `models.Slider` are. It **counts rows, not pixels**, so a panel cannot land half a line off, and it
//     is a drag because the input vocabulary has no wheel.
//
// # Which music a screen plays
//
// `score.go` is one `state.ActiveScreen` table, read **every frame**, and no scene decides it.
// `music.PlayTrack` does nothing when the track is already sounding, so there is no previous screen to
// remember and no way for the table and what is audible to drift. A call in each scene's `Init` and a
// matching one to put the score back would be two edits per screen, and the forgotten one leaves the
// shop's loop playing over a duel.
//
//   - **Every screen is listed, including the ones that play the score.** `music.Score` is a name rather
//     than the empty string, because a zero value standing for a real thing is the pattern this project
//     rejects everywhere else. `TestEveryScreenNamesItsMusic` walks `state.ActiveScreen`, so a new screen
//     fails the suite rather than falling back quietly.
//   - **A screen gives one of two answers**: it names a piece of music, or it **leaves whatever is
//     sounding alone**. Settings, Achievements, Credits, the debug gallery and an opened sealed good are
//     reached *from* somewhere and go back to it, so they are transparent to the music the way they are
//     transparent to the run — a player who opens the volume bar to turn the shop's loop up must not have
//     the loop stop to let them. Coming back asks for a track that is already sounding, so the loop plays
//     through the visit instead of starting over.
//   - **It is deliberately not `chromeShowing`.** That predicate answers whether the frame draws its own
//     controls, and it is wrong for music in three places — `Title` and `PostBattle` stand the chrome down
//     while naming their own music, and `RunOver` is a destination rather than an overlay. Two questions
//     that agree about five screens and disagree about three are two tables.
//   - **One table rather than a map of tracks beside a set of exceptions**, because a screen written into
//     both would be a contradiction nothing catches.
//   - **A track nothing was loaded under gets the score**, so a clone with no bundle synced plays the
//     score everywhere rather than falling silent. `TestEveryNamedTrackIsInTheBundle` holds this table
//     against the committed manifest — **the manifest, not the directory**, since the audio is not in git
//     and a test looking for files would fail on every clean clone — and *reports* a bundled loop no
//     screen plays rather than failing on one.
package game
