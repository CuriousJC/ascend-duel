package session

// Rings: the catalogue, what the run is wearing, and the moments that fire outside combat.
//
// **`rings.json` is parsed here for the reason the worms are** — a ring belongs to a *run*. Three of
// its ten moments are `deck-built`, `fight-start` and `fight-won`, none of which happen inside
// `internal/combat` at all, and the accumulator a growing ring carries has to survive a fight. This
// package is what survives one.
//
// **A fourth, `card-drawn`, is fired by a screen and answered here.** The flip belongs to a run and
// the draw pile belongs to a fight, so `DrawnAs` is what the combat screen calls once per card as it
// deals one. See combat.MomentCardDrawn.
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
// **Empty since 2026-08-21** *(owner's call)*, the day the shop landed. It held fire, ice and
// lightning for four days, which was always written down as temporary: it existed because a ring
// could not otherwise be got at all, and the shop is the thing it was waiting for. A run now opens
// bare and buys its first ring out of what the first fights pay.
//
// **What that means for a launch, said plainly:** every element is inert until the first ring is
// bought — an ice Strike is a plain Strike with a blue border — so the opening fights carry no
// statuses at all. That is the intended shape of a run rather than an oversight, and it is the
// thing to look at first if the early tower reads as flat.
//
// **It stays as a list rather than being deleted**, because it is the ring counterpart of
// `deckSeedName`: filling it in is how a ring gets onto a hand without playing to a shop, which is
// the only way to look at one in a launched game. Empty is the shipped value.
//
// It lived in `internal/screens` until 2026-08-17, where it could not survive a fight.
var StartingRings []string

// registeredRings is every ring in the catalogue, registered with the rules at package init and
// indexed by record key.
//
// **Walked in sorted key order**, per the determinism rules: `LoadRings` hands back a map, and
// registering in map order would deal a different set of RingIDs every launch. Nothing may serialize
// one, but a tool printing them would still tell a different story each run.
var registeredRings, ringPrices, ringSells, ringWeights = registerRings()

// ringKeys is registeredRings the other way round: which record a rules-level ring came from.
// **Built from the same map rather than from a second walk of the file**, so the two cannot disagree
// about what a RingID is.
var ringKeys = func() map[combat.RingID]string {
	out := make(map[combat.RingID]string, len(registeredRings))
	for key, id := range registeredRings {
		out[id] = key
	}
	return out
}()

