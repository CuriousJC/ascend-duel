package combat

// Rings, as rules. The grammar, the three closed vocabularies, and the four moments that fire
// inside this package.
//
// **A ring is the only collected thing that is never played** *(2026-08-17)*. A card resolves in the
// turn you queued it, a worm fires when you pick it, a hand is scored when the attack phase runs —
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
	// **Per card is the point.** A form ring doubles *every* card that matches, so three slash
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

	// MomentTurnTaken fires once at the end of each of this duelist's own turns, whatever the turn
	// held — an empty turn is still a turn taken.
	//
	// **A rule's `If` is matched against the turn as a whole**: it fires when *any* card of the turn
	// matches, which is what lets "a turn with a plan card in it" be said without the predicates
	// needing a negation. See TurnTaken.
	//
	// **Appended, because the enum is append-only.**
	MomentTurnTaken

	// MomentBlowFormed fires once per blow, in handEvent, after the hand is matched and while the
	// base sum is being added up. **It is the only moment that sees the blow rather than a card**,
	// which is what an echo needs: which card leads a blow is a fact about the whole turn.
	//
	// **Appended, because the enum is append-only.**
	MomentBlowFormed

	// MomentCardDrawn fires per card as it leaves the draw pile for the hand: the combat screen's
	// drawHand.
	//
	// **It is the element flip's moment, and the flip is all it holds** *(owner's call,
	// 2026-08-24)*. It used to be a `deck-built` verb, recolouring the whole fight deck once as it
	// came out of the run — which deals the same cards, since a flip is unconditional over an
	// element, and says the wrong thing about *when*. Every one of these rings is worded "every X
	// card is dealt as a Y card", and dealing is what a draw is.
	//
	// **The draw pile therefore holds cards as the run owns them**, and the flip is applied on the
	// way into the hand. That is the invariant the reshuffle has to keep: a discarded card is put
	// back as the run owns it, or a second flip would land on the colour the first one made and two
	// rings would chain a deck to one colour between them. See screens/combat_deck.go.
	//
	// **A card drawn under a flip does not remember what it was.** It carries the colour it became
	// and nothing else, so a rule firing later — a card-damage ring keyed on ice — matches the card
	// in the hand rather than the card in the run. What the original is still reachable *from* is
	// the card's ID, which is a handle for the layers above the rules and never something a rule
	// reads.
	//
	// **Appended, because the enum is append-only.**
	MomentCardDrawn
)

// Moments is every moment in a fixed order, for anything that walks them.
func Moments() []Moment {
	return []Moment{MomentCardCost, MomentCardDamage, MomentAttackLands, MomentDeckBuilt,
		MomentFightStart, MomentFightWon, MomentPrizesDealt, MomentBlowFormed, MomentTurnTaken,
		MomentCardDrawn}
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
	case MomentBlowFormed:
		return "blow-formed"
	case MomentTurnTaken:
		return "turn-taken"
	case MomentCardDrawn:
		return "card-drawn"
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
	case MomentCardCost, MomentCardDamage, MomentAttackLands, MomentDeckBuilt, MomentBlowFormed,
		MomentTurnTaken, MomentCardDrawn:
		return true
	default:
		return false
	}
}

