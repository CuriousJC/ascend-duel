package session

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// startingVitae is what a run opens with. It was a constant on the combat screen, reset on every
// visit, which is exactly what "run-level state living on a scene" looks like.
const startingVitae = 5

// Session is one run: everything the player is carrying up the tower.
//
// **It is snapshotted, not replayed** *(owner's call, 2026-08-25)*. Two runs from the same seed may
// end up holding different decks, because a deck edit is a *choice* rather than something derived
// from the seed — so *replaying* a run means a seed plus a choice log, and that is still true. See
// the `randomness` skill. **Resuming is a different question**: it wants the state the player is in
// rather than the path they took to it, which is what save.go writes down. The two do not conflict
// and the distinction is worth keeping: a snapshot is not a replay and cannot be used as one.
type Session struct {
	// deck is every card the player owns, in no particular order, including the ones currently
	// sitting in a pile on the combat screen. **The piles are copies of this**, dealt fresh at
	// the start of each fight; this is the list they are dealt from.
	//
	// Unexported so it cannot be appended to from a screen. Everything that changes it goes
	// through a method, which is what keeps an index handed out by `Deck()` meaningful for as
	// long as a caller holds it.
	deck []combat.Card

	// nextCardID is the counter behind every card's identity. **It only ever goes up**, so a
	// number is never handed out twice inside one run and an id belonging to a removed card is
	// never quietly reused by a card added later.
	//
	// Identity is the run's to give: the rules have no idea a run exists, and every card outside
	// one — an enemy's deck, a test literal — keeps the zero id and is none the worse for it. See
	// combat.Card.ID for what the number buys.
	nextCardID int

	// fight is how many rooms in the run has got: zero on the first fight, incremented on a win.
	//
	// **It lives here rather than on the combat screen because two scenes read it** — the screen
	// picks the opponent and scales it, and the post-battle screen seeds its offer from it. It
	// used to be `CombatScene.fightIndex`, which survived `Init` and was invisible to everything
	// else. The screen keeps a copy per visit for its draw paths; this is the authority.
	fight int

	// vitae is the purse. **Run-level**: awarded by the post-battle screen and by propagation, and
	// spent in the shop. It stopped being a constant on the combat screen on 2026-08-17 and gained
	// something to be spent on four days later.
	vitae int

	// worn is what the player is wearing, by record key, **in worn order** — which is a rule and not
	// a presentation detail: relics fire left to right and compound, so the order has to be one the
	// player can see. See relic.go.
	worn []string

	// climb is the run's fight order — who stands in each room, in the order they will be met.
	// Nil on a run built by New, which is a test's run; a real one comes from Start. See climb.go.
	climb *pyramid.Pyramid

	// phase is where in the loop the run is: the fight, the reward, the shop, the room choice.
	// See flow.go, which is the one place that moves it.
	phase Phase

	// lifeLeft is the life the fighter finished the last fight on. **Run-level because a screen
	// after the fight has to draw the duelist as they came out of it** — the reward screen puts the
	// player's card up beside their relics, and there is no combatant left to ask by then.
	lifeLeft int

	// hurt is how much life the run is down, carried from room to room and cleared only by a
	// stairway win. **A wound rather than a life total**, because the ceiling it sits under is
	// rebuilt every fight and moved by whatever is worn. See life.go.
	hurt int

	// dmgBonus and lifeBonus are what the run has drunk: the damage and the max life the potions
	// have added to the duelist's own record. **Two totals rather than a list of bottles** — a
	// potion is gone the moment it is drunk, so what a run carries is the sum. See potion.go, and
	// Equip, which is where they reach the fighter.
	dmgBonus  int
	lifeBonus int

	// bossWins is how many stairway protectors the run has beaten. It is the run's max-life
	// multiplier, kept as a count rather than as a product. See life.go.
	bossWins int

	// spoils is what the last win still owes the player, decided by WonFight and paid out by the
	// post-battle screen as it narrates each part. See spoils.go.
	spoils Spoils

	// grown is each growing relic's accumulator, keyed by record. **Keyed by record rather than by
	// position**, because it is the first relic state that will have to be serialized and a position
	// would mean nothing in a save file.
	grown map[string]int

	// stones is how many stones the run has put on each rung of the hand ladder, keyed by hand key.
	// **Keyed by hand rather than by stone record** — one stone per rung, so the record is a name
	// for the rung and the rung is what the ladder is actually read against. See stone.go.
	stones map[string]int

	// plays is how many times the run has formed each rung, keyed by hand key. **A tally, not an
	// upgrade** *(owner's call, 2026-09-05)*: `stones` changes what a rung pays and this changes
	// nothing at all. It is here rather than on the combat screen because a count belonging to one
	// fight would be reset by the next `Init`, and it is the run's whole climb that is interesting.
	plays map[string]int

	// held is the bucket: every parasite the run is carrying, by record key, in the order they
	// were acquired. **A list rather than a count per key** — two of the same are two things to
	// spend, and the board piece draws a card for each. See parasite.go.
	held []string

	// duplicated is what the last duplicate parasite minted, so the screen can seat the copy in
	// the hand it was spent from. **Deliberately not snapshotted**: it is a handover between two
	// calls a frame apart, not a fact about the run, and a resumed run has no hand to seat it in.
	// See Session.Duplicated.
	duplicated []combat.Card

	// pouch is the stones the run is carrying but has not spent, by record key, in the order they
	// were acquired. **A list rather than counts**, unlike `stones` — see stone.go, where the
	// distinction is written down.
	pouch []string

	// granted is the stones the last rock-shower parasite handed over, so the dialog can show what
	// the player just got. **Not snapshotted**, for the reason duplicated is not: it is a handover
	// between two calls a frame apart, and the stones themselves are already on their rungs in
	// `stones`, which is saved.
	granted []Stone

	// lastParasite is the record key of the parasite this run spent most recently, which is what a
	// chimera copies. **Saved**, unlike `granted` and `duplicated`, because the memory is the
	// run's rather than the fight's: a chimera carried out of one duel still copies what was spent
	// in the previous one. See luck.go.
	//
	// **A chimera never writes itself here** — `rememberParasite` records the resolved record — so
	// two of them in a row both fire the thing behind them.
	lastParasite string

	// ledger is the run's account of itself: every fight, round by round, in already-worded
	// lines. **Run-level because that is the whole feature** — it used to be this fight's events
	// on the combat screen, thrown away by the next Init. See ledger.go.
	ledger Ledger

	// tutorial is the teaching run, or nil for a run nobody is being taught. See tutorial.go for
	// why a step cursor belongs to the run rather than to the screen that happens to be up.
	tutorial *tutorial.Run

	// roundLimit is how many rounds a fight of this run gets before the clock kills the duelist.
	// **The run's number, not the rules'** — `combat.DefaultRoundLimit` is what a run opens at and
	// this is what it is actually on, so a relic or a brand that buys a sixth round has one field to
	// move rather than a constant it cannot reach. See clock.go, and Equip, which is where it
	// reaches a fighter.
	roundLimit int

	// relicSlots is how many relics this run may wear at once. **The run's number, not the rules'** —
	// `combat.DefaultRelicSlots` is what a run opens at and this is what it is actually on, for the
	// reason roundLimit is a field: a brand that buys a sixth finger has one place to write. See
	// relic.go, and Equip, which is where it reaches a fighter.
	relicSlots int
}

