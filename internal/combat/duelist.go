package combat

// The duelist: who is fighting, what they are carrying into the round, and the two things
// that are spent during one — action points and raised defences.
//
// **A Duelist is a value, not an object.** Every rule in this package takes one and returns a
// new one rather than mutating in place, which is what lets the resolver work out a whole round
// before anything is shown and what lets the balance tool replay the same fight from the same
// start. Nothing here reaches for a clock or a random source.
//
// Split out of combat.go on 2026-08-21, which held the duelist, the event vocabulary, the
// resolver and the planner in one file.

// Side identifies which duelist an event belongs to. The engine is deliberately
// symmetric — it has no notion of "player" — so callers map A and B onto whatever
// they like. Side A takes its whole turn before side B takes any of it.
type Side int

const (
	SideA Side = iota
	SideB
)

func (s Side) String() string {
	if s == SideA {
		return "A"
	}
	return "B"
}

// Duelist is a combatant's stats plus the combat state that persists between rounds.
// entities.Combatant embeds this and adds the sprite, which keeps graphics out of
// the rules entirely.
//
// Every field is comparable on purpose: TestRoundIsDeterministic compares two resolved
// duelists with ==, so nothing here may become a slice or a map. The defend queue is a fixed
// array plus a count rather than a slice for exactly that reason.
//
// **Three stats, and every one of them is the number it sounds like** *(2026-08-16)*. Constitution
// and Speed went with the same argument that took Strength the day before: each existed only to be
// converted into something else — `Con * 5` was life and `4 + Spd/10` was the action-point budget
// — so the player had to learn a number they could never act on directly. Speed was the clearer
// case of the two: twenty-four distinct values across the roster produced three distinct budgets.
type Duelist struct {
	// DMG is what a 1x attack deals in this duelist's hands, and it is the figure on the fighter
	// card. The ladder scales off it: a card declares its multiplier and the arithmetic is
	// `DMG * Amount / 100`.
	DMG int

	// Actions is this duelist's action-point budget before anything banked. It is what a round is
	// spent out of, and cards cost 1 to 3 of it.
	Actions int

	MaxLife     int
	CurrentLife int

	// Defends is the plan cards this duelist has raised and not yet spent, and DefendCount is
	// how many of the array is in use.
	//
	// **One card reaches it: Defend** *(2026-08-15)*. It is raised at the end of a turn and stands
	// until the start of its owner's next turn — long enough to cover the opponent's whole turn
	// once, whichever side raised it. See ClearDefenses. It is still a set rather than a flag
	// because two Defends in a round is a legal turn and has to mean something.
	//
	// **Every raised card answers the opponent's one blow, and they compose multiplicatively**
	// *(2026-08-14)*. A turn resolves a single attack, so "which card meets which blow" is a
	// question with no content: a second Defend takes half of what is left after the first has
	// taken half, and the order they were raised in changes nothing. `reductionFor` is what each is
	// worth, and it reads the card's own declared Amount.
	//
	// **A fixed array, not a slice**, because TestRoundIsDeterministic compares two resolved
	// duelists with == and nothing on this struct may stop being comparable. It is a set rather
	// than a queue now; the array is simply how a comparable set of at most five things is held.
	Defends     [maxPendingDefends]PendingDefend
	DefendCount int

	// The two halves of Prepare. GatheredAP is what has been banked *this* round; at the
	// round boundary it becomes BonusAP, which ActionPoints adds to the budget for the
	// round after. Splitting them is what makes the bonus arrive next round rather than
	// funding the turn that bought it.
	//
	// BonusAP is overwritten rather than added to at the boundary, so gathering twice in
	// one round is worth +4 next round while gathering once a round is worth a flat +2.
	// Stacking within a round is deliberate and is what puts the five-attack hand in
	// reach without a ring; carrying across rounds would compound without limit.
	BonusAP    int
	GatheredAP int

	// The two halves of Plan, and the same shape as Prepare above for the same reason: DrewCards
	// is what has been earned *this* round and BonusDraw is what the round after may draw.
	//
	// **The rules cannot draw a card, and this is how they ask for one** *(2026-08-15)*. There is
	// no deck in this package — the shuffle lives on the combat screen, which is what keeps the
	// engine free of randomness — so a Plan cannot hand its owner two cards. It records that two
	// are owed, and whoever holds the deck honours it when the round is over.
	//
	// **It is a bonus on the refill rather than an immediate draw**, and it has to be. The hand
	// refills to a fixed size at the round boundary, so cards handed over mid-round would simply
	// be two fewer drawn at the boundary and Plan would do nothing at all. Overwritten rather than
	// added to, exactly as BonusAP is: two Plans in one round is a hand of twelve next round, and
	// a Plan every round is a flat +2 rather than a hand that grows forever.
	BonusDraw int
	DrewCards int

	// Statuses is what has been done to this duelist, **indexed by status** — see status.go for
	// the lifecycle, which is one rule for all of them.
	//
	// **It was indexed by element until 2026-08-17**, which is the array the ring grammar could not
	// use: one element applying two statuses is the case that breaks it, and a status arriving from
	// something that is not a colour at all has no seat in it. The price moves with the index —
	// `statuses.json` is now the append-only file, because inserting a record mid-file re-points
	// every status a duelist is carrying.
	//
	// An array rather than named fields for the reason it always was: a new status does not grow
	// this struct, and *"consume the status this card applies"* stays expressible.
	//
	// The defences above deliberately stay where they are. Defend is a card effect rather than a
	// status, and filing it in this table would say it was one.
	Statuses [MaxStatuses]Status

	// Rings is what this duelist is wearing, in worn order, and RingCount is how many of the
	// array is in use. See ring.go for the grammar and WornRings for why the order is a rule.
	//
	// **It is what makes an element do anything at all** *(2026-08-16)*. A fire attack from a
	// duelist with no fire ring is a plain attack with a red border: it counts toward a hand, it
	// is discounted by nothing, and it applies no burn. See status.go for the argument, which is
	// that statuses given away free left the first three rings with no mechanic of their own.
	//
	// **It was `[ElementCount]bool` until 2026-08-17**, and the grammar is what took the flags
	// away: a form multiplier and a vitae ring have no element to be a bit under.
	//
	// **The ring is read off the attacker, never the victim.** Your fire ring makes *your* fire
	// attacks burn; it does nothing when a fire attack is aimed at you.
	//
	// **A fixed array plus a count rather than a slice**, exactly like the defend set above and
	// for the same reason: Duelist has to stay comparable. A WornRing is an ID and a number, so
	// the ring's own rules stay in the registry where they can be a slice.
	//
	// **Enemies never wear one.** The zero value is an empty hand and nothing sets it for them, so
	// an enemy's elements are inert by construction rather than by a rule written down somewhere
	// else. Statuses reaching the player by some other route later is expected; it will not be by
	// an enemy putting on jewellery.
	Rings     [MaxWornRings]WornRing
	RingCount int

	// SoloAttacks makes this duelist's attack cards resolve **one at a time, in the order they
	// were queued**, each landing its own blow — instead of being read as a set and scored
	// through the hand table.
	//
	// **It is what an enemy is** *(2026-08-17, owner's call)*. Hands are the player's mechanic:
	// the hands are counted off concepts, and an enemy has no axis to play with — every enemy
	// card in `data/enemies.json` is authored `basic` and `FormNone`, so an opponent's "hand"
	// was whatever its planner happened to afford. Now an
	// enemy holding three cards swings three times and the player can read the round off the
	// cards on the table.
	//
	// **The default is false, so a plain `Duelist{}` hands.** Hands are the norm and this is the
	// exception, which is why the field is named for the exception rather than for the norm: a
	// `Hands bool` would have made every existing literal — the whole test suite, the balance
	// tool's fighter — quietly stop hand-forming.
	//
	// **It is a flag on the duelist rather than a rule about SideB.** The engine has no idea which
	// side is a person, and it must not learn: the balance tool plays both sides headlessly, and
	// a rule keyed on the side would be a rule that cannot be tested from the other end.
	//
	// The alternative considered and rejected was deriving it from the cards — an enemy card is
	// `FormNone`, so "no form, no hand" needs no field at all. It was rejected because it
	// couples two things that are not the same thing: affixes are designed to *transform* an
	// enemy deck, and a card that gained a form would silently gain hands with it.
	SoloAttacks bool
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

// PendingDefend is one raised plan card waiting for the opponent's blow.
//
// **It is just the card.** It carried a charge count until 2026-08-14, when a turn stopped
// resolving more than one attack — counting incoming blows is meaningless when there is only
// ever one.
type PendingDefend struct {
	// Card is the whole card rather than its concept, **because what a defence is worth is a
	// property of the card** *(2026-08-17)*: a worm can scale one Defend without touching the
	// others. Storing the ID lost that the moment it was raised.
	Card Card
}

// maxPendingDefends bounds the defend set. A turn is capped at MaxActions cards and every one of
// them could be a defence, so this is everything a legal turn can raise.
const maxPendingDefends = baseMaxActions

// reductionFor is what one raised card takes off the blow: its own declared Amount, as a
// percentage.
//
// **Nothing reduces a blow to zero, and that is a rule rather than a number.** A turn lands one
// figure however many cards went into it, so total negation would be a whole opposing turn deleted
// by a single card — a dominant strategy rather than a decision. Something always lands, so the
// opponent is always still playing. `RegisterConcept` refuses a card declaring 100 or more, and
// `TestNoDefenceStopsABlowOutright` holds the resolver to it.
func reductionFor(card Card) int {
	if card.Spec().Verb != VerbDefend {
		return 0
	}
	return card.Amount()
}

// raiseDefend adds a defend card to the set.
//
// **An overflow is dropped rather than growing the set or panicking.** MaxActions caps a legal
// turn at five actions, so the array holds everything a legal turn can raise; ResolveRound
// deliberately trusts what it is handed so a balance sim can probe outside the rules, and a sim
// that queues six defends should get five of them rather than a crash.
func (d Duelist) raiseDefend(card Card) Duelist {
	if d.DefendCount >= len(d.Defends) {
		return d
	}
	d.Defends[d.DefendCount] = PendingDefend{Card: card}
	d.DefendCount++
	return d
}

// ClearDefenses drops every pending defence a turn put up.
//
// Exported because the combat screen resets a duelist between fights and has to be able to
// clear this without knowing what is in it — a screen that listed the fields by hand is how a
// raised defence once survived into the next duel.
func ClearDefenses(d Duelist) Duelist {
	d.Defends = [maxPendingDefends]PendingDefend{}
	d.DefendCount = 0
	return d
}

// baseMaxActions is how many actions one duelist may take in a round, whatever they cost.
const baseMaxActions = 5

// MaxActions is the second of the two bounds on a round. **A round is bounded by cost and
// by count, independently and on purpose**: the budget gates what can be afforded, and this
// gates how much can happen at all — which still bites when discounts have taken cards to
// free, and is what stops a swarm from becoming unbounded as its speed grows.
//
// It is a method rather than the bare constant it used to be, and it lives here rather than
// on the screen where `maxSelected` used to. Both were deliberate: it is a **rule**, so the
// opponent's planner has to obey it exactly as the player's selection does, and making it a
// function of the duelist is what gives a ring or a brand raising the cap somewhere to bite
// without touching a single call site. See MECHANICS.md.
func (d Duelist) MaxActions() int { return baseMaxActions }

// ActionPoints is how much this duelist has to spend in a round, including anything
// banked by a Prepare in the round before.
//
// **It is the stat plus the bank, and nothing else** *(2026-08-16)*. It used to be
// `4 + Spd/10 + BonusAP`, a conversion whose only observable effect was to flatten
// twenty-four distinct Speed values into three budgets. Now the file says what the budget is.
//
// **No status touches it.** A chill did until 2026-08-16, and it is now a card off the front of
// the turn instead — see playTurn. What that costs is the one thing the old version had going for
// it: an AP cut was felt while the player was still choosing, and a card taken off a committed
// turn is felt after they have. What it buys is a status a player can name.
func (d Duelist) ActionPoints() int {
	return d.Actions + d.BonusAP
}

// CanAfford reports whether a queued set fits inside this duelist's budget. The UI
// enforces this while the player builds a set; ResolveRound trusts what it is given
// so that a balance sim can deliberately probe outside the rules.
//
// **It is the duelist's own costs that are totalled** — see CostOf in ring.go — because a discount
// ring makes a cost a property of the pairing rather than of the card.
func (d Duelist) CanAfford(cards []Card) bool {
	return d.CostOf(cards) <= d.ActionPoints()
}
