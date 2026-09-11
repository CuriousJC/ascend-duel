package combat

// Relics, as rules. The grammar, the three closed vocabularies, and the four moments that fire
// inside this package.
//
// **A relic is the only collected thing that is never played** *(2026-08-17)*. A card resolves in the
// turn you queued it, a worm fires when you pick it, a hand is scored when the attack phase runs —
// each already knows *when* it happens. A relic waits, so it has to say so itself, and that is the
// third part the card language does not need. `.claude/skills/relics/SKILL.md` is the whole grammar;
// MECHANICS.md holds the argument for its shape.
//
// **A relic is a list of `When` / `If` / `Then` rules.** A list rather than one rule because a
// growing stat relic needs two moments — one to accumulate and one to apply — and `Then` is a list
// too, which is what buys a relic that shocks *and* chills with no new vocabulary.
//
// **This package holds the vocabulary and refuses a rule that misuses it; it does not read
// `relics.json`.** The file lives beside the worms in `internal/session`, which parses the strings
// and calls RegisterRelic with rules types — so the engine never sees an art key, and a relic's own
// record can carry one. Same division `buildStartingDeck` and `decks.EnemyCards` draw for cards.
//
// **Three of the seven moments fire outside combat**, in `session` and on the post-battle screen,
// which is what makes a relic a *run* concept the rules consult rather than a combat one. The
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
	// **Per card is the point.** A form relic doubles *every* card that matches, so three slash
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
	// matches, which is what lets "a turn with a defend card in it" be said without the predicates
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
	// element, and says the wrong thing about *when*. Every one of these relics is worded "every X
	// card is dealt as a Y card", and dealing is what a draw is.
	//
	// **The draw pile therefore holds cards as the run owns them**, and the flip is applied on the
	// way into the hand. That is the invariant the reshuffle has to keep: a discarded card is put
	// back as the run owns it, or a second flip would land on the colour the first one made and two
	// relics would chain a deck to one colour between them. See screens/combat_deck.go.
	//
	// **A card drawn under a flip does not remember what it was.** It carries the colour it became
	// and nothing else, so a rule firing later — a card-damage relic keyed on ice — matches the card
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
// that quietly never fires is indistinguishable from a relic that does nothing.
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

// RelicVerb is what a rule does. **One word carrying both the operation and its subject** —
// `scale-damage`, not an operation crossed with a subject *(owner's call, 2026-08-17)*: two crossing
// lists would buy a grid that is mostly meaningless cells, and `apply-status` sits on neither axis.
// The same argument that took the mixes out of `hands.json`.
//
// **Each verb belongs to exactly one moment**, and a verb used at the wrong one is refused at
// registration rather than ignored. See verbMoment.
type RelicVerb int

const (
	// DoAdjustCost makes a matching card cheaper or dearer by a signed Amount.
	DoAdjustCost RelicVerb = iota

	// DoScaleDamage scales a matching card's damage by Amount percent; 200 is double.
	DoScaleDamage

	// DoApplyStatus puts a status on whoever took the blow.
	DoApplyStatus

	// DoSetElement is the flip: it recolours a matching card **as that card is drawn**.
	//
	// **The only verb at MomentCardDrawn**, and it moved there on 2026-08-24 from `deck-built`,
	// where it recoloured the whole fight deck in one pass. The cards dealt are the same either way
	// — a flip is unconditional over an element — so what changed is what the game *says*: these
	// relics are all worded "every X card is dealt as a Y card", and a draw is the dealing.
	DoSetElement

	// DoAddDMG is flat DMG for the fight.
	DoAddDMG

	// DoAddHP is flat maximum life for the fight.
	DoAddHP

	// DoGrowOnWin adds Amount to **this relic's own accumulator** once per fight won, which every one
	// of its other effects then reads on top of its own figure. See WornRelic.
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
	// landing at full damage. Two is the pair of form relics: a stab card played twice.
	//
	// **Full-strength copies where DoEchoAttack diminishes**, and that is the difference between
	// the two verbs rather than an oversight: an echo is one card ringing on, a repeat is the card
	// played again. Both seat extra terms in the same sum and neither reaches the hand matcher.
	DoRepeatCard

	// DoGrowOnTurn adds Amount to **this relic's own accumulator** at the end of a turn the rule
	// matched — Momentum, which is worth more the longer a duelist keeps swinging.
	DoGrowOnTurn

	// DoResetGrowth puts this relic's accumulator back to zero at the end of a matching turn. It is
	// the only verb that takes an Amount of nothing, because it names no quantity.
	//
	// **A relic that can reset is a relic whose growth belongs to the fight**, not to the run — see
	// KeepsGrowth, which is what stops a streak being banked between fights.
	DoResetGrowth

	// DoGrowOnHit adds Amount to **this relic's own accumulator** every blow that lands with a
	// matching card in it — where DoGrowOnWin does the same once per fight won.
	//
	// **Once per hit** *(owner's call, 2026-08-22)*, where a status is once per blow: two fire cards
	// in a hand are two hits, and a fire card an echo relic seats three times is three. That is the
	// point of it — the accumulator measures how many times something connected, so the relics that
	// multiply landings and the relics that grow per landing are meant to compound.
	//
	// **The step reads the effect's raw Amount**, never `Amount + Grown` — a growth that grew would
	// compound, and every growing relic in the game is linear by decision.
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
	// first verb written for a *drawback* — see the Onslaught relic — which is why it is the one
	// scaling verb an author is expected to send below 100.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoScaleHP

	// DoAddHandDamage adds Amount flat to the blow's base sum when the hand named by the rule's
	// `Hand` predicate is what formed.
	//
	// **It is added last and multiplied with everything else** *(owner's call, 2026-09-05)*. Every
	// other term of the sum is a card; this one is the hand itself, so it joins after the cards are
	// counted and before the multiplier is applied — which is what makes a rung's bonus worth more
	// on the rung that pays more, without the relic saying so twice.
	//
	// **A blow forms exactly one hand**, so at most one rule per relic can fire, and two relics
	// naming the same rung add rather than compound.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoAddHandDamage

	// DoAddDamagePerHeld adds Amount to the blow for **every card still in the hand** that matches
	// the rule's predicate — the cards kept back, not the cards played.
	//
	// **It is the one verb that pays for what a turn did not do** *(owner's call, 2026-09-05)*.
	// Every other element and form relic pays for spending a card; this pays for holding one, so a
	// run wearing both is being pulled in two directions on purpose.
	//
	// **The held hand is already something the rules see** — `blowDMG` reads it for the parasite
	// riders — so this needed no new moment. It lands in the blow's base sum beside the hand's own
	// term, and is multiplied with the cards for the same reason that one is.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoAddDamagePerHeld

	// DoGrowPerCard adds Amount to **this relic's own accumulator once for every card of the turn
	// the rule matched** — where DoGrowOnTurn takes one step for a turn holding any match at all.
	//
	// **Per card rather than per turn** *(owner's call, 2026-09-05)*, which is what lets a relic be
	// worth more for defending twice than for defending once. It is the only turn-taken verb that
	// counts rather than fires.
	DoGrowPerCard

	// DoAddDamagePerVitae adds Amount flat damage to every blow **for each vitae the run is
	// carrying** — Rampant, which pays a duelist for not spending.
	//
	// **The rate is fixed at fight-start and the purse is not.** The verb sits at fight-start
	// because that is when the relics are put on, but the figure it produces is re-asked at every
	// blow against Duelist.Vitae, which moves during a fight. A relic resolved once at the door
	// would pay a turn-three blow at turn-one prices.
	DoAddDamagePerVitae

	// DoScaleHandDamage scales **the blow** by Amount percent when it formed the rung the rule
	// names — the rung relics that multiply, where add-hand-damage is the pair that adds a flat term.
	//
	// **It is a second multiplier, never a bigger hand** *(owner's call, 2026-09-05)*. `Multiplier`
	// is the ladder's own figure and stays it, because the banner, the hand row and the sum all
	// show the rung the player built; a relic that grew that number would make the ladder look
	// wrong. The scaling is applied to the result instead, and drawn as its own term.
	DoScaleHandDamage

	// DoScaleDamagePerVitae scales a matching card's damage by **Amount percentage points for each
	// vitae the run holds** — so 1 is +1% a vitae, which is the tenth of a multiplier per ten vitae
	// Fire of Life is written as.
	//
	// **It reads Duelist.Vitae at the card**, like every other per-vitae rule, because the purse
	// moves inside a fight.
	DoScaleDamagePerVitae
)