// RingVerb is what a rule does. **One word carrying both the operation and its subject** —
// `scale-damage`, not an operation crossed with a subject *(owner's call, 2026-08-17)*: two crossing
// lists would buy a grid that is mostly meaningless cells, and `apply-status` sits on neither axis.
// The same argument that took the mixes out of `hands.json`.
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

	// DoSetElement is the flip: it recolours a matching card **as that card is drawn**.
	//
	// **The only verb at MomentCardDrawn**, and it moved there on 2026-08-24 from `deck-built`,
	// where it recoloured the whole fight deck in one pass. The cards dealt are the same either way
	// — a flip is unconditional over an element — so what changed is what the game *says*: these
	// rings are all worded "every X card is dealt as a Y card", and a draw is the dealing.
	DoSetElement

	// DoAddDMG is flat DMG for the fight.
	DoAddDMG

	// DoAddHP is flat maximum life for the fight.
	DoAddHP

	// DoGrowOnWin adds Amount to **this ring's own accumulator** once per fight won, which every one
	// of its other effects then reads on top of its own figure. See WornRing.
	//
	// **Named for its moment, like DoGrowOnHit** *(owner's call, 2026-08-22)*. It was `grow`, from
	// when there was only one way to grow; a verb whose name does not say when it fires reads as
	// the default and makes the other one look like the special case.
	DoGrowOnWin

	// DoScalePropagation scales vitae propagation by Amount percent, *after* its cap.
	DoScalePropagation

	// DoAdjustPicks changes how many post-battle choices are offered.
	DoAdjustPicks

	// DoAdjustPrizeVitae changes what a won room pays, flat. **It was the vitae prize card's until
	// 2026-08-22**, when that card was removed and the figure it moved became the room award every
	// win pays — same moment, same flat addition, a figure that is now always there.
	DoAdjustPrizeVitae

	// DoEchoAttack makes the blow's **lead attack card** land more than once. Amount is how many
	// times it lands in total — 3 is full, two thirds, one third — and the echoes are added into
	// the blow's base sum, so the hand still multiplies one figure and a turn still lands one blow.
	//
	// **The ladder is even fractions counting down**, which is what makes one number enough: at
	// Amount n the k-th landing is worth (n-k+1)/n of the card. See EchoBonus.
	DoEchoAttack

	// DoRepeatCard makes **every card the rule matches** land Amount times inside the blow, each
	// landing at full damage. Two is the pair of form rings: a stab card played twice.
	//
	// **Full-strength copies where DoEchoAttack diminishes**, and that is the difference between
	// the two verbs rather than an oversight: an echo is one card ringing on, a repeat is the card
	// played again. Both seat extra terms in the same sum and neither reaches the hand matcher.
	DoRepeatCard

	// DoGrowOnTurn adds Amount to **this ring's own accumulator** at the end of a turn the rule
	// matched — Momentum, which is worth more the longer a duelist keeps swinging.
	DoGrowOnTurn

	// DoResetGrowth puts this ring's accumulator back to zero at the end of a matching turn. It is
	// the only verb that takes an Amount of nothing, because it names no quantity.
	//
	// **A ring that can reset is a ring whose growth belongs to the fight**, not to the run — see
	// KeepsGrowth, which is what stops a streak being banked between fights.
	DoResetGrowth

	// DoGrowOnHit adds Amount to **this ring's own accumulator** every blow that lands with a
	// matching card in it — where DoGrowOnWin does the same once per fight won.
	//
	// **Once per hit** *(owner's call, 2026-08-22)*, where a status is once per blow: two fire cards
	// in a hand are two hits, and a fire card an echo ring seats three times is three. That is the
	// point of it — the accumulator measures how many times something connected, so the rings that
	// multiply landings and the rings that grow per landing are meant to compound.
	//
	// **The step reads the effect's raw Amount**, never `Amount + Grown` — a growth that grew would
	// compound, and every growing ring in the game is linear by decision.
	DoGrowOnHit

	// DoDemoteCard steps a matching attack card Amount rungs **down its own form's ladder** as the
	// fight's deck is dealt: a 3 AP Lunge becomes a 2 AP Thrust, same form, one rung cheaper and
	// half the damage.
	//
	// **It walks `Neighbour`, so the ladder stays a consequence of `duelist_cards.json`** rather
	// than a table here to keep in step with it. A card with no rung below it is left alone — the
	// bottom of a form is the bottom.
	DoDemoteCard

	// DoScaleHP scales maximum life for the fight by Amount percent; 75 takes a quarter off.
	//
	// **A percentage where DoAddHP is flat**, and both exist on purpose: a flat +25 is worth less
	// every floor as the duelist's own life grows, where a scaling stays worth the same. It is the
	// first verb written for a *drawback* — see the Onslaught ring — which is why it is the one
	// scaling verb an author is expected to send below 100.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoScaleHP
)

