package combat

// Rings, as rules. The grammar, the three closed vocabularies, and the four moments that fire
// inside this package.
//
// **A ring is the only collected thing that is never played** *(2026-08-17)*. A card resolves in the
// turn you queued it, a worm fires when you pick it, a combo is scored when the attack phase runs —
// each already knows *when* it happens. A ring waits, so it has to say so itself, and that is the
// third part the card language does not need. `.claude/skills/rings/SKILL.md` is the whole grammar;
// MECHANICS.md holds the argument for its shape.
//
// **A ring is a list of `When` / `If` / `Then` rules.** A list rather than one rule because a
// growing stat ring needs two moments — one to accumulate and one to apply — and `Then` is a list
// too, which is what buys a ring that shocks *and* chills with no new vocabulary.
//
// **This package holds the vocabulary and refuses a rule that misuses it; it does not read
// `rings.json`.** The file lives beside the worms in `internal/session`, which parses the strings
// and calls RegisterRing with rules types — so the engine never sees an art key, and a ring's own
// record can carry one. Same division `buildStartingDeck` and `decks.EnemyCards` draw for cards.
//
// **Three of the seven moments fire outside combat**, in `session` and on the post-battle screen,
// which is what makes a ring a *run* concept the rules consult rather than a combat one. The
// appliers for those live here anyway — they are rules, and a screen deriving one from the rule list
// would be a second implementation of the grammar.

import (
	"fmt"
	"sort"
)

// Moment is when a rule wakes up. **Closed, and every one has a seat that already existed** — the
// grammar was designed against the code rather than the other way round.
type Moment int

const (
	// MomentCardCost fires per card whenever a cost is asked for: Duelist.CardCost.
	MomentCardCost Moment = iota

	// MomentCardDamage fires per card inside the blow's base sum: Duelist.CardDamage.
	//
	// **Per card is the point.** A family ring doubles *every* card that matches, so three slash
	// cards in a turn are three doublings inside the same blow.
	MomentCardDamage

	// MomentAttackLands fires once per landed blow, in resolveAttackPhase and resolveSoloAttacks.
	MomentAttackLands

	// MomentDeckBuilt fires once as a fight's deck is dealt out of the run: session.FightDeck.
	MomentDeckBuilt

	// MomentFightStart fires once per fight, as the duelist is put together.
	MomentFightStart

	// MomentFightWon fires once per win, after it.
	MomentFightWon

	// MomentPrizesDealt fires once as the post-battle cards go down.
	MomentPrizesDealt
)

// Moments is every moment in a fixed order, for anything that walks them.
func Moments() []Moment {
	return []Moment{MomentCardCost, MomentCardDamage, MomentAttackLands, MomentDeckBuilt,
		MomentFightStart, MomentFightWon, MomentPrizesDealt}
}

func (m Moment) String() string {
	switch m {
	case MomentCardDamage:
		return "card-damage"
	case MomentAttackLands:
		return "attack-lands"
	case MomentDeckBuilt:
		return "deck-built"
	case MomentFightStart:
		return "fight-start"
	case MomentFightWon:
		return "fight-won"
	case MomentPrizesDealt:
		return "prizes-dealt"
	default:
		return "card-cost"
	}
}

// ParseMoment resolves a moment from its name, reporting failure rather than falling back. A rule
// that quietly never fires is indistinguishable from a ring that does nothing.
func ParseMoment(name string) (Moment, bool) {
	for _, m := range Moments() {
		if m.String() == name {
			return m, true
		}
	}
	return MomentCardCost, false
}

// readsACard reports whether this moment has a card to match an `If` against. The three moments
// outside combat do not, which is why a predicate on one of them is refused at registration rather
// than silently matching everything.
func (m Moment) readsACard() bool {
	switch m {
	case MomentCardCost, MomentCardDamage, MomentAttackLands, MomentDeckBuilt:
		return true
	default:
		return false
	}
}

// RingVerb is what a rule does. **One word carrying both the operation and its subject** —
// `scale-damage`, not an operation crossed with a subject *(owner's call, 2026-08-17)*: two crossing
// lists would buy a grid that is mostly meaningless cells, and `apply-status` sits on neither axis.
// The same argument that took the mixes out of `combos.json`.
//
// **Each verb belongs to exactly one moment**, and a verb used at the wrong one is refused at
// registration rather than ignored. See verbMoment.
type RingVerb int

