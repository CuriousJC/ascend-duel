package combat

// The duelist: who is fighting, what they are carrying into the round, and the two things
// that are spent during one — action points and raised defenses.
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

// Other is the side that is not this one.
func (s Side) Other() Side { return other(s) }

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

	// Actions is this duelist's action-point budget. It is what a round is
	// spent out of, and cards cost 1 to 3 of it.
	Actions int

	MaxLife     int
	CurrentLife int

	// Vitae is the purse the run behind this duelist is carrying, **live for the length of the
	// fight**: seeded by session.Equip and stepped by every KindVitae the round announces.
	//
	// **The rules hold a copy, not the purse itself** *(owner's call, 2026-09-05)*. A rider paying
	// vitae for a card kept in hand is still announced rather than applied — the run's own figure
	// is the screen's to move — but a relic that reads the purse has to see what an earlier turn of
	// the same fight paid, so a duel cannot be handed one number at fight-start and left with it.
	// The copy is rebuilt on the next Equip, so it can only ever drift inside one fight.
	Vitae int

	// Shields is how many incoming attacks this duelist can still eat outright, **counted by the
	// element of the card that raised them** — see ShieldStack. It is what the player's defend cards
	// buy: Brace for one and Block for two. Guard is a third rung the file still declares at zero
	// copies, so the rules can resolve a 3-shield card that nothing deals.
	//
	// **A count rather than a percentage, because an enemy turn is several attacks.** Every
	// creature in the game is a solo attacker, so its turn resolves card by card with a figure
	// each; a shield takes one of those away entirely, which is what makes "how many hits am I
	// taking this round" a question the player can answer exactly rather than approximately.
	//
	// **The element is a rule, not a colour.** A shield eats a hit of any element, but one that eats
	// a hit of its own element banks an action point for its owner's next turn — see Surge and
	// shieldedHits, which spends the matching shields first.
	//
	// **It is raised and expires on the same schedule Defends do** — up at the end of a turn,
	// standing through the opponent's whole turn, gone at the start of its owner's next. See
	// ClearDefenses, which drops both, and expireDefenses, which says when.
	//
	// **An unspent shield is lost rather than kept**, which is what makes a defense a read of the
	// turn in front of you: raising more than the round throws away is a wasted point, so the
	// question the card asks is how hard *this* turn hits and never how long you can stockpile.
	//
	// **Nothing in the game gives one to an enemy**, and the asymmetry is deliberate rather than
	// unfinished — see VerbShield. A count meeting a hand-forming attacker would delete that
	// duelist's whole turn, since a hand lands one figure however many cards went into it.
	Shields ShieldStack

	// Surge is action points banked for this duelist's next turn and no other: one for every hit a
	// shield of the hit's own element ate. ActionPoints adds it to the budget, and the start of the
	// duelist's own turn spends it — so it buys exactly the turn after the blocks, whatever that
	// turn costs. **It is not capped**: five matched blocks are five more points.
	//
	// **It never outlives a fight.** The screen's reset between duels drops it with the shields.
	Surge int

	// Element is the duelist's own element, and **Basic means it has none** — which is the player,
	// and every bare `Duelist{}`. A creature is dealt one with its floor.
	//
	// **A hit of the target's own element fizzles**: it lands nothing at all. See fizzles. The
	// player has no element, so the rule runs one way by construction; the mirror of it is the
	// shield that matches the hit it eats, which banks a Surge.
	Element Element

	// Statuses is what has been done to this duelist, **indexed by status** — see status.go for
	// the lifecycle, which is one rule for all of them.
	//
	// **It was indexed by element until 2026-08-17**, which is the array the relic grammar could not
	// use: one element applying two statuses is the case that breaks it, and a status arriving from
	// something that is not a color at all has no seat in it. The price moves with the index —
	// `statuses.json` is now the append-only file, because inserting a record mid-file re-points
	// every status a duelist is carrying.
	//
	// An array rather than named fields for the reason it always was: a new status does not grow
	// this struct, and *"consume the status this card applies"* stays expressible.
	//
	// The defenses above deliberately stay where they are. Defend is a card effect rather than a
	// status, and filing it in this table would say it was one.
	Statuses [MaxStatuses]Status

	// Relics is what this duelist is wearing, in worn order, and RelicCount is how many of the
	// array is in use. See relic.go for the grammar and WornRelics for why the order is a rule.
	//
	// **It is what makes an element do anything at all** *(2026-08-16)*. A fire attack from a
	// duelist with no fire relic is a plain attack with a red border: it counts toward a hand, it
	// is discounted by nothing, and it applies no burn. See status.go for the argument, which is
	// that statuses given away free left the first three relics with no mechanic of their own.
	//
	// **It was `[ElementCount]bool` until 2026-08-17**, and the grammar is what took the flags
	// away: a form multiplier and a vitae relic have no element to be a bit under.
	//
	// **The relic is read off the attacker, never the victim.** Your fire relic makes *your* fire
	// attacks burn; it does nothing when a fire attack is aimed at you.
	//
	// **A slice, and there is no width at all** *(owner's call, 2026-09-17)*. It was a fixed array
	// plus a count until then, sized by a `MaxWornRelics` constant — a width rather than a rule,
	// but a limit all the same, and one that put a ceiling on a thing the design has no ceiling
	// for. **How many relics a duelist may wear is `relicSlots()` and nothing else**; how many they
	// *could conceptually* wear is now unbounded.
	//
	// **What the array was buying was value semantics, and that is now bought explicitly.** This
	// package hands duelists around by value — `ResolveRound` takes two and returns two, and the
	// rules step `Relics[i].Grown` on their own copy so growth lands on the run only once the round
	// is settled, exactly as the purse does. A slice aliases, so every place that takes a duelist
	// it intends to modify clones this first: see `cloned`, which is called at the top of
	// resolveRound and by Wearing. **Forgetting that is invisible** — the growth simply lands early,
	// no test goes red — so a new entry point that copies a Duelist calls `cloned` or it is wrong.
	//
	// **It also cost Duelist and Event their comparability**, which only ever had two readers:
	// TestRoundIsDeterministic, which compares with reflect.DeepEqual now, and the screen's face
	// cache, which keys on cards.Spec and never held one of these.
	//
	// **Enemies never wear one.** The zero value is an empty hand and nothing sets it for them, so
	// an enemy's elements are inert by construction rather than by a rule written down somewhere
	// else. Statuses reaching the player by some other route later is expected; it will not be by
	// an enemy putting on jewelry.
	Relics []WornRelic

	// RelicSlots is how many of those seats this duelist may actually fill. **Zero means
	// DefaultRelicSlots**, which is the five every duelist in the tower fights on — see
	// relicSlots(), where that reading lives, and session/relic.go, where a run hands its own
	// number over. It is a separate field from the array's width because the width is a fact
	// about keeping Duelist comparable and the cap is a rule a brand can move.
	RelicSlots int

	// SoloAttacks makes this duelist's attack cards resolve **one at a time, in the order they
	// were queued**, each landing its own hit at face damage — instead of being read as a set and
	// every hit multiplied through the hand table.
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

	// HandStones is how many stones this duelist holds for each rung of the hand ladder, indexed
	// by the rung's seat in the catalog — see stone.go, which owns the seats and the arithmetic.
	//
	// **A run's opinion about the ladder, carried by the fighter rather than by the catalog.**
	// `handTable` is package state shared by every fight and every tool, so a run raising a rung in
	// place would raise it for the enemy planner and for the review sheets. Equipping is where a
	// run's stones reach a duelist, which is the same seat `Relics` arrives in.
	//
	// **A fixed array rather than a map**, exactly like the defend set and the relic row above and
	// for the same reason: Duelist has to stay comparable, and `TestRoundIsDeterministic` compares
	// two resolved duelists with `==`.
	//
	// **Enemies never hold one.** Nothing sets it for them, so the zero value is a duelist reading
	// the catalog as written — and an enemy has `SoloAttacks` anyway, so it forms no hands to
	// raise.
	HandStones [MaxHandSlots]int

	// RoundLimit is how many rounds this duelist may fight before the clock kills them. **Zero is
	// no clock at all**, which is what every enemy and every bare `Duelist{}` in a test carries.
	//
	// **Zero means unlimited rather than the default on purpose.** A default of five here would
	// have put every existing multi-round test on a timer it was never written against, and the
	// safe direction for a rule nobody asked for is off. The run is what turns the clock on — see
	// session.Session.RoundLimit and DefaultRoundLimit, which is the number it turns it on at.
	//
	// **It is a property of the duelist rather than an argument to ResolveRound** for the reason
	// HandStones is one: the run's opinion reaches a fight through Equip, which is the same seat
	// the relics and the stones arrive in, and a relic or a brand that moves the limit later moves
	// this field rather than a signature every caller and test would have to grow.
	//
	// **The engine has no idea which side is a person**, so both sides are asked about their own
	// clock rather than the player's being read off SideA. Nothing sets it for a creature.
	RoundLimit int
}