// RelicVerbs is every verb in a fixed order.
func RelicVerbs() []RelicVerb {
	return []RelicVerb{DoAdjustCost, DoScaleDamage, DoApplyStatus, DoSetElement, DoAddDMG,
		DoAddHP, DoGrowOnWin, DoScalePropagation, DoAdjustPicks, DoAdjustPrizeVitae, DoScaleHP,
		DoEchoAttack, DoRepeatCard, DoDemoteCard, DoGrowOnHit, DoGrowOnTurn, DoResetGrowth,
		DoAddHandDamage, DoAddDamagePerHeld, DoGrowPerCard, DoAddDamagePerVitae,
		DoScaleHandDamage, DoScaleDamagePerVitae}
}

func (v RelicVerb) String() string {
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
	case DoGrowPerCard:
		return "grow-per-card"
	case DoAddDamagePerVitae:
		return "add-damage-per-vitae"
	case DoScaleHandDamage:
		return "scale-hand-damage"
	case DoScaleDamagePerVitae:
		return "scale-damage-per-vitae"
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
	case DoAddHandDamage:
		return "add-hand-damage"
	case DoAddDamagePerHeld:
		return "add-damage-per-held"
	default:
		return "adjust-cost"
	}
}

// ParseRelicVerb resolves a verb from its name.
func ParseRelicVerb(name string) (RelicVerb, bool) {
	for _, v := range RelicVerbs() {
		if v.String() == name {
			return v, true
		}
	}
	return DoAdjustCost, false
}

// verbMoment is the one moment each verb belongs to. **A table rather than a check at each applier**,
// because the failure it prevents is a rule that loads, never fires, and looks exactly like a relic
// that does nothing.
func verbMoment(v RelicVerb) Moment {
	switch v {
	case DoAdjustCost:
		return MomentCardCost
	case DoScaleDamage, DoScaleDamagePerVitae:
		return MomentCardDamage
	case DoApplyStatus, DoGrowOnHit:
		return MomentAttackLands
	case DoGrowOnTurn, DoResetGrowth, DoGrowPerCard:
		return MomentTurnTaken
	case DoSetElement:
		return MomentCardDrawn
	case DoDemoteCard:
		return MomentDeckBuilt
	case DoAddDMG, DoAddHP, DoScaleHP, DoAddDamagePerVitae:
		return MomentFightStart
	case DoEchoAttack, DoRepeatCard, DoAddHandDamage, DoAddDamagePerHeld, DoScaleHandDamage:
		return MomentBlowFormed
	case DoGrowOnWin, DoScalePropagation:
		return MomentFightWon
	default:
		return MomentPrizesDealt
	}
}

// RelicCondition is a rule's `If`. **A zero condition always fires**, which is what the stat relics and
// the two vitae relics want and why the field is optional in the file.
//
// Comparable, and each predicate carries its own "is it set" flag because every one of the three has
// a meaningful zero value: Basic is an element, FormNone is a form, and concept zero is the
// player's first card.
type RelicCondition struct {
	Element    Element
	HasElement bool

	Form    Form
	HasForm bool

	Concept    ConceptID
	HasConcept bool

	// Tier narrows a rule to cards sitting on one rung of their form's ladder, which for the
	// player's nine attacks is **the cost printed on the card** — 1, 2 or 3 *(2026-08-22)*.
	//
	// **The declared cost, never the wearer's.** A discount relic makes a Lunge cost 2 to its
	// wearer, and a rule matching `Tier: 3` still has to see a Lunge — otherwise two relics worn
	// together would silently stop each other working, and which one won would depend on the order
	// they were bought in. `Concept.Tier` is the same reading a worm takes, and for the same reason.
	Tier    int
	HasTier bool

	// Lead narrows a rule to the **first attack card of the blow**, and it is the one predicate
	// that is not a fact about the card *(2026-08-22)*. It exists because Echo says "your first
	// attack" where the form relics say "every stab": with it, one verb pair covers both and the
	// scope is written in the file rather than hidden inside a verb.
	//
	// **Only `blow-formed` knows which card leads**, so a rule setting this at any other moment is
	// refused at registration — see checkRule.
	Lead bool

	// Hand narrows a rule to blows that formed one named rung of the ladder, and it is the second
	// predicate that is not a fact about a card *(2026-09-05)*.
	//
	// **It is asked of the blow, never of a card**, so `Matches` does not read it: a rung is a
	// property of the whole set, and a per-card test would have to answer "is this card part of
	// the hand", which is a different question with a different answer. `HandBonus` is the one
	// caller and it asks directly.
	//
	// **Only `blow-formed` knows what formed**, so a rule setting this at any other moment is
	// refused at registration, exactly as Lead is.
	Hand    HandID
	HasHand bool

	// MinForms narrows a rule to blows whose **scoring cards cover at least this many distinct
	// forms** — Dual Wield, which pays for a pair built out of two different weapons.
	//
	// **The third predicate that is not a fact about a card** *(2026-09-05)*, and like Hand it is
	// asked of the blow: `Matches` does not read it, because "how many forms are in this set" has
	// no per-card answer. It is legal alongside `Hand` for the same reason — both narrow the same
	// set — and refused anywhere but `blow-formed`, which is the only moment that knows what
	// formed.
	MinForms int
}

// Any reports whether this condition constrains anything at all.
func (c RelicCondition) Any() bool {
	return c.HasElement || c.HasForm || c.HasConcept || c.HasTier || c.Lead || c.HasHand ||
		c.MinForms > 0
}