// New starts a run from a deck list — `startingDeck`, in practice, expanded to one entry per
// card. The slice is copied, so the caller's starting list cannot be edited by a worm.
//
// **It opens wearing StartingRelics**, which is empty as shipped — see relic.go, where the list and
// the reason live. A run buys its relics.
func New(deck []combat.Card) *Session {
	s := &Session{deck: make([]combat.Card, len(deck)), vitae: startingVitae, grown: map[string]int{},
		stones: map[string]int{}, plays: map[string]int{}, roundLimit: combat.DefaultRoundLimit,
		relicSlots: combat.DefaultRelicSlots}
	copy(s.deck, deck)

	// **Identity is stamped here and nowhere else on the way in.** `StartingDeck()` hands over a
	// list of descriptions — four copies of a fire Bash are four equal values — and a run is
	// where they stop being interchangeable.
	for i := range s.deck {
		s.deck[i].ID = s.mintCardID()
	}

	// **The fingers are counted before the relics go on**, which is the whole reason
	// StartingRelicSlots is a var rather than something set on the run afterwards: Wear checks the
	// cap, so a sixth relic named by a fixture is refused by a hand that has not been widened yet.
	if StartingRelicSlots > 0 {
		s.SetRelicSlots(StartingRelicSlots)
	}

	for _, key := range StartingRelics {
		s.Wear(key)
	}
	// **The bucket is filled the same way the fingers are**, and a key the catalogue has not got is
	// dropped rather than held — `Hold` is what refuses it. See StartingParasites, which is empty
	// as shipped.
	//
	// **It goes past the cap on purpose** *(2026-09-06)*. `Hold` refuses a third parasite because
	// `MaxHeld` is a rule about *acquiring* one, and this is a fixture planting a bucket rather than
	// a run buying one — the same exception `internal/scenario`'s check() already writes down for a
	// hand longer than the game's own. Four fixtures exist to walk six parasites through the dialog
	// and trimming them to two would leave four Notes describing cards that are no longer there.
	// The pane draws the first two seats and the count reads the honest number, so an over-full
	// bucket looks like what it is.
	for _, key := range StartingParasites {
		s.hold(key)
	}
	// **And the pouch the same way**, with a key the catalogue has not got dropped rather than
	// carried — `Carry` is what refuses it. See StartingStones, which is empty as shipped.
	for _, key := range StartingStones {
		s.Carry(key)
	}
	return s
}

