// Package journal is what the player chose, in order, on its way to the disk.
//
// **It is the answer to "how do I get back to this?"** A run seed rebuilds the journey, the motifs,
// the elements and every shuffle, and it is still not enough: the deck changes with what the player
// takes and spends, and the hands follow from the deck. Same seed and different choices is a
// different fight two. This file is what closes that gap.
//
// # Inputs, never outputs
//
// **A journal records choices and the ledger records what became of them**, and the two are not
// interchangeable. A record in internal/session is what combat.ResolveRound produced; it cannot
// re-drive the resolver, and a replay fed from one would be comparing the engine to a recording of
// itself. What can put a run back where it was is the clicks, which is all that is written here.
//
// So nothing in this package knows what a card is worth, what a blow came to or who won. It knows
// which card was picked out of which seat, which button was pressed and which relic was bought.
//
// **Legible rather than exhaustive**, because replay is a person at a keyboard: a record names a
// relic key, a card's label and a seat, not a drag path and not a pointer position.
//
// # One file, and the run it belongs to
//
// `journal.jsonl` in the profile's own directory, beside the two files internal/profile already
// keeps and moved with them by DUELLO_PROFILE. **Starting a run truncates it.** A run that
// ended without going wrong is a run nobody is going to ask about, so keeping it would be a
// directory growing for no reader — and one fixed name means nothing has to sweep up after it.
//
// **A crash takes a copy under its own name**, which is what makes the one-file rule safe: see
// internal/crashlog, which writes `crash-<utc>-<code>.jsonl` beside the report it is already
// writing. Without that the one run worth retracing is the one the next launch overwrites.
//
// **A resumed run appends rather than truncating.** The journal on disk belongs to that same
// climb, and a Continue that cleared it would throw away everything before the last launch.
//
// # The envelope
//
// One record per line, appended as it happens:
//
//	{"t":12345,"kind":"select","card":31,"label":"Jab","seat":3}
//
// Four rules hold it:
//
//   - **`t` is the simulation tick**, state.GlobalState.Count, never a wall clock. The rule
//     internal/trace is under: a tick lines up with a replay of the same seed and a clock does not.
//   - **`kind` is one word from a closed vocabulary**, append-only, and a name rather than an
//     ordinal — the rule every saved file in this game is under, and this one outlives its build
//     harder than a save does.
//   - **Fields are flat, named and few.** No nested objects, and one array: the cards a choice was
//     aimed at.
//   - **The first line is the header**: the schema, the build, the run code, the install id, the
//     platform and when it was opened.
//
// **Appended a line at a time rather than written whole.** A document written at a phase boundary
// has nothing to say about the frame that went wrong, and the whole audience for this file is
// somebody reading it after the process that wrote it died. See profile.Store.AppendLine.
//
// # What it may never do
//
// **Nothing here may ever be fatal**, and a journal that cannot be written is told to the player
// once rather than once per click — see internal/crashlog's two verbs. A read-only directory is
// "this run is not being retraced", which is the rule internal/profile and the audio device are
// already under.
//
// **Capture may never change an outcome.** It joins playback speed, the debug flags,
// internal/trace, internal/idle and the scripted demo: combat.ResolveRound never sees any of it.
//
// **It writes through profile.Store rather than through `os`**, on the storage-boundary rule:
// nothing above the backend may know a save is a file, and a second thing reaching for `filepath`
// is a second thing to port.
//
// **internal/combat may never import it**, for internal/trace's reason: the rules package stays
// free of everything, which is what makes it testable without a window.
//
// # How it is wired
//
//   - **A card is named by identity**, `combat.Card.ID`, for the reason a rune's targets are: three piles
//     hold copies of the same cards, so a position names a different card a moment later.
//   - **A rune's gamble is not written down.** The roll comes off the run seed, so a replay reaching that
//     line with the same choices behind it rolls the same thing.
//   - **A resumed run's header says `resumed`**, so two headers in one file is one journey played across
//     two launches.
//   - **A click has its line where its function is; a screen and a phase are diffed once a frame.** The
//     first is where the choice actually is. The second two are reached from a dozen places — a button, a
//     run advancing, a crash, a scenario opening the game halfway through the realms — so a call beside
//     each is a list the next one gets left off. `game.journalWatch` is the diff, on `screens.RunWatch`'s
//     argument.
//   - **`CombatScene.choices` is the one stored handle**, taken at `Init`. Every other screen writes its
//     lines from a method already holding a `gs`; the hand row's click handlers are reached from a button
//     or a drag with nothing but the scene in hand.
//   - **Every method is safe on a nil receiver**, which is what lets a scene write a line without asking
//     whether there is a journal — a test scene and a review tool both run without one.
//   - **A journal that cannot be written gives up and is a `Tell` once**, not once per click: a box per
//     click is a queue the player has to fight their way out of to keep playing. **The wording lives in
//     `main`** rather than here, because `crashlog` has to be able to name the journal file and only one
//     of the two may point at the other.
//   - **The record is compact rather than indented**, unlike every other write in `internal/profile`:
//     one record per line is what makes the file appendable without parsing what is above it.
package journal
