package combat

import "math/rand"

// ResolveRound plays out one round and returns its event log along with the state
// both sides end in. ResolutionOrder decides the order it plays them in.
//
// Inputs are taken by value and never mutated, so a caller can re-run a round from
// the same starting state — the returned duelists are the authority on what changed.
// **`src` is the round's randomness and every field of it may be nil.** It is the seat CLAUDE.md's
// determinism rules require — injected sources, never a package global — and there are two of them:
// the shock roll and the gamble a golden or a silver card takes. **The zero value rolls nothing**,
// which is what a caller with no business being random should pass: a preview, or a test pinning
// the parts of the engine that are still exact. See Sources, which is where the rule that the two
// streams are never interchanged is written down.
func ResolveRound(a, b Duelist, aCards, bCards []Card, round int, src Sources) (events []Event, aAfter, bAfter Duelist) {
	return resolveRound(a, b, aCards, bCards, nil, nil, round, handTable, src)
}

// ResolveRoundHolding is ResolveRound told what each side did **not** play.
//
// **It exists because four riders read the unplayed hand** — see rider.go, where the in-hand
// riders are. A card kept back is not a card the resolver would otherwise ever see: the turn is
// what was queued, and everything else was invisible to this package until a rune made
// holding a card worth something.
//
// **ResolveRound is kept and delegates**, rather than every caller and test growing two nil
// arguments. A side that holds nothing plays exactly the round it always did, which is what makes
// that safe.
//
// `aHeld` and `bHeld` are the cards still in each side's hand at the moment the round was
// committed — not the draw pile, and not the cards being played. An opponent has no hand to hold,
// so `bHeld` is nil for every fight the game plays today.
func ResolveRoundHolding(a, b Duelist, aCards, bCards, aHeld, bHeld []Card, round int, src Sources) (events []Event, aAfter, bAfter Duelist) {
	return resolveRound(a, b, aCards, bCards, aHeld, bHeld, round, handTable, src)
}

// resolveRound is ResolveRound with the catalog injected. It exists so a test can drive a
// synthetic hand through the whole engine rather than only through the matcher.
func resolveRound(a, b Duelist, aCards, bCards, aHeld, bHeld []Card, round int, hands []Hand, src Sources) (events []Event, aAfter, bAfter Duelist) {
	// **Both duelists' relic rows are cloned before a single rule runs** *(2026-09-17)*. This
	// package hands duelists around by value and the rules step `Relics[i].Grown` on their own copy
	// — the caller settles that growth onto the run when the round is over, exactly as it settles
	// the purse. `Relics` is a slice now, so without this the step writes straight through into the
	// caller's duelist and a relic grows mid-round whether or not the round is ever settled.
	//
	// **It is here rather than at every call site**, because this is the one door: ResolveRound and
	// ResolveRoundHolding both come through it. See Duelist.cloneRelics, which says what is being
	// bought, and TestResolvingARoundDoesNotGrowTheCallersRelics, which is what goes red without it.
	a.Relics, b.Relics = a.cloneRelics(), b.cloneRelics()

	events = make([]Event, 0, 16)
	events = append(events, Event{Kind: KindRoundStart, Round: round})

	// A defense expires at the start of its owner's next turn, so it covers exactly one
	// opposing turn whichever side raised it. Expiry is a rule about *turns* rather than
	// about the action sequence, which is why it lives here and not in ResolutionOrder —
	// a side that queues nothing still has a turn, and still loses its guard in it.
	//
	// A whole turn each, A then B. This used to be one flat loop over ResolutionOrder with a
	// flag watching for the handover; hands made a turn a thing with its own beginning —
	// a chill is spent at it, and a hand's position is an index *within* it — so the turn
	// became worth naming. ResolutionOrder is still the authority on order: playTurn walks
	// exactly the slots it produced for that side.
	events, a, b = playTurn(events, SideA, a, b, appendTurn(nil, SideA, aCards), aHeld, round, hands, src)

	// B still loses its standing defenses even in a round it never gets to act in, which is
	// why this is not inside playTurn's early return: expiry is a property of the turn
	// arriving, not of anything happening in it.
	if a.Alive() && b.Alive() {
		events, b, a = playTurn(events, SideB, b, a, appendTurn(nil, SideB, bCards), bHeld, round, hands, src)
	} else {
		events, b = expireDefenses(events, SideB, b, round)
	}

	// **A always burns before B**, which is the same order the turns were played in and needs no
	// tie-break.
	events, a = endRound(events, SideA, a, round)
	events, b = endRound(events, SideB, b, round)

	// **The clock is read last, after every other way the round could have ended.** A duelist who
	// died to the final blow or to a burn tick is not out of time — they are simply dead — and
	// `callTime` says so by asking whether they are still alive. A fight that finishes on the last
	// round is a fight nobody ran out of. See clock.go.
	//
	// **A fight already decided is not timed out.** A duelist who killed their opponent on the
	// final round beat the clock; one who died to the final blow died to the blow. Only a duel
	// still standing on both sides has run out of anything. See FightOver.
	//
	// **A always before B**, the same order the turns and the burns took, so a round that times
	// both sides out reads in one order rather than in whichever the map felt like.
	if !FightOver(a, b) {
		events, a = callTime(events, SideA, a, round)
		events, b = callTime(events, SideB, b, round)
	}

	events = append(events, Event{Kind: KindRoundEnd, Round: round})
	return events, a, b
}

