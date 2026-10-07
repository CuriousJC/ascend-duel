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
//     `advancePlayback` is the one reader. It is for a step that lands *inside* a playing round, on
//     `shield-broke` — no step in the shipped script does — since a round has three acts (the duelist swings, the shields break what they can reach, the
//     creature swings with what is left) and the middle one needs a beat of its own. **Only a NEXT step
//     holds**, which is what stops a step waiting on an outcome from stopping the thing it is waiting
//     for; `TestOnlyANextStepHoldsTheRound` is the tripwire. It changes pacing and cannot change an
//     outcome.
//   - **A break lives inside its own round.** `seatEnemyCards` drops the marks, because the opponent's
//     row is re-planned the moment a round ends, so a mark left standing cracks whichever card the
//     planner has just put in that seat. Anything wanting to point at a break has to do it during the
//     round, which is why a step on `shield-broke` holds one.
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
//     *element* do not**, so a lesson matching on element can have a card it did not name sitting
//     between two it did. Lit and clickable, that card can be queued.
//     `TestTheMatchingCardsGateLightsOnlyTheTaughtCards` walks every seat of the real dealt hand through
//     `InputAllowed`. **The spotlight shades everything but the holes, on both axes** —
//     `screens.scrimAround` — so an anchor naming things in different corners, like `fight-frame`,
//     lights those things and nothing between them.
//   - **The tutorial runs on the real deck, and `matching-cards` is what pays for that.** A real hand of
//     eight against a five-card cap and a six-point budget leaves cards behind by the rules of the game,
//     so the lesson cannot wait on an emptied hand. The anchor is the largest matching set in the hand
//     and `matching-queued` is its condition; because the lock leaves only those cards clickable, the
//     hand the player builds is the hand Bob just described. **It is the one anchor computed from the
//     cards rather than from a layout** — `CombatScene.matchingCards` is the single answer both the
//     square and the condition read.
//   - **Which axis a set is counted on is authored, not assumed.** The script's `Match` is `concept`,
//     `form` or `element`, and a script pointing at a matching set without naming one is **refused at
//     load**. The lesson matches on `concept`.
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
// # The taught run, and the test that holds its promise
//
// The script carries the run it needs — `Seed`, `Enemy` and `Match` — because a promise and the thing
// that makes it true belong in one file. **The realm's element is not in the script**: it is the
// seed's, so the seed is chosen for a realm one in the element the steps name.
//
// **The taught fight is three rounds, one lesson each.**
//
//   - **Round one is a hand.** The opening deal holds exactly three Slices, none in the creature's
//     element, and no other set of three — a Card Three of a Kind, which is the set `matching-cards`
//     lights on the `concept` axis. They wound the creature without killing it.
//   - **Round two is a surge and a pair.** The opening deal also holds the Block in the creature's
//     element, and the step names two more cards beside it — `named-cards`, counted by
//     `cards-queued` — to make a Pair. The creature throws exactly two attacks, and both shields eat
//     a hit of their own element, banking two action points. Round two's hand also holds an attack in
//     the creature's element, which is what `element-attack` points at to say a hit of it fizzles.
//   - **Round three is the kill**, which some play in that hand makes on the surged budget. **Whether
//     it needs the surge is reported, not required** *(owner's call)*.
//
// **The creature is an inner-room record of the realm's own motif**, standing in the first room: an
// outer-room creature has too little life to survive the first two rounds.
//
// `TestTheTutorialsFightPlaysAsTaught` in `internal/screens` plays all three rounds headlessly — the
// screen's own deal and refill, the creature's planner, `ResolveRound`, round two as the script names
// it — and fails on any of the above.
// **If it goes red the answer is a new seed, not a weaker check.** `TestFindATaughtFight`, skipped unless
// `SEEDSEARCH=1` is set, is the search: it walks run codes for a realm one in the element, plays the
// fight against every creature of that realm's motif with a picture in it — round two as the Block
// alone, since the cards named beside it are a fact about one deal — and prints the ones that hold. It proposes and asserts nothing. **Re-run it after touching the player's deck, the hand ladder,
// a creature's deck or stats, or the growth curve** — every one of them moves the fight.
package tutorial