// Matches reports whether a card satisfies every predicate that is set. **Every one, not any** — two
// predicates on one rule narrow it, which is what a "fire slash" relic would want.
func (c RelicCondition) Matches(card Card) bool {
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

// RelicEffect is one entry in a rule's `Then`. Which fields mean anything depends on the verb, the
// same way a card's Amount is read against its verb.
type RelicEffect struct {
	Do RelicVerb

	// Amount is the figure, read against the verb: a signed cost delta, a percentage, flat DMG or
	// HP, or how much an accumulator grows. Unused by apply-status and set-element, which name a
	// thing rather than a quantity.
	Amount int

	// Status is what apply-status applies.
	Status StatusID

	// Element is what set-element recolours a card to.
	Element Element
}

// RelicRule is one `When` / `If` / `Then`.
type RelicRule struct {
	When Moment
	If   RelicCondition
	Then []RelicEffect
}

// Relic is one relic's rules, whole. The art key and the long-press text stay in `data` — see the file
// comment on why this package never reads the record.
type Relic struct {
	// Key is the record key, and the identity anything outside the process has to use: a save file
	// writes it, and the accumulator on `Session` is keyed by it.
	Key   string
	Name  string
	Rules []RelicRule
}

// RelicID identifies a registered relic. An index into a registry, so **registration-ordered and never
// serialized** — the hazard ConceptID and StatusID carry, and the reason WornRelic is resolved from a
// key rather than stored as a number.
type RelicID int

// NoRelic is the absence of one.
const NoRelic RelicID = -1

var (
	relicRegistry []Relic
	relicBy       = map[string]RelicID{}
)

// RegisterRelic adds one relic and returns its ID, or reports why it could not.
//
// **It is where the grammar is enforced**, and the four failures it catches are the four a file can
// produce: an effect whose verb belongs to another moment, a predicate on a moment with no card to
// match, an `apply-status` naming a status no file defines, and a figure that makes the effect do
// nothing. Every one of them would otherwise load cleanly and look like a relic with no rules.
func RegisterRelic(key, name string, rules []RelicRule) (RelicID, error) {
	if key == "" {
		return NoRelic, fmt.Errorf("a relic has no record key")
	}
	if id, taken := relicBy[key]; taken {
		return id, fmt.Errorf("%s is registered twice", key)
	}
	if len(rules) == 0 {
		return NoRelic, fmt.Errorf("%s has no rules, so wearing it does nothing", key)
	}

	for _, rule := range rules {
		if len(rule.Then) == 0 {
			return NoRelic, fmt.Errorf("%s has a %s rule with nothing in its Then", key, rule.When)
		}
		if rule.If.Any() && !rule.When.readsACard() {
			return NoRelic, fmt.Errorf("%s has a %s rule with an If, and %s has no card to match one against",
				key, rule.When, rule.When)
		}
		if rule.If.Lead && rule.When != MomentBlowFormed {
			return NoRelic, fmt.Errorf("%s narrows a %s rule to the lead card, and only blow-formed knows which card leads",
				key, rule.When)
		}
		if rule.If.HasHand && rule.When != MomentBlowFormed {
			return NoRelic, fmt.Errorf("%s narrows a %s rule to a hand, and only blow-formed knows what formed",
				key, rule.When)
		}
		if rule.If.HasHand && (rule.If.HasElement || rule.If.HasForm || rule.If.HasConcept ||
			rule.If.HasTier || rule.If.Lead) {
			return NoRelic, fmt.Errorf("%s narrows a rule by both a hand and a card, and a hand is a fact about the whole blow",
				key)
		}
		if rule.If.HasConcept && (rule.If.Concept < 0 || int(rule.If.Concept) >= ConceptCount()) {
			return NoRelic, fmt.Errorf("%s names a concept the registry does not hold", key)
		}

		for _, e := range rule.Then {
			// **A held card has no place in the blow**, so neither predicate about one applies:
			// `Lead` names the blow's first *played* card and `Hand` names the rung the played
			// cards formed. Either alongside this verb is a rule whose two halves are about
			// different piles.
			// Same seam as Hand: a form count is a fact about the set that formed, and only one
			// moment knows what formed.
			if rule.If.MinForms > 0 && rule.When != MomentBlowFormed {
				return NoRelic, fmt.Errorf("%s counts the forms of a blow at %s, and only %s knows what formed",
					key, rule.When, MomentBlowFormed)
			}
			if e.Do == DoAddDamagePerHeld && (rule.If.Lead || rule.If.HasHand || rule.If.MinForms > 0) {
				return NoRelic, fmt.Errorf("%s pays per held card and also narrows by the blow, and a held card is in neither", key)
			}
			// A per-card step with nothing to count by is grow-on-turn wearing a longer name,
			// and the two would then differ only by how many cards the turn happened to hold.
			if e.Do == DoGrowPerCard && !rule.If.Any() {
				return NoRelic, fmt.Errorf("%s grows per card and names no card to count", key)
			}
			if want := verbMoment(e.Do); want != rule.When {
				return NoRelic, fmt.Errorf("%s does %s at %s, and %s belongs to %s",
					key, e.Do, rule.When, e.Do, want)
			}
			if err := checkEffect(key, e); err != nil {
				return NoRelic, err
			}
		}
	}

	id := RelicID(len(relicRegistry))
	relicRegistry = append(relicRegistry, Relic{Key: key, Name: name, Rules: rules})
	relicBy[key] = id
	return id, nil
}

// checkEffect holds each verb to the figure it needs. **A zero is refused rather than clamped**,
// unlike a worm's amount: a worm is a reward the player chose and a silent nothing would be worse
// than a ceiling, where a relic is authored once and a zero there is a typo.
func checkEffect(key string, e RelicEffect) error {
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
		// Signed on purpose: a discount is negative and a relic with a drawback is expressible.
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

// RelicOf is the relic behind an ID. An unknown ID is a relic with no rules, which does nothing.
func RelicOf(id RelicID) Relic {
	if id < 0 || int(id) >= len(relicRegistry) {
		return Relic{Key: "?", Name: "?"}
	}
	return relicRegistry[id]
}

// RelicByKey finds a registered relic by its record key.
func RelicByKey(key string) (RelicID, bool) {
	id, ok := relicBy[key]
	return id, ok
}

// MustRelic is RelicByKey for callers that would rather fail at startup than wear a relic that does
// nothing.
func MustRelic(key string) RelicID {
	id, ok := relicBy[key]
	if !ok {
		panic("combat: no relic named " + key)
	}
	return id
}

// RelicCount is how many relics are registered.
func RelicCount() int { return len(relicRegistry) }

// RelicKeys is every registered key, sorted, for a tool or a test walking the catalogue without
// depending on registration order.
func RelicKeys() []string {
	out := make([]string, 0, len(relicRegistry))
	for _, r := range relicRegistry {
		out = append(out, r.Key)
	}
	sort.Strings(out)
	return out
}

// MaxWornRelics is the width of a duelist's relic array, and **a width rather than a design cap**
// — exactly like MaxEchoLandings and MaxStatuses. Duelist has to stay comparable, so the hand is a
// fixed array; this is how long that array is and nothing about it is a rule.
//
// **What a duelist may actually wear is DefaultRelicSlots, and it is five.** The two were one number
// until 2026-09-11, which meant the cap could not be moved for a fixture without moving the width
// for the shipped game — so a scenario wanting to look at six relics at once had nowhere to write.
// Splitting them costs three more seats in a handful of Event arrays and buys a cap that a run can
// carry, which is the shape a brand will want anyway.
const MaxWornRelics = 8

// DefaultRelicSlots is how many relics a duelist may wear. **Five, until brands expand it** — see
// MECHANICS.md, where the cap is deliberately never displayed and surfaces when a sixth is bought.
//
// It is what a run opens on, in the way DefaultRoundLimit is: `session.Session` carries the number
// actually in force and hands it to the fighter through Equip, so something that buys a sixth
// finger has one field to move. See Duelist.RelicSlots.
const DefaultRelicSlots = 5

// WornRelic is one relic on a duelist's hand: which relic, and how far its accumulator has grown.
//
// **The accumulator travels with the worn relic rather than living in the registry**, because it
// belongs to a run and the registry belongs to the process. `Session` is what keeps it between
// fights, keyed by record, and sets this field as the duelist is put together — which is also why a
// growing relic is the first relic state that will have to be serialized.
//
// **Every one of the relic's own effect amounts is read as `Amount + Grown`.** A relic holds exactly
// one numeric effect if it grows *(owner's call, 2026-08-17)*, so there is never a question of which
// figure the accumulator feeds.
type WornRelic struct {
	Relic RelicID
	Grown int
}

// relicSlots is how many relics this duelist may wear, with the two ways the field can be wrong
// answered in one place.
//
// **Zero is the default, not "no relics"** — every bare `Duelist{}` in a test and every enemy carries
// a zero here, and reading that as a hand with no fingers would take relics off half the suite. That
// is the opposite reading from RoundLimit's zero, and deliberately so: an unlimited clock is a
// coherent fight and a duelist who can wear nothing is not.
//
// **And it can never exceed the array.** The cap travels from a run, which is free to be wrong; the
// width is a fact about this struct.
func (d Duelist) relicSlots() int {
	n := d.RelicSlots
	if n <= 0 {
		n = DefaultRelicSlots
	}
	if n > MaxWornRelics {
		n = MaxWornRelics
	}
	return n
}

// WornRelics is what this duelist is wearing, in worn order.
//
// **Left to right, and it compounds.** That is a determinism rule rather than a preference:
// multiplicative effects are order-sensitive, so the order has to be one a rule can name, and worn
// order is the only order the player can actually see. Two slash relics are x4 and that is a build.
func (d Duelist) WornRelics() []WornRelic {
	if d.RelicCount <= 0 {
		return nil
	}
	n := d.RelicCount
	if slots := d.relicSlots(); n > slots {
		n = slots
	}
	return d.Relics[:n]
}

// Wearing returns this duelist with one more relic on, or unchanged if the hand is full. It returns a
// copy like everything else in this package.
func (d Duelist) Wearing(w WornRelic) Duelist {
	if d.RelicCount >= d.relicSlots() {
		return d
	}
	d.Relics[d.RelicCount] = w
	d.RelicCount++
	return d
}

// WearsRelic reports whether this duelist has one particular relic on. **A query rather than a flag**
// *(2026-08-17)*: it was an array of bools indexed by element until the grammar landed, which a
// form multiplier had no element to be a bit under.
func (d Duelist) WearsRelic(id RelicID) bool {
	for _, w := range d.WornRelics() {
		if w.Relic == id {
			return true
		}
	}
	return false
}

// RelicEffectsAt is every effect a worn set fires at one moment, in worn order, with each figure
// already carrying its relic's accumulator.
//
// **The card is what an `If` is matched against**, and a zero Card is what the three cardless moments
// pass — a rule with a predicate at one of those is refused at registration, so nothing here has to
// decide what a form means at `fight-start`.
func RelicEffectsAt(worn []WornRelic, m Moment, card Card) []RelicEffect {
	src := RelicContributionsAt(worn, m, card)
	if len(src) == 0 {
		return nil
	}

	out := make([]RelicEffect, 0, len(src))
	for _, c := range src {
		out = append(out, c.Effect)
	}
	return out
}

// RelicContribution is one effect and the worn relic that produced it.
//
// **The pair travels together because a screen needs both and can derive neither**, which is the
// argument appliedStatus already makes one moment over: an effect says a card is doubled, and only
// the relic says what doubled it. A tooltip explaining where a figure came from is a picture of the
// second half.
type RelicContribution struct {
	Relic  RelicID
	Effect RelicEffect
}

// RelicContributionsAt is RelicEffectsAt with each effect still attached to its relic, in worn order.
// **This is the walk**, and RelicEffectsAt is the view of it that does not care where an effect came
// from — two walks would be two chances to disagree about which rules fire.
func RelicContributionsAt(worn []WornRelic, m Moment, card Card) []RelicContribution {
	var out []RelicContribution
	for _, w := range worn {
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != m || !rule.If.Matches(card) {
				continue
			}
			for _, e := range rule.Then {
				e.Amount += w.Grown
				out = append(out, RelicContribution{Relic: w.Relic, Effect: e})
			}
		}
	}
	return out
}

// relicEffects is RelicEffectsAt for the relics this duelist is wearing.
func (d Duelist) relicEffects(m Moment, card Card) []RelicEffect {
	return RelicEffectsAt(d.WornRelics(), m, card)
}

// CardCost is what one card takes out of this duelist's budget, discounts included.
//
// **The relics are read here rather than on the card**, because a cost is a property of the *pairing*:
// the same Strike costs 3 to a duelist wearing the crush discount and 4 to one who is not. `Card.Cost`
// is still the card's own figure and is what a contact sheet or a deck panel draws.
func (d Duelist) CardCost(c Card) int { return CostWith(d.WornRelics(), c) }

// CostWith is CardCost for a caller that has a worn set and no duelist — the post-battle screen
// drawing a card out of the run deck, which has to print the price the fight will actually charge.
//
// **A card face and the AP bar must never disagree**, which is the whole reason a cost is asked of
// the wearer rather than of the card: three dashes on a card the budget charges two for is a screen
// contradicting the engine.
func CostWith(worn []WornRelic, c Card) int {
	cost := c.Cost()
	for _, e := range RelicEffectsAt(worn, MomentCardCost, c) {
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
// defence — the card's own figure scaled by every relic that matches it.
//
// **Compounding, left to right**, which is what makes two matching relics x4 rather than x2. The floor
// is the one `Card.Damage` holds for the same reason: a card that is meant to deal nothing is not an
// attack.
func (d Duelist) CardDamage(c Card) int {
	dmg := c.Damage(d.DMG)
	if dmg == 0 {
		return 0
	}
	for _, e := range d.relicEffects(MomentCardDamage, c) {
		// **The per-vitae scaler is the one that is not a plain percentage.** Its Amount is
		// percentage points *per vitae*, read against the duelist's live purse, so it composes
		// into the same left-to-right product as everything else rather than needing its own pass.
		if e.Do == DoScaleDamagePerVitae {
			dmg = dmg * (100 + e.Amount*d.Vitae) / 100
			continue
		}
		dmg = dmg * e.Amount / 100
	}
	if dmg < 1 {
		dmg = 1
	}
	return dmg
}

// statusesFrom is every status this duelist's relics put on a target for a landed blow, in the order
// they will be applied.
//
// **Deduplicated, so one blow lands one of each.** Two fire cards in a hand match a fire relic twice,
// and applying a status twice is the same as applying it once — see the no-stacking rule — but it
// would announce itself twice, and a feed saying "sets them burning" twice for one blow is a feed
// describing two things that did not happen.
//
// **Relic order outer, cards inner**, per the worn-order rule.
//
// **It says which relic applied each status, not just which statuses landed** *(2026-08-18)*. It
// always knew - the relic is the thing being walked - and threw the answer away, which left the
// screen unable to fly a CHILLED out of the relic that caused it without inventing an element-to-relic
// table of its own. That table would be a second rule about the same thing, and it would be wrong
// the first time a form relic or a concept relic applied a status, which this grammar already
// allows. See Event.Relic.
//
// **The first relic to apply a status is the one credited**, which falls out of the dedup and out of
// worn order being left to right. Two relics that both set something burning are one burn, and it
// belongs to the one worn first - the same tie-break every other compounding effect takes.
func (d Duelist) statusesFrom(cards []Card) []appliedStatus {
	var out []appliedStatus
	seen := make(map[StatusID]bool)

	for _, w := range d.WornRelics() {
		for _, rule := range RelicOf(w.Relic).Rules {
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
					out = append(out, appliedStatus{Relic: w.Relic, Status: e.Status})
				}
			}
		}
	}
	return out
}

// appliedStatus is one status a blow lands, and the worn relic that put it there.
//
// **The pair travels together because the screen needs both and can derive neither.** A status
// names what landed; only the relic names where it came from, and "where it came from" is a card
// the player is looking at.
type appliedStatus struct {
	Relic  RelicID
	Status StatusID
}

// AddedDMG and AddedHP are what a worn set adds for the fight about to start. They take the worn
// slice rather than a duelist because `session` applies them while the duelist is still being put
// together — the stat they add to is the one that has not been set yet.
func AddedDMG(worn []WornRelic) int { return sumAmounts(worn, MomentFightStart, DoAddDMG) }

// HPScale is what every worn relic does to maximum life, as a percentage — 100 when nothing scales
// it. **Compounding left to right**, like every other multiplicative relic effect, so two relics each
// taking a quarter off leave 56% rather than half.
func HPScale(worn []WornRelic) int {
	out := 100
	for _, e := range RelicEffectsAt(worn, MomentFightStart, Card{}) {
		if e.Do == DoScaleHP {
			out = out * e.Amount / 100
		}
	}
	return out
}

// LandingAmounts is what one card of a blow pays, term by term: its own damage first, then a term
// for every extra landing its relics buy. One entry when nothing repeats or echoes it, which is
// almost every card in the game.
//
// **Two verbs land here and they stack in a fixed order** *(2026-08-22)*: `repeat-card` adds
// full-strength copies, then `echo-attack` adds its diminishing ladder. Repeats first because they
// are the card being played again — an echo of a repeated card would be an echo of something that
// already happened twice, which is a fact about the blow rather than about the card.
//
// **Extra landings add rather than compound**, so two relics landing a card three times land it five
// times, not nine. `MaxEchoLandings` is the ceiling, and it is a width on the event's arrays as much
// as a rule.
//
// `lead` says whether this is the blow's first attack card, which is the only thing the `Lead`
// predicate reads.
func LandingAmounts(worn []WornRelic, card Card, lead bool, damage int) []int {
	shape := LandingsOf(worn, card, lead)

	out := make([]int, 0, shape.Count())
	for j := 0; j < shape.Count(); j++ {
		out = append(out, shape.Amount(j, damage))
	}
	return out
}

// LandingShape is how many times one card lands inside a blow and how those landings are priced:
// full-strength copies from `repeat-card`, then the diminishing ladder from `echo-attack`.
//
// **It is separate from the figures as of 2026-08-26**, because the figures are no longer decided
// once. A growing relic now steps between the landings of a blow, so each landing's base damage is
// asked for again at the accumulator the previous one left behind — where the *shape* is settled
// when the card is reached and does not move under it. Splitting the two is what lets the sum walk
// forward without an echo ladder that changes length halfway down.
type LandingShape struct {
	// Copies is how many extra full-strength landings, and Echoes how many diminishing ones. Both
	// are already clamped against MaxEchoLandings.
	Copies, Echoes int
}

// Count is how many times the card lands in total, the original included.
func (s LandingShape) Count() int { return 1 + s.Copies + s.Echoes }

// Amount is what the j-th landing pays, counting from zero, given what the card is worth at the
// moment that landing is counted.
//
// **The original and every repeat are full strength; the echoes count down.** j is a position in the
// blow rather than a rung of the ladder, so the echo index is measured from the end of the repeats.
func (s LandingShape) Amount(j, damage int) int {
	if j <= s.Copies {
		return damage
	}
	return EchoBonus(damage, j-s.Copies+1, s.Echoes+1)
}

// LandingSeats reports which worn seats are the reason a card lands more than once — the relics whose
// `repeat-card` or `echo-attack` rules matched it.
//
// **It is what makes an extra landing attributable** *(owner's call, 2026-08-26)*. Every figure in a
// blow's sum is accompanied by the card that produced it shaking; a card's own damage comes from the
// card and a multiplier comes from its relic, but an echo's extra *term* is produced by a relic that
// contributes no multiplier at all — so without this the Echo Ring would sit still through the three
// terms it is single-handedly responsible for.
//
// **Every contributing seat, not one.** Extra landings add across relics, so two echo relics are both
// the reason and both shake.
func LandingSeats(worn []WornRelic, card Card, lead bool) [MaxWornRelics]bool {
	var out [MaxWornRelics]bool

	for seat, w := range worn {
		if seat >= MaxWornRelics {
			break
		}
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentBlowFormed || !rule.If.Matches(card) {
				continue
			}
			if rule.If.Lead && !lead {
				continue
			}
			for _, e := range rule.Then {
				if e.Amount+w.Grown < 2 {
					continue
				}
				switch e.Do {
				case DoRepeatCard, DoEchoAttack:
					out[seat] = true
				}
			}
		}
	}
	return out
}

