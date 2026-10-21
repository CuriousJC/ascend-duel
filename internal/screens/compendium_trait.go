package screens

// **The Compendium's trait book: each trait worn by a real relic or card, drawn the way the game
// draws it.**
//
// A trait is a property of a wearing rather than a thing, so it has no picture of its own: in play a
// weightless relic and an effortless card ride aloft, an ephemeral relic is faded with its border
// broken, a mirrored card carries its echo, and an armored relic has a line of light running round its
// edge. The book shows
// exactly that, on a sample — a relic from the open shelf for a relic trait, a card from the starting
// deck for a card trait.
//
// **The sample changes from run to run and holds within one**, drawn off its own stream so opening the
// book rerolls nothing. What the picture says comes off the trait's own rules — whether it takes a
// slot, whether it lasts the fight, whether the card is counted or lands twice — never off its name,
// so a new trait arrives drawn.

import (
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// traitsDialog is every trait: the relic traits, then the card traits, each in its file's order. A
// relic trait is had once a relic has been worn with it — bought so, or a cantrip's relic cast — and a
// card trait once a card carrying it has been taken out of an action bundle.
//
// **The tooltip is the trait's own sentence and its odds**, so the book is also where a player reads
// how rare each one is.
func traitsDialog(gs *state.GlobalState) *compendiumDialog {
	// **The card pictures are read from a stash a drawing door fills**, and a book opened before
	// anything has drawn a card would build its samples with no picture.
	ui.UseImages(gs)
	all := everything(gs)
	rng := rand.New(rand.NewSource(seeds.For(gs.RunSeed, seeds.CompendiumSample)))
	relics := sampleRelics(gs)
	pool := session.ActionPool()
	pickRelic := distinct(rng, len(relics))
	pickCard := distinct(rng, len(pool))

	var out []compendiumEntry
	for _, t := range session.ShelfTraits() {
		r := t.Record
		var spec cards.Spec
		if len(relics) > 0 {
			spec = ui.RelicSpec(gs, relics[pickRelic()], "", true, false)
		}
		e := entry(spec, bleedStyle(true),
			all.Total(session.StatRelicTrait+r.RelicTrait), "WORN", r.Name,
			[]string{"RELIC TRAIT", r.Text, traitOdds(t.Percent, "relics on the shelf")})
		e.aloft = r.Slot != nil && !*r.Slot
		e.ephemeral = r.LastsFight()
		e.edge = t.Trait.AddsRules()
		out = append(out, e)
	}
	for _, t := range session.CardTraitCatalog() {
		r := t.Record
		var spec cards.Spec
		aloft := false
		if len(pool) > 0 {
			c := pool[pickCard()]
			c.Traits |= t.Trait
			spec = ui.CardSpec(c, ui.HeldBy(combat.Duelist{}, c), true, false)
			aloft = !c.Counted()
		}
		e := entry(spec, cards.Hand, all.Total(session.StatCardTrait+r.CardTrait), "TAKEN", r.Name,
			[]string{"CARD TRAIT", r.Text, traitOdds(t.Percent, "cards in an action bundle")})
		e.aloft = aloft
		out = append(out, e)
	}
	return catalogDialog(gs, "TRAITS", out)
}

// sampleRelics is what a relic trait may be shown on: the relics on every shelf from the first run,
// in file order. **Never a locked one**, so the book gives away nothing an achievement is keeping.
func sampleRelics(gs *state.GlobalState) []data.RelicData {
	var out []data.RelicData
	for _, key := range data.RelicFileOrder() {
		if r, ok := gs.Relics[key]; ok && r.Unlock == "" {
			out = append(out, r)
		}
	}
	return out
}

// distinct draws indices out of n without repeating one until all n have been drawn, so two traits
// are never shown on the same relic or the same card — the same sample twice reads as one picture
// standing for two traits.
func distinct(rng *rand.Rand, n int) func() int {
	var order []int
	return func() int {
		if len(order) == 0 {
			order = rng.Perm(n)
		}
		i := order[0]
		order = order[1:]
		return i
	}
}
