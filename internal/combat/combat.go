package combat

import "math/rand"

// ResolveRound plays out one round and returns its event log along with the state
// both sides end in. ResolutionOrder decides the order it plays them in.
//
// Inputs are taken by value and never mutated, so a caller can re-run a round from
// the same starting state — the returned duelists are the authority on what changed.
// **`rng` is the round's randomness and may be nil.** It is the seat CLAUDE.md's determinism
// rules require — an injected source, never a package global — and today the only thing that
// draws from it is a shock roll. A nil source means no roll ever lands, which is what a caller
// with no business being random should pass: a preview, or a test pinning the parts of the
// engine that are still exact.
func ResolveRound(a, b Duelist, aCards, bCards []Card, round int, rng *rand.Rand) (events []Event, aAfter, bAfter Duelist) {
	return resolveRound(a, b, aCards, bCards, round, handTable, rng)
}

// resolveRound is ResolveRound with the catalogue injected. It exists so a test can drive a
// synthetic hand through the whole engine rather than only through the matcher.
func resolveRound(a, b Duelist, aCards, bCards []Card, round int, hands []Hand, rng *rand.Rand) (events []Event, aAfter, bAfter Duelist) {
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
	events, a, b = playTurn(events, SideA, a, b, appendTurn(nil, SideA, aCards), round, hands, rng)

	// B still loses its standing defenses even in a round it never gets to act in, which is
	// why this is not inside playTurn's early return: expiry is a property of the turn
	// arriving, not of anything happening in it.
	if a.Alive() && b.Alive() {
		events, b, a = playTurn(events, SideB, b, a, appendTurn(nil, SideB, bCards), round, hands, rng)
	} else {
		b = expireDefenses(b)
	}

	// **A always burns before B**, which is the same order the turns were played in and needs no
	// tie-break.
	events, a = endRound(events, SideA, a, round)
	events, b = endRound(events, SideB, b, round)

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
	round int,
	hands []Hand,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	actor = expireDefenses(actor)

	// A chill comes off the front, which needs no tie-break and so is the only pick that is
	// deterministic without inventing a rule. **The front of a turn is its attacks** — the phase
	// order puts them before the plans — so what a chill costs first is the blow, which is what
	// makes it worth planning around rather than merely suffering.
	//
	// **The action points are not refunded.** They were committed when the cards were queued,
	// and letting them come back would make a chill pure tempo; keeping them spent makes it
	// tempo and economy both.
	//
	// **The chill is read off the status and nowhere else** *(2026-08-17)*. A duelist used to
	// carry a separate counter as well, banked by a hand that took actions off the opponent's
	// next turn; hands buy damage alone now, so the status is the only thing that can take a
	// card and there is one place to look for how many.
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

	// **The attack phase is one blow, whatever it was made of.** Every attack card queued is
	// announced, then the hand they form is announced, then a single figure of damage lands. Five
	// Strikes are not five hits; they are one Four of a Kind.
	events, actor, target = resolveAttackPhase(events, side, actor, target, turn, round, hands, rng)

	// **The plan phase comes second, and that is what a defence needs** *(2026-08-15)*. A Defend
	// answers the *opponent's* blow, and the opponent acts after this turn ends — so a defence
	// raised at the end of a turn is the only one that is standing when anything is aimed at it.
	// Prepare and Plan both bank for the round after and do not care where they sit.
	//
	// **It is skipped if either side fell**, since a corpse raising a shield is a line in the log
	// nobody wants and a duel that is over does not need one.
	if !actor.Alive() || !target.Alive() {
		return events, actor, target
	}
	for _, slot := range turn {
		if slot.Card.Category() != CategoryPlan {
			continue
		}
		events, actor, target = resolvePlan(events, side, actor, target, slot.Card, round)
	}

	// **The turn is closed after everything in it has resolved**, which is what makes "a turn with
	// no plan card in it" a question this can answer. A duelist who fell mid-turn returns above and
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

// resolvePlan runs one plan card. **The plans are the only cards that still resolve one at a
// time**, because each does something to its own duelist rather than contributing to a shared
// blow.
func resolvePlan(
	events []Event,
	side Side,
	actor, target Duelist,
	card Card,
	round int,
) ([]Event, Duelist, Duelist) {
	events = append(events, Event{
		Kind:    KindAction,
		Side:    side,
		Action:  card.Concept,
		Element: card.Element,
		Round:   round,
	})

	// **The switch is on the verb, not on the card** *(2026-08-16)*. It used to name Prepare, Plan
	// and Defend one at a time, which meant an enemy's `Congeal` could not shield anything however
	// obviously it was a defence. Three verbs, any number of cards.
	spec := card.Spec()
	switch spec.Verb {
	case VerbBank:
		actor.GatheredAP += card.Amount()
		events = append(events, Event{
			Kind:   KindGathered,
			Side:   side,
			Amount: card.Amount(),
			Round:  round,
		})

	case VerbDraw:
		// **Recorded, not drawn.** There is no deck in this package; what is banked here is honoured
		// by whoever holds one when the round is over. See Duelist.BonusDraw.
		actor.DrewCards += card.Amount()
		events = append(events, Event{
			Kind:   KindDrew,
			Side:   side,
			Amount: card.Amount(),
			Round:  round,
		})

	case VerbDefend:
		// Raised, not spent. What it is worth is `reductionFor`, and it is read when the opponent's
		// blow arrives — see resolveAttackPhase.
		actor = actor.raiseDefend(card)
	}

	return events, actor, target
}

// expireDefenses drops everything the previous turn put up. Called at the start of a
// side's own turn, never at the round boundary — side B acts last, so a defense cleared
// at the boundary would have protected B from nothing at all.
//
// The clearing itself is ClearDefenses, which the combat screen also calls between fights; the
// timing rule is what lives here.
func expireDefenses(d Duelist) Duelist { return ClearDefenses(d) }

// endRound rolls what was banked this round into next round's budget, ticks a burn, and counts
// every status down one. Assignment rather than addition for the bank: two Prepares in one round
// are worth +4 next round, and preparing every round is worth a flat +2 rather than compounding
// forever.
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
		tick := d.Statuses[id].Amount
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

	d = tickStatuses(d)

	d.BonusAP = d.GatheredAP
	d.GatheredAP = 0

	// Same shape and the same reason: assignment, so a Plan widens exactly one hand.
	d.BonusDraw = d.DrewCards
	d.DrewCards = 0
	return events, d
}