// HandBonus is the flat damage a worn set adds to a blow for the rung it formed, and which seats
// are the reason.
//
// **It is asked of the hand and not of a card**, which is what separates it from every other
// blow-formed verb: `repeat-card` and `echo-attack` both answer "what does this relic do to this
// card", where this answers "what does this relic do because the turn built a Full House". That is
// why `RelicCondition.Hand` is read here rather than inside `Matches`.
//
// **Two relics naming one rung add.** They are flat terms in a sum, so there is nothing to compound
// — unlike the multipliers, where worn order decides the result.
//
// The seats come back for the same reason `LandingSeats` reports them: the bonus is a term in the
// bracket that no card produced, so without this the relic paying for it would sit still while its
// own figure landed.
// DamagePerVitae is the flat damage every blow gains for each vitae the run holds, summed over the
// worn relics that say so.
//
// **It is the rate, not the payment.** The multiplication is done at each blow against the
// duelist's live purse, because vitae moves inside a fight: a card kept in hand pays one. Asking
// once at fight-start was the first version of this and it was wrong.
func DamagePerVitae(worn []WornRelic) int {
	total := 0
	for _, w := range worn {
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentFightStart {
				continue
			}
			for _, e := range rule.Then {
				if e.Do == DoAddDamagePerVitae {
					total += e.Amount
				}
			}
		}
	}
	return total
}

