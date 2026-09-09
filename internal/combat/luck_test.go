package combat

import (
	"math/rand"
	"testing"
)

// grantedBy is what one side was granted across a whole round, in DMG and in life.
func grantedBy(events []Event, side Side) (dmg, life int) {
	for _, e := range events {
		if e.Side != side {
			continue
		}
		switch e.Kind {
		case KindGrantedDMG:
			dmg += e.Amount
		case KindGrantedLife:
			life += e.Amount
		}
	}
	return dmg, life
}

// gamble resolves one round with a luck source, which is the only way to reach the metals.
func gamble(a, b Duelist, aCards []Card, seed int64) ([]Event, Duelist) {
	events, after, _ := resolveRound(a, b, aCards, nil, nil, nil, 1, handTable,
		Sources{Luck: rand.New(rand.NewSource(seed))})
	return events, after
}

// **A golden card rolls every time it is played, and the two rewards are exclusive.** One d5 with
// three outcomes, not two rolls — see rollGolden, which is where the argument against a 4% double
// payout is written down.
func TestAGoldenCardGrantsAtMostOneThingPerPlay(t *testing.T) {
	card := Plain(Strike).SetRider(Rider{Kind: RiderGolden, Amount: LuckOutcomes})

	dmgs, lives, duds := 0, 0, 0
	for seed := int64(0); seed < 60; seed++ {
		events, _ := gamble(duelist(10, 3, 100), duelist(0, 0, 100000), []Card{card}, seed)
		dmg, life := grantedBy(events, SideA)
		if dmg > 0 && life > 0 {
			t.Fatalf("seed %d granted %d DMG and %d life from one play", seed, dmg, life)
		}
		switch {
		case dmg > 0:
			dmgs++
			if dmg != LuckDMG {
				t.Errorf("seed %d granted %d DMG, and a win is worth %d", seed, dmg, LuckDMG)
			}
		case life > 0:
			lives++
			if life != LuckLife {
				t.Errorf("seed %d granted %d life, and a win is worth %d", seed, life, LuckLife)
			}
		default:
			duds++
		}
	}

	// **All three faces have to be reachable**, which is the whole of what LuckOutcomes is for: a
	// die with no losing face is a purchase, and one with no winning face is a card that lies.
	if dmgs == 0 || lives == 0 || duds == 0 {
		t.Errorf("over 60 plays: %d DMG, %d life, %d nothing — one outcome never came up",
			dmgs, lives, duds)
	}
}

// **The grant reaches the fighting duelist, not only the log.** That is what makes the point of DMG
// worth something for the rest of the fight; the event is what makes it worth something for the
// rest of the run, and both halves are needed. See screens.settleGrants.
func TestAGrantMovesTheDuelistItWasRolledFor(t *testing.T) {
	card := Plain(Strike).SetRider(Rider{Kind: RiderGolden, Amount: LuckOutcomes})

	for seed := int64(0); seed < 60; seed++ {
		a := duelist(10, 3, 100)
		a.CurrentLife = 50
		events, after := gamble(a, duelist(0, 0, 100000), []Card{card}, seed)

		dmg, life := grantedBy(events, SideA)
		if got := after.DMG - a.DMG; got != dmg {
			t.Fatalf("seed %d announced %d DMG and moved the duelist by %d", seed, dmg, got)
		}
		// **The ceiling and the floor together**, or a maximum that rose while the player stood
		// where they were would read as nothing having happened.
		if got := after.MaxLife - a.MaxLife; got != life {
			t.Fatalf("seed %d announced %d life and moved the maximum by %d", seed, life, got)
		}
		if got := after.CurrentLife - 50; got != life {
			t.Fatalf("seed %d announced %d life and moved what they are standing on by %d",
				seed, life, got)
		}
	}
}

// **A silver card pays into the purse and announces a KindVitae.** It needs no grant event, because
// vitae already travels out of a resolved round as the difference between the purse the duel opened
// with and the one it closes with — see screens.payHeldVitae.
func TestASilverCardPaysThePurse(t *testing.T) {
	card := Plain(Strike).SetRider(Rider{Kind: RiderSilver, Amount: LuckOutcomes})

	paid, duds := 0, 0
	for seed := int64(0); seed < 60; seed++ {
		a := duelist(10, 3, 100)
		events, after := gamble(a, duelist(0, 0, 100000), []Card{card}, seed)

		if dmg, life := grantedBy(events, SideA); dmg != 0 || life != 0 {
			t.Fatalf("seed %d: silver granted %d DMG and %d life, and it grants neither", seed, dmg, life)
		}
		switch earned := after.Vitae - a.Vitae; earned {
		case SilverVitae:
			paid++
		case 0:
			duds++
		default:
			t.Fatalf("seed %d paid %d vitae, and a win is worth %d", seed, earned, SilverVitae)
		}
	}
	if paid == 0 || duds == 0 {
		t.Errorf("over 60 plays silver paid %d times and missed %d — it is not a gamble", paid, duds)
	}
}

// **A nil luck source rolls nothing**, which is what every test and every headless caller passes and
// what keeps `internal/combat` integer arithmetic when it is asked to be. It is the contract
// TestRoundIsDeterministic depends on, one field over.
func TestNoLuckSourceTakesNoGamble(t *testing.T) {
	for _, kind := range []RiderKind{RiderGolden, RiderSilver} {
		card := Plain(Strike).SetRider(Rider{Kind: kind, Amount: LuckOutcomes})
		a := duelist(10, 3, 100)
		events, after, _ := resolveRound(a, duelist(0, 0, 100000), []Card{card}, nil, nil, nil,
			1, handTable, Sources{})

		if dmg, life := grantedBy(events, SideA); dmg != 0 || life != 0 {
			t.Errorf("%s granted %d DMG and %d life off a nil source", kind, dmg, life)
		}
		if after.Vitae != a.Vitae {
			t.Errorf("%s paid %d vitae off a nil source", kind, after.Vitae-a.Vitae)
		}
	}
}

// **A card that is not a metal never gambles**, which is the shape every rider reader takes: ask the
// card, get nothing, carry on. Nothing branches on whether a card is ridden.
func TestAnOrdinaryCardHasNoOdds(t *testing.T) {
	c := Plain(Strike)
	if c.GoldenOdds() != 0 || c.SilverOdds() != 0 {
		t.Errorf("a plain card reported odds: gold %d, silver %d", c.GoldenOdds(), c.SilverOdds())
	}
}