// resolveAttackPhase is the whole of one side's offence: every attack card it queued, the hand
// they form, and the single blow that follows.
//
// **One blow per turn** *(2026-08-14)*. Attack cards no longer resolve one at a time; they are
// announced, and then `BlowFor` reads them as a set and says what they amount to. Cards that
// contribute to no hand are announced and then ignored — `Strike, Jab, Strike` is a Pair and the
// Jab is not in it, so it adds nothing to the figure.
//
// The order inside the blow is: shock roll, base damage from the hand's own cards, the hand
// multiplier, the attacker's earth weight, then the defender's raised cards. **Weight sits where
// it does because it is a property of the attacker** — it says how hard they can still swing — so
// everything the defender does happens to a blow that has already been blunted.
func resolveAttackPhase(
	events []Event,
	side Side,
	actor, target Duelist,
	turn []Slot,
	round int,
	hands []Hand,
	rng *rand.Rand,
) ([]Event, Duelist, Duelist) {
	targetSide := other(side)

	// **A solo attacker takes a different phase entirely, not a special case inside this one.**
	// The two are different shapes: one announces everything and then lands a single figure, the
	// other resolves each card completely before the next one starts. Threading a flag through
	// the blow, the multiplier and the hand event would leave a function whose every step had
	// two readings.
	if actor.SoloAttacks {
		return resolveSoloAttacks(events, side, actor, target, turn, round, rng)
	}

	// Every attack card is announced whether or not it ends up in the hand. **A slot that
	// resolved has to produce a beat**, because the screen counts one per slot to know how far
	// through the round playback is — see TestEverySlotIsEitherTakenOrChilled.
	attacks := 0
	for _, slot := range turn {
		if slot.Card.Category() != CategoryAttack {
			continue
		}
		attacks++
		events = append(events, Event{
			Kind:    KindAction,
			Side:    side,
			Action:  slot.Card.Concept,
			Element: slot.Card.Element,
			Round:   round,
		})
	}
	if attacks == 0 {
		return events, actor, target
	}

	blow := blowFor(turn, hands)
	if len(blow.Cards) == 0 {
		return events, actor, target
	}

	// The hand is announced before the blow lands, so a boosted figure never arrives before the
	// reason for it. **Every turn with an attack in it announces a hand** — a lone attack is the
	// High Card, which is a catalogue entry like any other rather than an absence.
	//
	// **It also carries the sum**, which is what the damage below is taken from — see handEvent.
	//
	// **A hand buys damage and nothing else** *(2026-08-17)*. It used to be able to bank action
	// points or take actions off the opponent's next turn, which is why there was a phase here
	// paying those out before the blow landed. Statuses come from elements and rings now, so the
	// multiplier is the whole reward and there is nothing to pay before the roll.
	swung := handEvent(side, blow, turn, actor, round)
	events = append(events, swung)

	// A shocked attacker may miss outright, and misses before anything else happens — no defence
	// spent, no status applied. The attack did not occur.
	//
	// **This is a roll**, and the only one in the package. See shockMissPct. Nothing is consumed
	// by it: a shock rolls on every attack it outlives, so the duelist comes back unchanged.
	if attackMisses(actor, rng) {
		events = append(events, Event{
			Kind:    KindMissed,
			Side:    side,
			Action:  turn[blow.Cards[0]].Card.Concept,
			Element: turn[blow.Cards[0]].Card.Element,
			Target:  targetSide,
			Round:   round,
		})
		return events, actor, target
	}

	// **Base damage is the cards in the hand, and the multiplier is DMG on top.** DMG is what one
	// Strike deals at this duelist's strength, which is the figure the duelist card shows — so
	// `20 + 10 x 1.5 = 35` for a pair of Strikes at Str 10, exactly as the design states it.
	//
	// That sum is the announcement's `Amount`, taken rather than repeated: the feed prints the
	// arithmetic, and a second copy of it here is the one way the printed sum could be wrong.
	dmg := blunt(swung.Amount, actor.weight())

	events, dmg = applyDefends(events, side, target, dmg, round)

	// Every defence is spent on the turn it answered.
	target = ClearDefenses(target)

	target.CurrentLife = reduce(target.CurrentLife, dmg)
	events = append(events, Event{
		Kind:   KindDamage,
		Side:   side,
		Target: targetSide,
		Amount: dmg,
		Life:   target.CurrentLife,
		Round:  round,
	})

	// **The blow lands whatever the attacker's rings say it does, and it does so because the hand
	// was formed rather than because the blow hurt.** A hand halved by a Defend still connected, and
	// making the status conditional on the final figure would mean a defensive card silently
	// un-applied something the attacker had already paid for.
	//
	// **Every status comes off a worn ring** *(2026-08-16, re-expressed in the grammar 2026-08-17)*.
	// A rainbow thrown by a duelist wearing two elemental rings lands two statuses; thrown by an
	// enemy it lands none. The colours still count toward the hand either way — what a ring buys is
	// the status, not the multiplier.
	//
	// The cards of the hand are what the rings match against, so a form ring or a concept ring
	// reaches this the same way an elemental one does. `statusesFrom` deduplicates, which is what
	// keeps two fire cards from announcing one burn twice.
	blowCards := make([]Card, 0, len(blow.Cards))
	for _, i := range blow.Cards {
		blowCards = append(blowCards, turn[i].Card)
	}
	// **The accumulator moves before the statuses are handed out and after the damage has landed.**
	// A growing ring is paid for the blow that just connected, never for the one it is about to
	// strengthen — otherwise the first fire attack of a fight would already be wearing its own
	// bonus.
	actor = actor.GrowOnHit(blowCards)

	for _, a := range actor.statusesFrom(blowCards) {
		applied, amount, ok := applyStatus(target, a.Status, actor)
		if !ok {
			continue
		}
		target = applied
		events = append(events, Event{
			Kind:   KindStatus,
			Side:   side,
			Target: targetSide,
			Status: a.Status,
			Ring:   a.Ring,
			Amount: amount,
			Life:   target.CurrentLife,
			Round:  round,
		})
	}

	if !target.Alive() {
		events = append(events, Event{
			Kind:   KindDefeated,
			Side:   side,
			Target: targetSide,
			Round:  round,
		})
	}

	return events, actor, target
}

