package combat

// Relics, as rules. The grammar, the three closed vocabularies, and the four moments that fire
// inside this package.
//
// **A relic is the only collected thing that is never played** *(2026-08-17)*. A card resolves in the
// turn you queued it, an essence fires when you pick it, a hand is scored when the attack phase runs —
// each already knows *when* it happens. A relic waits, so it has to say so itself, and that is the
// third part the card language does not need. `.claude/skills/relics/SKILL.md` is the whole grammar;
// MECHANICS.md holds the argument for its shape.
//
// **A relic is a list of `When` / `If` / `Then` rules.** A list rather than one rule because a
// growing stat relic needs two moments — one to accumulate and one to apply — and `Then` is a list
// too, which is what buys a relic that shocks *and* chills with no new vocabulary.
//
// **This package holds the vocabulary and refuses a rule that misuses it; it does not read
// `relics.json`.** The file lives beside the essences in `internal/session`, which parses the strings
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
	// 2026-08-24)*. It used to be a `deck-built` verb, recoloring the whole fight deck once as it
	// came out of the run — which deals the same cards, since a flip is unconditional over an
	// element, and says the wrong thing about *when*. Every one of these relics is worded "every X
	// card is dealt as a Y card", and dealing is what a draw is.
	//
	// **The draw pile therefore holds cards as the run owns them**, and the flip is applied on the
	// way into the hand. That is the invariant the reshuffle has to keep: a discarded card is put
	// back as the run owns it, or a second flip would land on the color the first one made and two
	// relics would chain a deck to one color between them. See screens/combat_deck.go.
	//
	// **A card drawn under a flip does not remember what it was.** It carries the color it became
	// and nothing else, so a rule firing later — a card-damage relic keyed on ice — matches the card
	// in the hand rather than the card in the run. What the original is still reachable *from* is
	// the card's ID, which is a handle for the layers above the rules and never something a rule
	// reads.
	//
	// **Appended, because the enum is append-only.**
	MomentCardDrawn

	// MomentTurnStart fires once at the top of each of this duelist's own turns, in playTurn,
	// **before the chill, the riders and both phases** — so a relic that puts life back does it in
	// time for the turn it is about to survive.
	//
	// **It is MomentTurnTaken's other end and it is deliberately a second moment rather than a
	// flag on that one.** What a turn-taken rule reads is the turn: Momentum grows because a turn
	// held no defense, which is a fact that does not exist until the turn is over. A turn-start
	// rule has no turn to read at all, which is why it takes no predicate.
	//
	// **It has no card, so it reads none** — see readsACard. A rule carrying any `If` here is
	// refused at registration rather than matching everything: "heal on a turn with a fire card in
	// it" is a sentence this moment cannot answer, because the cards have not been committed to
	// anything yet.
	//
	// **Appended, because the enum is append-only.**
	MomentTurnStart

	// MomentEssenceSpent fires as an essence is pointed at the run's deck — the reward screen's
	// offer, the shop's vial, and one carried into a duel out of the satchel.
	//
	// **It is a question rather than an event, which is the shape MomentPrizesDealt already has.**
	// Nothing about the round or the run has happened yet; what the moment answers is how many
	// cards the essence about to be spent may take, and Session.EssenceTargets is the seat all
	// three spend sites ask through.
	//
	// **It has no card, so it reads none** — see readsACard. A rule narrowed to a fire card here
	// would be asking about a card the player has not picked yet.
	//
	// **Appended, because the enum is append-only.**
	MomentEssenceSpent
)

