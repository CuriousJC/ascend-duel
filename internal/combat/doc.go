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
//   - **Defenses cover the opponent's next turn, not the rest of the round.** Side B acts
//     last, so a defense that expired at the round boundary would never protect B from
//     anything. They expire at the start of their owner's own next turn instead, which is
//     the one rule that is symmetric under a resolution order that is not.
//
// # What lives in which file
//
// The package was one 1500-line file until 2026-08-21. It is now split by concern, and the split
// is the map:
//
//   - card.go — what a card is: a concept plus an element, two ints and comparable. Its Category
//     says when it resolves, its Form says what kind of thing it is, and nothing else about it is
//     stored here — a name, a cost and a picture are all the concept's.
//   - concept.go — a card's rules, as data. A Concept is a label, a verb, an amount, a cost, a
//     target and a form, registered at load and named by a ConceptID. It replaced a closed enum of
//     fourteen constants with cost, damage and category as switch statements over it, which held
//     twelve player cards and could not hold the several hundred that per-enemy decks produce. IDs
//     are registration-ordered and must never be serialized.
//   - element.go — the four colours plus Basic, which is the absence of one rather than a fifth.
//   - status.go — a status is a record in statuses.json rather than an element: a key, a name, a
//     badge, one of four closed effect kinds, an amount and a duration. They share one lifecycle
//     and nothing stacks; a second hit resets the clock. A status only happens if a worn ring says
//     so, which is what left the elemental rings something to be.
//   - duelist.go — who is fighting, and the two things spent during a round: action points and
//     raised defences. A Duelist is a value; every rule takes one and returns a new one.
//   - event.go — the vocabulary a resolved round hands back, and the play order both the resolver
//     and the screen read. This is the whole contract between the rules and the pictures.
//   - hand.go and hand_table.go — a hand is a damage multiplier and nothing else, matched on three
//     axes: concept, form and element. Exactly one applies, winning on its multiplier, and a tie
//     goes to the narrowest axis. The multiplier multiplies the cards; there is no third term.
//     Adding a rung is one entry in data/hands.json.
//   - ring.go — the ring grammar: seven moments, three predicates, ten effect verbs, with
//     registration refusing a verb used at the wrong moment or a status no file holds. Worn order
//     is a rule, because effects compound left to right. An enemy wears no rings, so an enemy's
//     colours are inert by construction.
//   - planner.go — what an enemy does with the hand it was dealt.
//   - combat.go — ResolveRound and the phase resolvers.
//
// # Rules that are easy to break without noticing
//
//   - No Ebitengine import, ever. That is what makes this package testable without a window, and
//     the property — not the package name — is the rule. internal/music and internal/cards are
//     tested for the same reason.
//   - It must never import internal/seeds either. The rules take an injected *rand.Rand and stay
//     ignorant of where it came from.
//   - A turn resolves exactly one attack for the player, scored as a hand; an enemy's attacks
//     resolve one at a time, each landing its own face damage. That is Duelist.SoloAttacks, and it
//     is a flag on the duelist rather than a rule about SideB — the engine has no idea which side
//     is a person and must not learn, because the balance tool plays both sides headlessly.
//   - Nothing reduces a blow to zero. A turn lands one figure however many cards made it, so total
//     negation would be a whole opposing turn deleted by one card.
//   - The rules cannot draw a card — there is no deck in this package. Plan records a bonus draw
//     and emits an event; the screen honours it on the next refill.
//   - Never change these rules to make a screen look right. If a screen contradicts the engine,
//     say so and let the owner decide which one is wrong. That is a game-design call and it
//     ripples into the tests and the balance.
package combat
