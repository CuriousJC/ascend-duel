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

	// KindRaised is a defend card putting shields up. Action is the card, Amount is how many it
	// raised, and Life carries the duelist's shield count afterwards, so a second Guard in the same
	// turn reads as five rather than as three twice.
	//
	KindRaised

	// KindBlocked is a shield eating one incoming hit outright. Action is the attack that was
	// stopped, Target is the duelist that spent the shield, Amount is how many shields are left
	// afterwards, and Element is **the shield's** element — which pip the row loses. Surged says the
	// shield matched the hit's own element and banked an action point for the defender's next turn.
	//
	// **It leaves no figure**, so a feed reading
	// `Amount` off the two would be reading a remaining blow in one case and a remaining shield in
	// the other. It is not a KindMissed either: a miss is the attacker's own failure and costs the
	// defender nothing, where this is something the defender paid for.
	KindBlocked

	// KindExpired is a duelist's standing shields lapsing unspent, at the start of their own next
	// turn. Amount is how many were lost.
	//
	// **It exists because a shield row has to empty when the shields do.** Every other change to
	// that count is announced — raised, and eaten one attack at a time — so an expiry that said
	// nothing would leave the readout showing a defense the engine had already taken away, for a
	// whole turn. This is the same argument that makes a chilled slot emit a beat.
	//
	// **Only shields raise it, and only when some were standing.** A percentage guard lapsing is
	// not drawn anywhere, so announcing it would be a beat with no picture; and a turn beginning
	// with nothing up is the ordinary case, which must not cost the feed a line.
	KindExpired

	// KindDamage is one hit landing. Slot is the card that threw it and, from a hand-forming
	// attacker, Hit is which term of the hand it was.
	KindDamage
	KindDefeated

	// KindHand says a hand formed, and carries the arithmetic of **every hit** the turn is about to
	// throw. It follows the turn's attack KindActions and comes before the first hit, so no figure
	// lands before the reason for it.
	KindHand

	// KindChilled is one action lost to a chill. One event per action, so a chill deep enough to
	// take several narrates as the several things it actually is.
	KindChilled

	// The three element events, added 2026-08-12 with the statuses.

	// KindStatus is one element status landing on a duelist. Element says which, Amount says how
	// much was added by this hit, Target is who is carrying it.
	//
	// It is a separate event rather than a field on KindDamage because a status is not the hit:
	// a chill that lands is felt a round later and against a completely different card, and a
	// Resolution feed that folded it into the damage line would announce it at the one moment it
	// does nothing.
	KindStatus

	// KindMissed is one hit that never happened because its owner was shocked. Action is the
	// attack that was lost, Side is whose it was, and Slot and Hit say which hit — a shock rolls
	// once per hit, so one card of a turn can miss while the rest land.
	//
	// Nothing of the defender's stopped it, and a log
	// saying a blow was "stopped cold" by a defense that was never raised would send the player
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
	// relic reading the purse must see what an earlier turn of the round paid — but that is a copy
	// for the length of one fight. What the run is actually paid is the difference between the
	// purse the duel was handed and the one it hands back; see screens.payHeldVitae. **Do not sum
	// these events to move a purse**: that was the old way and it made two counters over one
	// figure.
	KindVitae

	// KindTimeUp is the round limit running out on a duelist who is still standing. Target and
	// Side are both that duelist, Amount is the life the clock took, and Life is what is left of
	// them, which is nothing.
	//
	// **It is emitted before the KindDefeated that follows it**, so the feed reads "out of time"
	// and then the fall, in that order — a death with no cause in front of it would read as the
	// engine having lost count.
	//
	// **Its own kind rather than a KindDamage.** A blow has an attacker and a card behind it, and
	// this has neither: nothing was swung, nobody landed it, and a damage event with an empty
	// Action would send the feed looking for a card that was never played. It is the same argument
	// KindHealed makes for not being a negative KindDamage.
	KindTimeUp

	// KindGrantedDMG is a golden card's gamble coming up damage. Action is the card that rolled it,
	// Amount is the DMG granted, and Side and Target are both the duelist who played it.
	//
	// **The rules have already moved the fighting duelist by the time this is emitted**, so the
	// grant is worth something for the rest of the fight. What it cannot do is move the *run*, which
	// is what owns a permanent bonus — so this is the announcement the layer above reads and acts
	// on. See screens.settleGrants, and KindVitae, which is the same division of labor.
	//
	// **Its own kind rather than a flag on one grant event.** The two grants travel to different
	// places on screen — a DMG figure to the stat, a life figure to the bar — and read as different
	// sentences in the feed, which is the argument KindHealed already makes for not being a negative
	// KindDamage.
	KindGrantedDMG

	// KindGrantedLife is a golden card's gamble coming up life. Amount is the life granted, and it
	// is added to the duelist's maximum *and* to what they are currently standing on — a bonus that
	// raised only the ceiling would read as nothing happening. Life is what they are on afterwards.
	KindGrantedLife

	// KindDrained is a worn relic turning part of a blow into life for the duelist who threw it.
	// Relic names which one and Amount is the life actually restored, **after the cap** — so a
	// drain on a duelist already at full life emits nothing at all rather than a beat worth zero,
	// which is the rule the heal rider is already under.
	//
	// **Its own kind rather than a KindHealed with a relic on it.** The two come off different
	// things and therefore fly from different places: a rider's heal leaves the card that carried
	// it, and a drain leaves the ring in the relic row. The choreography table is one entry per
	// kind, so a shared kind would be one anchor for two sources and the relic would fire without
	// saying which relic fired.
	KindDrained

	// KindRegenerated is a worn relic putting life on its wearer at the top of their own turn.
	// Relic names which one and Amount is the life actually restored, after the cap — so a duelist
	// at full life emits nothing at all, like every other heal.
	//
	// **Its own kind rather than a KindDrained, and the difference is where the life came from.**
	// A drain takes its share out of a blow, so the figure flies out of the body it was taken
	// from; a regeneration is made by the relic out of nothing, so it flies out of the ring. That
	// is the same distinction KindStatus draws by taking anchorRelic, and the choreography table is
	// one entry per kind.
	KindRegenerated

	// KindFizzled is one hit of the target's own element landing nothing at all: no damage, no
	// drain, no status, and no relic stepping. Action is the card, Element its element, and Slot
	// and Hit say which hit — like a miss, one card of a turn can fizzle while the rest land.
	//
	// **Its own kind rather than a KindMissed.** A miss is the attacker's shock and a matter of
	// luck; a fizzle is the target's nature and a matter of the player's choice, and a feed saying
	// "shocked" over an ice card thrown at an ice creature would send the player looking for a
	// status that is not there.
	KindFizzled

	KindRoundEnd
)

