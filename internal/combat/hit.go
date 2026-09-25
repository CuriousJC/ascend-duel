package combat

// The hand-forming attack phase: a hand is read off the turn, and then **every card of the turn
// lands its own hit**, defenses included.
//
// **A hit is a landing, not a card.** A card lands once, plus once for every extra landing a worn
// relic buys it — an echo ladder, a form repeat — so a five-card turn is at least five hits and can
// be more. Each hit is its own arithmetic from end to end:
//
//	(the card's damage at the hand's DMG, with every relic that prices the card)
//	  x the hand's multiplier
//	  x every relic that scales the hand
//	  then the attacker's weight and the target's vulnerability
//
// **The hand's DMG is where every relic that raises the duelist lands** — a rung relic, the cards
// kept back, the purse — so each card grows by its own multiplier and nothing is added to a hit
// afterwards.
//
// **Everything in that list is paid per hit.** A status lands on every hit that connects, a drain is a share of each hit's figure, and a
// shock rolls once per hit. Nothing about the attack phase belongs to the turn as a whole except the
// hand that names it.
//
// **Each hit rounds on its own**, so a turn's total is the sum of rounded hits rather than one
// rounded sum.

import "math/rand"

// resolveAttackPhase is the whole of one side's offense: every attack card it queued, the hand
// they form, and the hits that follow.
//
// The log it writes is: a KindAction per attack card, one KindHand carrying every hit's arithmetic,
// then per hit a KindFizzled, a KindMissed, a KindBlocked or a KindDamage — each followed by whatever that hit
// drained and whatever statuses it landed, and a KindDefeated on the hit that killed. **Hits stop at
// a death**: the terms after it are on the hand event and no event says they were thrown.
func resolveAttackPhase(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	held []Card,
	round int,
	hands []Hand,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	// **A solo attacker takes a different phase entirely, not a special case inside this one.** It
	// reads no hand, so it has no multiplier and no hand event; each card is its own hit at face
	// damage.
	if actor.SoloAttacks {
		return resolveSoloAttacks(events, side, actor, target, turn, round, rng)
	}

	// Every attack card is announced whether or not it ends up in the hand. **A slot that resolved
	// has to produce a beat**, because the screen counts one per slot to know how far through the
	// round playback is — see TestEverySlotIsEitherTakenOrChilled.
	for _, slot := range turn {
		if slot.Card.Category() != CategoryAttack {
			continue
		}
		events = append(events, Event{
			Kind:    KindAction,
			Side:    side,
			Action:  slot.Card.Concept,
			Element: slot.Card.Element,
			Round:   round,
		})
	}

	// **The ladder is read through the actor's own stones**, so a run that has bought a Pair stone
	// plays a different ladder from its opponent. See stone.go.
	//
	// **A turn with no attack in it still forms a hand.** A hand of nothing but shields is named and
	// written into the account like any other; it has no hits, so it spends nothing of the target's.
	blow := blowFor(turn, actor.handsFrom(hands))
	if len(blow.Cards) == 0 {
		return events, actor, target
	}

	hand, hits, actor, target := strike(side, blow, turn, held, actor, target, round, rng)
	events = append(events, hand)
	return append(events, hits...), actor, target
}

// landing is one hit before it is thrown: which card of the turn, which of that card's landings,
// and the shape its relics gave the card.
type landing struct {
	seat  int
	nth   int
	shape LandingShape
}

// landingsOf lays a turn out as the hits it will throw, in turn order and each card's landings
// together.
//
// **Every card of the turn throws a hit** *(owner's call)*, defenses included, whether or not it
// made the hand. A defense deals nothing of its own, so its hit comes to nothing, and it lands its
// card's statuses like any other hit — one rule, with nothing special for
// a verb.
//
// **The lead is the turn's first attack card**, which is the only thing the `Lead` predicate reads.
func landingsOf(blow Blow, turn []Slot, worn []WornRelic) []landing {
	var out []landing
	lead := leadSlot(blow, turn)
	for i, slot := range turn {
		shape := LandingsOf(worn, slot.Card, i == lead)
		for n := 0; n < shape.Count(); n++ {
			out = append(out, landing{seat: i, nth: n, shape: shape})
		}
	}
	return out
}

