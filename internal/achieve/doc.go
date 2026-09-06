// Package achieve decides what the player has earned, and nothing else.
//
// **It is the seat every word in `data/achievements.json` is refused at**, the same job
// `internal/session` does for a worm's target and `internal/combat` does for a card's verb. A
// trigger kind, a clause mode, an axis, a moment name and a counter name are all closed
// vocabularies here; a file inventing one fails the launch rather than producing an achievement
// that can never land, which is the failure this shape exists to prevent — an achievement nobody
// can earn looks exactly like an achievement nobody has earned yet.
//
// **Three trigger kinds, because the things they watch are three different shapes.** This is the
// one design decision worth reading before changing anything:
//
//   - A **turn** is gone the moment it resolves, so it has to be judged as it happens. It is also
//     the family that is pure grammar: Spectrum, Weaponmaster, Arsenal, Prism and Elementalist are
//     patterns over the cards a turn put on the table, and four of the five were rungs of the hand
//     ladder until 2026-09-05 — cut because the ladder could not *price* them, not because they
//     could not be matched. This is where they went.
//   - A **count** accumulates across every run the player has ever played, so it cannot be a
//     predicate over anything the process is holding. It is a tally on the profile.
//   - A **moment** is a named thing the code already reaches — a duel won, a floor climbed, a card
//     altered. Short, closed, and the only family that costs a line of Go each.
//
// **The tutorial's model deliberately does not transfer.** `internal/tutorial` publishes facts once
// a frame and every condition is a predicate over what is true *now*, which is right for a lesson
// and cannot express "three hundred times". So moments and turns are pushed at this package rather
// than polled by it, and the silent-failure hazard that shape carries is answered the other way:
// every moment name a record may write is a constant in this file, so a record naming one the code
// never pushes is refused at load instead of quietly never firing.
//
// **It imports `data` and `combat` and nothing else of ours**, which puts it beside `internal/decks`
// in the graph and keeps it free of Ebitengine — so the whole catalogue is walked in a test rather
// than by playing to the end of it.
//
// **It never touches the profile and never writes to disk.** It is handed what has happened and
// hands back keys; `internal/screens/achieve.go` is what turns a key into an award, an unlock and a
// toast, and `internal/screens/save.go` remains the only thing in the game that writes a file.
package achieve