// playTurn runs one side's whole turn: expiry, then whatever a chill has taken off the
// front of it, then **every hand the surviving cards form**, and only then the cards
// themselves.
//
// **Hands are matched against what is left after a chill, not against the queue.** The
// player queued five attacks; a chill that ate two means three happened, and a hand scored
// off cards a chill deleted would let a chilled duelist swing with a turn they did not take.
// That ordering is the reason the hand phase sits *inside* a turn rather than at the top of
// the round: a round-wide hand phase would score B's hands before A's ice had taken
// anything off B.
func playTurn(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	held []Card,
	round int,
	hands []Hand,
	src Sources,
) ([]Event, Duelist, Duelist) {
	// **Regeneration is the first thing that happens in a turn**, ahead of the chill, the riders
	// and both phases — a relic that puts life back does it in time for the turn it is about to
	// survive rather than after it. See MomentTurnStart and DoHealShare.
	events, actor = healAtTurnStart(events, side, actor, round)

	events, actor = expireDefenses(events, side, actor, round)

	// **The surge is spent by the turn arriving**, exactly as the shields lapse: it bought this
	// turn's budget, which the cards were already committed against, and it buys no other. See
	// Duelist.Surge.
	actor.Surge = 0

	// A chill comes off the front, which needs no tie-break and so is the only pick that is
	// deterministic without inventing a rule.
	//
	// **The front of a turn is its defenses as of 2026-09-15**, because the phase order flipped —
	// see Categories. So what a chill costs first is now the guard rather than the blow. **That is
	// a real change to what ice does and it was taken rather than worked around**: the alternative
	// is naming attacks explicitly here, which is exactly the invented rule this picks the front to
	// avoid. If ice should go back to eating the blow first, this is the line, and it needs a
	// tie-break rule of its own.
	//
	// **The action points are not refunded.** They were committed when the cards were queued,
	// and letting them come back would make a chill pure tempo; keeping them spent makes it
	// tempo and economy both.
	//
	// **The chill is read off the status and nowhere else** *(2026-08-17)*. A hand buys damage and
	// only damage, so nothing else in the game can take a card off a turn — which means there is
	// exactly one place to look for how many, and a second counter would be a second answer to a
	// question with one.
	//
	// **It bites on every turn it outlives**, rather than being spent when it bites — the status
	// counting down is what ends it. The asymmetry phases impose is carried by the status too:
	// side A acts first, so ice A lands takes a card from B the same round, while ice B lands
	// finds A has already acted and bites in the round after.
	lost := actor.chillCards()
	if lost > len(turn) {
		lost = len(turn)
	}
	for i := 0; i < lost; i++ {
		events = append(events, Event{
			Kind:    KindChilled,
			Side:    side,
			Action:  turn[i].Card.Concept,
			Element: turn[i].Card.Element,
			Round:   round,
		})
	}
	turn = turn[lost:]

	// **Riders fire here: after the chill, before the blow.** A rider belongs to one card rather
	// than to the duelist, so the moment it wants is "this card was played" — and a card a chill
	// ate was never played. Putting it in front of the attack phase is what makes a heal arrive in
	// time to matter to the turn it was spent in, rather than after the round it was meant to
	// survive. See rider.go.
	events, actor = playRiders(events, side, actor, turn, held, round, src.Luck)

	// **The defend phase comes first as of 2026-09-15** *(owner's call)*, reversing the 2026-08-15
	// order. The argument for putting it last was that a defense answers the *opponent's* blow and
	// the opponent acts after this turn — which is true and does not depend on within-turn order:
	// `expireDefenses` runs at the start of a side's *own* turn, so a shield raised anywhere in
	// this turn is standing through the opponent's either way, and this side's attacks are aimed at
	// the other duelist rather than at itself. See Categories, where the ordering lives.
	//
	// What it buys is what a turn reads as: raise the guard, then swing. A defend card used to pay
	// a visible 0 into the hand's sum and then do the thing it was actually for several beats
	// later, on a card the player had stopped watching.
	//
	// **It is skipped if either side fell**, since a corpse raising a shield is a line in the log
	// nobody wants and a duel that is over does not need one. **That guard has moved with the
	// phase and now only protects the phase below it** — nothing can have fallen this early in a
	// turn, because the only thing in front of the defenses is the riders.
	for at, slot := range turn {
		if slot.Card.Category() != CategoryDefend {
			continue
		}
		events, actor, target = resolveDefend(events, side, actor, target, slot.Card, at, round)
	}

	if !actor.Alive() || !target.Alive() {
		return events, actor, target
	}

	// **The attack phase is a hit per landing.** Every attack card queued is announced, then the
	// hand they form is announced, then every card lands its own hit — five Bashes are five hits
	// under one Four of a Kind. See hit.go.
	events, actor, target = resolveAttackPhase(events, side, actor, target, turn, held, round, hands, src.Roll)

	// **Nothing follows the hits**, so a duelist who fell to one closes no turn: the streak below is
	// a fact about turns taken and a corpse takes none.
	if !actor.Alive() || !target.Alive() {
		return events, actor, target
	}

	// **The turn is closed after everything in it has resolved**, which is what makes "a turn with
	// no defend card in it" a question this can answer. A duelist who fell mid-turn returns above and
	// never reaches this: a streak is a fact about turns taken, and a corpse takes none.
	actor = actor.TurnTaken(cardsOf(turn))

	return events, actor, target
}