// maxHandTerms is the width of a hand event's arrays: **every hit a legal turn can throw** — each
// of its cards, each landing as many times as an echo or a repeat relic allows. A repeat matches on
// form, so five crush cards under Aftershock is ten hits. An over-long turn drops terms from the
// *arithmetic* rather than from the fight.
const maxHandTerms = baseMaxActions * MaxEchoLandings

// Event is one entry in the replayable log for a single round.
type Event struct {
	Kind   EventKind
	Side   Side      // who acted
	Action ConceptID // set on KindAction, on KindRaised for the card that raised the shields, on KindBlocked for the attack a shield ate, on KindChilled for the action lost, on KindMissed for the attack that never landed, on KindDamage for the card that landed, and on KindHand for the card the hand led with
	Amount int       // damage dealt, shields raised or left standing, status applied, or on KindHand what its hits add up to
	Target Side      // who took the damage
	Life   int       // target's life after the event
	Round  int

	// Slot is which of the acting side's turn a card event is about, as an index into the turn as
	// it was played — the same sequence HandCards indexes and the same one this side's KindActions
	// arrive in.
	//
	// **It is set on every hit event — KindDamage, KindMissed, KindBlocked, and the KindDrained and
	// KindStatus a hit produced — on KindRaised, and on the rider events.** A block names its attack
	// by ConceptID, which is a *kind* of card rather than one of them: a creature queuing two Nips
	// and having one of them eaten gives a screen reading `Action` no way to say which, and shields
	// pick the heaviest blow, so the card a shield ate may be the third of five and the screen has
	// to shatter that one. A raise carries it for the same reason one layer over: the defenses fire
	// as a bundle with no card lit, so the pips have nothing but this to leave from.
	//
	// **The zero value is a real slot**, like Status's and Relic's, so it is read only on the kind
	// that sets it.
	Slot int

	// Hit is which term of the turn's KindHand a hit event belongs to: an index into HandCards and
	// every array beside it. **Set on the hit events of a hand-forming attacker** — KindDamage,
	// KindMissed, KindBlocked, and the KindDrained and KindStatus that hit produced — and read on no
	// others. A solo attacker has no hand event to point into, so its hits leave it zero, which is a
	// real term, like Slot's zero is a real slot.
	//
	// **It is what lets a screen pair a hit with its arithmetic** when one card throws several hits:
	// Slot names the card and cannot say which of its landings this was.
	Hit int

	// Rider is which rider on the card is responsible for this event, and RiderNone - the zero
	// value - is every event no rider caused.
	//
	// **It is here for Event.Relic's reason: the thing that caused this is something the player can
	// see, and nothing else on the event can name it** *(2026-09-10)*. Two riders emit KindVitae -
	// a played RiderSilver coming up heads, and a held RiderVitaeInHand paying for being kept back
	// - and until this field existed the two arrived indistinguishable. The fight log printed "kept
	// back for 8 vitae" over a card that had just been played, which is a sentence that is wrong
	// about the one thing the player did.
	//
	// **It also says where a signal comes from.** A rider on a played card fires from its seat on
	// the table and a held one fires from the hand, and RiderVitaeInHand is the only one of the two
	// that can be held - so the screen reads the rider rather than searching the turn for a card
	// that might match. See screens.cardSignal.
	//
	// **Set on every event a rider produces**, which is KindHealed, KindRaised from a shielding
	// attack, KindVitae from either metal, and both grants. It is deliberately not set on the
	// events a *card* produces - a Brace's KindRaised carries RiderNone, because the card raising
	// shields is the card doing its own job.
	Rider RiderKind

	// Surged is set on a KindBlocked whose shield was the hit's own element, and says that block
	// banked one action point for the defender's next turn. See Duelist.Surge.
	Surged bool

	// Element is the card's element on KindAction, KindMissed and KindStatus. Basic everywhere
	// else, which is also the zero value — an event with nothing to say about color says `basic`,
	// exactly as a plain card does.
	Element Element

	// Status is which status is meant, on KindStatus and KindBurned.
	//
	// **It replaced reading Element for it** *(2026-08-17)*, because a status is no longer the same
	// object as a color: two relics can put two different statuses on the same fire card, and an
	// event naming the color could not say which had landed. Element still carries the card's own
	// color on a KindStatus, which is what the feed's swatch and its sentence are drawn from.
	//
	// **The zero value is a real status**, the first one registered — the hazard Action carries for
	// concepts. It is set on the two kinds that mean it and read on no others.
	Status StatusID

	// Relic is the worn relic behind the event, on KindStatus, KindDrained and KindRegenerated.
	//
	// **It is here because a status has a cause the player can see** *(2026-08-18)*. The screen
	// flies the word out of the relic that caused it, and there is no other honest way for it to
	// know which relic that was: reading it off the card's element would be a second rule about
	// something the grammar already decides, and it would be wrong the first time a form relic or
	// a concept relic applied a status - both of which RegisterRelic accepts today.
	//
	// **Which relic, not which slot.** A RelicID says something in a trace and in a test; a worn
	// index says nothing outside one duelist's array. The screen finds its position by walking the
	// worn list, which is at most five entries and is the same order the relic row is drawn in.
	//
	// **The zero value is a real relic**, exactly as Status's is a real status, so it is set on the
	// kinds that mean it and read on no others. NoRelic is the absence, for a caller that wants
	// to say so explicitly.
	Relic RelicID

	// Hand is set on KindHand and names what the attack phase formed. The screen looks it up
	// with HandByID rather than being told its name here, so a hand renamed is renamed once.
	//
	// **It always names a hand**, because `blowFor` falls back to the catalog's No Hand: a turn
	// the engine could not name is the one failure this model can have. `HandNone` is the zero
	// value and reaches a screen only on an event that is not a KindHand.
	Hand HandID

	// Multiplier is the turn's damage multiplier in percent — the hand's, so 150 is the 1.5x a
	// pair earns. It is on the event because the screen has no business re-deriving a number the
	// resolver already worked out.
	Multiplier int

	// HandCards and HandCardCount are set on KindHand alongside Hand: **the hits this turn
	// throws**, one term per hit, each naming the card that throws it as an index into the turn *as
	// it was played*. Every card played is here, defenses included. See RungCards
	// for the cards that made the hand.
	//
	// **They are here so a screen never has to work out which cards earned a hand.** The matcher
	// already knows, and re-deriving it from the hand's pattern would be a second matcher. It would
	// also be wrong: a counted hand is not contiguous, so Two Pair can be two cards, a card that
	// earned nothing, and two more.
	//
	// **A fixed array rather than a slice, because Event has to stay comparable** —
	// TestHandsDoNotBreakDeterminism compares two logs entry by entry with ==. It is sized to
	// maxHandTerms, and a turn throwing more hits than that has the extra ones dropped from the
	// arithmetic rather than from the fight.
	//
	// **A term is a landing, not a card.** An echoed card seats the same index two or three times
	// with a smaller amount each time, and each of those is a hit of its own.
	//
	// The indices count the actions that actually resolved, chilled ones already removed, which is
	// the same sequence as this side's KindAction events. The screen seats the whole turn at DUEL!
	// rather than a card at a time, so the cards are there to point at.
	HandCards     [maxHandTerms]int
	HandCardCount int

	// RungCards and RungCardCount are **which cards actually made the hand** — `Blow.Rung`, in turn
	// order.
	//
	// **Two sets, because the hits and the rung are different questions.** Every attack the turn
	// played throws a hit whether or not it agreed with anything, and a shield that made the rung
	// throws none. The screen raises these on the hand's announcement, raising being the whole of
	// what says which cards made the rung; the hits walk HandCards.
	//
	// **A landing is not a term here.** This is cards, so it holds each of the rung's cards once.
	//
	// A fixed array for HandCards' reason, and sized the same way.
	RungCards     [maxHandTerms]int
	RungCardCount int

	// HandAmounts is what each hit's card deals, in the same order and to the same count — the
	// card's own term, before the flat bonuses and the multipliers.
	//
	// **It is here so the screen can show the arithmetic rather than assert it.** Re-deriving one
	// on the screen would mean the screen owning `CardDamage`, the Strength scaling and every relic
	// that touches a card's damage — a second resolver.
	//
	// A fixed array for the reason HandCards is one: Event has to stay comparable.
	HandAmounts [maxHandTerms]int

	// HandCardBase is what each landing was worth **before any worn relic touched it** — the card's
	// own damage at the wielder's DMG, with an echo's fraction already taken off.
	//
	// **It is here so a hit can be written the way it is worked out** *(owner's call)*:
	// `(10 x 2) x 2.5 = 50` rather than `20 x 2.5 = 50`, with the relic's figure beside the term it
	// priced.
	//
	// **A screen may not divide HandAmounts by HandRelicScale to get it back.** Every relic rounds
	// and CardDamage floors at 1, so the quotient is wrong exactly where the arithmetic is
	// interesting. That is the reason this is a field rather than a reading of two others.
	HandCardBase [maxHandTerms]int

	// HandCardPct[i] is the percentage term i applied to HandDMG — the card's own multiplier, with
	// an echo's fraction already taken off it. 300 is a 3x card; 200 is that card's second landing
	// under one echo.
	//
	// **It is here so a hit can be written the way the game works it out** *(owner's call)*:
	// `(12 x 3) x 1 = 36` rather than `36`, which makes the DMG the whole hand swings at — the one
	// number a rung relic raises — visible in every hit. **Neither this nor HandDMG is what the hit
	// deals**: HandAmounts is the card's term and HitAmounts the hit.
	//
	// **A screen may not divide HandCardBase by HandDMG to get it back**, for HandCardBase's own
	// reason: an echo fraction is taken off the damage rather than off the percentage, so the two
	// roundings need not agree. Where they disagree the working is written the flat way — see
	// ui.TermSplit, which is the one place that comparison is made.
	HandCardPct [maxHandTerms]int

	// HandDMG is the DMG every hit of this turn was swung at, and HandDMGBare is what that figure
	// would have been with no rung relic worn.
	//
	// **The difference between them is what a rung relic actually put into every hit**, which is not
	// HandBonus: the riders scale DMG after the raise is folded in, so a +2 under a held rider is
	// worth more than 2. A screen showing the duelist's figure climbing shows this difference.
	//
	// Both are the whole hand's, not a hit's — one duelist swings one hand at one DMG.
	HandDMG     int
	HandDMGBare int

	// HitAmounts[i] is what hit i comes to before the attacker's weight and the target's
	// vulnerability: the card's term, plus every flat bonus, times the hand's multiplier, times
	// HandScale — each step rounded toward zero, on this hit alone.
	//
	// **It is the figure the hit's arithmetic ends on**, and `Amount` on this event is the sum of
	// them — what the hand was worth, not what landed. What landed is each hit's KindDamage, and a
	// hit that missed, was blocked, or came after a death has a figure here and nothing there.
	HitAmounts [maxHandTerms]int

	// HandRelicScale[i][seat] is what the relic on that worn seat multiplied hit i's card term by, as
	// a percent, and 0 for a seat that did not touch it.
	//
	// **A relic's figure belongs to the hit, never to the card face** *(owner's call)*: the face says
	// what the card does, because a growing relic steps between the hits of one turn and the same card
	// is worth different things in different queue positions. So each hit's line is where the relics
	// are accounted for — each says its own figure beside the term it priced, and its card bounces on
	// that beat. See
	// combat.CardScaleBySeat, which is the only place these are worked out.
	//
	// **Per seat, so the screen knows which relic to bounce.** A product would say what the term came
	// to and leave five fingers unaccounted for.
	HandRelicScale [maxHandTerms][]int

	// HandLanding[i][seat] reports whether the relic on that seat is why term i exists at all: an
	// extra landing bought by `repeat-card` or `echo-attack`. False on a card's own first landing,
	// which no relic had to seat.
	//
	// **It is separate from HandRelicScale because those relics contribute no multiplier.** An echo
	// relic buys a *term*, not a figure, so it has nothing to say beside the number — and without
	// this it would be the one thing in the hand dialog with no card accounting for it while the
	// player watches three hits it alone is responsible for. See combat.LandingSeats.
	HandLanding [maxHandTerms][]bool

	// HandGrown[i][seat] is what the relic on that worn seat had accumulated **after** hit i was
	// thrown — unchanged by a hit that missed, was blocked or was never thrown, since only a hit
	// that connects grows a relic. The relic row reads it to step each badge as the hits land.
	//
	// **Indexed by worn seat**, which is stable for the length of a turn: the row can be reordered
	// between rounds and not inside one. A screen that wants a relic's identity has the row itself.
	//
	// It is the widest thing on an Event by some way — a hand of five, each landing five times, over
	// five fingers. That is affordable because a KindHand event happens once per turn, and the
	// alternative is a screen re-deriving which relic grew.
	HandGrown [maxHandTerms][]int

	// HandBonus is DMG a worn relic added to the duelist **because of the rung this blow formed**,
	// and HandBonusSeats is which seats paid it.
	//
	// **It is base damage, not a term** *(owner's call)*: a duelist on 14 wearing a Twinned Ring
	// swings a Pair at 16, so a 1x card in it gains 2 and a 0.5x card gains 1. That makes the relic
	// worth more to a bigger hand, and it is the same fold the damage riders take — see
	// combat.blowDMG.
	//
	// **So it is on the event to be *said*, never to be added.** Every figure in `HandAmounts`
	// already has it inside; a screen that also wrote it as a term would print a hit that comes to
	// more than its own total. What a screen does with it is name the relic that raised the figure.
	//
	// Zero when nothing worn names this rung, which is the usual case.
	HandBonus      int
	HandBonusSeats []bool

	// HeldBonus is flat damage a worn relic adds to **every hit** for the cards the turn kept back,
	// and HeldBonusSeats is which seats paid it. It joins each hit after the card's term and before
	// the multiplier.
	//
	// **HeldBonusCards is how many held cards paid it**, across every seat that did. The run's
	// account writes the term as `Jar of Ice (4 cards)  20`, and a count is the one thing a
	// player reading the working back cannot re-derive: the hand it was counted over is three
	// turns gone.
	//
	// **HeldBonusEach is the same tally one card at a time** *(owner's call)*, so a hit on the combat
	// screen writes six `+5` terms where six earth cards were kept back rather than one `+30`, each
	// flying out of the card it was paid for. Every entry names the seat that paid.
	HeldBonus      int
	HeldBonusCards int
	HeldBonusSeats []bool
	HeldBonusEach  []HeldPay

	// VitaeBonus is the duelist's Bounty as it joins every hit, beside HeldBonus, and
	// VitaeBonusSeats is which worn relics put it there.
	VitaeBonus      int
	VitaeBonusSeats []bool

	// HandScale is the percentage the worn relics multiply every hit by after the hand's own
	// Multiplier — 100 when nothing did — and HandScaleSeats is which relics paid. **Multiplier does
	// not include it**, so the ladder's own figure is what the banner shows.
	HandScale      int
	HandScaleSeats []bool
}