const (
	// DoAdjustCost makes a matching card cheaper or dearer by a signed Amount.
	DoAdjustCost RingVerb = iota

	// DoScaleDamage scales a matching card's damage by Amount percent; 200 is double.
	DoScaleDamage

	// DoApplyStatus puts a status on whoever took the blow.
	DoApplyStatus

	// DoSetElement is the flip: it recolours every matching card as the fight's deck is dealt.
	DoSetElement

	// DoAddDMG is flat DMG for the fight.
	DoAddDMG

	// DoAddHP is flat maximum life for the fight.
	DoAddHP

	// DoGrow adds Amount to **this ring's own accumulator**, which every one of its other effects
	// then reads on top of its own figure. See WornRing.
	DoGrow

	// DoScalePropagation scales vitae propagation by Amount percent, *after* its cap.
	DoScalePropagation

	// DoAdjustPicks changes how many post-battle choices are offered.
	DoAdjustPicks

	// DoAdjustPrizeVitae changes what the vitae prize card pays, flat.
	DoAdjustPrizeVitae
)

// RingVerbs is every verb in a fixed order.
func RingVerbs() []RingVerb {
	return []RingVerb{DoAdjustCost, DoScaleDamage, DoApplyStatus, DoSetElement, DoAddDMG,
		DoAddHP, DoGrow, DoScalePropagation, DoAdjustPicks, DoAdjustPrizeVitae}
}

func (v RingVerb) String() string {
	switch v {
	case DoScaleDamage:
		return "scale-damage"
	case DoApplyStatus:
		return "apply-status"
	case DoSetElement:
		return "set-element"
	case DoAddDMG:
		return "add-dmg"
	case DoAddHP:
		return "add-hp"
	case DoGrow:
		return "grow"
	case DoScalePropagation:
		return "scale-propagation"
	case DoAdjustPicks:
		return "adjust-picks"
	case DoAdjustPrizeVitae:
		return "adjust-prize-vitae"
	default:
		return "adjust-cost"
	}
}

// ParseRingVerb resolves a verb from its name.
func ParseRingVerb(name string) (RingVerb, bool) {
	for _, v := range RingVerbs() {
		if v.String() == name {
			return v, true
		}
	}
	return DoAdjustCost, false
}

// verbMoment is the one moment each verb belongs to. **A table rather than a check at each applier**,
// because the failure it prevents is a rule that loads, never fires, and looks exactly like a ring
// that does nothing.
func verbMoment(v RingVerb) Moment {
	switch v {
	case DoAdjustCost:
		return MomentCardCost
	case DoScaleDamage:
		return MomentCardDamage
	case DoApplyStatus:
		return MomentAttackLands
	case DoSetElement:
		return MomentDeckBuilt
	case DoAddDMG, DoAddHP:
		return MomentFightStart
	case DoGrow, DoScalePropagation:
		return MomentFightWon
	default:
		return MomentPrizesDealt
	}
}

// RingCondition is a rule's `If`. **A zero condition always fires**, which is what the stat rings and
// the two vitae rings want and why the field is optional in the file.
//
// Comparable, and each predicate carries its own "is it set" flag because every one of the three has
// a meaningful zero value: Basic is an element, FamilyNone is a family, and concept zero is the
// player's first card.
type RingCondition struct {
	Element    Element
	HasElement bool

	Family    Family
	HasFamily bool

	Concept    ConceptID
	HasConcept bool
}

// Any reports whether this condition constrains anything at all.
func (c RingCondition) Any() bool { return c.HasElement || c.HasFamily || c.HasConcept }

// Matches reports whether a card satisfies every predicate that is set. **Every one, not any** — two
// predicates on one rule narrow it, which is what a "fire slash" ring would want.
func (c RingCondition) Matches(card Card) bool {
	if c.HasElement && card.Element != c.Element {
		return false
	}
	if c.HasFamily && card.Family() != c.Family {
		return false
	}
	if c.HasConcept && card.Concept != c.Concept {
		return false
	}
	return true
}

// RingEffect is one entry in a rule's `Then`. Which fields mean anything depends on the verb, the
// same way a card's Amount is read against its verb.
type RingEffect struct {
	Do RingVerb

	// Amount is the figure, read against the verb: a signed cost delta, a percentage, flat DMG or
	// HP, or how much an accumulator grows. Unused by apply-status and set-element, which name a
	// thing rather than a quantity.
	Amount int

	// Status is what apply-status applies.
	Status StatusID

	// Element is what set-element recolours a card to.
	Element Element
}

// RingRule is one `When` / `If` / `Then`.
type RingRule struct {
	When Moment
	If   RingCondition
	Then []RingEffect
}