// RingVerbs is every verb in a fixed order.
func RingVerbs() []RingVerb {
	return []RingVerb{DoAdjustCost, DoScaleDamage, DoApplyStatus, DoSetElement, DoAddDMG,
		DoAddHP, DoGrowOnWin, DoScalePropagation, DoAdjustPicks, DoAdjustPrizeVitae, DoScaleHP,
		DoEchoAttack, DoRepeatCard, DoDemoteCard, DoGrowOnHit, DoGrowOnTurn, DoResetGrowth}
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
	case DoGrowOnWin:
		return "grow-on-win"
	case DoGrowOnHit:
		return "grow-on-hit"
	case DoGrowOnTurn:
		return "grow-on-turn"
	case DoResetGrowth:
		return "reset-growth"
	case DoScalePropagation:
		return "scale-propagation"
	case DoAdjustPicks:
		return "adjust-picks"
	case DoAdjustPrizeVitae:
		return "adjust-prize-vitae"
	case DoScaleHP:
		return "scale-hp"
	case DoEchoAttack:
		return "echo-attack"
	case DoRepeatCard:
		return "repeat-card"
	case DoDemoteCard:
		return "demote-card"
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
	case DoApplyStatus, DoGrowOnHit:
		return MomentAttackLands
	case DoGrowOnTurn, DoResetGrowth:
		return MomentTurnTaken
	case DoSetElement:
		return MomentCardDrawn
	case DoDemoteCard:
		return MomentDeckBuilt
	case DoAddDMG, DoAddHP, DoScaleHP:
		return MomentFightStart
	case DoEchoAttack, DoRepeatCard:
		return MomentBlowFormed
	case DoGrowOnWin, DoScalePropagation:
		return MomentFightWon
	default:
		return MomentPrizesDealt
	}
}

// RingCondition is a rule's `If`. **A zero condition always fires**, which is what the stat rings and
// the two vitae rings want and why the field is optional in the file.
//
// Comparable, and each predicate carries its own "is it set" flag because every one of the three has
// a meaningful zero value: Basic is an element, FormNone is a form, and concept zero is the
// player's first card.
type RingCondition struct {
	Element    Element
	HasElement bool

	Form    Form
	HasForm bool

	Concept    ConceptID
	HasConcept bool

	// Tier narrows a rule to cards sitting on one rung of their form's ladder, which for the
	// player's nine attacks is **the cost printed on the card** — 1, 2 or 3 *(2026-08-22)*.
	//
	// **The declared cost, never the wearer's.** A discount ring makes a Lunge cost 2 to its
	// wearer, and a rule matching `Tier: 3` still has to see a Lunge — otherwise two rings worn
	// together would silently stop each other working, and which one won would depend on the order
	// they were bought in. `Concept.Tier` is the same reading a worm takes, and for the same reason.
	Tier    int
	HasTier bool

	// Lead narrows a rule to the **first attack card of the blow**, and it is the one predicate
	// that is not a fact about the card *(2026-08-22)*. It exists because Echo says "your first
	// attack" where the form rings say "every stab": with it, one verb pair covers both and the
	// scope is written in the file rather than hidden inside a verb.
	//
	// **Only `blow-formed` knows which card leads**, so a rule setting this at any other moment is
	// refused at registration — see checkRule.
	Lead bool
}

// Any reports whether this condition constrains anything at all.
func (c RingCondition) Any() bool {
	return c.HasElement || c.HasForm || c.HasConcept || c.HasTier || c.Lead
}