// Slot is one card's place in a round's resolution order: whose it is, where it sits
// in that side's queue, and what it is.
//
// **It holds a whole Card rather than a bare concept** since 2026-08-12. A slot is what both
// the engine and the screen walk, so anything that has to know a card's element while a round is
// being ordered — a hand matching on color, a row drawing a border — reads it here.
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

// GrownAt is what the relic on `seat` had accumulated after term `term`, and **zero for a seat this
// event does not describe**.
//
// **A total reading rather than an index** *(2026-09-17)*. The seat rows became slices when the
// duelist's relic row lost its fixed width, and they are built one per term — so they are all the
// same length in an event the resolver wrote, and need not be in one a test or a tool assembled by
// hand. A reader walking one row and indexing another is the shape that panics, and it panicked.
func (e Event) GrownAt(term, seat int) int {
	if term < 0 || term >= len(e.HandGrown) {
		return 0
	}
	row := e.HandGrown[term]
	if seat < 0 || seat >= len(row) {
		return 0
	}
	return row[seat]
}

// TermSplit is one hit's card term written the way the game worked it out: the DMG the hand was
// swung at, and the percentage this card applied to it. It reports false when the two do not come
// to the term's own figure, and a caller that gets false writes the flat figure instead.
//
// **It exists so that no screen divides one field of this event by another.** Every reading of a
// term's working goes through here — the hand dialog and the run's account are two drawings of one
// event, and a split each of them derived would be two drawings of two.
//
// **False is reachable and is not a bug.** An echo takes its fraction off the damage while
// HandCardPct takes the same fraction off the percentage, so the two roundings can land a point
// apart; a card whose damage hit CardDamage's floor of 1 is the other case. The working is written
// without the split there rather than printing a product that is not the term.
func (e Event) TermSplit(term int) (dmg, pct int, ok bool) {
	if term < 0 || term >= len(e.HandCardPct) {
		return 0, 0, false
	}
	dmg, pct = e.HandDMG, e.HandCardPct[term]
	if dmg <= 0 || pct <= 0 {
		return 0, 0, false
	}
	if scaleDamage(dmg, pct) != e.HandCardBase[term] {
		return 0, 0, false
	}
	return dmg, pct, true
}

// DMGRaise is what a rung relic put into the DMG this blow was swung at — the climb the duelist's
// own figure makes, which is what a screen animates. Zero when nothing raised it.
//
// **Not HandBonus**, which is the relic's raw figure before the riders scaled it. See HandDMG.
func (e Event) DMGRaise() int {
	if e.HandBonus == 0 {
		return 0
	}
	return e.HandDMG - e.HandDMGBare
}