// Ring is one ring's rules, whole. The art key and the long-press text stay in `data` — see the file
// comment on why this package never reads the record.
type Ring struct {
	// Key is the record key, and the identity anything outside the process has to use: a save file
	// writes it, and the accumulator on `Session` is keyed by it.
	Key   string
	Name  string
	Rules []RingRule
}

// RingID identifies a registered ring. An index into a registry, so **registration-ordered and never
// serialized** — the hazard ConceptID and StatusID carry, and the reason WornRing is resolved from a
// key rather than stored as a number.
type RingID int

// NoRing is the absence of one.
const NoRing RingID = -1

var (
	ringRegistry []Ring
	ringBy       = map[string]RingID{}
)

// RegisterRing adds one ring and returns its ID, or reports why it could not.
//
// **It is where the grammar is enforced**, and the four failures it catches are the four a file can
// produce: an effect whose verb belongs to another moment, a predicate on a moment with no card to
// match, an `apply-status` naming a status no file defines, and a figure that makes the effect do
// nothing. Every one of them would otherwise load cleanly and look like a ring with no rules.
func RegisterRing(key, name string, rules []RingRule) (RingID, error) {
	if key == "" {
		return NoRing, fmt.Errorf("a ring has no record key")
	}
	if id, taken := ringBy[key]; taken {
		return id, fmt.Errorf("%s is registered twice", key)
	}
	if len(rules) == 0 {
		return NoRing, fmt.Errorf("%s has no rules, so wearing it does nothing", key)
	}

	for _, rule := range rules {
		if len(rule.Then) == 0 {
			return NoRing, fmt.Errorf("%s has a %s rule with nothing in its Then", key, rule.When)
		}
		if rule.If.Any() && !rule.When.readsACard() {
			return NoRing, fmt.Errorf("%s has a %s rule with an If, and %s has no card to match one against",
				key, rule.When, rule.When)
		}
		if rule.If.HasConcept && (rule.If.Concept < 0 || int(rule.If.Concept) >= ConceptCount()) {
			return NoRing, fmt.Errorf("%s names a concept the registry does not hold", key)
		}

		for _, e := range rule.Then {
			if want := verbMoment(e.Do); want != rule.When {
				return NoRing, fmt.Errorf("%s does %s at %s, and %s belongs to %s",
					key, e.Do, rule.When, e.Do, want)
			}
			if err := checkEffect(key, e); err != nil {
				return NoRing, err
			}
		}
	}

	id := RingID(len(ringRegistry))
	ringRegistry = append(ringRegistry, Ring{Key: key, Name: name, Rules: rules})
	ringBy[key] = id
	return id, nil
}

// checkEffect holds each verb to the figure it needs. **A zero is refused rather than clamped**,
// unlike a worm's amount: a worm is a reward the player chose and a silent nothing would be worse
// than a ceiling, where a ring is authored once and a zero there is a typo.
func checkEffect(key string, e RingEffect) error {
	switch e.Do {
	case DoApplyStatus:
		if e.Status < 0 || int(e.Status) >= StatusCount() {
			return fmt.Errorf("%s applies a status that is in no file", key)
		}
	case DoSetElement:
		// A flip to basic is the absence of a flip, and Basic is the zero value — so this is also
		// what catches an effect that forgot to name an element at all.
		if e.Element == Basic {
			return fmt.Errorf("%s flips cards to basic, which is the absence of an element", key)
		}
	case DoAdjustCost, DoAdjustPicks, DoAdjustPrizeVitae:
		// Signed on purpose: a discount is negative and a ring with a drawback is expressible.
		if e.Amount == 0 {
			return fmt.Errorf("%s does %s by 0", key, e.Do)
		}
	default:
		if e.Amount <= 0 {
			return fmt.Errorf("%s does %s with Amount %d", key, e.Do, e.Amount)
		}
	}
	return nil
}

// RingOf is the ring behind an ID. An unknown ID is a ring with no rules, which does nothing.
func RingOf(id RingID) Ring {
	if id < 0 || int(id) >= len(ringRegistry) {
		return Ring{Key: "?", Name: "?"}
	}
	return ringRegistry[id]
}

// RingByKey finds a registered ring by its record key.
func RingByKey(key string) (RingID, bool) {
	id, ok := ringBy[key]
	return id, ok
}

// MustRing is RingByKey for callers that would rather fail at startup than wear a ring that does
// nothing.
func MustRing(key string) RingID {
	id, ok := ringBy[key]
	if !ok {
		panic("combat: no ring named " + key)
	}
	return id
}

// RingCount is how many rings are registered.
func RingCount() int { return len(ringRegistry) }

