package combat

import (
	"math/rand"
	"testing"
)

// The hand-forming attack phase is a hit per landing. These pin the parts of that which are a
// property of *hits* rather than of any one relic: every attack throws one, each is its own
// arithmetic, and everything that happens to a hit — a roll, a shield, a status, a drain, a death —
// happens to that hit alone.

// cardTerms is what a hand's cards deal between them, before any flat term or multiplier.
func cardTerms(e Event) int {
	sum := 0
	for i := 0; i < e.HandCardCount; i++ {
		sum += e.HandAmounts[i]
	}
	return sum
}

// hitsWorth is what a hand's hits come to when each is its card's term plus `flat`, times the
// multiplier and the hand scale, **rounded one hit at a time** — the rule written out a second way.
func hitsWorth(e Event, flat int) int {
	sum := 0
	for i := 0; i < e.HandCardCount; i++ {
		hit := scaleDamage(e.HandAmounts[i]+flat, e.Multiplier)
		if e.HandScale != 0 && e.HandScale != 100 {
			hit = scaleDamage(hit, e.HandScale)
		}
		sum += hit
	}
	return sum
}

// hitEvents is every hit outcome one side threw, in log order.
func hitEvents(events []Event, by Side) []Event {
	var out []Event
	for _, e := range events {
		switch e.Kind {
		case KindDamage, KindMissed, KindFizzled:
			if e.Side == by {
				out = append(out, e)
			}
		case KindBlocked:
			if e.Side != by {
				out = append(out, e)
			}
		}
	}
	return out
}

// sequenceSource hands back its rolls in order and then repeats the last, so a test can say
// "the first hit misses and the rest land" without hunting for a seed.
type sequenceSource struct {
	rolls []int64
	at    *int
}

func (s sequenceSource) Int63() int64 {
	r := s.rolls[min(*s.at, len(s.rolls)-1)]
	*s.at++
	return r
}
func (s sequenceSource) Seed(int64) {}

func rolls(values ...int) *rand.Rand {
	at := 0
	src := sequenceSource{at: &at}
	for _, v := range values {
		src.rolls = append(src.rolls, int64(v)<<32)
	}
	return rand.New(src)
}

func TestEveryAttackCardLandsItsOwnHit(t *testing.T) {
	a, b := duelist(10, 8, 5000), duelist(10, 8, 5000)
	turn := PlainCards(Bash, Bash, Jab, Cut, Smash)

	events, _, after := resolve(a, b, turn, nil, 1)
	e := handEventOf(t, events, SideA)

	hits := hitEvents(events, SideA)
	if len(hits) != len(turn) {
		t.Fatalf("five attack cards threw %d hits, want five", len(hits))
	}
	landed := 0
	for n, h := range hits {
		if h.Kind != KindDamage {
			t.Errorf("hit %d was a %v, want damage", n, h.Kind)
		}
		if h.Hit != n {
			t.Errorf("hit %d names term %d; hits are thrown in term order", n, h.Hit)
		}
		if h.Slot != e.HandCards[n] {
			t.Errorf("hit %d names seat %d, want the seat its term names, %d", n, h.Slot, e.HandCards[n])
		}
		if h.Amount != e.HitAmounts[n] {
			t.Errorf("hit %d landed %d, want its own figure %d — nothing stands between them here",
				n, h.Amount, e.HitAmounts[n])
		}
		landed += h.Amount
	}

	// **The total is the hits', and the target lost exactly that.**
	if landed != e.Amount {
		t.Errorf("the hits landed %d between them against the hand's %d", landed, e.Amount)
	}
	if want := b.CurrentLife - landed; after.CurrentLife != want {
		t.Errorf("the target is on %d, want %d", after.CurrentLife, want)
	}
}

func TestAnEchoedCardThrowsAHitPerLanding(t *testing.T) {
	echo := relic(t, "hit-echo", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Lead: true},
		Then: []RelicEffect{{Do: DoEchoAttack, Amount: 3}},
	})

	a := duelist(10, 8, 5000).Wearing(WornRelic{Relic: echo})
	events, _, _ := resolve(a, duelist(10, 8, 5000), PlainCards(Bash, Bash), nil, 1)
	e := handEventOf(t, events, SideA)

	// Two cards, the first landing three times: four hits.
	hits := hitEvents(events, SideA)
	if len(hits) != 4 || e.HandCardCount != 4 {
		t.Fatalf("an echoed pair threw %d hits over %d terms, want four", len(hits), e.HandCardCount)
	}
	for n, want := range []int{0, 0, 0, 1} {
		if hits[n].Slot != want {
			t.Errorf("hit %d came off seat %d, want %d", n, hits[n].Slot, want)
		}
	}

	// **Every landing is multiplied by the hand**, the echoes included.
	for n := 0; n < e.HandCardCount; n++ {
		if want := scaleDamage(e.HandAmounts[n], e.Multiplier); e.HitAmounts[n] != want {
			t.Errorf("hit %d came to %d, want its term %d times the hand = %d",
				n, e.HitAmounts[n], e.HandAmounts[n], want)
		}
	}
}