// applyDefends runs every card the target has raised over one incoming blow and reports what is
// left of it, announcing each as it bites.
//
// **It does not spend them, and the caller clears them once the turn is over.** A defence covers
// exactly one opposing *turn* — see expireDefenses — which is one blow from a hand-forming duelist and
// several from a solo one. Spending them on the first blow would make a Defend nearly worthless
// against the very opponents that swing more than once.
//
// **They compose multiplicatively and the order is not read.** Multiplying what is left rather
// than adding the percentages is what stops two cards reaching zero by accident while keeping each
// one worth something: two Defends take three quarters rather than the whole thing, and a third
// takes seven eighths, which is a curve that never arrives.
func applyDefends(events []Event, side Side, target Duelist, dmg, round int) ([]Event, int) {
	for i := 0; i < target.DefendCount; i++ {
		card := target.Defends[i].Card
		pct := reductionFor(card)
		if pct <= 0 {
			continue
		}
		dmg = dmg * (100 - pct) / 100

		events = append(events, Event{
			Kind:   KindNegated,
			Side:   other(side),
			Action: card.Concept,
			Target: side,
			Amount: dmg,
			Round:  round,
		})
	}
	return events, dmg
}

// resolveSoloAttacks is the attack phase of a duelist whose cards form no hands: **every attack
// resolves completely, in queue order, before the next one starts**.
//
// **No hand is read and no hand event is emitted** *(2026-08-17)*. That is the whole of the
// difference — there is no set to score, so there is no multiplier. What lands is the sum of what
// was played, one figure at a time, and the screen writes a sentence per card because there is no
// phase line to carry them.
//
// Three things it keeps deliberately in step with the hand-forming phase, because they are rules about
// attacking rather than rules about hands:
//
//   - **One beat per slot.** Every attack card announces itself with a KindAction, so playback can
//     still count how far through the round it is — see TestEverySlotIsEitherTakenOrChilled.
//   - **One shock roll for the turn, not one per card.** A shock is "the turn's attack misses", and
//     rolling per card would both change what the status means and advance the one random stream in
//     the package a different number of times per round. A shocked solo attacker misses with
//     everything and says so on each card.
//   - **Weight, then defences, then statuses**, in that order, for the reason the other phase gives:
//     weight is a property of the attacker, so everything the defender does happens to a blow that
//     has already been blunted.
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
	missed := false
	rolled := false

	for _, slot := range turn {
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

		// The roll happens on the first card that could actually swing, and once only. Rolling
		// before the loop would advance the stream for a turn that never reaches one.
		if !rolled {
			missed, rolled = attackMisses(actor, rng), true
		}
		if missed {
			events = append(events, Event{
				Kind:    KindMissed,
				Side:    side,
				Action:  slot.Card.Concept,
				Element: slot.Card.Element,
				Target:  targetSide,
				Round:   round,
			})
			continue
		}

		dmg := blunt(actor.CardDamage(slot.Card), actor.weight())
		events, dmg = applyDefends(events, side, target, dmg, round)

		target.CurrentLife = reduce(target.CurrentLife, dmg)
		events = append(events, Event{
			Kind:   KindDamage,
			Side:   side,
			Target: targetSide,
			Amount: dmg,
			Life:   target.CurrentLife,
			Round:  round,
		})

		// One card, and the same rings the other phase reads. An enemy wears none, so this does
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
				Ring:    a.Ring,
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

	// **The defences are spent only if something was swung at them**, which is the hand-forming
	// phase's rule too: a turn with no attacks in it returns before clearing, and expireDefenses
	// takes them at the start of their owner's next turn instead.
	if attacked {
		target = ClearDefenses(target)
	}
	return events, actor, target
}