// RingKeys is every registered key, sorted, for a tool or a test walking the catalogue without
// depending on registration order.
func RingKeys() []string {
	out := make([]string, 0, len(ringRegistry))
	for _, r := range ringRegistry {
		out = append(out, r.Key)
	}
	sort.Strings(out)
	return out
}

// MaxWornRings is how many rings can be worn at once. **Five, until brands expand it** — see
// MECHANICS.md, where the cap is deliberately never displayed and surfaces when a sixth is bought.
const MaxWornRings = 5

// WornRing is one ring on a duelist's hand: which ring, and how far its accumulator has grown.
//
// **The accumulator travels with the worn ring rather than living in the registry**, because it
// belongs to a run and the registry belongs to the process. `Session` is what keeps it between
// fights, keyed by record, and sets this field as the duelist is put together — which is also why a
// growing ring is the first ring state that will have to be serialized.
//
// **Every one of the ring's own effect amounts is read as `Amount + Grown`.** A ring holds exactly
// one numeric effect if it grows *(owner's call, 2026-08-17)*, so there is never a question of which
// figure the accumulator feeds.
type WornRing struct {
	Ring  RingID
	Grown int
}

// WornRings is what this duelist is wearing, in worn order.
//
// **Left to right, and it compounds.** That is a determinism rule rather than a preference:
// multiplicative effects are order-sensitive, so the order has to be one a rule can name, and worn
// order is the only order the player can actually see. Two slash rings are x4 and that is a build.
func (d Duelist) WornRings() []WornRing {
	if d.RingCount <= 0 {
		return nil
	}
	n := d.RingCount
	if n > MaxWornRings {
		n = MaxWornRings
	}
	return d.Rings[:n]
}

// Wearing returns this duelist with one more ring on, or unchanged if the hand is full. It returns a
// copy like everything else in this package.
func (d Duelist) Wearing(w WornRing) Duelist {
	if d.RingCount >= MaxWornRings {
		return d
	}
	d.Rings[d.RingCount] = w
	d.RingCount++
	return d
}

// WearsRing reports whether this duelist has one particular ring on. **A query rather than a flag**
// *(2026-08-17)*: it was an array of bools indexed by element until the grammar landed, which a
// family multiplier had no element to be a bit under.
func (d Duelist) WearsRing(id RingID) bool {
	for _, w := range d.WornRings() {
		if w.Ring == id {
			return true
		}
	}
	return false
}

// RingEffectsAt is every effect a worn set fires at one moment, in worn order, with each figure
// already carrying its ring's accumulator.
//
// **The card is what an `If` is matched against**, and a zero Card is what the three cardless moments
// pass — a rule with a predicate at one of those is refused at registration, so nothing here has to
// decide what a family means at `fight-start`.
func RingEffectsAt(worn []WornRing, m Moment, card Card) []RingEffect {
	var out []RingEffect
	for _, w := range worn {
		for _, rule := range RingOf(w.Ring).Rules {
			if rule.When != m || !rule.If.Matches(card) {
				continue
			}
			for _, e := range rule.Then {
				e.Amount += w.Grown
				out = append(out, e)
			}
		}
	}
	return out
}

// ringEffects is RingEffectsAt for the rings this duelist is wearing.
func (d Duelist) ringEffects(m Moment, card Card) []RingEffect {
	return RingEffectsAt(d.WornRings(), m, card)
}

// CardCost is what one card takes out of this duelist's budget, discounts included.
//
// **The rings are read here rather than on the card**, because a cost is a property of the *pairing*:
// the same Strike costs 3 to a duelist wearing the crush discount and 4 to one who is not. `Card.Cost`
// is still the card's own figure and is what a contact sheet or a deck panel draws.
func (d Duelist) CardCost(c Card) int { return CostWith(d.WornRings(), c) }

// CostWith is CardCost for a caller that has a worn set and no duelist — the post-battle screen
// drawing a card out of the run deck, which has to print the price the fight will actually charge.
//
// **A card face and the AP bar must never disagree**, which is the whole reason a cost is asked of
// the wearer rather than of the card: three dashes on a card the budget charges two for is a screen
// contradicting the engine.
func CostWith(worn []WornRing, c Card) int {
	cost := c.Cost()
	for _, e := range RingEffectsAt(worn, MomentCardCost, c) {
		cost += e.Amount
	}
	if cost < minCardCost {
		cost = minCardCost
	}
	return cost
}

// CostOf totals the action-point cost of a queued set, for this duelist.
func (d Duelist) CostOf(cards []Card) int {
	total := 0
	for _, c := range cards {
		total += d.CardCost(c)
	}
	return total
}