// strike throws every hit of a blow and packages the hand event that describes them.
//
// **The hand event is built alongside the hits and emitted in front of them**, so the screen has
// every hit's arithmetic from the beat the hand is named — which is what lets it work all of them
// out on screen at once — and the log still reads in the order things happened.
//
// **A hit's figure is asked at the accumulator the hits before it left**, and a growing relic steps
// only on a hit that connected: a miss or a block pays no relic.
func strike(
	side Side,
	blow Blow,
	turn []Slot,
	held []Card,
	actor, target Duelist,
	round int,
	rng *rand.Rand,
) (Event, []Event, Duelist, Duelist) {
	targetSide := other(side)
	worn := actor.WornRelics()

	lead := turn[leadSlot(blow, turn)].Card
	e := Event{
		Kind:       KindHand,
		Side:       side,
		Hand:       blow.Hand.ID,
		Multiplier: blow.Multiplier,
		Action:     lead.Concept,
		Element:    lead.Element,
		Round:      round,
	}

	// **The DMG every hit is swung at.** The damage riders and a rung relic's raise are folded into
	// it for the length of this blow alone, so a +10 arrives in every hit — see blowDMG. It is put
	// back before the actor is returned; a DMG left raised would make the bonus permanent.
	//
	// **The cards kept back and the purse raise it the same way**, both read at the blow: a held
	// card pays again every turn it is held, and vitae moves inside a fight — a card kept in hand
	// pays one, and riders fire before the attack phase.
	baseDMG := actor.DMG
	e.HandBonus, e.HandBonusSeats = HandBonus(worn, blow.Satisfied)
	e.HeldDMG, e.HeldDMGCards, e.HeldDMGSeats = HeldDMG(worn, held)
	e.VitaeDMG, e.VitaeDMGSeats = DMGPerVitae(worn)*actor.Vitae, seatsDoing(worn, DoAddDMGPerVitae)
	actor.DMG = blowDMG(baseDMG+e.HandBonus+e.HeldDMG+e.VitaeDMG, turn, held, blow)
	e.HandDMG, e.HandDMGBare = actor.DMG, blowDMG(baseDMG, turn, held, blow)

	// **A rung relic is a second multiplier, never a bigger hand.** `Multiplier` stays the ladder's
	// own figure, so the banner and the hand row show the rung the player built; this is applied
	// after it on every hit.
	e.HandScale, e.HandScaleSeats = HandScale(worn, blow.Satisfied, scoringCards(blow, turn))

	for _, i := range blow.Rung {
		if e.RungCardCount >= len(e.RungCards) {
			break
		}
		e.RungCards[e.RungCardCount] = i
		e.RungCardCount++
	}

	landings := landingsOf(blow, turn, worn)

	// **Which hits the shields eat is decided before the first one is thrown**, the matching
	// element first and then the heaviest, ranked on each landing's own figure at the turn's
	// opening state. See shieldedHits, whose argument this is.
	//
	// **A hit of nothing is never eaten** *(owner's call)*: a shield is spent on something that
	// would have hurt, or a turn of shields would strip the target's for free. A hit that fizzles is
	// a hit of nothing.
	planned := make([]int, len(landings))
	elems := make([]Element, len(landings))
	for h, l := range landings {
		card := turn[l.seat].Card
		elems[h] = card.Element
		planned[h] = scaleDamage(l.shape.Amount(l.nth, actor.CardDamage(card)), blow.Multiplier)
		if planned[h] <= 0 || fizzles(card, target) {
			planned[h] = -1
		}
	}
	eaten := shieldedHits(planned, elems, target.Shields)

	var hits []Event
	thrown := false
	for h, l := range landings {
		card := turn[l.seat].Card

		d := l.shape.Amount(l.nth, actor.CardDamage(card))
		figure := scaleDamage(d, blow.Multiplier)
		if e.HandScale != 0 && e.HandScale != 100 {
			figure = scaleDamage(figure, e.HandScale)
		}

		// **The hit's arithmetic goes on the hand event whether or not it is thrown**, so the screen
		// can work every hit out at once. A term past the array's width is dropped from the
		// *arithmetic* rather than from the fight: the hit is still thrown.
		recorded := e.HandCardCount < len(e.HandCards)
		if recorded {
			at := e.HandCardCount
			e.HandCards[at] = l.seat
			e.HandAmounts[at] = d
			e.HandCardBase[at] = l.shape.Amount(l.nth, card.Damage(actor.DMG))
			// **Only an attack applies a percentage to the DMG.** A defense's Amount is how many
			// shields it raises, and recorded here it would print as a multiplier of 0.01 per shield;
			// zero sends its term to the flat figure, which is the 0 it deals.
			if card.Spec().Verb == VerbAttack {
				e.HandCardPct[at] = l.shape.Amount(l.nth, card.Amount())
			}
			e.HandRelicScale[at] = CardScaleBySeat(worn, card)
			e.HitAmounts[at] = figure
			if l.nth > 0 {
				// **Only the extra landings are attributed to a relic.** The card's own first
				// landing is the card being played, which needed no relic to seat it.
				e.HandLanding[at] = LandingSeats(worn, card, l.seat == leadSlot(blow, turn))
			} else {
				e.HandLanding[at] = make([]bool, len(worn))
			}
			e.HandCardCount++
		}
		e.Amount += figure

		// **A hit after a death is not thrown.** Its arithmetic is on the hand event, and nothing
		// in the log says it landed. **A hit of nothing is still thrown**: it can miss and land its
		// card's statuses like any other, and it spends nothing of the target's.
		if target.Alive() {
			thrown = thrown || figure > 0
			hits, actor, target = throwHit(hits, side, targetSide, actor, target, card, l.seat, h, figure, eaten[h], round, rng)
		}

		if recorded {
			// **After the hit, not before**, so the row of badges reads as the number this hit has
			// just earned rather than the number it was counted at.
			grown := actor.WornRelics()
			row := make([]int, len(grown))
			for seat, w := range grown {
				row[seat] = w.Grown
			}
			e.HandGrown[e.HandCardCount-1] = row
		}
	}

	// **Every standing shield is spent on the turn it answered**, if anything that could hurt was
	// swung at it.
	if thrown {
		target = ClearDefenses(target)
	}

	actor.DMG = baseDMG
	return e, hits, actor, target
}