// Alive reports whether this duelist can still fight.
func (d Duelist) Alive() bool { return d.CurrentLife > 0 }

// ClearDefenses drops everything a turn put up, which is the shields.
//
// **It stays a function and stays exported even though it now clears one field** *(2026-09-16)*.
// It answers "this duelist is no longer defending", and that is a question the combat screen asks
// between fights without wanting to know what defending consists of — a screen that cleared the
// field by hand is how a raised defense once survived into the next duel. It cleared two mechanics
// until the percentage guard was deleted, and the argument for one door over two outlives the
// second mechanic.
//
// **An unspent shield is dropped rather than kept.** It expires with the turn it was raised
// against; see Duelist.Shields.
func ClearDefenses(d Duelist) Duelist {
	d.Shields = ShieldStack{}
	return d
}

// raiseShields adds to the shield count, and it is the whole of what VerbShield does.
//
// **A duelist's total is not capped; maxShields bounds one card** *(owner's call, 2026-09-09)*.
// The clamp here said the opposite for nine days, against maxShields' own doc comment — three
// Guards is nine shields "and is meant to be" — so one of the two was wrong and this was it. What
// bounds a turn is the action budget, exactly as it bounds nine attacks; a second bound on the
// total was a cap on a resource the player had already paid for.
//
// **The argument the clamp was written on does not survive the round limit.** It said a sixth
// shield could never be spent, because a turn holds MaxActions cards and so throws at most five
// attacks. That is true of *one* turn and shields last exactly one — but it prices a shield at
// what it stops rather than at what it cost, and the player buying a tenth is buying insurance
// against a turn they cannot see. Overpaying for it is a poor play, not an impossible one.
//
// The readout is what actually pays for this, and it is a screen problem rather than a rule:
// the pip row on the duelist card fits six at its current pitch. See screens.maxShieldPips.
func (d Duelist) raiseShields(e Element, n int) Duelist {
	d.Shields[e] += n
	return d
}

