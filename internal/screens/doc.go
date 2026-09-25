// Package screens is one Scene per screen of the game: the title, the duel, the between-fight
// scenes, the credits. Each owns its own state and its own widgets and draws itself; the game
// loop above it does nothing but pick which one is active.
//
// # Where to start
//
// Scene is the contract — Init, Update, Draw — and it lives in internal/ui/scene.go, which explains
// the lifecycle including the one rule that catches people out: Init may run more than once,
// because a screen can be re-entered, and re-entering the combat screen is how the next fight
// starts.
//
// flow.go is the loop. A scene that has finished calls advanceRun; where that leads is the run's
// business, not the scene's. Adding a screen to the game is a phase in internal/session, an entry
// in this package's phaseScreens table, and an entry in the registry in internal/game — and
// nothing else, because no scene names its successor.
//
// **The drawing left on 2026-09-17.** Everything a scene draws *through* — the table, the clock,
// the movers, the card faces, the panels belonging to no screen, the prose — is internal/ui now,
// and what is here is the scenes. If you are looking for a symbol the combat-screen skill names and
// it is not in this package, it is in that one.
//
// # The rules this package works under
//
//   - internal/combat decides rounds; a screen only replays them. Never change a rule to make a
//     screen look right. Say which of the two is wrong and let the owner decide.
//   - Presentation may never change an outcome. A whole round is resolved before playback begins,
//     so playback speed, a flight, a dialog that pauses the cursor and every debug view may alter
//     pacing and must not alter results.
//   - A screen's working state lives on its scene, never on GlobalState. If exactly one screen
//     reads a field, it belongs to that screen. If it has to outlive a fight, it belongs to
//     internal/session.
//   - Clicks and drags only. No right click, no hotkeys, and one typed-text field in the whole
//     game — the seed. Anything that wants a keyboard needs a different design.
//   - Cards fly; they never appear. Anything that changes where it is on screen travels there.
//   - **Nothing here may be reached from internal/ui.** That package draws for whoever asks and
//     knows about no screen at all; a shared helper that needs a scene's geometry is a helper on
//     the wrong side of the line. Run the `audit` skill's pkgsplit to see whether it still holds.
//
// # Testing
//
// This package links Ebitengine, so most of it cannot be tested without a window. The tests that
// exist are a deliberate narrow exception: they compare constants and walk switch statements,
// create no images, and guard cross-package invariants a compiler cannot see. They are not license
// to test the rest of the screen, and nothing here should reach for a window to keep one alive.
// What cannot be unit-tested gets a tool instead — see the demoplay build tag.
//
// # The files
//
// **A grouping, not an inventory, and that is a decision** *(2026-09-17)*. This section used to
// carry a paragraph per file, duplicating each file's own header comment — and because the two
// said the same thing, only one of them ever got edited. By the time it was measured it named 37
// of the package's 85 files, described a button that had been deleted and a panel that had been
// replaced. **Each file's own header comment under its package clause is the detail**; what
// belongs here is only which group a file is in, so that an edit knows where to start looking.
//
// A file boundary is not a reason to change what a function does. Moving something between these
// files is a move, not a rewrite.
//
//   - **The loop and the run** — scene wiring in flow.go, run.go and save.go. run.go is a run's
//     whole lifecycle: BootRun, NewRun, ContinueRun, AbandonRun, and EndRunInDefeat, which is
//     AbandonRun under a second name because dying and giving up are one event.
//   - **The combat screen** — combat.go and the combat_*.go files. The scene, the piles, the hand
//     and its drag, the table, the relic row, the HUD, and one file per thing that moves on it:
//     the flights, the hits, the shields, the shatter, the signals, the hand dialog, the deal. Read the
//     `combat-screen` skill before touching any of them.
//   - **The between-fight scenes** — postbattle*.go and shop*.go. Pick an essence and the card it
//     eats; then three relics on a shelf, the sealed goods, the pouch and the hooded creature who
//     greets you.
//   - **The screens that are not stations of a run** — title.go, credits.go, settings*.go,
//     achievements.go, runover.go, animations.go, ascend.go. None appears in flow.go, which is
//     what "not a station" means mechanically.
//   - **The tutorial's half** — tutorial.go and the *_tutorial.go files. The state machine is
//     internal/tutorial; what is here is Bob's bubble, the lit rectangle, and each scene
//     publishing the facts a step is a predicate over.
//   - **The run's account** — ledger*.go, and combat_ledger.go, which is the three call sites that
//     put a duel into it. The panel itself is chrome, held by internal/game.
//   - **Things a scene owns that internal/ui deliberately does not** — controlcolumn.go (placement
//     measured from the hand and the action-point bar), buildband.go, consumables.go,
//     stoneflight.go, targeting.go, seeds.go, achieve.go.
package screens