// handEvent packages what the attack phase formed for the screen: which hand, the multiplier,
// which cards of the turn earned it, and the arithmetic they come to.
//
// **It is the attack phase's one line in the feed** *(2026-08-14)*, so it carries everything that
// line has to say. The individual attack cards are still announced — a slot that resolved has to
// produce a beat — but the screen draws no sentence for them: five cards making one blow read as
// five blows, which is the thing one-blow-per-turn was meant to stop saying.
func handEvent(side Side, blow Blow, turn []Slot, actor Duelist, round int) Event {
	e := Event{
		Kind:       KindHand,
		Side:       side,
		Hand:       blow.Hand.ID,
		Multiplier: blow.Multiplier,
		Round:      round,
	}

	// **The blow is added up here and nowhere else.** The attack phase takes its damage figure off
	// this event rather than recomputing it, so the sentence the feed prints and the damage that
	// lands cannot be two different sums — and the dialog flying the figures down reads the same
	// per-card amounts the sum was made of.
	//
	// A card past the array's width is dropped from the *bracket* rather than from the sum, which
	// is the posture HandCards already takes: the arithmetic on screen may be short of a term
	// before the damage that lands is wrong.
	// **An echo seats the lead card again, at a smaller figure, right behind itself** *(2026-08-22,
	// owner's call)*. It is deliberately a term in this sum rather than a second blow: the turn
	// still lands once, the hand still multiplies one figure, and what the player sees is the first
	// card paying three times — "seven cards played, the first one three of them".
	//
	// **The echo does not reach the matcher.** `blowFor` has already run, so an echoed Strike does
	// not turn a Pair into Trips; it pays into the hand the real cards formed.
	worn := actor.WornRings()
	for n, i := range blow.Cards {
		card := turn[i].Card
		for t, d := range LandingAmounts(worn, card, n == 0, actor.CardDamage(card)) {
			e.Base += d
			if e.HandCardCount >= len(e.HandCards) {
				continue
			}
			e.HandCards[e.HandCardCount] = i
			e.HandAmounts[e.HandCardCount] = d
			e.HandCardCount++
			if t > 0 {
				e.EchoTerms++
			}
		}
	}

	lead := turn[blow.Cards[0]].Card

	// **The multiplier multiplies the cards** *(2026-08-18)*. There is no separate swing term: a
	// hand is worth a proportion of what its own cards deal, so a Pair of Lunges is worth more
	// than a Pair of Jabs by exactly the margin the cards themselves are worth.
	e.Amount = scaleDamage(e.Base, blow.Multiplier)

	e.Action = lead.Concept
	e.Element = lead.Element

	return e
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