// spendShield takes one shield of the given element if there is one, and reports whether an
// incoming attack was eaten.
//
// **A shield negates one hit outright — no damage, no partial figure.** That is the whole mechanic:
// one shield buys one of several hits, whichever kind of attacker is swinging. Which shield eats
// which hit is decided up front by shieldedHits; this only takes the one it named.
func (d Duelist) spendShield(e Element) (Duelist, bool) {
	if e < 0 || int(e) >= ElementCount || d.Shields[e] <= 0 {
		return d, false
	}
	d.Shields[e]--
	return d, true
}

// ShieldStack is a duelist's standing shields, counted by the element of the card that raised
// each. **Indexed by element and fixed-width**, so a Duelist stays a value: a slice here would
// alias between the copies the resolver hands around.
type ShieldStack [ElementCount]int

// Count is how many shields are standing, of every element.
func (s ShieldStack) Count() int {
	n := 0
	for _, c := range s {
		n += c
	}
	return n
}

// baseMaxActions is how many actions one duelist may take in a round, whatever they cost.
const baseMaxActions = 5

// MaxEchoLandings is the most times one card can land in a turn — the most hits it can throw —
// echoes and repeats included.
//
// **A width rather than a design cap**, exactly like MaxStatuses: Event's hand
// arrays are fixed so an Event stays comparable, and every landing is a term in them. Five is
// generous against the one echo relic that exists, which lands a card three times.
const MaxEchoLandings = 5

// MaxActions is the second of the two bounds on a round. **A round is bounded by cost and
// by count, independently and on purpose**: the budget gates what can be afforded, and this
// gates how much can happen at all — which still bites when discounts have taken cards to
// free, and is what stops a swarm from becoming unbounded as its speed grows.
//
// It is a method rather than the bare constant it used to be, and it lives here rather than
// on the screen where `maxSelected` used to. Both were deliberate: it is a **rule**, so the
// opponent's planner has to obey it exactly as the player's selection does, and making it a
// function of the duelist is what gives a relic or a brand raising the cap somewhere to bite
// without touching a single call site. See MECHANICS.md.
func (d Duelist) MaxActions() int { return baseMaxActions }

// ActionPoints is how much this duelist has to spend in a round: the stat, plus whatever Surge
// the blocks of the opponent's last turn banked. Nothing else adds to it, so a player can plan
// against it for a whole fight and read the surge as a one-turn bonus on top.
//
// **It stays a method rather than becoming a field read**, for the reason MaxActions is one: a
// relic or a brand raising a budget wants somewhere to bite that is not every call site.
//
// **No status touches it.** A chill is a card off the front of the turn instead — see playTurn.
func (d Duelist) ActionPoints() int { return d.Actions + d.Surge }

// CanAfford reports whether a queued set fits inside this duelist's budget. The UI
// enforces this while the player builds a set; ResolveRound trusts what it is given
// so that a balance sim can deliberately probe outside the rules.
//
// **It is the duelist's own costs that are totaled** — see CostOf in relic.go — because a discount
// relic makes a cost a property of the pairing rather than of the card.
func (d Duelist) CanAfford(cards []Card) bool {
	return d.CostOf(cards) <= d.ActionPoints()
}