// registerRings hands back all three maps from one walk of the file, rather than reading it twice.
// **The price, the sell-back and the draw weight are registered here and not with the rules**: `internal/combat` resolves a round and
// has no purse, so what a ring costs is the run's business in exactly the way its art is a
// screen's — the same line `RegisterRing` already draws.
func registerRings() (map[string]combat.RingID, map[string]int, map[string]int, map[string]int) {
	records := data.LoadRings()

	out := make(map[string]combat.RingID, len(records))
	prices := make(map[string]int, len(records))
	sells := make(map[string]int, len(records))
	weights := make(map[string]int, len(records))
	for _, key := range data.RingOrder(records) {
		rules, err := ringRules(records[key])
		if err != nil {
			panic("rings.json: " + err.Error())
		}
		id, err := combat.RegisterRing(key, records[key].Name, rules)
		if err != nil {
			panic("rings.json: " + err.Error())
		}
		// **A ring with no rarity is refused rather than given away.** Every other word in this
		// file is resolved rather than trusted, and an absent tier is the same failure as a
		// misspelled one: a ring that reaches the shelf costing nothing and never being offered.
		rarity := records[key].Rarity
		if !rarity.Valid() {
			panic(fmt.Sprintf("rings.json: %s has rarity %q, which is not one of common, "+
				"uncommon or rare", key, rarity))
		}
		out[key] = id
		prices[key] = rarity.Price()
		sells[key] = rarity.Sell()
		weights[key] = rarity.Weight()
	}
	return out, prices, sells, weights
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
	if in.Tier != 0 {
		if in.Tier < 1 {
			return cond, fmt.Errorf("%s matches tier %d, and a rung is 1 or more", key, in.Tier)
		}
		cond.Tier, cond.HasTier = in.Tier, true
	}
	cond.Lead = in.Lead
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
	if !s.canWear(key) {
		return false
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

// AbsorbGrowth takes back whatever a fight grew. **The run's accumulators are the duelist's, after
// the fight** *(2026-08-22)*.
//
// A `grow-on-hit` ring is the first accumulator that moves *during* a fight rather than after one:
// combat bumps the duelist's own copy as each blow lands, so the second fire attack of a fight is
// already stronger than the first. That copy is thrown away when the screen leaves, which is what
// this reads before it happens.
//
// **It takes the larger of the two figures rather than trusting the duelist outright**, because the
// duelist is put together by Equip from these same numbers: a caller handing back a duelist that
// never wore the ring — a test, a screen that rebuilt its fighter — must not be able to wind a
// run's accumulator backwards.
//
// **A ring that resets itself keeps nothing** — see combat.KeepsGrowth. Momentum's streak is a fact
// about the turns of one duel, and banking it would make it a permanent bonus that one plan card
// had once wiped.
//
// **It is called on a win and not on a defeat**, which needs no rule of its own: a lost fight ends
// the run.
func (s *Session) AbsorbGrowth(d combat.Duelist) {
	for _, w := range d.WornRings() {
		key, ok := ringKeys[w.Ring]
		if !ok || !combat.KeepsGrowth(w.Ring) {
			continue
		}
		if w.Grown > s.grown[key] {
			s.grown[key] = w.Grown
		}
	}
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
	}

	// **Scaled after the flat adds**, so Bulwark's +25 is worth a quarter less under Onslaught
	// rather than surviving it whole. The alternative — scale the base, then add — would make the
	// two rings' order of application a thing to remember, and a drawback nothing else touches is
	// not a drawback.
	//
	// **A duelist never scales below 1 life.** A stack of drawbacks reaching zero is a run that
	// cannot start a fight, which is a worse failure than a ring being weaker than its text.
	if pct := combat.HPScale(worn); pct != 100 {
		d.MaxLife = d.MaxLife * pct / 100
		if d.MaxLife < 1 {
			d.MaxLife = 1
		}
	}

	if d.CurrentLife > d.MaxLife {
		d.CurrentLife = d.MaxLife
	}
	return d
}

// FightDeck is the draw pile this fight opens with: every card the run owns, with every
// `deck-built` rule applied.
//
// **It is the demotions and nothing else as of 2026-08-24.** The element flip used to be applied
// here too and now fires per card at `card-drawn` — see combat.MomentCardDrawn for why. The cards
// a fight ends up playing are unchanged; what changed is that this pile holds them in the colours
// the run owns, and the flip lands on the way into the hand.
//
// **The stored deck is untouched.** Each card is read as the run owns it, so two demoting rings
// land on their own sources rather than walking one card two rungs between them, and a ring bought
// mid-run applies from the next fight without rewriting anything.
//
// **Identity survives**, which is what lets the screen put a drawn card back the way it found it
// when the discard is reshuffled: a card here is the run's card with its concept possibly stepped
// down, carrying the same ID.
func (s *Session) FightDeck() []combat.Card {
	deck := s.Deck()

	worn := s.WornRings()
	if len(worn) == 0 {
		return deck
	}

	for i, card := range deck {
		if id, demoted := combat.DemoteConcept(worn, card); demoted {
			deck[i].Concept = id
		}
	}
	return deck
}

// AlteredAs is a card the run owns, shown as the rings will actually hand it over: the
// `deck-built` demotion and the `card-drawn` flip, both read off the owned card.
//
// **It is a preview, and the only caller is the deck panel.** Nothing in a fight goes through it —
// the demotion is applied once as the draw pile is built and the flip once as a card is drawn, each
// at its own moment, by the code that owns that moment. This is what those two would produce, asked
// ahead of time, so that a player can see the deck they are about to be dealt rather than the list
// they happen to own.
//
// **The card handed in must be the run's own.** Both verbs read the original, so a card that has
// already been through a draw would have the flip read the flip's own answer.
func (s *Session) AlteredAs(c combat.Card) combat.Card {
	worn := s.WornRings()
	if len(worn) == 0 {
		return c
	}

	out := c
	if id, demoted := combat.DemoteConcept(worn, c); demoted {
		out.Concept = id
	}
	if e, flipped := combat.FlipElement(worn, c); flipped {
		out.Element = e
	}
	return out
}

// DrawnAs is what a card is dealt as: the colour the worn flips make it, or the card unchanged
// when none of them match it.
//
// **It is the `card-drawn` moment, and the run is where it lives** because the fight's piles are a
// screen's and the worn rings are the run's. The caller passes the card *as the run owns it* — see
// combat.FlipElement, which reads the original element and would chain if handed a card it had
// already recoloured.
func (s *Session) DrawnAs(c combat.Card) combat.Card {
	if e, flipped := combat.FlipElement(s.WornRings(), c); flipped {
		c.Element = e
	}
	return c
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

// PrizeVitae is what a win's room award pays, given the room's own base. **Flat, not a scaling** —
// Soul Taker turns 3, 4, 5 into 8, 9, 10 rather than multiplying whatever the room happens to be
// worth.
//
// **It used to be the vitae prize card's** *(retargeted 2026-08-22)*, which went when the reward
// screen stopped offering money as a third card. It is the same `prizes-dealt` moment and the same
// flat addition; what moved is which figure it lands on — the one thing a win always pays.
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

// propagation is vitae earning interest: **+1 for every 5 held, capped at +5**, then scaled by every
// ring that scales it.
//
// **It is a rule of the run rather than a ring** *(owner's call, 2026-08-17)*, which is what lets
// Banker bend it — a ring may only ever bend a rule the game already has.
//
// **The cap binds the base rate and a ring scales what the cap produced.** At 25 held that is +5
// bare and +10 wearing Banker. An absolute cap on the figure that finally lands would leave the ring
// doing nothing past 25 held, which is a ring that stops working exactly when a run can afford it.
// Rounded down, like every other integer rule in the game.
// **It reports the figure rather than adding it** *(2026-08-22)*, because the post-battle screen
// narrates the interest as its own sentence and the purse has to climb when that sentence lands.
// See spoils.go: deciding a payout and paying it are separate here.
func (s *Session) propagation() int {
	base := s.vitae / 5
	if base > maxPropagation {
		base = maxPropagation
	}
	if base <= 0 {
		return 0
	}
	return combat.ScalePropagation(s.WornRings(), base)
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