// cardsOf is a turn's cards without their slots, for the rules that ask about the turn as a whole.
func cardsOf(turn []Slot) []Card {
	out := make([]Card, len(turn))
	for i, slot := range turn {
		out[i] = slot.Card
	}
	return out
}

// resolveDefend runs one card of the second phase. **They are the only cards that still resolve one
// at a time**, because each does something to its own duelist rather than contributing to a shared
// blow.
func resolveDefend(
	events []Event,
	side Side,
	actor, target Duelist,
	card Card,
	at int,
	round int,
) ([]Event, Duelist, Duelist) {
	events = append(events, Event{
		Kind:    KindAction,
		Side:    side,
		Action:  card.Concept,
		Element: card.Element,
		Round:   round,
	})

	// **The switch is on the verb, not on the card** *(2026-08-16)*. It used to name the player's
	// three cards one at a time, which meant an enemy's `Congeal` could not guard anything however
	// obviously it was a defense. Two verbs, any number of cards.
	spec := card.Spec()
	switch spec.Verb {
	case VerbShield:
		// Raised, not spent. Each one eats a whole incoming attack when the opponent swings — see
		// blockedByShield, and Duelist.Shields for when they expire.
		actor = actor.raiseShields(card.Element, card.Amount())
		// **`Slot` names the card that raised them.** The defenses fire as one bundle and nothing
		// lifts, so the event is the only thing that can say which card the pips come out of.
		events = append(events, Event{
			Kind:    KindRaised,
			Side:    side,
			Action:  card.Concept,
			Element: card.Element,
			Slot:    at,
			Amount:  card.Amount(),
			Life:    actor.Shields.Count(),
			Round:   round,
		})
	}

	return events, actor, target
}

