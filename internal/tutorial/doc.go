// Package tutorial is the teaching run's state machine: which step is up, what it points at,
// and what has to happen before it moves on.
//
// **It knows nothing about drawing and imports no Ebitengine**, which is the same rule
// `internal/combat` and `internal/decks` hold and it is here for the same payoff: the script can
// be walked end to end in a test, with no window and no duel, and a step whose condition can
// never fire is caught by `go test` rather than by a player stuck on realm one.
//
// # The three vocabularies, and why all of them are closed
//
// A step names an Anchor (what to point at), a Condition (what advances it) and whether it Gates
// (whether anything outside the anchor still accepts a click). Anchors and conditions are both
// enums parsed from strings at load, and a word the file invents is refused rather than ignored.
//
// That is the rule `data/SKILL.md` states as "do not grow a rules vocabulary in JSON ahead of the
// rules", and the failure it prevents here is specific: a misspelled anchor draws a spotlight
// around the empty rectangle at the origin, and a misspelled condition produces a step nothing
// can satisfy. Both look like a hung tutorial and neither looks like a typo.
//
// # Facts, rather than events
//
// A step advances when something happens in the game. The obvious way to do that is an event call
// at every site where something can happen — `tut.Did("duel-pressed")` in the DUEL! handler, one
// in the essence handler, one per screen — and it is the wrong way, because the failure of a
// forgotten call is a tutorial that silently stops advancing.
//
// So the traffic goes the other way. Each scene publishes a [Facts] once a frame, describing what
// is true right now, and a condition is a predicate over that. A scene that forgets to publish
// reports the zero value, which fails visibly and immediately rather than at one step in twelve.
// It also means the whole script can be driven in a test by writing structs.
//
// # What is deliberately not here
//
// **No rectangles.** An anchor is a name; `internal/screens` owns the geometry, because a
// rectangle is a fact about a layout and this package must stay window-free.
// `TestEveryAnchorHasARectangle` over there is what stops the two drifting apart — the same
// tripwire an `EventKind` has with its choreography entry.
//
// **No persistence.** Whether a given player has seen the tutorial is a profile question, and the
// answer lives in `internal/profile` rather than here — this package walks a script and has no
// opinion about who has already walked it. `screens.tutorialForThisRun` reads the profile and calls
// `session.Teach`; the combat screen's overlay marks it seen when the script ends or is skipped. A
// scenario carrying `"Teach": true` forces it regardless, which is the only way to see it twice.
//
// # Where it lives, and when it fires
//
// `data/tutorial.json` is the script and `internal/screens/tutorial.go` is Bob's bubble, the red square
// and the leader line to it. **It ships**: unlike trace, idle, the demo and the scenario fixture, a
// tutorial is a feature the player is meant to meet, so there is no build tag.
//
// **It fires on its own**, off the profile: a player `profile.json` has not recorded as taught gets
// taught, on the first fight of a fresh run. `screens.tutorialForThisRun` is the whole trigger, and it
// declines for a resumed run and for a scenario — a lesson that opens by describing the hand you are
// holding cannot begin halfway through the realms. **A launch on a clean machine therefore opens into
// the tutorial.** `"Teach": true` in a scenario forces it whatever the profile says; `DUELLO_PROFILE`
// pointed at an empty directory makes any launch a new player's.
//
// # The screen's half of it
//
//   - **A step waiting for NEXT holds the round where it is.** `Run.HoldsRound` is the predicate and
//     `advancePlayback` is the one reader. It exists for the shield step, which lands *inside* a playing
//     round: a round has three acts (the duelist swings, the shields break what they can reach, the
//     creature swings with what is left) and the middle one needs a beat of its own. **Only a NEXT step
//     holds**, which is what stops a step waiting on an outcome from stopping the thing it is waiting
//     for; `TestOnlyANextStepHoldsTheRound` is the tripwire. It changes pacing and cannot change an
//     outcome.
//   - **A break lives inside its own round.** `seatEnemyCards` drops the marks, because the opponent's
//     row is re-planned the moment a round ends, so a mark left standing cracks whichever card the
//     planner has just put in that seat. Anything wanting to point at a break has to do it during the
//     round, which is why the step above holds one.
//   - **An anchor names what the step is *asking for*, not what it is about.** `matching-cards` and
//     `matching-cards-left` are the same set minus what is already queued, and they are two because the
//     steps using them say different things: "take the other three" asks, and "one of those four is a
//     Brace" describes. One anchor for both lights the card already taken — and since the anchor is the
//     click gate, that card becomes the one thing the step invites you to click, which undoes the step
//     before it. `TestTheStepAsksOnlyForTheCardsStillToTake` is the tripwire. **The red comes off each
//     card as it is taken**, so the row says how much is left without a counter.
//   - **A card is tinted, a control is framed.** An anchor naming cards gets the scrim and no rectangle:
//     the cards wear `cards.MarkHighlit`. A frame outside a card is a thing *near* the card where a
//     tinted card is the card answering, and round a set of cards a frame is a lot of loose rectangles.
//     `Anchor.NamesCards` is the closed table saying which; `ui.MarksFor` reads the same
//     `gs.InputFocus` list the spotlight is handed, so lit and clickable stay one set by construction.
//   - **The lit square and the one legal click are the same rectangle**, computed once. A lit hole the
//     player cannot click, or a clickable region that is not lit, would each be worse than no tutorial.
//   - **An anchor may name several rectangles, and for a *set* of cards it must.** `tutorialRects` and
//     `state.InputFocus` are both lists because `matching-cards` and `shattered-cards` point at cards
//     that need not be adjacent, and the bounding box round them is the set *plus whatever is between
//     two of them*. **Cards sharing a *concept* land together whichever sort key leads; cards sharing an
//     *element* do not** — and the lesson matches on element, so under the default cost-led sort the
//     taught four can sit either side of a card worth 2 AP the taught set needs. Lit and clickable, it
//     can be queued, and then the fourth taught card cannot be paid for.
//     `TestTheMatchingCardsGateLightsOnlyTheTaughtCards` walks every seat of the real dealt hand through
//     `InputAllowed`. The spotlight scrims the gaps between holes, so lit and clickable stay one area.
//   - **The tutorial runs on the real deck, and `matching-cards` is what pays for that.** A real hand of
//     eight against a five-card cap and a six-point budget leaves cards behind by the rules of the game,
//     so the lesson cannot wait on an emptied hand. The anchor is the largest matching set in the hand
//     and `matching-queued` is its condition; because the lock leaves only those cards clickable, the
//     hand the player builds is the hand Bob just described. **It is the one anchor computed from the
//     cards rather than from a layout** — `CombatScene.matchingCards` is the single answer both the
//     square and the condition read.
//   - **Which axis a set is counted on is authored, not assumed.** The script's `Match` is `concept`,
//     `form` or `element`, and a script pointing at a matching set without naming one is **refused at
//     load**. The lesson matches on `element`.
//   - **The ledger step is the one anchor naming a control the frame owns.** `state.LedgerOpens` is a
//     tally bumped by `internal/game` when the panel opens, published as a fact and read by
//     `ledger-opened` against a baseline — the same trick `round-done` uses, because the account is
//     reachable from every screen and an opening from three steps ago is not this step's. **It advances
//     one frame late on purpose**: the panel takes the whole frame and the scene beneath it is not
//     updated, so the step gives way when the player closes the account rather than while it covers Bob.
//   - **`gs.InputGated` / `gs.InputFocus` is the shield**, and it gates on the *cursor* rather than per
//     widget — one predicate in `systems.UpdateButton` plus the handful of places in `internal/screens`
//     that read the mouse directly. A per-widget rule is a list a new widget is missing from.
//     **`internal/game` clears the gate every tick and the tutorial re-asserts it**, exactly as
//     `state.ModalOpen` works, so a screen left mid-step cannot leave the session unclickable.
//   - **It is deliberately not a `ui.ModalToggle`.** Every other dialog takes one footprint and scrims
//     the whole screen; a thing whose job is to point at what is underneath cannot be the thing covering
//     it. That is a second dialog shape, decided on purpose.
//
// **The machinery refuses the mistakes it can detect** — an ungated action step, a click with nothing
// named to click, a lock disagreeing with its condition. **What it cannot check is whether an anchor
// shows the player how to satisfy the step's condition**, and that is the mistake to look for: a step
// pointing at the shop shelf while waiting for the player to press *Leave* reads as a lock-up. Read
// each new step against its own condition.
//
// # The taught run, and the tests that hold its promise
//
// The script carries the run it needs — `Seed`, `Enemy` and `Match` — because a promise and the thing
// that makes it true belong in one file.
//
// **The taught fight is two rounds, and the shield is why.** Run code `0019QS` deals `Jab Brace Thrust
// Bash`, all arcane, for exactly 6 AP — an Elemental Four of a Kind that wounds the GiantBat without
// killing it. **One of the four is a Brace**, which teaches what a hand of pure attacks cannot: a
// defense carries an element and joins a hand like anything else, bringing no damage with it. Because
// it brings none, the creature lives, takes its turn — Swoop, Drain, Nip — and **the Brace's one shield
// eats the Drain whole while the other two land**. A creature that dies to the player's turn never
// swings, so a lesson about shields cannot be taught in a round that kills. The player then reads the
// ledger and finishes it. **The Drain is the bat's one big card**, which is what the heaviest-hit rule
// makes visible, and the step that explains it has a broken card on the table to point at. The other
// four cards dealt are an arcane, an earth, a fire and an ice, so there is no competing set, and the
// first card dealt is one of the four — which the opening step needs, since it queues `first-card`.
//
// **Three tests hold the promise, and they check each other.**
//
//   - `TestTheTutorialsBlowWoundsTheTutorialsEnemyWithoutKillingIt` in `internal/combat` proves the
//     rules resolve that turn to a wound. **It is two-sided**, failing if the hits start killing, if
//     they leave more than half the creature standing, or if the taught set stops holding exactly one
//     shield. It is the one to keep: four files tuned for their own reasons can break the promise
//     silently in either direction — the taught cards' `Amount`, the ladder's multiplier, the duelist's
//     `DMG`, the bat's `HP`.
//   - `TestTheTutorialsSeedDealsTheHandTheLessonDescribes` in `internal/screens` proves the seed deals
//     it — the set's size, that it is the only one that size, that the first card belongs to it, that it
//     is affordable, that it does *not* kill, and that its four cards are the four the combat test writes
//     out by hand.
//   - `TestTheTutorialsShieldEatsTheCreaturesHeaviestBlow` in `internal/screens` plans the bat's turn
//     exactly as the screen does, resolves the whole round, and checks that one hit is blocked, that it
//     is the heaviest, that the heaviest is the *only* card that size, and that the step naming the card
//     names the right one.
//
// **If any goes red the answer is a new seed, not a weaker check.** `TestFindATutorialSeed`, skipped
// unless `SEEDSEARCH=1` is set, is the search: it is a test rather than a tool because the shop
// internals it deals from are unexported, and `tools/seeds` tallies concepts where the tutorial matches
// on element. **It proposes and asserts nothing** — take a candidate, pin it, and let the tests above
// confirm it. **Prefer a marked candidate**, which keeps the cards the steps name; a seed dealing a
// different four means re-authoring the lesson. **Expect to re-run it whenever `relics.json` gains,
// loses or renames a record**: the shelf is a weighted draw over the catalog's sorted keys, so any of
// those reshuffles what the taught seed lands on, and the shop step is the one part of the lesson a
// catalog edit can break silently. Pinning the taught shop is the fix to argue for if that keeps
// happening.
package tutorial