// CardDamage is what one card deals in this duelist's hands, before any hand multiplier, blunting or
// defence — the card's own figure scaled by every ring that matches it.
//
// **Compounding, left to right**, which is what makes two matching rings x4 rather than x2. The floor
// is the one `Card.Damage` holds for the same reason: a card that is meant to deal nothing is not an
// attack.
func (d Duelist) CardDamage(c Card) int {
	dmg := c.Damage(d.DMG)
	if dmg == 0 {
		return 0
	}
	for _, e := range d.ringEffects(MomentCardDamage, c) {
		dmg = dmg * e.Amount / 100
	}
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// statusesFrom is every status this duelist's rings put on a target for a landed blow, in the order
// they will be applied.
//
// **Deduplicated, so one blow lands one of each.** Two fire cards in a hand match a fire ring twice,
// and applying a status twice is the same as applying it once — see the no-stacking rule — but it
// would announce itself twice, and a feed saying "sets them burning" twice for one blow is a feed
// describing two things that did not happen.
//
// **Ring order outer, cards inner**, per the worn-order rule.
func (d Duelist) statusesFrom(cards []Card) []StatusID {
	var out []StatusID
	seen := make(map[StatusID]bool)

	for _, w := range d.WornRings() {
		for _, rule := range RingOf(w.Ring).Rules {
			if rule.When != MomentAttackLands {
				continue
			}
			for _, card := range cards {
				if !rule.If.Matches(card) {
					continue
				}
				for _, e := range rule.Then {
					if e.Do != DoApplyStatus || seen[e.Status] {
						continue
					}
					seen[e.Status] = true
					out = append(out, e.Status)
				}
			}
		}
	}
	return out
}

// AddedDMG and AddedHP are what a worn set adds for the fight about to start. They take the worn
// slice rather than a duelist because `session` applies them while the duelist is still being put
// together — the stat they add to is the one that has not been set yet.
func AddedDMG(worn []WornRing) int { return sumAmounts(worn, MomentFightStart, DoAddDMG) }

// AddedHP is flat maximum life for the fight.
func AddedHP(worn []WornRing) int { return sumAmounts(worn, MomentFightStart, DoAddHP) }

// AddedPicks is how many extra post-battle choices a worn set offers.
func AddedPicks(worn []WornRing) int { return sumAmounts(worn, MomentPrizesDealt, DoAdjustPicks) }

// AddedPrizeVitae is what a worn set adds to the vitae prize card. **Flat, not a percentage** — Soul
// Taker turns 5 into 10 rather than doubling whatever the card happens to pay.
func AddedPrizeVitae(worn []WornRing) int {
	return sumAmounts(worn, MomentPrizesDealt, DoAdjustPrizeVitae)
}

func sumAmounts(worn []WornRing, m Moment, do RingVerb) int {
	total := 0
	for _, e := range RingEffectsAt(worn, m, Card{}) {
		if e.Do == do {
			total += e.Amount
		}
	}
	return total
}

// Growth is what one worn ring's accumulator gains from a win. It is per ring rather than summed
// because each ring grows its own — see WornRing.
func Growth(w WornRing) int {
	total := 0
	for _, rule := range RingOf(w.Ring).Rules {
		if rule.When != MomentFightWon {
			continue
		}
		for _, e := range rule.Then {
			if e.Do == DoGrow {
				total += e.Amount
			}
		}
	}
	return total
}

// ScalePropagation applies every propagation-scaling ring to a figure the run's own rule already
// produced and capped.
//
// **The cap binds the base rate and the ring scales what the cap produced** *(owner's call,
// 2026-08-17)*. An absolute cap on the figure that finally lands would leave Banker doing nothing
// past 25 held — a ring that stops working exactly when a run can afford it.
//
// Left to right and compounding, like every other ring effect.
func ScalePropagation(worn []WornRing, base int) int {
	for _, e := range RingEffectsAt(worn, MomentFightWon, Card{}) {
		if e.Do == DoScalePropagation {
			base = base * e.Amount / 100
		}
	}
	return base
}

// FlipElement is what colour a card is dealt as, given a worn set. It reports false when no ring
// touches it, so a caller can leave the card alone rather than writing its own colour back over it.
//
// **Every flip reads the card's original element**, which is what stops two of them chaining a deck
// to one colour: the later ring matches on what the card *is*, not on what the earlier ring made it.
// The last matching flip wins, and worn order is what decides which that is.
func FlipElement(worn []WornRing, card Card) (Element, bool) {
	out, flipped := Basic, false
	for _, e := range RingEffectsAt(worn, MomentDeckBuilt, card) {
		if e.Do == DoSetElement {
			out, flipped = e.Element, true
		}
	}
	return out, flipped
}