// expireDefenses drops the shields the previous turn put up. Called at the start of a side's own
// turn, never at the round boundary — side B acts last, so a defense cleared at the boundary would
// have protected B from nothing at all.
//
// **This is the whole of "a shield lasts the turn after it was played".** Raised at the end of
// your turn, standing through the opponent's, gone before you act again.
//
// The clearing itself is ClearDefenses, which the combat screen also calls between fights; the
// timing rule is what lives here.
func expireDefenses(events []Event, side Side, d Duelist, round int) ([]Event, Duelist) {
	// **The announcement is for the shields alone**, because they are the only half of this the
	// screen draws. A guard lapsing unspent has no readout to correct, and a beat with no picture
	// is the thing the choreography table exists to refuse.
	if d.Shields.Count() > 0 {
		events = append(events, Event{
			Kind:   KindExpired,
			Side:   side,
			Target: side,
			Amount: d.Shields.Count(),
			Round:  round,
		})
	}
	return events, ClearDefenses(d)
}

// endRound ticks a burn and counts every status down one.
//
// **The burn ticks before the countdown**, so a fire hit lands damage at the end of the round it
// was struck in as well as the round after. MECHANICS.md says a DoT "lands at end of round" and
// this is the end of the round it was applied in; making it wait would mean a fire attack did
// nothing at all in a duel that ended on the round it was played.
//
// **A dead duelist does not burn.** The first version ticked regardless, on the grounds that
// skipping a corpse would make the order of two deaths matter — it does not, because whether a
// duelist is dead is settled before either side's round-end runs. What it did instead was
// announce a second `KindDefeated` over a body, and the Resolution feed duly read
// "Goblin falls / Goblin burns for 2 / Goblin falls". Statuses still tick down, so a duelist
// somehow revived does not wake up carrying an expired burn.
func endRound(events []Event, side Side, d Duelist, round int) ([]Event, Duelist) {
	// **Every damage-over-time status ticks, one at a time**, in registration order. There is one
	// such status in the game today; walking them is what stops a second one being silently
	// ignored, and the order is fixed because which tick killed a duelist decides what the feed
	// says they fell to.
	for _, id := range d.tickingStatuses() {
		if !d.Alive() {
			break
		}
		// **A tick is amplified by whatever the carrier is vulnerable to**, exactly as a blow is. A
		// burn is damage this duelist takes, and a rule that exempted it would be "damage, except the
		// kind that arrives at the end of the round" — see EffectDamageAmplification.
		tick := amplify(d.Statuses[id].Amount, d.vulnerability())
		d.CurrentLife = reduce(d.CurrentLife, tick)

		// Side and Target are both this duelist, because nobody acted. The status was applied by an
		// attack rounds ago and whoever applied it may not even be alive to see this.
		events = append(events, Event{
			Kind:   KindBurned,
			Side:   side,
			Target: side,
			Status: id,
			Amount: tick,
			Life:   d.CurrentLife,
			Round:  round,
		})

		if !d.Alive() {
			events = append(events, Event{
				Kind:   KindDefeated,
				Side:   side,
				Target: side,
				Round:  round,
			})
		}
	}

	return events, tickStatuses(d)
}