// seatsDoing is which worn seats carry a verb at all — the attribution a flat fight-start term
// cannot recover from its own figure, since by the time a blow lands it is one number.
func seatsDoing(worn []WornRelic, do RelicVerb) [MaxWornRelics]bool {
	var seats [MaxWornRelics]bool
	for seat, w := range worn {
		if seat >= MaxWornRelics {
			break
		}
		for _, rule := range RelicOf(w.Relic).Rules {
			for _, e := range rule.Then {
				if e.Do == do {
					seats[seat] = true
				}
			}
		}
	}
	return seats
}

// HandScale is what the worn relics multiply a blow by **after its hand multiplier has been
// applied**, in percent, and which seats paid for it. 100 is the identity.
//
// **It is deliberately not folded into the hand's own multiplier.** That figure belongs to the
// ladder and is what the banner and the hand row say; see DoScaleHandDamage.
//
// **Relics compound, left to right by seat**, like every other multiplier in the game: two 200 relics
// on one rung are 4x rather than 3x, and which seat a relic sits in therefore matters exactly as
// much as it does for the card scalers.
//
// **`cards` is the scoring set, not the turn** — it is what `MinForms` counts, so a card the turn
// played that paid nothing into the hand is not a second form.
func HandScale(worn []WornRelic, hand HandID, cards []Card) (int, [MaxWornRelics]bool) {
	var seats [MaxWornRelics]bool
	pct := 100

	if hand == HandNone {
		return pct, seats
	}
	for seat, w := range worn {
		if seat >= MaxWornRelics {
			break
		}
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentBlowFormed {
				continue
			}
			if rule.If.HasHand && rule.If.Hand != hand {
				continue
			}
			if rule.If.MinForms > 0 && distinctForms(cards) < rule.If.MinForms {
				continue
			}
			for _, e := range rule.Then {
				if e.Do != DoScaleHandDamage {
					continue
				}
				pct = pct * e.Amount / 100
				seats[seat] = true
			}
		}
	}
	return pct, seats
}