// Matches reports whether a card satisfies every predicate that is set. **Every one, not any** — two
// predicates on one rule narrow it, which is what a "fire slash" ring would want.
func (c RingCondition) Matches(card Card) bool {
	if c.HasElement && card.Element != c.Element {
		return false
	}
	if c.HasForm && card.Form() != c.Form {
		return false
	}
	if c.HasConcept && card.Concept != c.Concept {
		return false
	}
	if c.HasTier && ConceptOf(card.Concept).Tier() != c.Tier {
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
		if rule.If.Lead && rule.When != MomentBlowFormed {
			return NoRing, fmt.Errorf("%s narrows a %s rule to the lead card, and only blow-formed knows which card leads",
				key, rule.When)
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
	case DoResetGrowth:
		// The one verb that names no quantity: it puts an accumulator to zero.
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
// form multiplier had no element to be a bit under.
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
// decide what a form means at `fight-start`.
func RingEffectsAt(worn []WornRing, m Moment, card Card) []RingEffect {
	src := RingContributionsAt(worn, m, card)
	if len(src) == 0 {
		return nil
	}

	out := make([]RingEffect, 0, len(src))
	for _, c := range src {
		out = append(out, c.Effect)
	}
	return out
}

// RingContribution is one effect and the worn ring that produced it.
//
// **The pair travels together because a screen needs both and can derive neither**, which is the
// argument appliedStatus already makes one moment over: an effect says a card is doubled, and only
// the ring says what doubled it. A tooltip explaining where a figure came from is a picture of the
// second half.
type RingContribution struct {
	Ring   RingID
	Effect RingEffect
}

// RingContributionsAt is RingEffectsAt with each effect still attached to its ring, in worn order.
// **This is the walk**, and RingEffectsAt is the view of it that does not care where an effect came
// from — two walks would be two chances to disagree about which rules fire.
func RingContributionsAt(worn []WornRing, m Moment, card Card) []RingContribution {
	var out []RingContribution
	for _, w := range worn {
		for _, rule := range RingOf(w.Ring).Rules {
			if rule.When != m || !rule.If.Matches(card) {
				continue
			}
			for _, e := range rule.Then {
				e.Amount += w.Grown
				out = append(out, RingContribution{Ring: w.Ring, Effect: e})
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
//
// **It says which ring applied each status, not just which statuses landed** *(2026-08-18)*. It
// always knew - the ring is the thing being walked - and threw the answer away, which left the
// screen unable to fly a CHILLED out of the ring that caused it without inventing an element-to-ring
// table of its own. That table would be a second rule about the same thing, and it would be wrong
// the first time a form ring or a concept ring applied a status, which this grammar already
// allows. See Event.Ring.
//
// **The first ring to apply a status is the one credited**, which falls out of the dedup and out of
// worn order being left to right. Two rings that both set something burning are one burn, and it
// belongs to the one worn first - the same tie-break every other compounding effect takes.
func (d Duelist) statusesFrom(cards []Card) []appliedStatus {
	var out []appliedStatus
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
					out = append(out, appliedStatus{Ring: w.Ring, Status: e.Status})
				}
			}
		}
	}
	return out
}

// appliedStatus is one status a blow lands, and the worn ring that put it there.
//
// **The pair travels together because the screen needs both and can derive neither.** A status
// names what landed; only the ring names where it came from, and "where it came from" is a card
// the player is looking at.
type appliedStatus struct {
	Ring   RingID
	Status StatusID
}

// AddedDMG and AddedHP are what a worn set adds for the fight about to start. They take the worn
// slice rather than a duelist because `session` applies them while the duelist is still being put
// together — the stat they add to is the one that has not been set yet.
func AddedDMG(worn []WornRing) int { return sumAmounts(worn, MomentFightStart, DoAddDMG) }

// HPScale is what every worn ring does to maximum life, as a percentage — 100 when nothing scales
// it. **Compounding left to right**, like every other multiplicative ring effect, so two rings each
// taking a quarter off leave 56% rather than half.
func HPScale(worn []WornRing) int {
	out := 100
	for _, e := range RingEffectsAt(worn, MomentFightStart, Card{}) {
		if e.Do == DoScaleHP {
			out = out * e.Amount / 100
		}
	}
	return out
}

// LandingAmounts is what one card of a blow pays, term by term: its own damage first, then a term
// for every extra landing its rings buy. One entry when nothing repeats or echoes it, which is
// almost every card in the game.
//
// **Two verbs land here and they stack in a fixed order** *(2026-08-22)*: `repeat-card` adds
// full-strength copies, then `echo-attack` adds its diminishing ladder. Repeats first because they
// are the card being played again — an echo of a repeated card would be an echo of something that
// already happened twice, which is a fact about the blow rather than about the card.
//
// **Extra landings add rather than compound**, so two rings landing a card three times land it five
// times, not nine. `MaxEchoLandings` is the ceiling, and it is a width on the event's arrays as much
// as a rule.
//
// `lead` says whether this is the blow's first attack card, which is the only thing the `Lead`
// predicate reads.
func LandingAmounts(worn []WornRing, card Card, lead bool, damage int) []int {
	copies, echoes := 0, 0
	for _, w := range worn {
		for _, rule := range RingOf(w.Ring).Rules {
			if rule.When != MomentBlowFormed || !rule.If.Matches(card) {
				continue
			}
			if rule.If.Lead && !lead {
				continue
			}
			for _, e := range rule.Then {
				amount := e.Amount + w.Grown
				if amount < 2 {
					continue
				}
				switch e.Do {
				case DoRepeatCard:
					copies += amount - 1
				case DoEchoAttack:
					echoes += amount - 1
				}
			}
		}
	}

	if copies+echoes == 0 {
		return []int{damage}
	}

	if total := 1 + copies + echoes; total > MaxEchoLandings {
		// Repeats are kept ahead of echoes when the ceiling bites, since a full-strength landing
		// is the one the player paid for.
		if copies > MaxEchoLandings-1 {
			copies = MaxEchoLandings - 1
		}
		echoes = MaxEchoLandings - 1 - copies
	}

	out := make([]int, 0, 1+copies+echoes)
	out = append(out, damage)
	for i := 0; i < copies; i++ {
		out = append(out, damage)
	}

	n := echoes + 1
	for k := 2; k <= n; k++ {
		out = append(out, EchoBonus(damage, k, n))
	}
	return out
}

// EchoBonus is what one echoed landing is worth: the k-th landing of a card that lands n times,
// where k counts from 1 and k=1 is the full-strength original.
//
// **Even fractions counting down** — at n=3 that is the card, two thirds of it, one third of it —
// which is what lets a ring say the whole ladder with one number. Never below 1, for the reason
// CardDamage is never below 1: a landing that announces itself and deals nothing is worse than a
// small figure.
func EchoBonus(cardDamage, k, n int) int {
	if k <= 1 || k > n || n < 2 {
		return 0
	}
	d := cardDamage * (n - k + 1) / n
	if d < 1 {
		d = 1
	}
	return d
}

// AddedHP is flat maximum life for the fight.
func AddedHP(worn []WornRing) int { return sumAmounts(worn, MomentFightStart, DoAddHP) }

// AddedPicks is how many extra post-battle choices a worn set offers.
func AddedPicks(worn []WornRing) int { return sumAmounts(worn, MomentPrizesDealt, DoAdjustPicks) }

// AddedPrizeVitae is what a worn set adds to a won room's vitae award. **Flat, not a percentage** — Soul
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
			if e.Do == DoGrowOnWin {
				total += e.Amount
			}
		}
	}
	return total
}

