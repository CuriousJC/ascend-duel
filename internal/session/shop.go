package session

import "github.com/curiousjc/ascend-duel/internal/combat"

// The shop's rules: what a ring costs, what it sells back for, and the two ways the row changes.
//
// **The screen draws a shelf; this decides what a purchase is.** Buying and selling both move the
// purse and both move the worn row, and those are two things the run owns — so they are one method
// each here rather than a scene reaching into `Wear` with a `SpendVitae` beside it and getting the
// order wrong on the day it fails.
//
// **A price is a fact about a ring and lives in `rings.json`** — a concept ring covering four cards
// and a form ring covering twelve are not the same object. What a ring *sells* for is not a field:
// it is a quarter of the price, rounded up, and it is one rule of the shop rather than seventeen
// numbers to keep in step with seventeen others.
//
// **A base ring is 3 and the ladder is scaled off it** *(owner's call, 2026-08-21)*. A base ring is
// one of the four that give a colour its status; everything else is read against that, 2 through 7.
//
// **Nothing prices these numbers but judgement.** `tools/balance` plays postures against the roster
// and knows nothing about rings, so what a doubling of every slash card is worth in vitae is a guess
// that has never been measured. Said out loud here because the alternative is a table of figures
// that looks derived.

// sellDivisor is what a ring sells back for: a quarter of what it cost, rounded up *(owner's call,
// 2026-08-21)*.
//
// **Rounded up so the cheapest ring is still worth something.** At the prices the catalogue actually
// carries — 2 to 7 — a quarter rounded down would be nothing for most of them, which would make
// selling a way of throwing a ring away rather than a trade. Rounded up, a base ring of 3 pays 1
// back. The loss on the round trip is still the point: a swap is meant to cost, or the shelf is a
// free rerolling of your hand every visit.
const sellDivisor = 4

// RingPrice is what the shop charges for a ring, and whether the catalogue holds one at all.
func RingPrice(key string) (int, bool) {
	p, ok := ringPrices[key]
	return p, ok
}

// SellValue is what taking a ring off pays back: a quarter of its price, rounded up. Zero for a
// ring the catalogue does not hold, which is a ring nothing can be wearing.
func SellValue(key string) int {
	price, ok := ringPrices[key]
	if !ok {
		return 0
	}
	return (price + sellDivisor - 1) / sellDivisor
}

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

// Sell takes a worn ring off and pays a quarter of its price back, rounded up. It reports whether
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