// Moments is every moment in a fixed order, for anything that walks them.
func Moments() []Moment {
	return []Moment{MomentCardCost, MomentCardDamage, MomentAttackLands, MomentDeckBuilt,
		MomentFightStart, MomentFightWon, MomentPrizesDealt, MomentBlowFormed, MomentTurnTaken,
		MomentCardDrawn, MomentTurnStart, MomentEssenceSpent}
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
	case MomentTurnStart:
		return "turn-start"
	case MomentEssenceSpent:
		return "essence-spent"
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
	case MomentTurnStart:
		// **Nothing has been committed yet**, so there is no card and no turn to ask about. A
		// predicate here would have to match everything, which is the silent-rule failure
		// readsACard exists to refuse.
		return false
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

	// DoSetElement is the flip: it recolors a matching card **as that card is drawn**.
	//
	// **The only verb at MomentCardDrawn**, and it moved there on 2026-08-24 from `deck-built`,
	// where it recolored the whole fight deck in one pass. The cards dealt are the same either way
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
	// fight's deck is dealt: a 3 AP Skewer becomes a 2 AP Thrust, same form, one rung cheaper and
	// half the damage.
	//
	// **It walks `Neighbor`, so the ladder stays a consequence of `duelist_cards.json`** rather
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

	// DoAddHandDMG adds Amount to the duelist's **DMG** for the length of one blow, when the blow
	// satisfied the rung named by the rule's `Hand` predicate.
	//
	// **It is base damage, not a term** *(owner's call, 2026-09-14)*. It was `add-hand-damage` and
	// a flat addition to `Base` from 2026-09-05 until then, which paid the same 2 whether the Pair
	// was two Jabs or two Skewers. What it does now is raise the figure every card of the hand
	// multiplies: a duelist on 14 swings a Pair at 16, so a 1x card in it gains 2 and a 0.5x card
	// gains 1, and the relic is worth more to a hand that is worth more. The name says `dmg` rather
	// than `damage` for exactly that reason — DMG is the duelist's stat, damage is a figure in a
	// sum. See combat.blowDMG, which is where it is folded in, and Event.HandBonus, which reports
	// it so a screen can *say* it without adding it a second time.
	//
	// **It is still inside the multiplier**, since it moves the cards the multiplier multiplies.
	//
	// **Two relics naming one rung add rather than compound**, and a blow can satisfy several rungs
	// at once — see Blow.Satisfied — so a Pair relic and a Three of a Kind relic both pay on trips.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoAddHandDMG

	// DoAddDamagePerHeld adds Amount to the blow for **every card still in the hand** that matches
	// the rule's predicate — the cards kept back, not the cards played.
	//
	// **It is the one verb that pays for what a turn did not do** *(owner's call, 2026-09-05)*.
	// Every other element and form relic pays for spending a card; this pays for holding one, so a
	// run wearing both is being pulled in two directions on purpose.
	//
	// **The held hand is already something the rules see** — `blowDMG` reads it for the rune
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
	// names — the rung relics that multiply, where add-hand-dmg is the pair that raises the duelist.
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

	// DoDrainDamage restores Amount percent of a landed blow to the duelist who threw it — the
	// first verb in the grammar that gives a relic's wearer life back.
	//
	// **Once per blow, not once per card**, which is DoApplyStatus's rule rather than DoGrowOnHit's:
	// the share is taken out of the blow, and a blow is one figure however many cards went into it.
	// A rule carrying a predicate asks whether *any* card of the blow matched, so an elemental drain
	// is "your fire hands drain" rather than a share paid per fire card.
	//
	// **It reads the figure that landed**, after weight, after the target's vulnerability, and after
	// the shield and the miss — so a blow that was eaten or never thrown drains nothing, and a
	// drain is never worth more than the blow the player watched.
	//
	// **Two relics add rather than compound**, like every other flat share of one figure.
	//
	// **Capped at full life by `restore`**, which is what keeps it a comeback rather than a ceiling:
	// nothing in the game heals above full.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoDrainDamage

	// DoHealShare restores Amount percent of the duelist's **maximum** life at the top of their own
	// turn — the regeneration verb, where DoDrainDamage is the one that takes its life off a blow.
	//
	// **Of the maximum, never of what is left.** A share of the current life is worth least at the
	// moment a duelist needs it and most when they need nothing, which is a comeback relic that
	// does not come back. This one is the same figure however badly the fight is going.
	//
	// **Two relics add rather than compound**, like every other share of one figure.
	//
	// **Capped at full life by `restore`**: nothing in the game heals above full, so this is worth
	// nothing to a duelist who has not been hit and everything to one who has.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoHealShare

	// DoScaleRolls scales **the numerator of every roll in the rules** by Amount percent: 200 is
	// twice as many winning faces on a gold or a silver card, and twice as likely to miss while
	// shocked.
	//
	// **The numerator, never the denominator.** Halving a d5 to a d2 is the obvious reading and it
	// is a trap: LuckOutcomes refuses a die with under three faces, so the card would silently stop
	// paying altogether. Widening the paying band leaves the die alone.
	//
	// **It must not change how many times a stream is drawn from.** Rolling twice and taking the
	// better would advance the luck cursor twice, so wearing this would reroll every later gamble
	// in the run — which is the determinism rule in the randomness skill, met by scaling the
	// comparison rather than the number of draws. See rollGolden and attackMisses, both of which
	// still take exactly one sample.
	//
	// **It reaches the shock as well, and that is a drawback rather than an oversight**
	// *(owner's call)*. attackMisses reads the *acting* duelist's own miss chance, so a worn relic
	// can only ever double its wearer's own chance of whiffing — there is no way from here to make
	// a shocked opponent miss more. A relic about luck that is only ever lucky would be the
	// dishonest version.
	//
	// **Two of them compound**, like DoScaleHP and unlike the flat shares — it is a multiplier.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoScaleRolls

	// DoAdjustRoundLimit moves **this fight's clock** by Amount rounds: -2 takes two off, +1 buys one.
	//
	// **A delta, never a figure** *(owner's call, 2026-09-19)*. A relic naming three rounds outright
	// could not be mixed with one that buys rounds — whichever was read last would simply win, and
	// which that was would depend on nothing the player can see. Deltas sum, so a relic taking two
	// and a relic giving one leave a fight one round shorter and both sentences stay true.
	//
	// **Summing is also what makes worn order irrelevant here**, which is the one place a relic verb
	// steps outside left-to-right compounding. Addition commutes; a figure would not, and a drawback
	// a second relic could cancel by sitting to its right is not a drawback.
	//
	// **Signed, so Amount may not be zero** — see checkEffect, where it joins DoAdjustCost. A relic
	// moving the clock by nothing is a typo, not a relic.
	//
	// **The result is clamped to one round and never to none.** Zero is no clock at all in the
	// rules, so a stack of drawbacks reaching it would take the mechanic off the fight rather than
	// making it harsher — the one direction a bug in this is invisible. Same clamp, and the same
	// reason, as session.SetRoundLimit.
	//
	// **A fight already on no clock stays on none.** Creatures and every bare Duelist in a test
	// carry a zero, and a delta off an unlimited fight is still unlimited.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoAdjustRoundLimit

	// DoAdjustEssenceTargets moves how many cards one essence is spent on, by a **signed** Amount.
	// 1 is two cards where the mechanic gives one.
	//
	// **A delta rather than a percentage, so worn order decides nothing** *(owner's call,
	// 2026-09-19)*. Every delta sums, because addition commutes — the argument DoAdjustRoundLimit
	// is already under — and what that buys here is a step whose worth does not run away with the
	// number of copies: a card at a time is a dial the essence catalog can be priced against, where
	// a doubling turns a second copy into four cards and a third into eight.
	//
	// **Floored at one card, never at none.** A delta that reached zero would take the essence
	// mechanic off the run rather than making it meaner — the clamp SetRoundLimit and
	// SetRelicSlots are both under.
	//
	// **Appended, because the enum is append-only**: the registry indexes by ordinal.
	DoAdjustEssenceTargets
)

