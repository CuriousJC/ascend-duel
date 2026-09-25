// Package combat is the duel rules engine. It imports nothing from Ebitengine and
// knows nothing about drawing: ResolveRound takes two duelists and the actions they
// have queued for one round and returns an ordered event log plus the state both
// sides end the round in. The combat screen replays that log; it never computes an
// outcome itself. That split is what makes the rules unit-testable and what would
// let a headless balance sim run thousands of duels with no window.
//
// A duel is a sequence of rounds. Each round both sides spend an action-point budget
// on a set of actions, and those resolve **in phases**: everything side A queued, in
// category order, and then everything side B queued. Control returns to the player to
// re-plan. Nothing here runs a duel to completion — that is the screen's loop, and the
// point is that the player re-evaluates between rounds.
//
// Phase resolution replaced alternation on 2026-08-06, on the grounds that interleaving is
// not graspable by players. See MECHANICS.md. Two consequences run through this file:
//
//   - **Initiative is gone.** With one contiguous turn per side there is no exchange for a
//     faster action to lead, so the whole lever was reporting a distinction the resolver no
//     longer made. See the TODO in TODO.md before bringing it back.
//   - **Shields cover the opponent's next turn, not the rest of the round.** Side B acts last, so
//     a defense that expired at the round boundary would never protect B from anything. Shields
//     last until the start of their owner's own next turn instead, which is the one rule that is
//     symmetric under a resolution order that is not. The percentage guard this was first written
//     about is gone — see VerbShield.
//
// # What lives in which file
//
// The package was one 1500-line file until 2026-08-21. It is now split by concern, and the split
// is the map:
//
//   - card.go — what a card is: a concept plus an element, two ints and comparable. Its Category
//     says when it resolves, its Form says what kind of thing it is, and nothing else about it is
//     stored here — a name, a cost and a picture are all the concept's.
//
//   - concept.go — a card's rules, as data. A Concept is a label, a verb, an amount, a cost, a
//     target and a form, registered at load and named by a ConceptID. It replaced a closed enum of
//     fourteen constants with cost, damage and category as switch statements over it, which held
//     twelve player cards and could not hold the several hundred that per-enemy decks produce. IDs
//     are registration-ordered and must never be serialized.
//
//   - element.go — the five colors plus Basic, which is the absence of one rather than a sixth.
//
//   - status.go — a status is a record in statuses.json rather than an element: a key, a name, a
//     badge, one of four closed effect kinds, an amount and a duration. They share one lifecycle
//     and nothing stacks; a second hit resets the clock. A status only happens if a worn relic says
//     so, which is what left the elemental relics something to be.
//
//   - duelist.go — who is fighting, and what a round spends: action points, and the shields a
//     defend card raised. A Duelist is a value; every rule takes one and returns a new one.
//
//   - rider.go — the one alteration a rune may write onto a card, and the vocabulary the rules
//     have to read to honour it. A card carries at most one, so a second replaces the first.
//
//   - stone.go — the arithmetic of a bought rung, read *through* the hand table so the ladder
//     every tool and test sees is still the shipped one.
//
//   - luck.go — the gamble a gold or silver card takes on every play, off its own stream.
//
//   - clock.go — the round limit: a duelist still standing at the end of the last round dies,
//     through the same door a killing blow uses.
//
//   - event.go — the vocabulary a resolved round hands back, and the play order both the resolver
//     and the screen read. This is the whole contract between the rules and the pictures.
//
//   - hand.go and hand_table.go — a hand is a damage multiplier and nothing else, matched on three
//     axes: concept, form and element. Exactly one applies, winning on its multiplier, and a tie
//     goes to the narrowest axis. The multiplier multiplies the cards; there is no third term.
//     Adding a rung is one entry in data/hands.json.
//
//   - relic.go — the relic grammar, with registration refusing a verb used at the wrong moment or
//     a status no file holds. Worn order is a rule, because effects compound left to right. An
//     enemy wears no relics, so an enemy's colors are inert by construction.
//
//     **The three vocabularies are deliberately not counted here.** They grow by authoring —
//     Moments(), RelicVerbs() and RelicCondition's fields are the lists — and a figure written
//     into a doc comment goes stale silently, which is exactly what happened to the three that
//     used to be on this line.
//
//   - planner.go — what an enemy does with the hand it was dealt.
//
//   - combat.go — ResolveRound and the phase resolvers.
//
// # Rules that are easy to break without noticing
//
//   - No Ebitengine import, ever. That is what makes this package testable without a window, and
//     the property — not the package name — is the rule. internal/music and internal/cards are
//     tested for the same reason.
//   - It must never import internal/seeds either. The rules take an injected *rand.Rand and stay
//     ignorant of where it came from.
//   - Every card lands its own hit. A hand-forming duelist's turn reads one hand off its cards and
//     multiplies every hit by it (hit.go); an enemy's attacks resolve one at a
//     time at face damage, with no hand. That is Duelist.SoloAttacks, and it is a flag on the
//     duelist rather than a rule about SideB — the engine has no idea which side is a person and
//     must not learn, because the balance tool plays both sides headlessly.
//   - Nothing reduces a hit to zero by arithmetic. A shield eats a whole hit or none of it.
//   - The rules cannot draw a card — there is no deck in this package, and nothing in the game
//     asks for one any more.
//   - A shield eats one whole incoming hit — its own element's heaviest first, then the heaviest
//     of what is left — whichever kind of attacker is swinging; see shieldedHits.
//   - Never change these rules to make a screen look right. If a screen contradicts the engine,
//     say so and let the owner decide which one is wrong. That is a game-design call and it
//     ripples into the tests and the balance.
package combat