// Deck is every card the player owns, as a copy.
//
// **A copy, for the reason `decks.EnemyCards` hands one back**: anything that sorted or
// shuffled the result would otherwise be reordering what every future fight is dealt, and the
// damage would outlive whatever did it.
func (s *Session) Deck() []combat.Card {
	out := make([]combat.Card, len(s.deck))
	copy(out, s.deck)
	return out
}

// Size is how many cards the run holds. The deck thins as worms remove cards, so this is not a
// constant and nothing should treat 48 as one.
func (s *Session) Size() int { return len(s.deck) }

// Card is one entry by index, and reports whether the index exists.
func (s *Session) Card(i int) (combat.Card, bool) {
	if i < 0 || i >= len(s.deck) {
		return combat.Card{}, false
	}
	return s.deck[i], true
}

// Remove takes a card out of the run for good.
//
// **Indices shift, and that is why a caller may not hold two of them across a call.** The
// alteration screen offers a hand, the player picks one card, and the offer is done — one
// action against one index. If that ever becomes several actions against one offer, the offer
// has to be re-resolved after each, or carry something stabler than a position.
func (s *Session) Remove(i int) bool {
	if i < 0 || i >= len(s.deck) {
		return false
	}
	s.deck = append(s.deck[:i], s.deck[i+1:]...)
	return true
}

// SetElement recolours a card. The concept is untouched: a worm varies a card the game already
// defines rather than inventing one, so what changes is which colour it counts as in a mix and
// which status it can apply.
func (s *Session) SetElement(i int, e combat.Element) bool {
	if i < 0 || i >= len(s.deck) {
		return false
	}
	s.deck[i].Element = e
	return true
}

// Vitae is what the run is carrying.
func (s *Session) Vitae() int { return s.vitae }