// RelicVerbs is every verb in a fixed order.
func RelicVerbs() []RelicVerb {
	return []RelicVerb{DoAdjustCost, DoScaleDamage, DoApplyStatus, DoSetElement, DoAddDMG,
		DoAddHP, DoGrowOnWin, DoScalePropagation, DoAdjustPicks, DoAdjustPrizeVitae, DoScaleHP,
		DoEchoAttack, DoRepeatCard, DoDemoteCard, DoGrowOnHit, DoGrowOnTurn, DoResetGrowth,
		DoAddHandDMG, DoAddDamagePerHeld, DoGrowPerCard, DoAddDamagePerVitae,
		DoScaleHandDamage, DoScaleDamagePerVitae, DoDrainDamage, DoHealShare, DoScaleRolls,
		DoAdjustRoundLimit, DoAdjustEssenceTargets}
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
	case DoAddHandDMG:
		return "add-hand-dmg"
	case DoAddDamagePerHeld:
		return "add-damage-per-held"
	case DoDrainDamage:
		return "drain-damage"
	case DoHealShare:
		return "heal-share"
	case DoScaleRolls:
		return "scale-rolls"
	case DoAdjustRoundLimit:
		return "adjust-round-limit"
	case DoAdjustEssenceTargets:
		return "adjust-essence-targets"
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
	case DoApplyStatus, DoGrowOnHit, DoDrainDamage:
		return MomentAttackLands
	case DoHealShare:
		return MomentTurnStart
	case DoGrowOnTurn, DoResetGrowth, DoGrowPerCard:
		return MomentTurnTaken
	case DoSetElement:
		return MomentCardDrawn
	case DoDemoteCard:
		return MomentDeckBuilt
	case DoAddDMG, DoAddHP, DoScaleHP, DoAddDamagePerVitae, DoScaleRolls, DoAdjustRoundLimit:
		return MomentFightStart
	case DoEchoAttack, DoRepeatCard, DoAddHandDMG, DoAddDamagePerHeld, DoScaleHandDamage:
		return MomentBlowFormed
	case DoGrowOnWin, DoScalePropagation:
		return MomentFightWon
	case DoAdjustEssenceTargets:
		return MomentEssenceSpent
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
	// **The declared cost, never the wearer's.** A discount relic makes a Skewer cost 2 to its
	// wearer, and a rule matching `Tier: 3` still has to see a Skewer — otherwise two relics worn
	// together would silently stop each other working, and which one won would depend on the order
	// they were bought in. `Concept.Tier` is the same reading an essence takes, and for the same reason.
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

	// Hands narrows a rule to blows that **satisfied** one of the named rungs of the ladder, and it
	// is the second predicate that is not a fact about a card *(2026-09-05)*.
	//
	// **Satisfied, not formed** *(owner's call, 2026-09-13)*. The ladder names a blow after the one
	// rung that pays the most, so four identical cards are a Card Four of a Kind and nothing else —
	// which switched off the Form Four of a Kind relic on a turn that plainly was one. A blow now
	// carries every rung it satisfies (`Blow.Satisfied`) and a rule fires if any rung it names is in
	// that set. The consequence taken deliberately with it: the ladder is cumulative downward too,
	// so a Pair relic pays on every multi-card hand.
	//
	// **A list because one sentence can name several rungs.** "Every Four of a Kind" is three
	// catalog entries; written as three rules it fired once per axis the turn satisfied, so a 4x
	// relic paid 16x on a hand that was a four of a kind two ways. One rule naming the set fires
	// once, whatever the blow satisfied.
	//
	// **It is asked of the blow, never of a card**, so `Matches` does not read it: a rung is a
	// property of the whole set, and a per-card test would have to answer "is this card part of
	// the hand", which is a different question with a different answer. `HandBonus` and `HandScale`
	// are the callers and they ask directly.
	//
	// **Only `blow-formed` knows what formed**, so a rule setting this at any other moment is
	// refused at registration, exactly as Lead is.
	Hands []HandID

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
	return c.HasElement || c.HasForm || c.HasConcept || c.HasTier || c.Lead || c.HasHand() ||
		c.MinForms > 0
}