// distinctForms is how many different forms a set of cards covers. **Defend counts like any other**
// — a shield in the scoring set is a second weapon by this reckoning, and the rungs that can hold
// one are the elemental and card hands rather than the form ones.
func distinctForms(cards []Card) int {
	// A fixed array rather than a map: this runs inside every blow, and a map here would be an
	// allocation a round does not need. FormDefend is the last of the enum.
	var seen [FormDefend + 1]bool
	n := 0
	for _, c := range cards {
		f := c.Form()
		if f == FormNone || int(f) >= len(seen) || seen[f] {
			continue
		}
		seen[f] = true
		n++
	}
	return n
}

func HandBonus(worn []WornRelic, hand HandID) (int, [MaxWornRelics]bool) {
	var seats [MaxWornRelics]bool
	total := 0

	if hand == HandNone {
		return 0, seats
	}
	for seat, w := range worn {
		if seat >= MaxWornRelics {
			break
		}
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentBlowFormed || !rule.If.HasHand || rule.If.Hand != hand {
				continue
			}
			for _, e := range rule.Then {
				if e.Do != DoAddHandDamage {
					continue
				}
				total += e.Amount
				seats[seat] = true
			}
		}
	}
	return total, seats
}

// HeldBonus is the flat damage a worn set adds to a blow for the cards the turn **kept back**, and
// which seats are the reason.
//
// **The predicate is matched against each held card**, so `{ Element: "fire" }` means "for every
// fire card still in hand" and the amount is paid once per match. That is the same reading a
// `card-damage` rule takes, applied to the other pile.
//
// **A card kept back is not spent**, so this pays again on every turn it is still being held — the
// same property the parasite riders have, and for the same reason: it is a fact about the hand at
// the moment the blow is added up rather than an event.
func HeldBonus(worn []WornRelic, held []Card) (int, [MaxWornRelics]bool) {
	var seats [MaxWornRelics]bool
	total := 0

	for seat, w := range worn {
		if seat >= MaxWornRelics {
			break
		}
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentBlowFormed {
				continue
			}
			for _, e := range rule.Then {
				if e.Do != DoAddDamagePerHeld {
					continue
				}
				for _, c := range held {
					if rule.If.Matches(c) {
						total += e.Amount
						seats[seat] = true
					}
				}
			}
		}
	}
	return total, seats
}

