// Package journal is what the player chose, in order, on its way to the disk.
//
// **It is the answer to "how do I get back to this?"** A run seed rebuilds the tower, the motifs,
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
// keeps and moved with them by ASCEND_DUEL_PROFILE. **Starting a run truncates it.** A run that
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
package journal