// throwHit rolls, blocks or lands one hit.
//
// **The order inside a hit is the order inside every attack**: the fizzle, the shock roll, then a shield, then
// weight and vulnerability, then the damage, then the growing relics step, then what the hit drains
// and the statuses it lands. A miss spends no shield and a blocked hit lands nothing, and neither
// drains, burns or grows.
func throwHit(
	hits []Event,
	side, targetSide Side,
	actor, target Duelist,
	card Card,
	seat, hit, figure int,
	shield Element,
	round int,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	// **A fizzle is decided before anything is rolled**: it is the target's nature rather than
	// luck, so a hit that was never going to land draws no shock from the stream either.
	if fizzles(card, target) {
		return append(hits, Event{
			Kind:    KindFizzled,
			Side:    side,
			Action:  card.Concept,
			Element: card.Element,
			Target:  targetSide,
			Slot:    seat,
			Hit:     hit,
			Round:   round,
		}), actor, target
	}

	if attackMisses(actor, rng) {
		return append(hits, Event{
			Kind:    KindMissed,
			Side:    side,
			Action:  card.Concept,
			Element: card.Element,
			Target:  targetSide,
			Slot:    seat,
			Hit:     hit,
			Round:   round,
		}), actor, target
	}

	if blocked := false; shield >= 0 {
		hits, target, blocked = blockedByShield(hits, side, target, card, shield, seat, hit, round)
		if blocked {
			return hits, actor, target
		}
	}

	dmg := amplify(blunt(figure, actor.weight()), target.vulnerability())
	target.CurrentLife = reduce(target.CurrentLife, dmg)
	hits = append(hits, Event{
		Kind:    KindDamage,
		Side:    side,
		Target:  targetSide,
		Action:  card.Concept,
		Element: card.Element,
		Slot:    seat,
		Hit:     hit,
		Amount:  dmg,
		Life:    target.CurrentLife,
		Round:   round,
	})

	// **A growing relic steps on every hit that connects**, before anything else reads it.
	actor = actor.GrowOnLanding(card)

	// **A drain is a share of the figure that landed**, so it follows the damage and a hit that
	// produced no figure drains nothing. See DoDrainDamage.
	for _, dr := range actor.drainsFrom([]Card{card}) {
		before := actor.CurrentLife
		actor.CurrentLife = restore(actor.CurrentLife, dmg*dr.Pct/100, actor.MaxLife)
		// **A drain that restored nothing writes no beat**: a duelist already at full life has a
		// relic that did not fire, and a flight carrying a zero out of the ring says it did.
		if actor.CurrentLife == before {
			continue
		}
		hits = append(hits, Event{
			Kind:   KindDrained,
			Side:   side,
			Target: side,
			Relic:  dr.Relic,
			Slot:   seat,
			Hit:    hit,
			Amount: actor.CurrentLife - before,
			Life:   actor.CurrentLife,
			Round:  round,
		})
	}

	// **Every status comes off a worn relic, and every hit that connects lands its card's.** The
	// hit's card is what the relics match against, so a form relic or a concept relic reaches this
	// the same way an elemental one does.
	for _, a := range actor.statusesFrom([]Card{card}) {
		applied, amount, ok := applyStatus(target, a.Status, actor)
		if !ok {
			continue
		}
		target = applied
		hits = append(hits, Event{
			Kind:    KindStatus,
			Side:    side,
			Target:  targetSide,
			Element: card.Element,
			Status:  a.Status,
			Relic:   a.Relic,
			Slot:    seat,
			Hit:     hit,
			Amount:  amount,
			Life:    target.CurrentLife,
			Round:   round,
		})
	}

	if !target.Alive() {
		hits = append(hits, Event{Kind: KindDefeated, Side: side, Target: targetSide, Round: round})
	}
	return hits, actor, target
}

