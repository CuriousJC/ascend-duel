// Package session holds what belongs to a *run* rather than to a fight or a screen.
//
// **It is the hole every run-level feature has been blocked on.** Rings cannot be bought, the
// deck cannot be altered and vitae cannot be spent for one reason: nothing survives a fight.
// `CombatScene` is rebuilt on every entry — `Init` is how the next fight starts — so anything
// kept there is thrown away between rooms. This is where a run keeps its things.
//
// **No Ebitengine, ever**, and no screen state. What lands here is what more than one scene
// needs and what has to outlive a fight: the deck today, the rings and the purse next. If a
// field is only read by one screen, it belongs on that screen.
//
// The package sits below `screens` and above `combat`, and `state.GlobalState` carries a
// pointer to it — which is why `state` transitively imports `combat` as of 2026-08-17. That
// reverses a line in CLAUDE.md written to stop *screen* state leaking into global state; a run
// is not screen state, and it is exactly what `ActiveScreen` and `NewScreen` already sit beside.
//
// # What is in a run
//
//   - deck.go — the list a run opens with, read out of data/duelist_cards.json.
//   - session.go — the deck, the purse, the room counter. The deck is unexported and every change
//     goes through a method, so an index handed to a screen stays meaningful for as long as the
//     caller holds it.
//   - climb.go — who stands in each room, over internal/pyramid. This is what the room choice will
//     write to.
//   - flow.go — where the run is in its loop, and the one place that moves it on.
//   - ring.go — the rings being worn, in worn order, and the ring moments that fire between fights
//     rather than during one.
//   - worm.go — the deck alterations offered after a fight.
//
// # Three of the seven ring moments fire here
//
// deck-built in FightDeck, fight-start in Equip, and fight-won in WonFight — which is also where
// vitae propagates and where a growing ring takes its step. The order matters and MECHANICS.md
// states it: propagation is interest on what the run walked out of the fight holding, not on what
// the prize card is about to pay it.
//
// # It is deliberately not persisted
//
// Two runs from the same seed may hold different decks, because a deck edit is a choice rather
// than something derived from the seed. The replay story is a seed plus a choice log — see the
// randomness skill.
package session