// HasHand reports whether this condition names any rung at all.
func (c RelicCondition) HasHand() bool { return len(c.Hands) > 0 }

// onHand reports whether a blow satisfying `satisfied` passes this condition's rung test. **A rule
// naming no rung passes**, so this reads as a narrowing like every other predicate; a rule naming
// several passes on any one of them, which is what makes "every Four of a Kind" one rule.
func (c RelicCondition) onHand(satisfied []HandID) bool {
	if !c.HasHand() {
		return true
	}
	for _, want := range c.Hands {
		for _, got := range satisfied {
			if want == got {
				return true
			}
		}
	}
	return false
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

	// Element is what set-element recolors a card to.
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
		if rule.If.HasHand() && rule.When != MomentBlowFormed {
			return NoRelic, fmt.Errorf("%s narrows a %s rule to a hand, and only blow-formed knows what formed",
				key, rule.When)
		}
		if rule.If.HasHand() && (rule.If.HasElement || rule.If.HasForm || rule.If.HasConcept ||
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
			if e.Do == DoAddDamagePerHeld && (rule.If.Lead || rule.If.HasHand() || rule.If.MinForms > 0) {
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
// unlike an essence's amount: an essence is a reward the player chose and a silent nothing would be worse
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
	case DoAdjustCost, DoAdjustPicks, DoAdjustPrizeVitae, DoAdjustRoundLimit,
		DoAdjustEssenceTargets:
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

// RelicKeys is every registered key, sorted, for a tool or a test walking the catalog without
// depending on registration order.
func RelicKeys() []string {
	out := make([]string, 0, len(relicRegistry))
	for _, r := range relicRegistry {
		out = append(out, r.Key)
	}
	sort.Strings(out)
	return out
}

// DefaultRelicSlots is how many relics a duelist may wear. **The only limit there is**
// *(owner's call, 2026-09-17)*: there was a `MaxWornRelics` beside it, the width of the duelist's
// relic array, and it is gone — how many relics could *conceptually* be worn is unbounded now, and
// how many actually are is this, or whatever the run is carrying. See Duelist.Relics.
//
// **Five, until brands expand it** — see
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
	return n
}

// WornRelics is what this duelist is wearing, in worn order.
//
// **Left to right, and it compounds.** That is a determinism rule rather than a preference:
// multiplicative effects are order-sensitive, so the order has to be one a rule can name, and worn
// order is the only order the player can actually see. Two slash relics are x4 and that is a build.
func (d Duelist) WornRelics() []WornRelic {
	n := len(d.Relics)
	if n <= 0 {
		return nil
	}
	if slots := d.relicSlots(); n > slots {
		n = slots
	}
	return d.Relics[:n]
}

// Wearing returns this duelist with one more relic on, or unchanged if the hand is full. It returns a
// copy like everything else in this package.
func (d Duelist) Wearing(w WornRelic) Duelist {
	if len(d.Relics) >= d.relicSlots() {
		return d
	}
	// **Cloned rather than appended in place.** append may write into the backing array this
	// duelist shares with whoever it was copied from, which would put a relic on *their* hand as
	// well. See Duelist.Relics for the rule every copy here is under.
	d.Relics = append(d.cloneRelics(), w)
	return d
}

// cloneRelics is this duelist's worn row as a slice nobody else holds.
//
// **The one place the value semantics are bought** *(2026-09-17)*. Everything in this package
// passes duelists around by value and expects a copy to be a copy — `ResolveRound` steps
// `Relics[i].Grown` on its own and hands back a duelist whose growth the caller then settles. A
// shared backing array turns that into a write on the run's own duelist, mid-round, and nothing
// anywhere goes red. So every entry point that takes a Duelist it intends to modify calls this
// first: resolveRound does, and Wearing does.
func (d Duelist) cloneRelics() []WornRelic {
	if len(d.Relics) == 0 {
		return nil
	}
	out := make([]WornRelic, len(d.Relics))
	copy(out, d.Relics)
	return out
}

// Cloned is this duelist with a relic row nobody else can write to. **Exported because the
// callers that matter are outside this package** — session.Equip hands a fighter its relics, and
// a screen holding a duelist across a round must not be sharing that row with the resolver.
func (d Duelist) Cloned() Duelist {
	d.Relics = d.cloneRelics()
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
// the same Bash costs 3 to a duelist wearing the crush discount and 4 to one who is not. `Card.Cost`
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
// defense — the card's own figure scaled by every relic that matches it.
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

// drainsFrom is every share of a blow this duelist's relics turn back into life, in worn order.
//
// **One entry per relic, where statusesFrom deduplicates by status.** Two relics that both drain
// both pay, because two shares of a figure are two different amounts; two relics that both set
// something burning are one burn. What is deduplicated here is the *cards*: a relic fires once for
// the blow however many of its cards matched, so an elemental drain is a share of the hand rather
// than a share per card of that color.
//
// **It says which relic each share came from**, for appliedStatus's reason: the screen flies the
// life out of the ring that produced it, and nothing else on the event can name which ring that was.
func (d Duelist) drainsFrom(cards []Card) []relicDrain {
	var out []relicDrain

	for _, w := range d.WornRelics() {
		pct := 0
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentAttackLands {
				continue
			}
			matched := false
			for _, card := range cards {
				if rule.If.Matches(card) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			for _, e := range rule.Then {
				if e.Do == DoDrainDamage {
					pct += e.Amount + w.Grown
				}
			}
		}
		if pct > 0 {
			out = append(out, relicDrain{Relic: w.Relic, Pct: pct})
		}
	}
	return out
}

// healsFrom is every share of maximum life this duelist's relics put back at the top of a turn, in
// worn order.
//
// **One entry per relic**, like drainsFrom and for the same reason: two shares of one figure are
// two different amounts, and each has to fly out of its own ring.
//
// **No predicate is read, because this moment has none to read** — see MomentTurnStart. The rules
// refuse a conditioned turn-start rule at registration, so there is nothing to check here.
func (d Duelist) healsFrom() []relicDrain {
	var out []relicDrain

	for _, w := range d.WornRelics() {
		pct := 0
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentTurnStart {
				continue
			}
			for _, e := range rule.Then {
				if e.Do == DoHealShare {
					pct += e.Amount + w.Grown
				}
			}
		}
		if pct > 0 {
			out = append(out, relicDrain{Relic: w.Relic, Pct: pct})
		}
	}
	return out
}

// relicDrain is one worn relic's share of a figure, as a percent. **Shared by the two life verbs**
// — a drain's share of a blow and a regeneration's share of a maximum — because what travels is the
// same pair either way: which ring, and how much of it.
type relicDrain struct {
	Relic RelicID
	Pct   int
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

// RollScale is what a worn set does to the numerator of every roll, as a percentage — 100 when
// nothing scales it, which is the identity and what every bare duelist carries.
//
// **Compounding left to right**, like HPScale and every other multiplicative relic effect.
//
// **It is read off the duelist at each roll rather than resolved once at fight-start**, which is
// the shape add-damage-per-vitae already has: the verb declares a rate and the product is taken
// where it is needed. Here it is because the two roll sites are in different phases and neither
// has a figure to cache on.
func RollScale(worn []WornRelic) int {
	out := 100
	for _, e := range RelicEffectsAt(worn, MomentFightStart, Card{}) {
		if e.Do == DoScaleRolls {
			out = out * e.Amount / 100
		}
	}
	return out
}

// rollScale is RollScale for a duelist, which is how both roll sites reach it.
func (d Duelist) rollScale() int { return RollScale(d.WornRelics()) }

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
func LandingSeats(worn []WornRelic, card Card, lead bool) []bool {
	out := make([]bool, len(worn))

	for seat, w := range worn {
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

// HandBonus is the flat damage a worn set adds to a blow for the rungs it satisfied, and which
// seats are the reason.
//
// **It is asked of the hand and not of a card**, which is what separates it from every other
// blow-formed verb: `repeat-card` and `echo-attack` both answer "what does this relic do to this
// card", where this answers "what does this relic do because the turn built a Full House". That is
// why `RelicCondition.Hand` is read here rather than inside `Matches`.
//
// **Two relics naming one rung add.** They are flat terms in a sum, so there is nothing to compound
// — unlike the multipliers, where worn order decides the result.
//
// **It is asked against every rung the blow satisfied**, not the one the ladder named it after, so
// a turn that is a four of a kind two ways pays both rings. One relic still pays once per rule: a
// sentence covering several rungs is one rule naming them all, never one rule each.
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
func seatsDoing(worn []WornRelic, do RelicVerb) []bool {
	seats := make([]bool, len(worn))
	for seat, w := range worn {
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
func HandScale(worn []WornRelic, satisfied []HandID, cards []Card) (int, []bool) {
	seats := make([]bool, len(worn))
	pct := 100

	if len(satisfied) == 0 {
		return pct, seats
	}
	for seat, w := range worn {
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentBlowFormed {
				continue
			}
			if !rule.If.onHand(satisfied) {
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

func HandBonus(worn []WornRelic, satisfied []HandID) (int, []bool) {
	seats := make([]bool, len(worn))
	total := 0

	if len(satisfied) == 0 {
		return 0, seats
	}
	for seat, w := range worn {
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentBlowFormed || !rule.If.HasHand() || !rule.If.onHand(satisfied) {
				continue
			}
			for _, e := range rule.Then {
				if e.Do != DoAddHandDMG {
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
// same property the rune riders have, and for the same reason: it is a fact about the hand at
// the moment the blow is added up rather than an event.
//
// **It reports how many cards paid as well as what they paid** *(2026-09-14)*, because the run's
// account writes the term as `Jar of Ice (4 cards)  20` — a figure with no count beside it is the
// one term in the working the player cannot check against the hand they were holding. It is the
// tally across every seat, which is the figure the merged term is: two jars paying for six cards
// between them is one term of six.
func HeldBonus(worn []WornRelic, held []Card) (total, cards int, seats []bool, pays []HeldPay) {
	// A named return, so it needs building like every other seat row here: one entry per worn
	// relic, all false until one of them pays.
	seats = make([]bool, len(worn))

	for seat, w := range worn {
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
						cards++
						seats[seat] = true
						pays = append(pays, HeldPay{Amount: e.Amount, Seat: seat, Card: c})
					}
				}
			}
		}
	}
	return total, cards, seats, pays
}

// HeldPay is one held card paying one worn relic's figure into the blow: what it paid, which worn
// seat paid it, and which card it was paid for.
//
// **It is the per-card reading of the same tally `HeldBonus` totals**, and it exists because the
// sum shows its working a card at a time: six earth cards kept back are six `+5` terms rather than
// one `+30`, so the player can count the jar's term against the hand still in front of them. The
// total stays on the event beside it, because the run's account writes the merged figure.
//
// Two jars paying for the same card are two entries, one per seat — which is what lets each term
// fly out of the relic that paid it.
// **The card is on it so a screen can point at it.** The figure flies out of the card that is
// still in the hand rather than out of the relic, and the only other way to find that card is to
// match a concept and an element back against the row — which is what the held riders have to do
// and is a second reading of a thing the rules already knew. See screens.mathScript.
type HeldPay struct {
	Amount int
	Seat   int
	Card   Card
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

// RoundLimitFor is the clock a worn set puts this fight on, given the limit the run is carrying.
//
// **Every delta is summed**, so two relics each taking a round take two. Worn order decides
// nothing, because addition commutes — see DoAdjustRoundLimit for why that is the point rather
// than a shortcut.
//
// **Never below one.** Zero is no clock at all here, so a stack of drawbacks reaching it would
// take the mechanic off the fight instead of tightening it.
//
// **A fight on no clock stays on none**, whatever is worn: there is nothing to move.
func RoundLimitFor(worn []WornRelic, base int) int {
	if base <= 0 {
		return base
	}
	limit := base
	for _, e := range RelicEffectsAt(worn, MomentFightStart, Card{}) {
		if e.Do == DoAdjustRoundLimit {
			limit += e.Amount
		}
	}
	if limit < 1 {
		limit = 1
	}
	return limit
}

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
	for i := range d.Relics {
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
	for i := range d.Relics {
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
func CardScaleBySeat(worn []WornRelic, card Card) []int {
	out := make([]int, len(worn))

	for seat, w := range worn {
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

// EssenceTargets is how many cards one essence is spent on, given a worn set.
//
// **One card is the mechanic and a relic moves it** — see DoAdjustEssenceTargets. **Every delta
// sums and worn order decides nothing**, because addition commutes: two relics at 1 make three
// cards, which is the shape RoundLimitFor already has.
//
// **Never below one.** An essence with no card to land on is a reward that is not one, so a stack
// of drawbacks reaching zero is clamped back up rather than taken as a refusal.
func EssenceTargets(worn []WornRelic) int {
	n := 1
	for _, e := range RelicEffectsAt(worn, MomentEssenceSpent, Card{}) {
		if e.Do == DoAdjustEssenceTargets {
			n += e.Amount
		}
	}
	if n < 1 {
		n = 1
	}
	return n
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
	return Neighbor(card.Concept, -deepest)
}

// FlipStep is one relic recoloring a card on its way out of the draw pile: which relic did it, and
// what the card became.
//
// **It exists because the cascade is something to watch.** A card dealt under two flips changes
// twice, once per ring, and a caller handed only the final color could draw one change out of two —
// see screens/combat_deal.go, which plays a beat per step with the ring that caused it rattling.
type FlipStep struct {
	Relic RelicID
	To    Element
}

// FlipSteps is every flip that touches a card as it is dealt, **in worn order, each one reading what
// the flip before it left behind**.
//
// **They chain** *(owner's call, 2026-09-15)*. Lightning-to-ice worn beside ice-to-earth deals a
// lightning card as earth, through ice, rather than leaving it at ice. Every flip used to match on
// the card's *original* element on the argument that chaining lets a run funnel a whole deck into
// one color — which it does, and which is now the intent rather than the hazard. The deck panel's
// alterations view is what answers "so what am I actually holding"; see session.AlteredAs, which
// reads the same walk.
//
// **A flip onto the color the card already wears is not a step.** Nothing happened, so there is
// nothing to draw and nothing to report — which is also what stops a relic listing itself as a
// contributor to a card it left alone.
func FlipSteps(worn []WornRelic, card Card) []FlipStep {
	var out []FlipStep

	running := card
	for _, w := range worn {
		to, matched := Basic, false
		for _, rule := range RelicOf(w.Relic).Rules {
			if rule.When != MomentCardDrawn || !rule.If.Matches(running) {
				continue
			}
			for _, e := range rule.Then {
				if e.Do == DoSetElement {
					to, matched = e.Element, true
				}
			}
		}
		if !matched || to == running.Element {
			continue
		}
		running.Element = to
		out = append(out, FlipStep{Relic: w.Relic, To: to})
	}
	return out
}

// FlipElement is what color a card is dealt as, given a worn set. It reports false when no relic
// touches it, so a caller can leave the card alone rather than writing its own color back over it.
//
// **It is the last step of FlipSteps**, which is the walk — two walks would be two chances to
// disagree about what a card is dealt as, and the screen draws every step of the one this answers
// the end of.
//
// **The card handed in must be the card the run owns.** A flip reads the running element now, so a
// card that has already been through here and is handed back would take a second trip: the
// discard folded into the draw pile is restored to the run's own color first, which is what
// screens.restoreToDeck is for. That was true before the flips chained and it matters more now.
func FlipElement(worn []WornRelic, card Card) (Element, bool) {
	steps := FlipSteps(worn, card)
	if len(steps) == 0 {
		return Basic, false
	}
	return steps[len(steps)-1].To, true
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
// which is most of the catalog and is what says "draw nothing".
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
		// a multiplier, and its absence on the other branch is what says the flat one is not — so
		// the letter was a third of the badge’s width spent restating what the figure already reads
		// as, on the one card whose number is meant to be caught at a glance.
		return fmt.Sprintf("%.1f", float64(e.Amount)/100)
	}
	// **No `+` either** *(owner’s call, 2026-09-16)*, on the `x`’s own argument one branch over: a
	// counter only ever counts upward, so the sign was a character of a two-character badge spent
	// saying something no relic in the catalog contradicts. A drawback that took a figure *down*
	// still prints its minus, which is `%d` and needs no case here.
	return fmt.Sprintf("%d", e.Amount)
}
