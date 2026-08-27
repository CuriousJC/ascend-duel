package session

import "github.com/curiousjc/ascend-duel/internal/combat"

// The shop's rules: what a ring costs, what it sells back for, and the two ways the row changes.
//
// **The screen draws a shelf; this decides what a purchase is.** Buying and selling both move the
// purse and both move the worn row, and those are two things the run owns — so they are one method
// each here rather than a scene reaching into `Wear` with a `SpendVitae` beside it and getting the
// order wrong on the day it fails.
//
// **A price is a fact about a ring's rarity** *(owner's call, 2026-08-22)*, not a number the ring
// writes down: `rings.json` names one of three tiers and `data.Rarity` turns it into both a price
// and a draw weight. A common ring is 3 — the base, the plainest thing the grammar can say — and the
// two tiers above it are 5 and 7. What a ring *sells* for comes off the same tier: 1, 2 or 3,
// written down rather than derived, because a quarter rounded up paid an uncommon and a rare alike.
//
// **Nothing prices these numbers but judgement.** Nothing in the repo measures what a ring does to a
// duel, so what a doubling of every slash card is worth in vitae is a guess that has never been
// checked. Said out loud here because the alternative is a table of figures that looks derived.

// RingPrice is what the shop charges for a ring, and whether the catalogue holds one at all.
func RingPrice(key string) (int, bool) {
	p, ok := ringPrices[key]
	return p, ok
}

// RingWeight is how many tickets a ring holds in the shelf draw, from its rarity. Zero for a ring
// the catalogue does not hold, which is a ring nothing can offer.
//
// **The shelf asks rather than reading a record**, the same line RingPrice draws: what a tier is
// worth in tickets is the shop's arithmetic, not a screen's.
func RingWeight(key string) int { return ringWeights[key] }

// SellValue is what taking a ring off pays back: the tier's own figure — 1, 2 or 3. Zero for a ring
// the catalogue does not hold, which is a ring nothing can be wearing.
func SellValue(key string) int { return ringSells[key] }

// CanBuy reports whether this run could buy that ring right now: the catalogue holds it, it is not
// already on, there is a finger free, and the purse covers it.
//
// **It exists so the shelf can dim a card rather than swallow a click.** Buy checks the same four
// things itself — this is the question, not the guard.
func (s *Session) CanBuy(key string) bool {
	price, ok := ringPrices[key]
	if !ok || price > s.vitae {
		return false
	}
	return s.canWear(key)
}

// canWear is the row's half of the question: the catalogue holds it, it is not already worn, and
// the fifth finger is not spoken for. Shared with Wear so the shelf and the purchase cannot come to
// different conclusions.
func (s *Session) canWear(key string) bool {
	if _, ok := registeredRings[key]; !ok {
		return false
	}
	if len(s.worn) >= combat.MaxWornRings {
		return false
	}
	for _, k := range s.worn {
		if k == key {
			return false
		}
	}
	return true
}

// Buy pays for a ring and puts it on, and **reports whether it could**. A caller that does not
// check has handed out a free ring.
//
// **The purse moves first and the ring goes on second**, and the order matters: `SpendVitae` is the
// one place a purse goes down and the one place that refuses to go into debt, so asking it before
// wearing anything means a refusal leaves the run exactly as it was.
//
// **A sixth ring is refused here rather than swapped for.** Selling is what frees a finger — see
// Sell — so the trade is two decisions with a price between them rather than one click that quietly
// throws a ring away.
func (s *Session) Buy(key string) bool {
	if !s.canWear(key) {
		return false
	}
	price, ok := ringPrices[key]
	if !ok || !s.SpendVitae(price) {
		return false
	}
	return s.Wear(key)
}

// Sell takes a worn ring off and pays its tier's sell-back figure. It reports whether
// the run was wearing it.
//
// **It is the only way a ring comes off** *(owner's call, 2026-08-21)*, which is what makes the
// fifth finger a decision: swapping means selling something at a loss first, so a shelf full of
// tempting rings cannot be worn one after another for free.
//
// **A growing ring's accumulator is reset, not kept** *(owner's call, 2026-08-21)*. `grown` is keyed
// by record precisely so a ring taken off and put back on is the same ring — and the decision is
// that it is not the same *number*: the growth is what wearing it through fights paid for, so
// selling it forfeits that and re-buying starts again at the record's own amount. It is what stops
// a Heart Ring being parked in the shop between fights.
func (s *Session) Sell(key string) bool {
	at := -1
	for i, k := range s.worn {
		if k == key {
			at = i
			break
		}
	}
	if at < 0 {
		return false
	}

	s.worn = append(s.worn[:at], s.worn[at+1:]...)
	delete(s.grown, key)
	s.AddVitae(SellValue(key))
	return true
}

// The two sealed goods: **a bag of rocks and a can of worms.**
//
// A ring on the shelf is a thing you can read before you buy it. These are not: what is inside is
// four of something, drawn when the bag is opened, and the choice is which one to keep. That is
// the whole design — the price buys the *choice*, and the three that are not chosen are gone.
//
// **Both cost the same and both offer four** *(owner's call, 2026-08-27)*. They are the same
// bargain pointed at the two catalogues the run can alter — the ladder and the deck — and pricing
// or sizing them differently would be saying one of those is worth more without anything to say it
// with.

// bagPrice and canPrice are what the two sealed goods cost.
//
// **Five vitae, which is a ring's middle tier** *(owner's call, 2026-08-27)*. A win pays three to
// five from the room plus a tenth of the life left, so a good is about a fight's takings — bought
// instead of a ring rather than alongside one.
const (
	bagPrice = 5
	canPrice = 5
)

// bagSize and canSize are how many are drawn from inside.
//
// **Four, against the reward screen's two worms.** The free offer after a fight is a choice
// between two; this is what five vitae buys over that, and it is why the can is worth opening at
// all when a worm arrives free every fight.
const (
	bagSize = 4
	canSize = 4
)

// BagPrice, CanPrice, BagSize and CanSize are the figures a screen writes on the cards. **Asked
// rather than repeated**: a price printed on a face and charged by a method are two numbers that
// can disagree, and the shop is where that has already been avoided once for rings.
func BagPrice() int { return bagPrice }
func CanPrice() int { return canPrice }
func BagSize() int  { return bagSize }
func CanSize() int  { return canSize }

// CanAffordBag and CanAffordCan report whether the purse covers one. **The question, not the
// guard** — `BuyBag` and `BuyCan` check the purse themselves, exactly as `CanBuy` sits beside
// `Buy`. They exist so a shelf can dim a card rather than swallow a click.
func (s *Session) CanAffordBag() bool { return s.vitae >= bagPrice }
func (s *Session) CanAffordCan() bool { return s.vitae >= canPrice }

// BuyBag and BuyCan pay for a sealed good and report whether they could.
//
// **The purse moves and nothing else does.** What is inside is drawn by the screen from its own
// seeded stream and applied by `UseStone` or `Apply` when the player picks one, so a purchase
// interrupted by a quit costs the vitae and hands back nothing — which is the same deal a shop
// makes anywhere. Rolling the contents here would put the offer in the run's state and mean
// snapshotting a bag nobody has opened yet.
func (s *Session) BuyBag() bool { return s.SpendVitae(bagPrice) }
func (s *Session) BuyCan() bool { return s.SpendVitae(canPrice) }