// AddVitae puts some in the purse. **Never negative** — spending is the shop's business and it
// will want its own method, so that the one place a purse can go down is the one place that has to
// check it can.
func (s *Session) AddVitae(n int) {
	if n <= 0 {
		return
	}
	s.vitae += n
}

// SpendVitae takes from the purse, and **reports whether it could**. A run cannot go into debt:
// a caller that does not check the result has bought something for free.
//
// **It is the one place a purse goes down**, which is why AddVitae refuses a negative rather than
// being the same method twice. `Buy` is its only caller.
func (s *Session) SpendVitae(n int) bool {
	if n <= 0 || n > s.vitae {
		return false
	}
	s.vitae -= n
	return true
}

// LifeLeft is the life the fighter walked out of the last fight with.
func (s *Session) LifeLeft() int { return s.lifeLeft }

// Fight is how far up the tower the run has got, zero-based.
func (s *Session) Fight() int { return s.fight }

// WonFight advances to the next room. **Losing does not call this**, which is what makes a defeat
// put the same opponent back up rather than skipping past it.
//
// **It is the `fight-won` moment**, so it is also where the win's payout is decided and where every
// growing relic takes its step. Both happen before the room counter moves, which is the order
// MECHANICS.md states: interest is on what the run walked out of the fight holding, not on what the
// win is about to pay it.
//
// **It decides the payout and pays none of it** *(2026-08-22)*. `lifeLeft` is what the fighter
// finished on — a tenth of it is part of the prize — and the three figures are frozen here and
// handed over by the post-battle screen a sentence at a time. See spoils.go.
// **It is also where the body is settled** *(owner's call, 2026-09-06)*. The wound the fight left
// is carried into the next room, unless the room just won was the floor's stairway — a boss win
// heals to full and raises the ceiling by a third, compounding. See life.go, and note that both
// happen *before* the counter moves, on the same terms the payout does: they belong to the fight
// that was won, not to the one about to be met.
func (s *Session) WonFight(lifeLeft, maxLife int) {
	s.lifeLeft = lifeLeft
	s.spoils = s.spoilsFor(lifeLeft)
	s.growRelics()

	if hurt := maxLife - lifeLeft; hurt > 0 {
		s.hurt = hurt
	} else {
		s.hurt = 0
	}
	if pyramid.RoomOf(s.fight) == pyramid.RoomStairway {
		s.bossWins++
		s.hurt = 0
	}

	s.fight++
}

// Add puts a card into the run. Nothing offers this yet — REMOVE and MODIFY are the two worms
// that exist — but the third one named in the design is "add", and it is one line.
//
// **The card is given a fresh identity, whatever it arrived carrying.** A caller handing over a
// copy of a card the run already owns would otherwise put two cards with one id into the deck, and
// everything that looks a card up by id would find whichever came first.
func (s *Session) Add(c combat.Card) {
	c.ID = s.mintCardID()
	s.deck = append(s.deck, c)
}

// mintCardID hands out the next identity.
func (s *Session) mintCardID() int {
	s.nextCardID++
	return s.nextCardID
}

// CardByID is the card the run owns under an identity — **what a card looked like before any relic
// touched it**, which is the question a drawn card cannot answer for itself.
//
// It reports false for an id the run has not got, which covers the two honest cases: a card with
// no identity at all (an enemy's, a test's) and a card whose original has since been eaten by a
// worm. A caller that cannot find the original should draw the card it actually has.
//
// **A linear walk, deliberately.** The deck is fifty-odd cards and this is asked while a panel is
// open, so an index would be a second structure to keep in step with `Remove` for no measurable
// gain.
func (s *Session) CardByID(id int) (combat.Card, bool) {
	if id == 0 {
		return combat.Card{}, false
	}
	for _, c := range s.deck {
		if c.ID == id {
			return c, true
		}
	}
	return combat.Card{}, false
}