// healAtTurnStart is every worn regeneration relic firing, at the top of this duelist's own turn.
//
// **It is a whole function rather than four lines inside playTurn** because it is the only thing
// that happens before the chill, and the order there is the argument — see MomentTurnStart.
//
// **A share that restored nothing writes no beat.** A duelist at full life has a relic that did not
// fire, and a figure leaving the ring carrying a zero would say it did. Same rule as the heal rider
// and the drain.
func healAtTurnStart(events []Event, side Side, actor Duelist, round int) ([]Event, Duelist) {
	for _, h := range actor.healsFrom() {
		before := actor.CurrentLife
		actor.CurrentLife = restore(actor.CurrentLife, actor.MaxLife*h.Pct/100, actor.MaxLife)
		if actor.CurrentLife == before {
			continue
		}

		events = append(events, Event{
			Kind:   KindRegenerated,
			Side:   side,
			Target: side,
			Relic:  h.Relic,
			Amount: actor.CurrentLife - before,
			Life:   actor.CurrentLife,
			Round:  round,
		})
	}
	return events, actor
}

// blockedByShield spends one of the target's shields — the one of element `shield` that
// shieldedHits named — against one incoming hit, and reports whether the hit was eaten. A blocked hit
// lands nothing at all: no damage, no life change, and no KindDamage for the feed to draw.
//
// **A shield of the hit's own element banks an action point** for the target's next turn — see
// Duelist.Surge. Basic is no element, so a plain shield eating a plain hit matches nothing.
//
// **It is checked before weight and vulnerability** — everything downstream shapes a figure, and a
// blocked hit never produces one. Ordering it after them would spend a shield on arithmetic nobody
// sees.
//
// `slot` is the card's seat in the turn and `hit` is which term of the hand this was; a solo
// attacker has no hand and passes zero.
func blockedByShield(events []Event, side Side, target Duelist, card Card, shield Element, slot, hit, round int) ([]Event, Duelist, bool) {
	target, spent := target.spendShield(shield)
	if !spent {
		return events, target, false
	}
	surged := shield != Basic && shield == card.Element
	if surged {
		target.Surge++
	}
	events = append(events, Event{
		Kind:    KindBlocked,
		Side:    other(side),
		Action:  card.Concept,
		Element: shield,
		Target:  other(side),
		Amount:  target.Shields.Count(),
		Slot:    slot,
		Hit:     hit,
		Surged:  surged,
		Round:   round,
	})
	return events, target, true
}

