package session

// Rings: the catalogue, what the run is wearing, and the three moments that fire outside combat.
//
// **`rings.json` is parsed here for the reason the worms are** — a ring belongs to a *run*. Three of
// its seven moments are `deck-built`, `fight-start` and `fight-won`, none of which happen inside
// `internal/combat` at all, and the accumulator a growing ring carries has to survive a fight. This
// package is what survives one.
//
// **What crosses the edge is a rules type, never a record.** `data.RingData` holds an art key and a
// long-press line; `combat.RegisterRing` takes a key, a name and `[]combat.RingRule`. So the engine
// never reads a file it has no business in, and this file never grows an opinion about what a
// status is worth. Same division `decks.EnemyCards` draws for enemy cards.
//
// **Bad records panic at load**, like every other catalogue: an unknown moment, a verb used at the
// wrong moment, a predicate the rules cannot resolve, or a status key that is in no file.

import (
	"fmt"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// StartingRings is what a run opens wearing, in worn order.
//
// **Temporary, and it is a live test of the gate as much as of the statuses** — earth is left off on
// purpose, so an earth attack in a launched game is a plain attack with a brown border. A run will
// start bare and buy its rings once a shop exists; this constant is the seat that replaces.
//
// It lived in `internal/screens` until 2026-08-17, where it could not survive a fight.
var StartingRings = []string{"fire-ring", "frozen-ring", "thunder-ring"}

// registeredRings is every ring in the catalogue, registered with the rules at package init and
// indexed by record key.
//
// **Walked in sorted key order**, per the determinism rules: `LoadRings` hands back a map, and
// registering in map order would deal a different set of RingIDs every launch. Nothing may serialize
// one, but a tool printing them would still tell a different story each run.
var registeredRings = registerRings()

func registerRings() map[string]combat.RingID {
	records := data.LoadRings()

	out := make(map[string]combat.RingID, len(records))
	for _, key := range data.RingOrder(records) {
		rules, err := ringRules(records[key])
		if err != nil {
			panic("rings.json: " + err.Error())
		}
		id, err := combat.RegisterRing(key, records[key].Name, rules)
		if err != nil {
			panic("rings.json: " + err.Error())
		}
		out[key] = id
	}
	return out
}

// Rings is every registered record key, sorted. For a tool or a screen that wants the catalogue
// rather than what is worn.
func Rings() []string {
	out := make([]string, 0, len(registeredRings))
	for key := range registeredRings {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// RingID is the rules-level ring behind a record key.
func RingID(key string) (combat.RingID, bool) {
	id, ok := registeredRings[key]
	return id, ok
}

// ringRules turns one record's strings into rules. **Every word is resolved rather than trusted** —
// a moment, a verb, an element, a form, a concept label and a status key are six vocabularies, and
// a misspelling in any of them is a ring that wears cleanly and does nothing.
func ringRules(r data.RingData) ([]combat.RingRule, error) {
	out := make([]combat.RingRule, 0, len(r.Rules))

	for _, rule := range r.Rules {
		when, ok := combat.ParseMoment(rule.When)
		if !ok {
			return nil, fmt.Errorf("%s wakes at %q, which is not a moment", r.RingRecord, rule.When)
		}

		cond, err := ringCondition(r.RingRecord, rule.If)
		if err != nil {
			return nil, err
		}

		then := make([]combat.RingEffect, 0, len(rule.Then))
		for _, e := range rule.Then {
			effect, err := ringEffect(r.RingRecord, e)
			if err != nil {
				return nil, err
			}
			then = append(then, effect)
		}

		out = append(out, combat.RingRule{When: when, If: cond, Then: then})
	}
	return out, nil
}

func ringCondition(key string, in *data.RingIfData) (combat.RingCondition, error) {
	var cond combat.RingCondition
	if in == nil {
		return cond, nil
	}

	if in.Element != "" {
		e, ok := combat.ParseElement(in.Element)
		if !ok {
			return cond, fmt.Errorf("%s matches element %q, which the rules do not have", key, in.Element)
		}
		cond.Element, cond.HasElement = e, true
	}
	if in.Form != "" {
		f, ok := combat.ParseForm(in.Form)
		if !ok {
			return cond, fmt.Errorf("%s matches form %q, which the rules do not have", key, in.Form)
		}
		cond.Form, cond.HasForm = f, true
	}
	if in.Concept != "" {
		// A concept is named by its label, which for the player's deck is its registry key. An
		// enemy's cards are scoped to that enemy and a ring can never match one, which is correct:
		// a ring is the duelist's only.
		id, ok := combat.ConceptByKey(in.Concept)
		if !ok {
			return cond, fmt.Errorf("%s matches concept %q, which is not a card in the player's deck", key, in.Concept)
		}
		cond.Concept, cond.HasConcept = id, true
	}
	return cond, nil
}

func ringEffect(key string, in data.RingEffectData) (combat.RingEffect, error) {
	var out combat.RingEffect

	do, ok := combat.ParseRingVerb(in.Do)
	if !ok {
		return out, fmt.Errorf("%s does %q, which is not an effect", key, in.Do)
	}
	out.Do, out.Amount = do, in.Amount

	if in.Status != "" {
		id, ok := combat.StatusByKey(in.Status)
		if !ok {
			return out, fmt.Errorf("%s applies status %q, which is in no file", key, in.Status)
		}
		out.Status = id
	}
	if in.Element != "" {
		e, ok := combat.ParseElement(in.Element)
		if !ok {
			return out, fmt.Errorf("%s names element %q, which the rules do not have", key, in.Element)
		}
		out.Element = e
	}
	return out, nil
}

// Wear puts a ring on, at the right-hand end of the row. It reports false for a record the catalogue
// does not hold, for one already worn, and for the sixth ring.
//
// **Worn order is the order rings fire in**, so appending is what makes the row on screen the rule:
// a ring bought later applies later, and the player can see which.
func (s *Session) Wear(key string) bool {
	if _, ok := registeredRings[key]; !ok {
		return false
	}
	if len(s.worn) >= combat.MaxWornRings {
		return false
	}
	for _, k := range s.worn {
		if k == key {
			return false
		}
	}
	s.worn = append(s.worn, key)
	return true
}

// Worn is what the run is wearing, in worn order, by record key. The ring row draws this and looks
// each record up for its art.
func (s *Session) Worn() []string {
	out := make([]string, len(s.worn))
	copy(out, s.worn)
	return out
}

// WornRings is the same set as rules, each carrying its accumulator. This is what a duelist is
// handed and what every moment outside combat is resolved against.
func (s *Session) WornRings() []combat.WornRing {
	out := make([]combat.WornRing, 0, len(s.worn))
	for _, key := range s.worn {
		id, ok := registeredRings[key]
		if !ok {
			continue
		}
		out = append(out, combat.WornRing{Ring: id, Grown: s.grown[key]})
	}
	return out
}

// Grown is how far one worn ring's accumulator has got. **Keyed by record rather than by position**,
// because a ring taken off and put back on is the same ring — and because this is the first ring
// state that will have to be serialized, where a position would mean nothing.
func (s *Session) Grown(key string) int { return s.grown[key] }

// Equip is the `fight-start` moment: the duelist puts the run's rings on and takes whatever they add
// for the fight.
//
// **The stats are added here rather than baked into the record**, so a ring taken off between fights
// stops paying. HP raises the ceiling and fills it, because a fight starts at full life; a duelist
// arriving hurt keeps the wound and gains the headroom.
func (s *Session) Equip(d combat.Duelist) combat.Duelist {
	worn := s.WornRings()
	for _, w := range worn {
		d = d.Wearing(w)
	}

	d.DMG += combat.AddedDMG(worn)

	if hp := combat.AddedHP(worn); hp > 0 {
		d.MaxLife += hp
		d.CurrentLife += hp
		if d.CurrentLife > d.MaxLife {
			d.CurrentLife = d.MaxLife
		}
	}
	return d
}

// FightDeck is the deck this fight is dealt from: every card the run owns, with every `deck-built`
// flip applied.
//
// **The stored deck is untouched**, which is what makes flips non-composing by construction: each
// card's colour is read off what the run actually owns, so two flips both land on their own sources
// and the order they were bought in cannot chain a deck to one colour. It also means a ring bought
// mid-run applies from the next fight without rewriting anything.
func (s *Session) FightDeck() []combat.Card {
	deck := s.Deck()

	worn := s.WornRings()
	if len(worn) == 0 {
		return deck
	}

	for i, card := range deck {
		if e, flipped := combat.FlipElement(worn, card); flipped {
			deck[i].Element = e
		}
	}
	return deck
}

// Picks is how many prizes the post-battle screen offers, at the `prizes-dealt` moment. One, plus
// whatever the rings add; never fewer than one, because a win that awards nothing is not a win.
func (s *Session) Picks() int {
	picks := 1 + combat.AddedPicks(s.WornRings())
	if picks < 1 {
		return 1
	}
	return picks
}

// PrizeVitae is what the vitae prize card pays, given what the screen offers as its base. **Flat, not
// a scaling** — Soul Taker turns 5 into 10 rather than doubling whatever the card happens to say.
func (s *Session) PrizeVitae(base int) int {
	out := base + combat.AddedPrizeVitae(s.WornRings())
	if out < 0 {
		return 0
	}
	return out
}

// CardCost is what one card costs the run as it stands, discounts included. The post-battle screen
// draws deck cards and has no duelist to ask.
func (s *Session) CardCost(c combat.Card) int { return combat.CostWith(s.WornRings(), c) }

// propagate is vitae earning interest: **+1 for every 5 held, capped at +5**, then scaled by every
// ring that scales it.
//
// **It is a rule of the run rather than a ring** *(owner's call, 2026-08-17)*, which is what lets
// Banker bend it — a ring may only ever bend a rule the game already has.
//
// **The cap binds the base rate and a ring scales what the cap produced.** At 25 held that is +5
// bare and +10 wearing Banker. An absolute cap on the figure that finally lands would leave the ring
// doing nothing past 25 held, which is a ring that stops working exactly when a run can afford it.
// Rounded down, like every other integer rule in the game.
func (s *Session) propagate() {
	base := s.vitae / 5
	if base > maxPropagation {
		base = maxPropagation
	}
	if base <= 0 {
		return
	}
	s.AddVitae(combat.ScalePropagation(s.WornRings(), base))
}

// maxPropagation is the ceiling on the base rate. **It is what stops it running away**: uncapped,
// +1 per 5 is roughly x1.2 a purse per fight, which compounds across 24 fights into a number no shop
// can be priced against. Capping the rate rather than the purse leaves a big purse worth having.
const maxPropagation = 5

// growRings is the other half of `fight-won`: every growing ring's accumulator takes its step.
//
// **Uncapped, by decision** *(2026-08-17)* — +5 a fight reaches +100 by the top of the tower and that
// is the intent, not an overflow.
func (s *Session) growRings() {
	for _, key := range s.worn {
		id, ok := registeredRings[key]
		if !ok {
			continue
		}
		if step := combat.Growth(combat.WornRing{Ring: id, Grown: s.grown[key]}); step != 0 {
			s.grown[key] += step
		}
	}
}