// TurnTaken is the duelist after one of their own turns has finished: every `turn-taken` rule that
// the turn matched has grown or reset this ring's accumulator.
//
// **A rule fires when any card of the turn matches it**, and a rule with no `If` fires on every
// turn — including an empty one, which is still a turn taken. That is what lets Momentum be written
// without a "not" in the predicates: one rule grows on every turn, a second resets on a turn holding
// a plan card, and the reset is applied second so a planning turn nets zero.
//
// **Growth first, then resets**, always. The other order would let a turn both bank and lose the
// same step depending on which rule the file happened to list first.
func (d Duelist) TurnTaken(cards []Card) Duelist {
	for i := 0; i < d.RingCount; i++ {
		step, reset := 0, false
		for _, rule := range RingOf(d.Rings[i].Ring).Rules {
			if rule.When != MomentTurnTaken {
				continue
			}
			if rule.If.Any() && !anyMatches(rule.If, cards) {
				continue
			}
			for _, e := range rule.Then {
				switch e.Do {
				case DoGrowOnTurn:
					step += e.Amount
				case DoResetGrowth:
					reset = true
				}
			}
		}

		d.Rings[i].Grown += step
		if reset {
			d.Rings[i].Grown = 0
		}
	}
	return d
}

// anyMatches reports whether any card of a turn satisfies a condition. **Any rather than every**,
// which is the reading a turn-wide predicate needs: "a turn with a plan card in it".
func anyMatches(c RingCondition, cards []Card) bool {
	for _, card := range cards {
		if c.Matches(card) {
			return true
		}
	}
	return false
}

// KeepsGrowth reports whether a ring's accumulator belongs to the **run** rather than to one fight.
//
// **A ring that can reset itself does not keep anything** *(2026-08-22)*: Momentum's streak is a
// fact about the turns of one duel, and banking it between fights would make it a permanent bonus
// that a single plan card once wiped. Heart, the growing stat rings and the Enflamed family hold no
// reset and are kept.
func KeepsGrowth(id RingID) bool {
	for _, rule := range RingOf(id).Rules {
		for _, e := range rule.Then {
			if e.Do == DoResetGrowth {
				return false
			}
		}
	}
	return true
}