// resolveSoloAttacks is the attack phase of a duelist whose cards form no hands: **every attack
// resolves completely, in queue order, before the next one starts**.
//
// **No hand is read and no hand event is emitted.** There is no set to score, so there is no
// multiplier: each card lands its own face damage as its own hit, and the screen writes a sentence
// per card because there is no hand line to carry them.
//
// What it shares with the hand-forming phase in hit.go, because these are rules about attacking
// rather than rules about hands:
//
//   - **One beat per slot.** Every attack card announces itself with a KindAction, so playback can
//     still count how far through the round it is — see TestEverySlotIsEitherTakenOrChilled.
//   - **A shock rolls once per hit.** Each attack is its own chance to miss.
//   - **Weight, vulnerability, then shields, then statuses**, in that order: weight is a property of
//     the attacker and vulnerability of the target, so everything the defender actively does happens
//     to a hit both of them have already shaped.
func resolveSoloAttacks(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	round int,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	targetSide := other(side)

	attacked := false

	// **Which attacks the shields eat is decided before the turn starts**, not as each hit arrives.
	// See shieldedSlots.
	eaten := shieldedSlots(actor, target, turn)

	for i, slot := range turn {
		if slot.Card.Category() != CategoryAttack {
			continue
		}
		attacked = true

		events = append(events, Event{
			Kind:    KindAction,
			Side:    side,
			Action:  slot.Card.Concept,
			Element: slot.Card.Element,
			Round:   round,
		})

		if attackMisses(actor, rng) {
			events = append(events, Event{
				Kind:    KindMissed,
				Side:    side,
				Action:  slot.Card.Concept,
				Element: slot.Card.Element,
				Target:  targetSide,
				Slot:    i,
				Round:   round,
			})
			continue
		}

		// **One shield, one hit**, and the hits it eats were chosen before the turn began — the
		// matching element first and then the heaviest, by shieldedSlots. Spending is here rather than up there because a missed
		// hit spends nothing: the roll above continues before this line.
		if blocked := false; eaten[i] >= 0 {
			events, target, blocked = blockedByShield(events, side, target, slot.Card, eaten[i], i, 0, round)
			if blocked {
				continue
			}
		}

		dmg := blunt(actor.CardDamage(slot.Card), actor.weight())
		dmg = amplify(dmg, target.vulnerability())

		target.CurrentLife = reduce(target.CurrentLife, dmg)
		events = append(events, Event{
			Kind:    KindDamage,
			Side:    side,
			Target:  targetSide,
			Action:  slot.Card.Concept,
			Element: slot.Card.Element,
			Slot:    i,
			Amount:  dmg,
			Life:    target.CurrentLife,
			Round:   round,
		})

		// One card, and the same relics the other phase reads. An enemy wears none, so this does
		// nothing for the only duelists that are solo attackers today — it is here because the rule
		// belongs to attacking, not to hand-forming.
		for _, a := range actor.statusesFrom([]Card{slot.Card}) {
			applied, amount, ok := applyStatus(target, a.Status, actor)
			if !ok {
				continue
			}
			target = applied
			events = append(events, Event{
				Kind:    KindStatus,
				Side:    side,
				Target:  targetSide,
				Element: slot.Card.Element,
				Status:  a.Status,
				Relic:   a.Relic,
				Amount:  amount,
				Life:    target.CurrentLife,
				Round:   round,
			})
		}

		if !target.Alive() {
			events = append(events, Event{Kind: KindDefeated, Side: side, Target: targetSide, Round: round})
			return events, actor, target
		}
	}

	// **The defenses are spent only if something was swung at them**, which is the hand-forming
	// phase's rule too: a turn with no attacks in it returns before clearing, and expireDefenses
	// takes them at the start of their owner's next turn instead.
	if attacked {
		target = ClearDefenses(target)
	}
	return events, actor, target
}