func TestAFlatBonusJoinsEveryHit(t *testing.T) {
	smolder := relic(t, "hit-smolder", RelicRule{
		When: MomentBlowFormed,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoAddDamagePerHeld, Amount: 5}},
	})

	played := PlainCards(Bash, Jab, Cut)
	held := []Card{Of(Bash, Fire)}
	events, _, _ := ResolveRoundHolding(duelist(10, 8, 5000).Wearing(WornRelic{Relic: smolder}),
		duelist(10, 8, 5000), played, nil, held, nil, 1, Sources{})
	e := handEventOf(t, events, SideA)

	for n := 0; n < e.HandCardCount; n++ {
		if want := scaleDamage(e.HandAmounts[n]+5, e.Multiplier); e.HitAmounts[n] != want {
			t.Errorf("hit %d came to %d, want its term plus the 5 held, multiplied = %d",
				n, e.HitAmounts[n], want)
		}
	}
}

func TestAShockRollsOncePerHit(t *testing.T) {
	a := shockedDuelist(t, duelist(10, 8, 5000))
	b := duelist(10, 8, 5000)

	// The first hit misses, the rest land.
	events, _, _ := resolveWith(rolls(0, 99, 99), a, b, PlainCards(Bash, Jab, Cut), nil, 1)

	hits := hitEvents(events, SideA)
	if len(hits) != 3 {
		t.Fatalf("three attacks threw %d hits, want three", len(hits))
	}
	if hits[0].Kind != KindMissed || hits[0].Hit != 0 {
		t.Errorf("the first hit was a %v on term %d, want a miss on term 0", hits[0].Kind, hits[0].Hit)
	}
	for n := 1; n < 3; n++ {
		if hits[n].Kind != KindDamage {
			t.Errorf("hit %d was a %v; a miss takes its own hit and nothing else", n, hits[n].Kind)
		}
	}
}

func TestHitsStopAtADeath(t *testing.T) {
	a := duelist(10, 8, 5000)
	// Three different attacks form no hand, so every hit is its card's face: enough life to take
	// the first and not the second.
	turn := PlainCards(Jab, Cut, Smash)
	b := duelist(10, 8, a.CardDamage(turn[0])+1)

	events, _, after := resolve(a, b, turn, nil, 1)
	e := handEventOf(t, events, SideA)

	if after.Alive() {
		t.Fatal("three Bashes left the target standing")
	}
	hits := hitEvents(events, SideA)
	if len(hits) != 2 {
		t.Errorf("%d hits were thrown, want two — nothing is thrown at a corpse", len(hits))
	}
	// **The arithmetic of the hit that never came is still on the hand event.**
	if e.HandCardCount != 3 {
		t.Errorf("the hand carries %d terms, want every hit's arithmetic", e.HandCardCount)
	}
	defeats := 0
	for _, ev := range events {
		if ev.Kind == KindDefeated {
			defeats++
		}
	}
	if defeats != 1 {
		t.Errorf("the target fell %d times", defeats)
	}
}

func TestAShieldEatsTheHeaviestHit(t *testing.T) {
	a := duelist(10, 8, 5000)
	b := duelist(10, 8, 5000)
	b.Shields = ShieldStack{Basic: 1}

	events, _, after := resolve(a, b, PlainCards(Jab, Smash, Cut), nil, 1)
	hits := hitEvents(events, SideA)
	if len(hits) != 3 {
		t.Fatalf("threw %d hits, want three", len(hits))
	}

	heaviest, best := -1, -1
	for i, c := range PlainCards(Jab, Smash, Cut) {
		if d := a.CardDamage(c); d > best {
			heaviest, best = i, d
		}
	}
	for _, h := range hits {
		blocked := h.Kind == KindBlocked
		if blocked != (h.Slot == heaviest) {
			t.Errorf("seat %d was %v; the one shield eats the heaviest hit, seat %d", h.Slot, h.Kind, heaviest)
		}
	}
	if after.Shields.Count() != 0 {
		t.Errorf("%d shields left standing after the turn", after.Shields.Count())
	}
}

func TestEveryHitLandsItsCardsStatus(t *testing.T) {
	burning := firstStatusOf(t, EffectDamageOverTime)
	lit := relic(t, "hit-kindling", RelicRule{
		When: MomentAttackLands,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoApplyStatus, Status: burning}},
	})

	a := duelist(10, 8, 5000).Wearing(WornRelic{Relic: lit})
	turn := []Card{Of(Bash, Fire), Of(Jab, Ice), Of(Cut, Fire)}
	events, _, _ := resolve(a, duelist(10, 8, 5000), turn, nil, 1)

	var burnedOn []int
	for _, e := range events {
		if e.Kind == KindStatus && e.Side == SideA {
			burnedOn = append(burnedOn, e.Hit)
		}
	}
	if len(burnedOn) != 2 || burnedOn[0] != 0 || burnedOn[1] != 2 {
		t.Errorf("the burn landed on hits %v, want [0 2] — once per fire hit", burnedOn)
	}
}

// firstStatusOf is the first registered status of one effect kind.
func firstStatusOf(t *testing.T, kind StatusEffect) StatusID {
	t.Helper()
	for _, id := range AllStatuses() {
		if StatusOf(id).Effect == kind {
			return id
		}
	}
	t.Fatalf("no status in the file is a %v", kind)
	return 0
}
