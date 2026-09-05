package combat

// The event vocabulary: what a resolved round hands back for a screen to replay.
//
// **This is the whole contract between the rules and the pictures.** ResolveRound decides an
// entire round before anything is drawn and returns this log; the screen walks it. Nothing on
// the far side of it may compute an outcome, and nothing here may know that a screen exists.
//
// **Two consequences worth keeping in mind when adding a kind.** A screen paces playback off
// this list, so a kind nobody draws still costs a beat — and internal/screens has a table that
// fails a build when a new kind arrives without a choreography entry, which is deliberate: an
// event with no picture and an event whose picture was forgotten otherwise look identical.
//
// **Slot and ResolutionOrder live here** because play order is part of what the log means: the
// resolver and the screen's two rows all read ResolutionOrder rather than deriving an order of
// their own, which is what stops the table showing a round in an order the engine did not play.
//
// Split out of combat.go on 2026-08-21.

type EventKind int

const (
	KindRoundStart EventKind = iota
	KindAction

	// KindNegated is the blow meeting a raised defence: Action is the card that answered it and
	// Amount is what is left of the blow afterwards.
	//
	// **One kind, one card.** It was three kinds for three cards that differed only in percentage;
	// `Action` already names which card it was, which is the whole distinction the feed was drawing.
	// Only creature guards reach it today, and the kind keeps its general name because what it
	// describes is a blow being reduced rather than one particular card doing it.
	KindNegated

	// KindRaised is a defend card putting shields up. Action is the card, Amount is how many it
	// raised, and Life carries the duelist's shield count afterwards, so a second Guard in the same
	// turn reads as five rather than as three twice.
	//
	KindRaised

	// KindBlocked is a shield eating one incoming attack outright. Action is the attack that was
	// stopped, Target is the duelist that spent the shield, and Amount is how many shields are
	// left afterwards.
	//
	// **It is not a KindNegated.** A guard leaves a figure and this leaves none, so a feed reading
	// `Amount` off the two would be reading a remaining blow in one case and a remaining shield in
	// the other. It is not a KindMissed either: a miss is the attacker's own failure and costs the
	// defender nothing, where this is something the defender paid for.
	KindBlocked

	// KindExpired is a duelist's standing shields lapsing unspent, at the start of their own next
	// turn. Amount is how many were lost.
	//
	// **It exists because a shield row has to empty when the shields do.** Every other change to
	// that count is announced — raised, and eaten one attack at a time — so an expiry that said
	// nothing would leave the readout showing a defence the engine had already taken away, for a
	// whole turn. This is the same argument that makes a chilled slot emit a beat.
	//
	// **Only shields raise it, and only when some were standing.** A percentage guard lapsing is
	// not drawn anywhere, so announcing it would be a beat with no picture; and a turn beginning
	// with nothing up is the ordinary case, which must not cost the feed a line.
	KindExpired

	KindDamage
	KindDefeated

	// KindHand says a hand formed. **Every one a turn forms is emitted before that turn's
	// first KindAction**, because the hand phase resolves before the cards do — so a boosted
	// hit is never shown before the reason for it. Several can arrive together and their runs
	// may overlap; see matchSlots.
	KindHand

	// KindChilled is one action lost to a chill. One event per action, so a chill deep enough to
	// take several narrates as the several things it actually is.
	KindChilled

	// The three element events, added 2026-08-12 with the statuses.

	// KindStatus is one element status landing on a duelist. Element says which, Amount says how
	// much was added by this hit, Target is who is carrying it.
	//
	// It is a separate event rather than a field on KindDamage because a status is not the blow:
	// a chill that lands is felt a round later and against a completely different card, and a
	// Resolution feed that folded it into the damage line would announce it at the one moment it
	// does nothing.
	KindStatus

	// KindMissed is an attack that never happened because its owner was shocked. Action is the
	// attack that was lost and Side is whose it was, which makes it the lightning counterpart of
	// KindChilled — a slot that resolves into nothing.
	//
	// It is deliberately not a KindNegated: nothing of the defender's stopped it, and a log
	// saying a blow was "stopped cold" by a defence that was never raised would send the player
	// looking for a card that is not there.
	KindMissed

	// KindBurned is a fire tick at the end of a round. Target is who burned; Side is the same,
	// because nobody acted.
	KindBurned

	// KindHealed is a rider restoring life to the duelist who played the card. Action is the card
	// that carried it, Amount is the life restored — **after the cap**, so a heal on full life
	// reports zero rather than the figure the rider names.
	//
	// **Its own kind rather than a negative KindDamage.** The two travel in opposite directions on
	// screen and read as opposite things in the log, and a feed that had to check the sign of an
	// amount to know which it was looking at is a feed one missing minus sign away from lying.
	KindHealed

	// KindVitae is a rider paying vitae for a card that stayed in the hand. Action is the card
	// that carried it, Amount is the figure, Side is whose hand it was.
	//
	// **It is an announcement to the feed, not the payment** *(and the payment moved on
	// 2026-09-05)*. The rules do step `Duelist.Vitae` when this fires — they have to, because a
	// ring reading the purse must see what an earlier turn of the round paid — but that is a copy
	// for the length of one fight. What the run is actually paid is the difference between the
	// purse the duel was handed and the one it hands back; see screens.payHeldVitae. **Do not sum
	// these events to move a purse**: that was the old way and it made two counters over one
	// figure.
	KindVitae

	KindRoundEnd
)