// playRiders fires every rider on every card of a turn, in queue order.
//
// **One event per rider that did something.** A heal on a duelist already at full life is a rider
// that fired and changed nothing, and emitting a zero would put a line in the feed saying life was
// restored when none was — so the cap is applied first and a no-op is silent. The rider is still
// spent, because it is a property of the card rather than a charge.
func playRiders(events []Event, side Side, actor Duelist, turn []Slot, held []Card, round int, luck *rand.Rand) ([]Event, Duelist) {
	for i, slot := range turn {
		// **Shields first, and from the same seat a defend card raises them.** A rider is not a
		// defend card — it is on a Jab, and the Jab is about to swing — so this cannot wait for
		// the defend phase without a shielding attack being the only card in the game whose two
		// halves happen in different phases. It goes through raiseShields, so the five-shield cap
		// and the pip row hold exactly as they do for a Guard.
		if up := slot.Card.ShieldOnPlay(); up > 0 {
			actor = actor.raiseShields(slot.Card.Element, up)
			events = append(events, Event{
				Kind:    KindRaised,
				Side:    side,
				Action:  slot.Card.Concept,
				Element: slot.Card.Element,
				Slot:    i,
				Rider:   RiderShieldOnPlay,
				Amount:  up,
				Life:    actor.Shields.Count(),
				Round:   round,
			})
		}

		// **The metals gamble here, on every play, for the rest of the run.** A golden card is not
		// a consumable that was spent once: it is what the card permanently became, so the roll
		// belongs to the moment the card is played and happens as often as the card is.
		//
		// **The grant lands on the fighting duelist and is announced for the run.** Moving `actor`
		// is what makes the point of DMG worth something for the rest of this fight; the event is
		// what lets `internal/screens` move the figure the run owns. Doing only one of the two
		// would be a bonus that evaporated at the round boundary, or one the player could not use
		// until the next fight.
		if odds := slot.Card.GoldenOdds(); odds > 0 {
			dmg, life := rollGolden(odds, actor.rollScale(), luck)
			if dmg > 0 {
				actor.DMG += dmg
				events = append(events, Event{
					Kind:    KindGrantedDMG,
					Side:    side,
					Target:  side,
					Action:  slot.Card.Concept,
					Slot:    i,
					Rider:   RiderGolden,
					Element: slot.Card.Element,
					Amount:  dmg,
					Life:    actor.CurrentLife,
					Round:   round,
				})
			}
			if life > 0 {
				// **The ceiling and the floor together.** A maximum that rose while the player
				// stood where they were would read as nothing having happened, which is the same
				// pairing `session.Equip` makes when it applies the run's life bonus.
				actor.MaxLife += life
				actor.CurrentLife += life
				events = append(events, Event{
					Kind:    KindGrantedLife,
					Side:    side,
					Target:  side,
					Action:  slot.Card.Concept,
					Slot:    i,
					Rider:   RiderGolden,
					Element: slot.Card.Element,
					Amount:  life,
					Life:    actor.CurrentLife,
					Round:   round,
				})
			}
		}

		// **Silver needs no grant event**, because vitae already has a way out of a resolved round:
		// the purse the duel closes with, less the one it opened with. See KindVitae.
		if odds := slot.Card.SilverOdds(); odds > 0 {
			if paid := rollSilver(odds, actor.rollScale(), luck); paid > 0 {
				actor.Vitae += paid
				events = append(events, Event{
					Kind:    KindVitae,
					Side:    side,
					Target:  side,
					Action:  slot.Card.Concept,
					Slot:    i,
					Rider:   RiderSilver,
					Element: slot.Card.Element,
					Amount:  paid,
					Round:   round,
				})
			}
		}

		heal := slot.Card.HealOnPlay()
		if heal <= 0 {
			continue
		}

		before := actor.CurrentLife
		actor.CurrentLife = restore(actor.CurrentLife, heal, actor.MaxLife)
		if actor.CurrentLife == before {
			continue
		}

		events = append(events, Event{
			Kind:    KindHealed,
			Side:    side,
			Target:  side,
			Action:  slot.Card.Concept,
			Slot:    i,
			Rider:   RiderHealOnPlay,
			Element: slot.Card.Element,
			Amount:  actor.CurrentLife - before,
			Life:    actor.CurrentLife,
			Round:   round,
		})
	}

	// **The held hand pays after the played cards, and it pays once per card.** A card kept back
	// is not spent, so this fires again on every turn it is still being held — which is the whole
	// bargain the in-hand riders offer: a card worth more in the hand than in the turn.
	//
	// **One event per card rather than one for the turn**, because the feed names the card that
	// paid and a single summed line would name none of them.
	for _, c := range held {
		paid := c.VitaeInHand()
		if paid <= 0 {
			continue
		}
		events = append(events, Event{
			Kind:    KindVitae,
			Side:    side,
			Target:  side,
			Action:  c.Concept,
			Rider:   RiderVitaeInHand,
			Element: c.Element,
			Amount:  paid,
			Round:   round,
		})

		// **The duelist's own copy of the purse moves with the announcement**, so a later turn of
		// this fight reads what this one paid. The run's figure is still the screen's to add — see
		// Duelist.Vitae, which says why there are two.
		actor.Vitae += paid
	}
	return events, actor
}

// restore is reduce's opposite, and it caps at the duelist's maximum.
//
// **Nothing in the game heals above full**, which is a rules decision rather than an arithmetic
// one: a life total above the bar the card draws would be a number with nowhere to be shown.
func restore(life, heal, max int) int {
	life += heal
	if life > max {
		return max
	}
	return life
}