// LandingsOf is how many times a card lands, and how, given what its wearer has on.
//
// **Two verbs land here and they stack in a fixed order** *(2026-08-22)*: `repeat-card` adds
// full-strength copies, then `echo-attack` adds its diminishing ladder. Repeats first because they
// are the card being played again — an echo of a repeated card would be an echo of something that
// already happened twice, which is a fact about the blow rather than about the card.
//
// **Extra landings add rather than compound**, so two relics landing a card three times land it five
// times, not nine. `MaxEchoLandings` is the ceiling, and it is a width on the event's arrays as much
// as a rule.
//
// `lead` says whether this is the blow's first attack card, which is the only thing the `Lead`
// predicate reads.
func LandingsOf(worn []WornRelic, card Card, lead bool) LandingShape {
	copies, echoes := 0, 0
	for _, w := range worn {
		for _, rule := range RelicOf(w.Relic).Rules {
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

	if total := 1 + copies + echoes; total > MaxEchoLandings {
		// Repeats are kept ahead of echoes when the ceiling bites, since a full-strength landing
		// is the one the player paid for.
		if copies > MaxEchoLandings-1 {
			copies = MaxEchoLandings - 1
		}
		echoes = MaxEchoLandings - 1 - copies
	}

	return LandingShape{Copies: copies, Echoes: echoes}
}

// EchoBonus is what one echoed landing is worth: the k-th landing of a card that lands n times,
// where k counts from 1 and k=1 is the full-strength original.
//
// **Even fractions counting down** — at n=3 that is the card, two thirds of it, one third of it —
// which is what lets a relic say the whole ladder with one number. Never below 1, for the reason
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
func AddedHP(worn []WornRelic) int { return sumAmounts(worn, MomentFightStart, DoAddHP) }

// AddedPicks is how many extra post-battle choices a worn set offers.
func AddedPicks(worn []WornRelic) int { return sumAmounts(worn, MomentPrizesDealt, DoAdjustPicks) }

// AddedPrizeVitae is what a worn set adds to a won room's vitae award. **Flat, not a percentage** — Soul
// Taker turns 5 into 10 rather than doubling whatever the card happens to pay.
func AddedPrizeVitae(worn []WornRelic) int {
	return sumAmounts(worn, MomentPrizesDealt, DoAdjustPrizeVitae)
}

func sumAmounts(worn []WornRelic, m Moment, do RelicVerb) int {
	total := 0
	for _, e := range RelicEffectsAt(worn, m, Card{}) {
		if e.Do == do {
			total += e.Amount
		}
	}
	return total
}

// Growth is what one worn relic's accumulator gains from a win. It is per relic rather than summed
// because each relic grows its own — see WornRelic.
func Growth(w WornRelic) int {
	total := 0
	for _, rule := range RelicOf(w.Relic).Rules {
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
// the turn matched has grown or reset this relic's accumulator.
//
// **A rule fires when any card of the turn matches it**, and a rule with no `If` fires on every
// turn — including an empty one, which is still a turn taken. That is what lets Momentum be written
// without a "not" in the predicates: one rule grows on every turn, a second resets on a turn holding
// a defend card, and the reset is applied second so a defending turn nets zero.
//
// **Growth first, then resets**, always. The other order would let a turn both bank and lose the
// same step depending on which rule the file happened to list first.
func (d Duelist) TurnTaken(cards []Card) Duelist {
	for i := 0; i < d.RelicCount; i++ {
		step, reset := 0, false
		for _, rule := range RelicOf(d.Relics[i].Relic).Rules {
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
				case DoGrowPerCard:
					step += e.Amount * countMatches(rule.If, cards)
				case DoResetGrowth:
					reset = true
				}
			}
		}

		d.Relics[i].Grown += step
		if reset {
			d.Relics[i].Grown = 0
		}
	}
	return d
}

// countMatches is how many cards of a turn satisfy a condition — the counting counterpart of
// anyMatches, and the whole difference between grow-per-card and grow-on-turn.
func countMatches(c RelicCondition, cards []Card) int {
	n := 0
	for _, card := range cards {
		if c.Matches(card) {
			n++
		}
	}
	return n
}

// anyMatches reports whether any card of a turn satisfies a condition. **Any rather than every**,
// which is the reading a turn-wide predicate needs: "a turn with a defend card in it".
func anyMatches(c RelicCondition, cards []Card) bool {
	for _, card := range cards {
		if c.Matches(card) {
			return true
		}
	}
	return false
}

// KeepsGrowth reports whether a relic's accumulator belongs to the **run** rather than to one fight.
//
// **A relic that can reset itself does not keep anything** *(2026-08-22)*: Momentum's streak is a
// fact about the turns of one duel, and banking it between fights would make it a permanent bonus
// that a single defend card once wiped. Heart, the growing stat relics and the Enflamed family hold no
// reset and are kept.
func KeepsGrowth(id RelicID) bool {
	for _, rule := range RelicOf(id).Rules {
		for _, e := range rule.Then {
			if e.Do == DoResetGrowth {
				return false
			}
		}
	}
	return true
}

// GrowOnLanding is the attacker after **one landing of one card** has been counted into the blow.
//
// **It steps inside the sum as of 2026-08-26** *(owner's call)*, where it used to be one step taken
// after the whole blow had landed. The rule the change buys is that the order of the cards in a turn
// decides what they are worth: two fire cards no longer both fire at the relic's opening figure — the
// first fires bare, steps the relic, and the second fires at the bigger multiplier. That makes the
// queue an ordering decision the player is meant to make, which is the whole point of it.
//
// **Landings, not cards** *(owner's call, 2026-08-22, and it survives the move)*. A card that an
// echo or a repeat seats three times *hit* three times, so it steps three times — and each of those
// landings is counted at the figure the one before it left, so the ladder compounds inside itself.
// That is the combination the relics are for: Echo plus Enflamed is meant to be a build.
//
// **It returns a duelist rather than writing through a pointer**, like everything else in this
// package — a round is resolved by passing duelists along, and an accumulator that moved by side
// effect would be the one piece of fight state a replay could not reproduce.
//
// **What it grows is the copy the fight is holding.** `Session` keeps the run's own figure and reads
// it back off the duelist when the fight is won — see Session.AbsorbGrowth — which is what makes the
// growth survive the fight without combat knowing a run exists.
func (d Duelist) GrowOnLanding(card Card) Duelist {
	for i := 0; i < d.RelicCount; i++ {
		step := 0
		for _, rule := range RelicOf(d.Relics[i].Relic).Rules {
			if rule.When != MomentAttackLands || !rule.If.Matches(card) {
				continue
			}
			for _, e := range rule.Then {
				if e.Do == DoGrowOnHit {
					step += e.Amount
				}
			}
		}
		d.Relics[i].Grown += step
	}
	return d
}

// CardScaleBySeat is what **each worn relic** is multiplying one card's damage by right now, as a
// percent per worn seat, and 0 for a seat that does not touch the card at all.
//
// **It is per seat rather than a product** *(owner's call, 2026-08-26)*, because the sum shows the
// relics one at a time: each contributing relic says its own figure beside the term and the relic's own
// card bounces on the beat. A combined multiplier would say what the term came to and leave the
// player to work out which of five fingers produced it.
//
// **Every damage relic, not only the growing ones.** A first pass showed growth alone and the fire
// relic's doubling stayed invisible — baked into the term's figure with nothing on screen accounting
// for it. Nothing reaches a card's printed damage any more, so the sum is the only place any of it
// can be seen, and a relic that fires without saying so is a rule the player has to take on trust.
//
// **Zero is "did not fire", which is why the identity is not stored.** A relic whose rule does not
// match the card contributes nothing and has no beat; a relic contributing exactly 100 has fired and
// changed nothing, which no relic in the file does but which the grammar allows.
func CardScaleBySeat(worn []WornRelic, card Card) [MaxWornRelics]int {
	var out [MaxWornRelics]int

	for seat, w := range worn {
		if seat >= MaxWornRelics {
			break
		}
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentCardDamage || !rule.If.Matches(card) {
				continue
			}
			for _, e := range rule.Then {
				if e.Do != DoScaleDamage {
					continue
				}
				if out[seat] == 0 {
					out[seat] = 100
				}
				out[seat] = out[seat] * (e.Amount + w.Grown) / 100
			}
		}
	}
	return out
}

// MoveRelic slides the relic at `from` to sit at `to`, shuffling everything between them along.
//
// **Worn order is firing order**, so this changes what the duelist's relics do — see WornRelics. It is
// the live counterpart of Session.MoveRelic: a reorder made during a fight has to reach the duelist
// the fight is holding, or the row on screen and the rules would disagree until the next fight.
//
// **Accumulators travel with their relics**, because the number belongs to the relic and not to the
// finger. An out-of-range index is a no-op rather than a panic: this is driven by a drag, and a drop
// resolved against a row that has changed underneath it must not take the frame with it.
func (d Duelist) MoveRelic(from, to int) Duelist {
	n := len(d.WornRelics())
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return d
	}

	w := d.Relics[from]
	if from < to {
		copy(d.Relics[from:to], d.Relics[from+1:to+1])
	} else {
		for i := from; i > to; i-- {
			d.Relics[i] = d.Relics[i-1]
		}
	}
	d.Relics[to] = w
	return d
}