// Event is one entry in the replayable log for a single round.
// maxHandTerms is the width of a hand event's two arrays: **every landing a legal turn can produce**
// — each of its cards, each landing as many times as an echo or a repeat ring allows.
//
// It went from "one echoed card" to "every card" on 2026-08-22, when the form repeat rings landed:
// a repeat matches on form, so five crush cards under Aftershock is five cards landing twice.
// Over-long turns still drop terms from the *bracket* rather than from the sum.
const maxHandTerms = baseMaxActions * MaxEchoLandings

type Event struct {
	Kind   EventKind
	Side   Side      // who acted
	Action ConceptID // set on KindAction, on KindNegated for the defense that stopped it, on KindRaised for the card that raised the shields, on KindBlocked for the attack a shield ate, on KindChilled for the action lost, on KindMissed for the attack that never landed, and on KindHand for the card the blow led with
	Amount int       // damage dealt, shields raised or left standing, status applied, or on KindHand what the hand adds up to
	Target Side      // who took the damage
	Life   int       // target's life after the event
	Round  int

	// Element is the card's element on KindAction, KindMissed and KindStatus. Basic everywhere
	// else, which is also the zero value — an event with nothing to say about colour says `basic`,
	// exactly as a plain card does.
	Element Element

	// Status is which status is meant, on KindStatus and KindBurned.
	//
	// **It replaced reading Element for it** *(2026-08-17)*, because a status is no longer the same
	// object as a colour: two rings can put two different statuses on the same fire card, and an
	// event naming the colour could not say which had landed. Element still carries the card's own
	// colour on a KindStatus, which is what the feed's swatch and its sentence are drawn from.
	//
	// **The zero value is a real status**, the first one registered — the hazard Action carries for
	// concepts. It is set on the two kinds that mean it and read on no others.
	Status StatusID

	// Ring is the worn ring that applied the status, on KindStatus.
	//
	// **It is here because a status has a cause the player can see** *(2026-08-18)*. The screen
	// flies the word out of the ring that caused it, and there is no other honest way for it to
	// know which ring that was: reading it off the card's element would be a second rule about
	// something the grammar already decides, and it would be wrong the first time a form ring or
	// a concept ring applied a status - both of which RegisterRing accepts today.
	//
	// **Which ring, not which slot.** A RingID says something in a trace and in a test; a worn
	// index says nothing outside one duelist's array. The screen finds its position by walking the
	// worn list, which is at most five entries and is the same order the ring row is drawn in.
	//
	// **The zero value is a real ring**, exactly as Status's is a real status, so it is set on the
	// one kind that means it and read on no others. NoRing is the absence, for a caller that wants
	// to say so explicitly.
	Ring RingID

	// Hand is set on KindHand and names what the attack phase formed. The screen looks it up
	// with HandByID rather than being told its name here, so a hand renamed is renamed once.
	//
	// **It always names a hand**, because `blowFor` falls back to the catalogue's High Card: a
	// turn with an attack in it produces a blow, and a blow the engine could not name is the one
	// failure this model can have. `HandNone` is the zero value and reaches a screen only on an
	// event that is not a KindHand. The comment here claimed the opposite until 2026-08-19, and a
	// dead branch in the log was written against it.
	Hand HandID

	// Multiplier is the turn's damage multiplier in percent — the hand's, so 150 is the 1.5x a
	// pair earns. It is on the event because the screen has no business re-deriving a number the
	// resolver already worked out.
	Multiplier int

	// Base is the other term of the blow's arithmetic on KindHand, and it is here for the same
	// reason Multiplier is: the Resolution feed prints the sum — `(20 + 20) x 1.5 = 60` — and a
	// screen working a damage figure out for itself would be a second resolver.
	//
	// Base is what the hand's own cards carry, added up. `Amount` is that figure after the hand's
	// multiplier, and there is no third term: **the multiplier multiplies the cards** *(2026-08-18,
	// owner's call)*. It used to be applied to a separate reference swing of one 1x attack at the
	// attacker's DMG, added on top of the cards — which meant a hand's percent bought a fixed
	// figure rather than a proportion, so 500% was worth 2.5x the base on Jabs and 0.6x on Lunges.
	//
	// **Amount is the blow before the attacker's weight and before anything the defender raised**,
	// so it is what the hand was worth rather than what landed. What landed is the KindDamage
	// after it, and the gap between the two figures is exactly what the defence was worth.
	Base int

	// HandCards and HandCardCount are set on KindHand alongside Hand: **which cards of this
	// side's turn formed it**, as indices into the turn *as it was played*.
	//
	// **They are here so a screen never has to work out which cards earned a hand.** The
	// matcher already knows, and re-deriving it from the hand's pattern would be a second
	// matcher — the drift ResolutionOrder exists to prevent. It would also be wrong: a counted
	// hand is not contiguous, so Two Pair can be two cards, a card that earned nothing, and two
	// more.
	//
	// **A fixed array rather than a slice, because Event has to stay comparable** —
	// TestHandsDoNotBreakDeterminism compares two logs entry by entry with ==. It is sized to
	// maxHandTerms — every card a legal turn can hold, plus the extra landings an echo ring can
	// add — and a balance sim deliberately queueing more gets its extra cards dropped from the
	// *bracket* rather than from the hand, the same posture raiseDefend takes on an over-long
	// defend list.
	//
	// **A term is a landing, not a card** *(2026-08-22)*. An echoed card seats the same index two
	// or three times with a smaller amount each time, which is what makes the sum on screen read
	// as the card being played again rather than as one card worth more.
	//
	// The indices count the actions that actually resolved, chilled ones already removed,
	// which is the same sequence as this side's KindAction events — **events that have not
	// happened yet when this one arrives**, since the hand phase runs first. The screen seats
	// the whole turn at DUEL! rather than a card at a time, so the cards are there to bracket.
	HandCards     [maxHandTerms]int
	HandCardCount int

	// HandAmounts is what each of those cards deals, in the same order and to the same count.
	//
	// **It is here so the screen can show the arithmetic rather than assert it** *(2026-08-18)*.
	// The hand dialog flies each card's own figure down into a sum, and re-deriving one on the
	// screen would mean the screen owning `CardDamage`, the Strength scaling and every ring that
	// touches a card's damage — a second resolver, exactly what Base and Multiplier are on the
	// event to prevent. `Base` is the sum of the first HandCardCount entries.
	//
	// A fixed array for the reason HandCards is one: Event has to stay comparable.
	HandAmounts [maxHandTerms]int

	// HandCardBase is what each landing was worth **before any worn ring touched it** — the card's
	// own damage at the wielder's DMG, with an echo's fraction already taken off.
	//
	// **It is here so the sum can be written the way it is worked out** *(owner's call,
	// 2026-09-02)*: `10 + 10 + (10 x 2) x 2.5 = 100` rather than `10 + 10 + 20 x 2.5 = 100`. The
	// ring's figure is beside the term it priced everywhere else — on the card in the hand dialog,
	// on the term line in the ledger — and the sum was the one place it was silently folded in.
	//
	// **A screen may not divide HandAmounts by HandRingScale to get it back.** Every ring rounds
	// and CardDamage floors at 1, so the quotient is wrong exactly where the arithmetic is
	// interesting. That is the reason this is a field rather than a reading of two others.
	HandCardBase [maxHandTerms]int

	// EchoTerms is how many of those terms are echoes rather than cards — the tail of the list.
	// Zero on almost every blow. It is here so a screen can say *why* one card paid three terms
	// without re-deriving the ring that did it.
	EchoTerms int

	// HandRingScale[i][seat] is what the ring on that worn seat multiplied term i by, as a percent,
	// and 0 for a seat that did not touch it.
	//
	// **Every ring's figure moved off the card and into the sum on 2026-08-26** *(owner's call)*.
	// Nothing a ring does reaches a card's printed damage any more: the face says what the card does,
	// because a growing ring steps between the cards of one blow and the same card is worth different
	// things in different queue positions. So the sum is where the rings are accounted for — each one
	// says its own figure beside the term it priced, and its card bounces on that beat. See
	// combat.CardScaleBySeat, which is the only place these are worked out.
	//
	// **Per seat, so the screen knows which ring to bounce.** A product would say what the term came
	// to and leave five fingers unaccounted for.
	HandRingScale [maxHandTerms][MaxWornRings]int

	// HandLanding[i][seat] reports whether the ring on that seat is why term i exists at all: an
	// extra landing bought by `repeat-card` or `echo-attack`. False on a card's own first landing,
	// which no ring had to seat.
	//
	// **It is separate from HandRingScale because those rings contribute no multiplier.** An echo
	// ring buys a *term*, not a figure, so it has nothing to say beside the number — and without
	// this it would be the one thing in the sum with no card accounting for it while the player
	// watches three terms it alone is responsible for. See combat.LandingSeats.
	HandLanding [maxHandTerms][MaxWornRings]bool

	// HandGrown[i][seat] is what the ring on that worn seat had accumulated **after** term i was
	// counted. The ring row reads it to step each badge on the beat the term lands, so the player
	// watches the number that is about to price the next card go up.
	//
	// **Indexed by worn seat**, which is stable for the length of a blow: the row can be reordered
	// between rounds and not inside one. A screen that wants a ring's identity has the row itself.
	//
	// It is the widest thing on an Event by some way — a hand of five, each landing five times, over
	// five fingers. That is affordable because a KindHand event happens once per turn, and the
	// alternative is a screen re-deriving which ring grew, which is the resolver-in-the-screen this
	// whole block of fields exists to prevent.
	HandGrown [maxHandTerms][MaxWornRings]int

	// HandBonus is flat damage a worn ring added to this blow **because of the rung it formed**,
	// and HandBonusSeats is which seats paid it.
	//
	// **It is a term of Base and it is the last one** *(owner's call, 2026-09-05)*: every other
	// entry in the bracket is a card, so this is added after the cards are counted and before the
	// multiplier is applied. The screen draws it as its own term at the end of the sum, in the
	// ground's own ink rather than a ring's pink, because it is the hand paying rather than a
	// number a ring moved.
	//
	// Zero when nothing worn names this rung, which is the usual case.
	HandBonus      int
	HandBonusSeats [MaxWornRings]bool

	// HeldBonus is flat damage a worn ring added to this blow **for the cards the turn kept back**,
	// and HeldBonusSeats is which seats paid it.
	//
	// **A second term of Base, beside HandBonus and on the same terms**: after the cards, before
	// the multiplier, drawn in the ground's own ink. The two are separate fields rather than one
	// because they are different sentences — one is what the hand formed, the other is what the
	// hand still holds — and a screen that merged them could not say which.
	HeldBonus      int
	HeldBonusSeats [MaxWornRings]bool

	// VitaeBonus is the duelist's Bounty as it joined this blow's Base, and VitaeBonusSeats is
	// which worn rings put it there.
	VitaeBonus      int
	VitaeBonusSeats [MaxWornRings]bool

	// HandScale is the percentage the worn rings moved this blow's Multiplier by — 100 when
	// nothing did — and HandScaleSeats is which rings paid. **Multiplier already has it applied**;
	// this is kept so a screen can say the hand was improved rather than only show a bigger figure.
	HandScale      int
	HandScaleSeats [MaxWornRings]bool
}