// reduce takes damage off a life total without letting it go negative.
func reduce(life, dmg int) int {
	life -= dmg
	if life < 0 {
		return 0
	}
	return life
}

// other is the side that is not this one.
func other(s Side) Side {
	if s == SideA {
		return SideB
	}
	return SideA
}

// shieldedSlots picks which of a solo attacker's cards the target's shields eat, and with which
// shield: **the matching element first, then the heaviest hits**, whatever order they were queued
// in. It reports, per card of the turn, the element of the shield that eats it or -1 for a card no
// shield eats.
//
// **What a shield costs to raise does not vary with the opponent's queue order, so what it is worth
// does not either.** Eating whichever attack came first would make a shield worth whatever the
// creature happened to lead with — a Giant Bat opening with a Nip would spend it on two damage and
// then land a Drain for ten.
//
// **Ranked on CardDamage alone, and that is the whole of the arithmetic rather than a shortcut.**
// Everything downstream of a card's own damage — the attacker's weight and the target's
// vulnerability — is one multiplier applied identically to every attack in the turn, so neither can
// reorder two cards. Projecting the whole pipeline per card would be a second resolver that agreed
// with the first.
//
// **It is a snapshot of the turn's opening state, knowingly.** A status landed by an early hit
// amplifies the ones after it, so a shield can be provably not-optimal in hindsight. That is the
// price of deciding up front, and deciding up front is what lets the screen show the whole exchange
// before the creature swings — see screens.shatter.
func shieldedSlots(actor, target Duelist, turn []Slot) []Element {
	planned := make([]int, len(turn))
	elems := make([]Element, len(turn))
	for i, slot := range turn {
		planned[i] = -1
		elems[i] = slot.Card.Element
		if slot.Card.Category() == CategoryAttack {
			planned[i] = actor.CardDamage(slot.Card)
		}
	}
	return shieldedHits(planned, elems, target.Shields)
}

// shieldedHits is the rule both attack phases share: given what each hit is planned to deal, -1 for
// an entry that is not a hit at all, and each hit's element, it says which shield eats which hit.
// The answer is per hit: the element of the shield that eats it, or -1 for a hit that lands.
//
// **Two passes, and the order is the rule** *(owner's call)*:
//
//  1. **Every shield takes the heaviest hits of its own element first**, because a matched block
//     banks an action point — see Duelist.Surge. Basic is no element and matches nothing.
//  2. **Whatever shields are left take the heaviest of what is left**, whatever its element,
//     spent lowest element first so the choice is a function of the stack and nothing else.
//
// So one ice shield against an ice Nip and a fire Drain eats the Nip: the point it banks is the
// trade the player made by raising ice.
//
// **Ties go to the earliest**, so the mask is a function of the lists and nothing else — a turn of
// three identical hits against one shield loses the first of them.
//
// A selection rather than a sort, because the list is at most a turn's landings long and the count
// taken is at most the shields standing, and it keeps the tie-break impossible to get wrong.
func shieldedHits(planned []int, elems []Element, shields ShieldStack) []Element {
	eaten := make([]Element, len(planned))
	for i := range eaten {
		eaten[i] = -1
	}
	heaviest := func(match func(i int) bool) int {
		best := -1
		for i, dmg := range planned {
			if eaten[i] >= 0 || dmg < 0 || !match(i) {
				continue
			}
			if best < 0 || dmg > planned[best] {
				best = i
			}
		}
		return best
	}

	for _, e := range AllElements {
		if e == Basic {
			continue
		}
		for shields[e] > 0 {
			best := heaviest(func(i int) bool { return elems[i] == e })
			if best < 0 {
				break
			}
			eaten[best] = e
			shields[e]--
		}
	}

	for _, e := range AllElements {
		for shields[e] > 0 {
			best := heaviest(func(int) bool { return true })
			if best < 0 {
				return eaten
			}
			eaten[best] = e
			shields[e]--
		}
	}
	return eaten
}