// fizzles reports whether a card's hit lands nothing on this target because it is the target's own
// element — an ice card thrown at an ice goblin. **Everything the hit would have done goes with
// it**: the damage, the drain, the statuses, and the growing relics' step. The card still counts
// toward the hand it formed; only its hit is wasted.
//
// **A wildcard never fizzles** *(owner's call)*. It counts as every element when a hand is formed,
// and a card that matched every target's element would fizzle on every attack in the game.
//
// **Basic is no element**, so a plain card never fizzles and a target with no element — the
// player, and every bare `Duelist{}` — takes every hit. That is what makes the rule one-way.
func fizzles(card Card, target Duelist) bool {
	return target.Element != Basic && card.Element == target.Element && !card.Wild(AxisElement)
}

// leadSlot is the turn index of the attack the blow is named by: the earliest attack card in the
// scoring set, or its first card for a blow that holds none.
//
// **Earliest rather than heaviest.** On the commonest turn those are the same card, and it is the
// card the `Lead` predicate reads and the hand's element comes from.
func leadSlot(blow Blow, turn []Slot) int {
	for _, i := range blow.Cards {
		if i >= 0 && i < len(turn) && turn[i].Card.formsBlow() {
			return i
		}
	}
	return blow.Cards[0]
}

// scoringCards is every card that paid into the blow, in turn order — the hand's own cards plus
// every attack the turn played. **The scoring set, not the turn**: a defense that made no hand is
// not here, having neither swung nor been counted.
func scoringCards(blow Blow, turn []Slot) []Card {
	out := make([]Card, 0, len(blow.Cards))
	for _, i := range blow.Cards {
		if i >= 0 && i < len(turn) {
			out = append(out, turn[i].Card)
		}
	}
	return out
}
