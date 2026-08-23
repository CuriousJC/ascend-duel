// Package data is the game's static content: JSON files next to a small Go loader.
//
// It is the bottom of the dependency graph. It imports nothing but the standard library and must
// never import upward — that is what lets every layer above read it, and it is the rule that
// decides several things that would otherwise look arbitrary.
//
// # Who may read which file
//
// Who consumes a file decides who may read it, not whether it is data. internal/combat reads
// exactly hands.json, duelist_cards.json and statuses.json, because those are rules. A portrait
// key and a floor band are a screen's business, so enemies.json is not the engine's to open —
// which is why enemy cards are registered by internal/decks rather than handed over from here.
// internal/decks exists for that one case, and bosses.json rides the same route.
//
// # Failing loudly
//
// A bad record panics at package init, so a file the rules cannot resolve fails on launch rather
// than mid-duel. combat.RegisterConcept is the validation. A deck quietly missing four cards is a
// balance change nobody made on purpose, and a game that starts anyway is a game that hides it.
//
// # Order
//
// Never walk a loaded map where order decides an outcome — Go randomises map iteration
// deliberately. EnemyOrder and RingOrder are the sorted walks, and they exist so a run seeded from
// one number deals the same game twice. See the randomness skill.
package data
