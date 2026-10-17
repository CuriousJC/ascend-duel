package session

// card_trait.go: data/card_traits.json as the run reads it — what a card trait is called and what it
// looks like, for the Compendium and the review sheet, and how often an action bundle's card comes
// with it. **What a trait does is `internal/combat`'s**, which reads the same file at init.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// CardTraitInfo is one card trait as a screen or a sheet reads it.
type CardTraitInfo struct {
	Trait   combat.CardTrait
	Record  data.CardTraitData
	Percent int
}

// cardTraits is the file, resolved against the rules' registry, in file order.
var cardTraits = func() []CardTraitInfo {
	var out []CardTraitInfo
	for _, r := range data.LoadCardTraits() {
		t, ok := combat.ParseCardTrait(r.CardTrait)
		if !ok {
			panic(fmt.Sprintf("card_traits.json: %q is not a card trait the rules registered", r.CardTrait))
		}
		out = append(out, CardTraitInfo{Trait: t, Record: r, Percent: r.Percent})
	}
	return out
}()

// CardTraitCatalog is every card trait, in file order — the order a bundled card rolls them.
func CardTraitCatalog() []CardTraitInfo { return cardTraits }

// TakeActionCard puts a card taken out of an action bundle into the run's deck, with a fresh identity,
// and counts it — the card by its concept, and each trait it carries.
func (s *Session) TakeActionCard(c combat.Card) {
	s.Add(c)
	s.count(StatCardTaken+combat.ConceptOf(c.Concept).Key, 1)
	for _, t := range combat.CardTraits() {
		if c.HasTrait(t) {
			s.count(StatCardTrait+t.String(), 1)
		}
	}
}

// ActionPool is every card an action bundle may hold: one of each card the starting deck is built
// from, every concept in every color it ships in, in file order. **One of each, not the deck's
// copies**, so a concept the deck holds three of is no likelier in a bundle than one it holds once.
func ActionPool() []combat.Card {
	out := make([]combat.Card, 0, len(startingDeck))
	for _, e := range startingDeck {
		out = append(out, e.card)
	}
	return out
}
