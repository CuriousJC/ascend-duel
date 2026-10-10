package session

// Relics: the catalog, what the run is wearing, and the moments that fire outside combat.
//
// **`relics.json` is parsed here for the reason the essences are** — a relic belongs to a *run*. Two
// of its moments are `equipped` and `fight-won`, neither of which happens inside
// `internal/combat` at all, and the accumulator a growing relic carries has to survive a fight. This
// package is what survives one.
//
// **A third, `card-drawn`, is fired by a screen and answered here.** The flips and demotions belong
// to a run and the draw pile belongs to a fight, so `DrawnAs` is what the combat screen calls once
// per card as it deals one. See combat.MomentCardDrawn.
//
// **What crosses the edge is a rules type, never a record.** `data.RelicData` holds an art key and a
// long-press line; `combat.RegisterRelic` takes a key, a name and `[]combat.RelicRule`. So the engine
// never reads a file it has no business in, and this file never grows an opinion about what a
// relic's effect is worth. Same division `decks.EnemyCards` draws for enemy cards.
//
// **Bad records panic at load**, like every other catalog: an unknown moment, a verb used at the
// wrong moment, or a predicate the rules cannot resolve.

import (
	"fmt"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// StartingRelics is what a run opens wearing, in worn order.
//
// **Empty since 2026-08-21** *(owner's call)*, the day the shop landed. It held fire, ice and
// lightning for four days, which was always written down as temporary: it existed because a relic
// could not otherwise be got at all, and the shop is the thing it was waiting for. A run now opens
// bare and buys its first relic out of what the first fights pay.
//
// **What that means for a launch, said plainly:** every element is inert until the first relic is
// bought — an ice Bash is a plain Bash with a blue border. That is the intended shape of a run rather than an oversight, and it is the
// thing to look at first if the early realms read as flat.
//
// **It stays as a list rather than being deleted**, because it is the relic counterpart of
// `deckSeedName`: filling it in is how a relic gets onto a hand without playing to a shop, which is
// the only way to look at one in a launched game. Empty is the shipped value.
//
// It lived in `internal/screens` until 2026-08-17, where it could not survive a fight.
var StartingRelics []string

// StartingRelicSlots is how many fingers a run opens with, or zero for combat.DefaultRelicSlots.
//
// **A package var beside StartingRelics, and set from the same seat, because of an ordering**: New
// wears StartingRelics as it builds the run, so a cap raised afterwards would arrive to find the
// extra relics already refused. A fixture wanting six relics has to be holding six fingers before
// the first one goes on.
//
// **Zero is the shipped value and means five.** It is a debug seat like the two lists around it —
// what moves this in a real run is a tonic, read over the top of it by RelicSlots.
var StartingRelicSlots int

// RelicSlots is how many relics this run may wear at once.
//
// **The run's base plus every relic-slot tonic it has drunk.** The tonics are read here rather than
// written into the base, so the save holds the base and a resume cannot add a tonic twice.
func (s *Session) RelicSlots() int {
	n := s.relicSlots
	if n <= 0 {
		n = combat.DefaultRelicSlots
	}
	return n + s.tonicTotal(TonicRelicSlots)
}

// SetRelicSlots moves the cap, and refuses to close the hand entirely.
//
// **A cap below one is clamped up rather than taken as "no relics"**, exactly as SetRoundLimit
// refuses to stop the clock: a drawback that reached zero here would take the whole relic mechanic
// off the run rather than making it harsher, which is the one direction a bug in this is invisible.
//
// **There is no clamp upward any more** *(owner's call, 2026-09-17)*. It used to stop at
// `combat.MaxWornRelics`, the width of the duelist's relic array, so that the shop and the fighter
// could not disagree about a number the array would silently refuse. The array is a slice now and
// there is no width, so a cap is whatever the run says it is.
func (s *Session) SetRelicSlots(n int) {
	if n < 1 {
		n = 1
	}
	s.relicSlots = n
}

// resumeRelicSlots is a saved cap read back, with an old save's silence answered. Zero means a file
// written before the cap was a field, not a run that may wear nothing — the trap resumeRoundLimit
// documents, one field over.
func resumeRelicSlots(saved int) int {
	if saved < 1 {
		return combat.DefaultRelicSlots
	}
	return saved
}

// registeredRelics is every relic in the catalog, registered with the rules at package init and
// indexed by record key.
//
// **Walked in sorted key order**, per the determinism rules: `LoadRelics` hands back a map, and
// registering in map order would deal a different set of RelicIDs every launch. Nothing may serialize
// one, but a tool printing them would still tell a different story each run.
var registeredRelics, relicPrices, relicSells, relicWeights = registerRelics()

// relicKeys is registeredRelics the other way round: which record a rules-level relic came from.
// **Built from the same map rather than from a second walk of the file**, so the two cannot disagree
// about what a RelicID is.
var relicKeys = func() map[combat.RelicID]string {
	out := make(map[combat.RelicID]string, len(registeredRelics))
	for key, id := range registeredRelics {
		out[id] = key
	}
	return out
}()

// registerRelics hands back all three maps from one walk of the file, rather than reading it twice.
// **The price, the sell-back and the draw weight are registered here and not with the rules**: `internal/combat` resolves a round and
// has no purse, so what a relic costs is the run's business in exactly the way its art is a
// screen's — the same line `RegisterRelic` already draws.
func registerRelics() (map[string]combat.RelicID, map[string]int, map[string]int, map[string]int) {
	records := data.LoadRelics()

	out := make(map[string]combat.RelicID, len(records))
	prices := make(map[string]int, len(records))
	sells := make(map[string]int, len(records))
	weights := make(map[string]int, len(records))
	for _, key := range data.RelicOrder(records) {
		rules, err := relicRules(records[key])
		if err != nil {
			panic("relics.json: " + err.Error())
		}
		id, err := combat.RegisterRelic(key, records[key].Name, rules)
		if err != nil {
			panic("relics.json: " + err.Error())
		}
		// **A relic with no rarity is refused rather than given away.** Every other word in this
		// file is resolved rather than trusted, and an absent tier is the same failure as a
		// misspelled one: a relic that reaches the shelf costing nothing and never being offered.
		rarity := records[key].Rarity
		if !rarity.Valid() {
			panic(fmt.Sprintf("relics.json: %s has rarity %q, which is not one of common, "+
				"uncommon or rare", key, rarity))
		}
		out[key] = id
		prices[key] = rarity.Price()
		sells[key] = rarity.Sell()
		weights[key] = rarity.Weight()
	}
	return out, prices, sells, weights
}

// CheckRelicRecord holds one record to everything registration would, without registering it.
//
// **It is for the archive**: a record in `data/archive/relics.json` is out of the game and never
// registered, and this is what keeps it one that would load if it were moved back. Every word is
// resolved as registerRelics resolves it, the rules go through `combat.CheckRelic`, the rarity
// has to be one of the three, and an `Unlock` has to be one some achievement grants.
func CheckRelicRecord(r data.RelicData) error {
	rules, err := relicRules(r)
	if err != nil {
		return err
	}
	if err := combat.CheckRelic(r.RelicRecord, rules); err != nil {
		return err
	}
	if !r.Rarity.Valid() {
		return fmt.Errorf("%s has rarity %q, which is not one of common, uncommon or rare",
			r.RelicRecord, r.Rarity)
	}
	return checkUnlock(r)
}

// Relics is every registered record key, sorted. For a tool or a screen that wants the catalog
// rather than what is worn.
func Relics() []string {
	out := make([]string, 0, len(registeredRelics))
	for key := range registeredRelics {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// RelicID is the rules-level relic behind a record key.
func RelicID(key string) (combat.RelicID, bool) {
	id, ok := registeredRelics[key]
	return id, ok
}

// relicRules turns one record's strings into rules. **Every word is resolved rather than trusted** —
// a moment, a verb, an element, a form and a concept label are five vocabularies, and
// a misspelling in any of them is a relic that wears cleanly and does nothing.
func relicRules(r data.RelicData) ([]combat.RelicRule, error) {
	return parseRelicRules(r.RelicRecord, r.Rules)
}

// parseRelicRules is relicRules for any rules written in the relic grammar, under the key they will
// be registered as — a catalog relic's, or a cantrip-relic's, which is written inline on its cantrip.
func parseRelicRules(key string, rules []data.RelicRuleData) ([]combat.RelicRule, error) {
	out := make([]combat.RelicRule, 0, len(rules))

	for _, rule := range rules {
		when, ok := combat.ParseMoment(rule.When)
		if !ok {
			return nil, fmt.Errorf("%s wakes at %q, which is not a moment", key, rule.When)
		}

		cond, err := relicCondition(key, rule.If)
		if err != nil {
			return nil, err
		}

		then := make([]combat.RelicEffect, 0, len(rule.Then))
		for _, e := range rule.Then {
			effect, err := relicEffect(key, e)
			if err != nil {
				return nil, err
			}
			then = append(then, effect)
		}

		out = append(out, combat.RelicRule{When: when, If: cond, Then: then})
	}
	return out, nil
}

func relicCondition(key string, in *data.RelicIfData) (combat.RelicCondition, error) {
	var cond combat.RelicCondition
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
		// enemy's cards are scoped to that enemy and a relic can never match one, which is correct:
		// a relic is the duelist's only.
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
	// `Hand` and `Hands` are the same field for one rung and for several. A record setting both
	// is refused rather than having one quietly win.
	if in.Hand != "" && len(in.Hands) > 0 {
		return cond, fmt.Errorf("%s names both Hand and Hands, and they are the same predicate", key)
	}
	keys := in.Hands
	if in.Hand != "" {
		keys = []string{in.Hand}
	}
	for _, k := range keys {
		id, ok := combat.HandIDForKey(k)
		if !ok {
			return cond, fmt.Errorf("%s matches hand %q, which is not a rung of the ladder", key, k)
		}
		cond.Hands = append(cond.Hands, id)
	}

	if in.MinForms > 0 {
		cond.MinForms = in.MinForms
	}
	cond.Lead = in.Lead
	return cond, nil
}

func relicEffect(key string, in data.RelicEffectData) (combat.RelicEffect, error) {
	var out combat.RelicEffect

	do, ok := combat.ParseRelicVerb(in.Do)
	if !ok {
		return out, fmt.Errorf("%s does %q, which is not an effect", key, in.Do)
	}
	out.Do, out.Amount = do, in.Amount

	if in.Element != "" {
		e, ok := combat.ParseElement(in.Element)
		if !ok {
			return out, fmt.Errorf("%s names element %q, which the rules do not have", key, in.Element)
		}
		out.Element = e
	}
	if in.Form != "" {
		f, ok := combat.ParseForm(in.Form)
		if !ok {
			return out, fmt.Errorf("%s names form %q, which the rules do not have", key, in.Form)
		}
		out.Form = f
	}
	return out, nil
}

// Wear puts a relic on, at the right-hand end of the row. It reports false for a record the catalog
// does not hold, for one already worn, and for the sixth relic.
//
// **Worn order is the order relics fire in**, so appending is what makes the row on screen the rule:
// a relic bought later applies later, and the player can see which.
func (s *Session) Wear(key string) bool {
	if !s.canWear(key) {
		return false
	}
	s.worn = append(s.worn, key)
	return true
}

// MoveRelic slides the worn relic at `from` to sit at `to`, shuffling everything between them along.
// It reports whether the row actually changed.
//
// **Worn order is the order relics fire in**, so this is a rule change and not a tidy-up — left to
// right, and multiplicative effects compound in the order they are applied. It is the run's half of
// a reorder; a fight already under way is holding its own copy of the row, so the caller has to move
// that too. See combat.Duelist.MoveRelic.
//
// **Accumulators are untouched, because they are keyed by record and not by position** — a relic
// moved along the row is the same relic with the same number, which is the same property that lets
// one be taken off and put back on.
//
// An out-of-range index is a no-op rather than a panic: this is driven by a drag, and a drop
// resolved against a row that changed underneath it must not take the frame with it.
func (s *Session) MoveRelic(from, to int) bool {
	n := len(s.worn)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return false
	}

	key := s.worn[from]
	if from < to {
		copy(s.worn[from:to], s.worn[from+1:to+1])
	} else {
		copy(s.worn[to+1:from+1], s.worn[to:from])
	}
	s.worn[to] = key
	return true
}

// Worn is what the run is wearing, in worn order, by record key. The relic row draws this and looks
// each record up for its art.
func (s *Session) Worn() []string {
	out := make([]string, len(s.worn))
	copy(out, s.worn)
	return out
}

// WornRelics is the same set as rules, each carrying its accumulator. This is what a duelist is
// handed and what every moment outside combat is resolved against.
func (s *Session) WornRelics() []combat.WornRelic {
	out := make([]combat.WornRelic, 0, len(s.worn))
	for _, key := range s.worn {
		id, ok := registeredRelics[key]
		if !ok {
			continue
		}
		out = append(out, combat.WornRelic{Relic: id, Grown: s.grown[key], Weightless: s.weightless[key]})
	}
	return out
}

// SetWeightless marks a relic as worn weightless or not, by record — before it goes on or while it is
// worn. **Any relic may be weightless**: it takes no finger, so a full row still takes it. Nothing in
// the shipped game sets it on a run relic yet; a cantrip-relic is weightless by its own wearing.
func (s *Session) SetWeightless(key string, on bool) {
	if s.weightless == nil {
		s.weightless = map[string]bool{}
	}
	if on {
		s.weightless[key] = true
		return
	}
	delete(s.weightless, key)
}

// IsWeightless reports whether a relic is worn weightless.
func (s *Session) IsWeightless(key string) bool { return s.weightless[key] }

// WeightedCount is how many of the worn relics take a finger: every one but the weightless.
func (s *Session) WeightedCount() int {
	n := 0
	for _, key := range s.worn {
		if !s.weightless[key] {
			n++
		}
	}
	return n
}

// StartingWeightless is which of StartingRelics a run opens wearing weightless. **A debug seat**,
// beside StartingRelics and set from the same place: `internal/scenario` fills it and nothing else
// may. Set before the relics go on, so a weightless one is not refused a finger.
var StartingWeightless []string

// AbsorbGrowth takes back whatever a fight grew. **The run's accumulators are the duelist's, after
// the fight** *(2026-08-22)*.
//
// A `grow-on-hit` relic is the first accumulator that moves *during* a fight rather than after one:
// combat bumps the duelist's own copy as each blow lands, so the second fire attack of a fight is
// already stronger than the first. That copy is thrown away when the screen leaves, which is what
// this reads before it happens.
//
// **It takes the larger of the two figures rather than trusting the duelist outright**, because the
// duelist is put together by Equip from these same numbers: a caller handing back a duelist that
// never wore the relic — a test, a screen that rebuilt its fighter — must not be able to wind a
// run's accumulator backwards.
//
// **A relic that can reset is the exception and is taken outright** — see combat.Resets. Momentum's
// streak carries from fight to fight until a defend card wipes it, and a wipe is the figure going
// down: taking the larger would keep a streak the player had already lost. Only a relic the duelist
// was wearing is read at all, so the guard above still holds for it.
//
// **It is called on a win and not on a defeat**, which needs no rule of its own: a lost fight ends
// the run.
func (s *Session) AbsorbGrowth(d combat.Duelist) {
	for _, w := range d.WornRelics() {
		key, ok := relicKeys[w.Relic]
		if !ok {
			continue
		}
		if combat.Resets(w.Relic) || w.Grown > s.grown[key] {
			s.grown[key] = w.Grown
		}
	}
}

// Grown is how far one worn relic's accumulator has got. **Keyed by record rather than by position**,
// because a relic taken off and put back on is the same relic — and because this is the first relic
// state that will have to be serialized, where a position would mean nothing.
func (s *Session) Grown(key string) int { return s.grown[key] }

// Equip is the `equipped` moment: the duelist puts the run's relics on and takes whatever they add
// for the fight. It is EquipWorn with the run's own relics, in the run's order.
func (s *Session) Equip(d combat.Duelist) combat.Duelist { return s.EquipWorn(d, s.WornRelics()) }

// EquipWearing is Equip with the fight's own relics worn after the run's — the cantrip-relics cast
// so far, in the order a fight starts them in.
func (s *Session) EquipWearing(d combat.Duelist, extra []combat.WornRelic) combat.Duelist {
	return s.EquipWorn(d, append(s.WornRelics(), extra...))
}

// EquipWorn puts a whole worn row on the duelist, in the order given — the run's relics and the
// fight's cantrip-relics, interleaved however the player has dragged them. **Every relic in it goes
// through every step**, so a cantrip-relic's `add-dmg` lands with the others' and its `scale-hp`
// scales the whole body, compounding left to right; the run is never written to, which is what makes
// a cantrip-relic last only the fight.
//
// **The stats are added here rather than baked into the record**, so a relic taken off between fights
// stops paying. HP raises the ceiling and fills it, because a fight starts at full life; a duelist
// arriving hurt keeps the wound and gains the headroom.
func (s *Session) EquipWorn(d combat.Duelist, worn []combat.WornRelic) combat.Duelist {
	// **The boss bonus goes on before any relic**. It is growth of the duelist's own body rather
	// than something worn, so a percentage relic scales the grown figure — the ordering this
	// function already documents below rather than a third rule. See life.go.
	// **The potions go on with it**, since a potion changes the duelist themself rather than being
	// something worn: the relics scale what the potions added too. The boss bonus is flat, so the
	// two add and neither grows the other.
	d.DMG += s.dmgBonus
	d.MaxLife += s.lifeBonus
	d.CurrentLife += s.lifeBonus

	d.MaxLife = s.raiseLifeForBosses(d.MaxLife)
	d.CurrentLife = s.raiseLifeForBosses(d.CurrentLife)

	for _, w := range worn {
		d = d.Wearing(w)
	}

	d.DMG += combat.AddedDMG(worn)

	// **The duel is seeded with the purse rather than with a product.** It was a resolved figure
	// for an hour, which was wrong: vitae moves *inside* a fight — a card kept in hand pays one —
	// so a relic reading the purse has to be re-asked each blow. What crosses the seam is the
	// opening balance; see combat.Duelist.Vitae.
	d.Vitae = s.vitae

	if hp := combat.AddedHP(worn); hp > 0 {
		d.MaxLife += hp
		d.CurrentLife += hp
	}

	// **Scaled after the flat adds**, so Bulwark's +25 is worth a quarter less under Onslaught
	// rather than surviving it whole. The alternative — scale the base, then add — would make the
	// two relics' order of application a thing to remember, and a drawback nothing else touches is
	// not a drawback.
	//
	// **A duelist never scales below 1 life.** A stack of drawbacks reaching zero is a run that
	// cannot start a fight, which is a worse failure than a relic being weaker than its text.
	if pct := combat.HPScale(worn); pct != 100 {
		d.MaxLife = d.MaxLife * pct / 100
		if d.MaxLife < 1 {
			d.MaxLife = 1
		}
	}

	if d.CurrentLife > d.MaxLife {
		d.CurrentLife = d.MaxLife
	}

	// **The clock goes on with the rest of what the run is carrying.** It touches none of the
	// figures above and none of them touch it — a fight's length is not a stat — so its place in
	// this function is not an ordering anybody has to remember. What matters is that it is *here*:
	// a fighter equipped without it carries a zero, and zero is no clock at all. See clock.go.
	//
	// **A relic may move it for the fight, and does so without writing to the run.** The run's own
	// number is the base and the worn set's deltas are summed over the top of it each time a
	// fighter is put together, so selling the relic gives the rounds straight back — where a relic
	// that called SetRoundLimit would leave the clock moved for the rest of the journey. Same shape
	// as AddedHP and HPScale above. See combat.DoAdjustRoundLimit.
	d.RoundLimit = combat.RoundLimitFor(worn, s.RoundLimit())

	// **The finger count goes over with it, and for the same reason.** A fighter equipped without
	// it carries a zero, which the rules read as the default — so this is not load-bearing today
	// and becomes load-bearing the moment anything moves it. See RelicSlots below.
	d.RelicSlots = s.RelicSlots()

	// **The tonics that bend the rules inside a fight go over with the rest.** A cost cut comes off
	// every card after the relics have had their say, and a vitae multiple scales every vitae the
	// round pays. See tonic.go.
	d.CostCut = s.CostCut()
	d.VitaeFactor = s.VitaeFactor()

	// **The stones go on last, and they touch nothing above.** Relics move DMG, life and what a card
	// costs; a stone moves a rung of the hand ladder, which is read at the moment a blow is scored
	// rather than mixed into the figures here. So the order between the two is not a decision
	// anybody has to remember. See stone.go.
	return s.equipStones(d)
}

// DrawnAs is what a card is dealt as: every worn flip and demotion walked over it, or the card
// unchanged when none of them match it.
//
// **It is the `card-drawn` moment, and the run is where it lives** because the fight's piles are a
// screen's and the worn relics are the run's. The deck panel asks it too, of a card the player owns,
// to show the deck they are about to be dealt rather than the list they happen to hold.
//
// **The caller passes the card as the run owns it.** The rings chain within one call — see
// combat.DealSteps — so a card handed back after a trip through here would take a second walk from
// wherever the first one left it. screens.restoreToDeck is what keeps that from happening when the
// discard is folded back into the draw pile.
func (s *Session) DrawnAs(c combat.Card) combat.Card {
	return combat.DealtAs(s.WornRelics(), c)
}

// DealStepsFor is the cascade a card goes through on its way into the hand: one entry per worn ring
// that changes it, in worn order.
//
// **DrawnAs is the end of this walk and this is the whole of it**, which is what the combat screen's
// deal needs — a card dealt under two rings changes twice, and a beat per ring is how the player
// sees which relic did which. See screens/combat_deal.go.
func (s *Session) DealStepsFor(c combat.Card) []combat.DealStep {
	return combat.DealSteps(s.WornRelics(), c)
}

// Picks is how many prizes the post-battle screen offers, at the `prizes-dealt` moment. One, plus
// whatever the relics add; never fewer than one, because a win that awards nothing is not a win.
func (s *Session) Picks() int {
	picks := 1 + combat.AddedPicks(s.WornRelics())
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
	out := base + combat.AddedPrizeVitae(s.WornRelics())
	if out < 0 {
		return 0
	}
	return out
}

// CardCost is what one card costs the run as it stands, discounts included. The post-battle screen
// draws deck cards and has no duelist to ask.
func (s *Session) CardCost(c combat.Card) int {
	return combat.CostWith(s.WornRelics(), s.CostCut(), c)
}

// propagation is vitae earning interest: **+1 for every 5 held, capped at +7**, then scaled by every
// relic that scales it.
//
// **It is a rule of the run rather than a relic** *(owner's call, 2026-08-17)*, which is what lets
// Banker bend it — a relic may only ever bend a rule the game already has.
//
// **The cap binds the base rate and a relic scales what the cap produced.** At 35 held that is +7
// bare and +14 wearing Banker. An absolute cap on the figure that finally lands would leave the relic
// doing nothing past 35 held, which is a relic that stops working exactly when a run can afford it.
// Rounded down, like every other integer rule in the game.
// **It reports the figure rather than adding it** *(2026-08-22)*, because the post-battle screen
// narrates the interest as its own sentence and the purse has to climb when that sentence lands.
// See spoils.go: deciding a payout and paying it are separate here.
// propagationPer is how much held vitae one point of interest is earned for.
const propagationPer = 5

func (s *Session) propagation() int {
	base := s.vitae / propagationPer
	if base > maxPropagation {
		base = maxPropagation
	}
	if base <= 0 {
		return 0
	}
	return combat.ScalePropagation(s.WornRelics(), base)
}

// maxPropagation is the ceiling on the base rate. **It is what stops it running away**: uncapped,
// +1 per 5 is roughly x1.2 a purse per fight, which compounds across 24 fights into a number no shop
// can be priced against. Capping the rate rather than the purse leaves a big purse worth having.
const maxPropagation = 7

// growRelics is the other half of `fight-won`: every growing relic's accumulator takes its step.
//
// **Uncapped, by decision** *(2026-08-17)* — +5 a fight reaches +100 by the top of the journey and that
// is the intent, not an overflow.
func (s *Session) growRelics() {
	for _, key := range s.worn {
		id, ok := registeredRelics[key]
		if !ok {
			continue
		}
		if step := combat.Growth(combat.WornRelic{Relic: id, Grown: s.grown[key]}); step != 0 {
			s.grown[key] += step
		}
	}
}