// ScalePropagation applies every propagation-scaling relic to a figure the run's own rule already
// produced and capped.
//
// **The cap binds the base rate and the relic scales what the cap produced** *(owner's call,
// 2026-08-17)*. An absolute cap on the figure that finally lands would leave Banker doing nothing
// past 25 held — a relic that stops working exactly when a run can afford it.
//
// Left to right and compounding, like every other relic effect.
func ScalePropagation(worn []WornRelic, base int) int {
	for _, e := range RelicEffectsAt(worn, MomentFightWon, Card{}) {
		if e.Do == DoScalePropagation {
			base = base * e.Amount / 100
		}
	}
	return base
}

// DemoteConcept is which concept a card is dealt as, given a worn set. It reports false when no
// relic steps it, so a caller can leave the card alone.
//
// **It reads the card as the run owns it, exactly like FlipElement**, so two demoting relics cannot
// walk one card two rungs down the ladder between them — the deepest single step wins and worn
// order decides a tie. A relic wanting two rungs says `Amount: 2`.
//
// **A card with no rung below it is left where it is.** Atrophy on a hand of Jabs is a relic doing
// nothing, which is a fact about that hand rather than a case to special-case.
func DemoteConcept(worn []WornRelic, card Card) (ConceptID, bool) {
	deepest := 0
	for _, e := range RelicEffectsAt(worn, MomentDeckBuilt, card) {
		if e.Do == DoDemoteCard && e.Amount > deepest {
			deepest = e.Amount
		}
	}
	if deepest == 0 {
		return NoConcept, false
	}
	return Neighbour(card.Concept, -deepest)
}

// FlipElement is what colour a card is dealt as, given a worn set. It reports false when no relic
// touches it, so a caller can leave the card alone rather than writing its own colour back over it.
//
// **Every flip reads the card's original element**, which is what stops two of them chaining a deck
// to one colour: the later relic matches on what the card *is*, not on what the earlier relic made it.
// The last matching flip wins, and worn order is what decides which that is.
//
// **"Original" is now a duty the caller carries** *(2026-08-24)*. While this fired at `deck-built`
// it was true by construction — the fight deck was built out of the run's own cards, once, and
// nothing had flipped anything yet. Firing per draw, the discard pile holds cards that have already
// been through here, so a caller that folds the discard back into the draw pile and hands those
// cards to this function is asking the second flip to read the first one's answer. The combat
// screen restores a card to the face the run owns before it can be drawn again; see
// screens/combat_deck.go.
func FlipElement(worn []WornRelic, card Card) (Element, bool) {
	out, flipped := Basic, false
	for _, e := range RelicEffectsAt(worn, MomentCardDrawn, card) {
		if e.Do == DoSetElement {
			out, flipped = e.Element, true
		}
	}
	return out, flipped
}

// Grows reports whether a relic holds an accumulator at all — a rule with any of the three growth
// verbs on it.
//
// **It is the question a badge asks**, and it is deliberately separate from KeepsGrowth: that one
// says whether the number survives the fight, this one says whether there is a number.
func Grows(id RelicID) bool {
	for _, rule := range RelicOf(id).Rules {
		for _, e := range rule.Then {
			switch e.Do {
			case DoGrowOnWin, DoGrowOnTurn, DoGrowOnHit, DoGrowPerCard:
				return true
			}
		}
	}
	return false
}

// Scaling reports whether a verb's Amount is a percentage rather than a flat figure. The three
// scaling verbs read 100 as "unchanged"; everything else reads 0 as "nothing".
//
// **It exists so a screen can say what a figure means without a second table of verbs.** A badge
// printing `50` where the relic is doing 1.5x, or `1.5x` where it is doing +150 HP, is a screen
// contradicting the engine — the same failure a cost on a card face is guarded against.
func Scaling(do RelicVerb) bool {
	switch do {
	case DoScaleDamage, DoScaleHP, DoScalePropagation:
		return true
	}
	return false
}

// GrowthEffect is the one numeric effect a growing relic's accumulator feeds, **at the size the
// accumulator has reached**: the effect's own Amount plus Grown, with the verb still attached so the
// caller knows whether that figure is a percentage.
//
// **A growing relic holds exactly one numeric effect** *(owner's call, 2026-08-17)*, which is what
// makes this answerable at all — there is never a question of which figure the accumulator feeds.
// The growth verbs themselves are skipped: `grow-on-hit 10` is the *step*, not the effect, and a
// badge showing the step would print the same number all run.
//
// It reports false for a relic that does not grow, and for one whose only effects carry no figure —
// a status or a flip. Neither exists today; refusing is what stops a badge inventing a number.
func GrowthEffect(w WornRelic) (RelicEffect, bool) {
	if !Grows(w.Relic) {
		return RelicEffect{}, false
	}
	for _, rule := range RelicOf(w.Relic).Rules {
		for _, e := range rule.Then {
			switch e.Do {
			case DoGrowOnWin, DoGrowOnTurn, DoGrowOnHit, DoGrowPerCard, DoResetGrowth,
				DoApplyStatus, DoSetElement:
				continue
			}
			e.Amount += w.Grown
			return e, true
		}
	}
	return RelicEffect{}, false
}

// CounterLabel is the accumulator badge's figure: what a growing relic is doing right now, written
// in the units its own effect is written in. A relic that does not grow reads as the empty string,
// which is most of the catalogue and is what says "draw nothing".
//
// **It is here rather than in the screen because it has two readers** — the worn-ring row and
// `tools/relicsheet`, which cannot import a package that links a window. Two copies of this
// formatting is two badges that can disagree about the same relic, which is the failure
// `tools/hands` exists to prevent one axis over.
//
// It is formatting and not a rule: it reads figures the resolver produced and computes none.
func CounterLabel(w WornRelic) string {
	e, ok := GrowthEffect(w)
	if !ok {
		return ""
	}
	if Scaling(e.Do) {
		// **No `x` after it** *(owner’s call, 2026-09-09)*. The decimal point is what says this is
		// a multiplier, and the `+` on the other branch is what says the flat one is not — so the
		// letter was a third of the badge’s width spent restating what the figure already reads
		// as, on the one card whose number is meant to be caught at a glance.
		return fmt.Sprintf("%.1f", float64(e.Amount)/100)
	}
	return fmt.Sprintf("%+d", e.Amount)
}
