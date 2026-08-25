package session

// The deck a run opens with.
//
// **It moved here from the combat screen on 2026-08-21.** A starting deck is a fact about a run,
// not about the screen that happens to deal from it first — and while it lived on that screen,
// `main` had to import `internal/screens` to start a run at all. The shop and the worms both edit
// this list; none of them should have to know which screen built it.
//
// **It is data, not a Go expression** *(2026-08-08)*: nine attack concepts x five colours, plus
// three plans in the same five = 60 cards, read out of data/duelist_cards.json. The deck's size is a
// consequence of a file the designer can read and edit.
//
// **The file holds more concepts than the deck holds cards** *(2026-08-24)*. The 0 AP and 4 AP rung
// of each attack form ship at `Copies: 0`, so they are registered with the rules — which is what
// lets a Shrink or a Grow worm step a card onto them — and expand to nothing here. A zero-copy
// record is a rung, not a card.

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// startingDeck is the deck the player opens a run with: **nine attack concepts x five colours,
// plus three plans in the same five = 60 cards**, built from `data/duelist_cards.json` rather than
// written out here.
//
// **48 until arcane landed on 2026-08-25.** Adding a colour adds a card to every concept at once,
// which is why one line in a JSON file moved the deck by a quarter — and why `tools/handodds` and
// `tools/seeds` both have to be re-run after it. See MECHANICS.md for what it did to the ladder.
//
// It became data on 2026-08-08, at the same time the concept grid was filled. The shape it
// replaced was a `concat` of `conceptDeck` calls — fine for six concepts, and a list nobody
// could count at a glance for twelve. What the JSON buys is that the deck's *size* is now a
// consequence of a file the designer can read and edit, rather than of a Go expression.
//
// **The rules moved with it on 2026-08-16.** Cost, damage, category and form used to live in
// `internal/combat` as switch statements, and this file declared a cost tier that was checked
// against them. A card carries its own rules now — `internal/combat` registers the concepts from
// this same file at init — so there is nothing left to cross-check and `CheckCostTiers` went with
// the duplication it guarded. What this function does is the other half: turning the concepts into
// a *pile*, which is a screen's business and not the rules'.
var startingDeck = buildStartingDeck()

// buildStartingDeck turns the data records into deck entries, in file order — which is grid
// order, which is the order the deck overlay sorts into anyway.
//
// **It panics on a bad record, and that is the right severity.** A concept the registry does not
// hold, or an element the rules do not know, would otherwise produce a deck quietly missing four
// cards — a balance change nobody made on purpose, and a game that starts anyway is a game that
// hides it. This runs at package init, so it fails on launch rather than mid-duel.
//
// A label the registry has not got means `internal/combat` refused the record when it registered
// the file, which it reports with its own reason; reaching here means the two loaders disagree
// about the same file, so the message says so.
func buildStartingDeck() []deckEntry {
	// Not named `cards`: that is the drawing package, imported above, and shadowing a
	// package name inside the one function that builds the deck is a trap.
	records := data.LoadDuelistCards()

	var out []deckEntry
	for _, c := range records {
		id, ok := combat.ConceptByKey(c.Label)
		if !ok {
			panic("duelist_cards.json: the rules did not register a card called " + c.Label)
		}
		for _, name := range c.Elements {
			e, ok := combat.ParseElement(name)
			if !ok {
				panic("duelist_cards.json: " + c.Label + " names unknown element " + name)
			}
			out = append(out, deckEntry{combat.Card{Concept: id, Element: e}, c.Copies})
		}
	}
	return out
}

// StartingDeck is the authored deck expanded to one entry per card — what a run opens with, and
// what `main` hands to `session.New`.
//
// Exported because a *run* starts outside this package now. Nothing here owns the deck any more:
// the piles are dealt from `GlobalState.Run` and a worm can thin or recolour it between fights,
// so this is the list a run begins from rather than the list it plays with.
func StartingDeck() []combat.Card {
	out := make([]combat.Card, 0, 48)
	for _, e := range startingDeck {
		for i := 0; i < e.count; i++ {
			out = append(out, e.card)
		}
	}
	return out
}

// deckEntry is one line of a deck list: a card and how many copies of it.
type deckEntry struct {
	card  combat.Card
	count int
}