// GrowOnHit is the attacker after a blow has landed: every ring whose `grow-on-hit` rule matched a
// card of the blow has taken its step, so the *next* attack of the same fight is already stronger.
//
// **It returns a duelist rather than writing through a pointer**, like everything else in this
// package — a round is resolved by passing duelists along, and an accumulator that moved by side
// effect would be the one piece of fight state a replay could not reproduce.
//
// **What it grows is the copy the fight is holding.** `Session` keeps the run's own figure and reads
// it back off the duelist when the fight is won — see Session.AbsorbGrowth — which is what makes the
// growth survive the fight without combat knowing a run exists.
func (d Duelist) GrowOnHit(cards []Card) Duelist {
	if len(cards) == 0 {
		return d
	}

	// **Landings, not cards** *(owner's call, 2026-08-22)*. A card that an echo or a repeat ring
	// seats three times *hit* three times, and a per-hit accumulator has to count all three — which
	// is the combination the rings are for: Echo plus Enflamed is meant to be a build, not two
	// rings that politely ignore each other.
	//
	// The count is the same list `handEvent` seats into the blow's sum, asked for again rather than
	// passed in: the damage figure does not change how many terms there are, so the two cannot
	// disagree about how many times a card landed.
	worn := d.WornRings()
	hits := make([]int, len(cards))
	for i, c := range cards {
		hits[i] = len(LandingAmounts(worn, c, i == 0, 100))
	}

	for i := 0; i < d.RingCount; i++ {
		step := 0
		for _, rule := range RingOf(d.Rings[i].Ring).Rules {
			if rule.When != MomentAttackLands {
				continue
			}
			landed := 0
			for n, c := range cards {
				if rule.If.Matches(c) {
					landed += hits[n]
				}
			}
			if landed == 0 {
				continue
			}
			for _, e := range rule.Then {
				if e.Do == DoGrowOnHit {
					step += e.Amount * landed
				}
			}
		}
		d.Rings[i].Grown += step
	}
	return d
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

// DemoteConcept is which concept a card is dealt as, given a worn set. It reports false when no
// ring steps it, so a caller can leave the card alone.
//
// **It reads the card as the run owns it, exactly like FlipElement**, so two demoting rings cannot
// walk one card two rungs down the ladder between them — the deepest single step wins and worn
// order decides a tie. A ring wanting two rungs says `Amount: 2`.
//
// **A card with no rung below it is left where it is.** Atrophy on a hand of Jabs is a ring doing
// nothing, which is a fact about that hand rather than a case to special-case.
func DemoteConcept(worn []WornRing, card Card) (ConceptID, bool) {
	deepest := 0
	for _, e := range RingEffectsAt(worn, MomentDeckBuilt, card) {
		if e.Do == DoDemoteCard && e.Amount > deepest {
			deepest = e.Amount
		}
	}
	if deepest == 0 {
		return NoConcept, false
	}
	return Neighbour(card.Concept, -deepest)
}

// FlipElement is what colour a card is dealt as, given a worn set. It reports false when no ring
// touches it, so a caller can leave the card alone rather than writing its own colour back over it.
//
// **Every flip reads the card's original element**, which is what stops two of them chaining a deck
// to one colour: the later ring matches on what the card *is*, not on what the earlier ring made it.
// The last matching flip wins, and worn order is what decides which that is.
//
// **"Original" is now a duty the caller carries** *(2026-08-24)*. While this fired at `deck-built`
// it was true by construction — the fight deck was built out of the run's own cards, once, and
// nothing had flipped anything yet. Firing per draw, the discard pile holds cards that have already
// been through here, so a caller that folds the discard back into the draw pile and hands those
// cards to this function is asking the second flip to read the first one's answer. The combat
// screen restores a card to the face the run owns before it can be drawn again; see
// screens/combat_deck.go.
func FlipElement(worn []WornRing, card Card) (Element, bool) {
	out, flipped := Basic, false
	for _, e := range RingEffectsAt(worn, MomentCardDrawn, card) {
		if e.Do == DoSetElement {
			out, flipped = e.Element, true
		}
	}
	return out, flipped
}