// Slot is one card's place in a round's resolution order: whose it is, where it sits
// in that side's queue, and what it is.
//
// **It holds a whole Card rather than a bare concept** since 2026-08-12. A slot is what both
// the engine and the screen walk, so anything that has to know a card's element while a round is
// being ordered — a hand matching on colour, a row drawing a border — reads it here.
type Slot struct {
	Side  Side
	Index int
	Card  Card
}

// ResolutionOrder is the sequence in which two queued sets resolve, and the single
// authority on that order. ResolveRound plays it and the combat screen's Resolution pane
// draws it; neither works the order out for itself, so the pane and the engine cannot
// drift apart.
//
// **A whole turn each, in category order.** Everything side A queued resolves — attacks first,
// then everything else — and only then does side B begin. Within a category the
// queued order is kept, which is where drag-to-reorder still bites and where sequence
// hands will match.
//
// Index is the action's position in its own side's queue, which is *not* its position
// here: reordering by category is the whole job. Consumers wanting "how far through the
// round are we" should count slots rather than read Index.
func ResolutionOrder(aCards, bCards []Card) []Slot {
	slots := make([]Slot, 0, len(aCards)+len(bCards))
	slots = appendTurn(slots, SideA, aCards)
	slots = appendTurn(slots, SideB, bCards)
	return slots
}

// appendTurn adds one side's whole turn, category by category.
func appendTurn(slots []Slot, side Side, cards []Card) []Slot {
	for _, cat := range Categories() {
		for i, c := range cards {
			if c.Category() == cat {
				slots = append(slots, Slot{Side: side, Index: i, Card: c})
			}
		}
	}
	return slots
}
