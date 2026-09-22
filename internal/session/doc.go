// Package session holds what belongs to a *run* rather than to a fight or a screen.
//
// **It is the hole every run-level feature has been blocked on.** Relics cannot be bought, the
// deck cannot be altered and vitae cannot be spent for one reason: nothing survives a fight.
// `CombatScene` is rebuilt on every entry — `Init` is how the next fight starts — so anything
// kept there is thrown away between rooms. This is where a run keeps its things.
//
// **No Ebitengine, ever**, and no screen state. What lands here is what more than one scene
// needs and what has to outlive a fight: the deck today, the relics and the purse next. If a
// field is only read by one screen, it belongs on that screen.
//
// The package sits below `screens` and above `combat`, and `state.GlobalState` carries a
// pointer to it — which is why `state` transitively imports `combat` as of 2026-08-17. That
// reverses a line in CLAUDE.md written to stop *screen* state leaking into global state; a run
// is not screen state, and it is exactly what `ActiveScreen` and `NewScreen` already sit beside.
//
// # What is in a run
//
// **This is a grouping rather than a list of every file** — each file's own header says what it
// holds, and an enumeration here is a second list that goes quietly out of date. The last one
// named seven files when there were twenty-four.
//
// The run itself:
//
//   - session.go — the deck, the purse, the room counter. The deck is unexported and every change
//     goes through a method, so an index handed to a screen stays meaningful for as long as the
//     caller holds it.
//   - deck.go — the list a run opens with, read out of data/duelist_cards.json.
//   - flow.go — where the run is in its loop, and the one place that moves it on.
//   - climb.go — who stands in each room, over internal/pyramid. This is what the room choice will
//     write to.
//   - clock.go, life.go — the round limit, and what the climb does to the body between fights.
//
// What a choice between two fights can change — holdings.go is the collective noun:
//
//   - relic.go — the relics being worn, in worn order, and the relic moments that fire between
//     fights rather than during one.
//   - essence.go — the deck alterations offered after a fight.
//   - rune.go, echo.go — the consumables carried into a fight and spent between its turns. The
//     essence's grammar plus a target *count*, aimed at card identities rather than deck
//     positions, and the one alteration that can happen while a duel is going on.
//   - stone.go, potion.go — the run's own opinion about what a hand is worth, and about the body.
//   - good.go, shop.go, spoils.go — what is sold, what it costs, and what winning pays.
//
// What the run says about itself:
//
//   - record.go, ledger.go, export.go — what happened as flat records, the account they make up,
//     and writing it out to a file.
//   - play.go, summary.go — how often each rung has actually been built, and what a run came to.
//   - save.go — a run written down and read back, as plain data internal/profile can put on disk.
//   - tutorial.go, jump.go — the teaching run parked on the run it teaches, and putting a run
//     somewhere it did not play its way to.
//
// # Four of the ten relic moments are answered here
//
// deck-built in FightDeck, fight-start in Equip, and fight-won in WonFight — which is also where
// vitae propagates and where a growing relic takes its step. The fourth, card-drawn, is answered by
// DrawnAs and *fired* by the combat screen, once per card as it deals one: the flip belongs to a run
// and the draw pile belongs to a fight. The order matters and MECHANICS.md
// states it: propagation is interest on what the run walked out of the fight holding, not on what
// the prize card is about to pay it.
//
// # It is deliberately not persisted
//
// Two runs from the same seed may hold different decks, because a deck edit is a choice rather
// than something derived from the seed. The replay story is a seed plus a choice log — see the
// randomness skill.
package session
