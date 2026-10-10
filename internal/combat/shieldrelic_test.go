package combat

import "testing"

// The shield relics: a helm raising a shield each turn, the Prismatic Shield turning shields to the
// foe's element, the Tower Shield keeping one past its lapse, and the three that pay out on a block.

func count(events []Event, k EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// TestAHelmRaisesItsShieldAfterTheExpiry. The helm's shield is standing through the creature's turn
// and eats the creature's hit of its own element, banking the point a matched block buys.
func TestAHelmRaisesItsShieldAfterTheExpiry(t *testing.T) {
	id := relic(t, "helm.ice", RelicRule{
		When: MomentTurnStart,
		Then: []RelicEffect{{Do: DoRaiseShield, Element: Ice, Amount: 1}},
	})
	a := duelist(10, 6, 500).Wearing(WornRelic{Relic: id})
	goblin := creature(Ice, 10, 6, 5000)

	events, a1, _ := resolve(a, goblin, nil, []Card{Of(Bash, Ice), Of(Bash, Ice)}, 1)

	if got := count(events, KindWarded); got != 1 {
		t.Fatalf("the helm warded %d times, want once", got)
	}
	if got := count(events, KindBlocked); got != 1 {
		t.Errorf("the helm's shield blocked %d hits, want one", got)
	}
	if a1.Surge != 1 {
		t.Errorf("an ice shield eating an ice hit banked %d AP, want 1", a1.Surge)
	}
}

// TestAWardIsRefusedWithACondition. turn-start has no card to read.
func TestAWardIsRefusedWithACondition(t *testing.T) {
	_, err := RegisterRelic("relictest.helm.conditioned", "conditioned", []RelicRule{{
		When: MomentTurnStart,
		If:   RelicCondition{Element: Fire, HasElement: true},
		Then: []RelicEffect{{Do: DoRaiseShield, Element: Fire, Amount: 1}},
	}})
	if err == nil {
		t.Fatal("a conditioned raise-shield registered")
	}
}

// TestAPrismaticShieldTakesTheFoesElementAndTheCardKeepsItsOwn. A fire Brace raised against an ice
// goblin is an ice shield, so it banks the point — and the raise still names the card.
func TestAPrismaticShieldTakesTheFoesElementAndTheCardKeepsItsOwn(t *testing.T) {
	id := relic(t, "prismatic", RelicRule{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoMatchFoeShields}},
	})
	a := duelist(10, 6, 500).Wearing(WornRelic{Relic: id})

	events, a1, _ := resolve(a, creature(Ice, 10, 6, 5000),
		[]Card{Of(Brace, Fire)}, []Card{Of(Bash, Ice)}, 1)

	for _, e := range events {
		if e.Kind == KindRaised && e.Element != Ice {
			t.Errorf("the raise announced a %v shield, want ice", e.Element)
		}
	}
	if a1.Surge != 1 {
		t.Errorf("a prismatic shield eating an ice hit banked %d AP, want 1", a1.Surge)
	}
}

// TestATowerShieldKeepsOneUnspentShield. Two shields, one hit: one is eaten, the other stands into
// the next round and eats that round's hit too.
func TestATowerShieldKeepsOneUnspentShield(t *testing.T) {
	id := relic(t, "tower", RelicRule{
		When: MomentEquipped,
		Then: []RelicEffect{{Do: DoKeepShields, Amount: 1}},
	})
	a := duelist(10, 6, 500).Wearing(WornRelic{Relic: id})
	goblin := creature(Ice, 10, 6, 5000)

	_, a1, _ := resolve(a, goblin, []Card{Of(Block, Fire)}, []Card{Of(Bash, Ice)}, 1)
	if a1.Shields.Count() != 1 {
		t.Fatalf("after one of two shields was eaten, %d stand, want 1", a1.Shields.Count())
	}

	events, a2, _ := resolve(a1, goblin, nil, []Card{Of(Bash, Ice)}, 2)
	if count(events, KindBlocked) != 1 || a2.CurrentLife != a1.CurrentLife {
		t.Errorf("the kept shield did not eat the next round's hit")
	}
	if count(events, KindExpired) != 0 {
		t.Errorf("a kept shield was announced as lapsing")
	}
}

// TestWithoutATowerEveryShieldLapses is the control: the same round keeps nothing.
func TestWithoutATowerEveryShieldLapses(t *testing.T) {
	_, a1, _ := resolve(duelist(10, 6, 500), creature(Ice, 10, 6, 5000),
		[]Card{Of(Block, Fire)}, []Card{Of(Bash, Ice)}, 1)
	if a1.Shields.Count() != 0 {
		t.Errorf("%d shields outlived the turn that swung at them", a1.Shields.Count())
	}
}

// blockRelic is one hit-blocked relic doing one thing.
func blockRelic(t *testing.T, key string, do RelicVerb, amount int) RelicID {
	t.Helper()
	return relic(t, key, RelicRule{When: MomentHitBlocked, Then: []RelicEffect{{Do: do, Amount: amount}}})
}

// TestAThornedShieldReturnsHalfTheBlockedHit. The creature's hit would have landed 10; 5 goes back.
func TestAThornedShieldReturnsHalfTheBlockedHit(t *testing.T) {
	a := duelist(10, 6, 500).Wearing(WornRelic{Relic: blockRelic(t, "thorned", DoReflectDamage, 50)})
	goblin := creature(Ice, 10, 6, 5000)

	events, _, g1 := resolve(a, goblin, []Card{Of(Brace, Fire)}, []Card{Of(Bash, Ice)}, 1)

	var blockedAt, reflected int = -1, 0
	for i, e := range events {
		if e.Kind == KindBlocked {
			blockedAt = i
		}
		if e.Kind == KindReflected {
			if i != blockedAt+1 {
				t.Errorf("the reflection is not right behind the block it answers")
			}
			reflected = e.Amount
		}
	}
	would := goblin.CardDamage(Of(Bash, Ice))
	if reflected != would/2 {
		t.Errorf("reflected %d of a %d hit, want %d", reflected, would, would/2)
	}
	if lost := goblin.CurrentLife - g1.CurrentLife; lost < reflected {
		t.Errorf("the goblin lost %d, want at least the %d reflected", lost, reflected)
	}
}

// TestAThornThatKillsStopsTheThrowersTurn. The creature dies to its own first blocked hit and throws
// nothing else.
func TestAThornThatKillsStopsTheThrowersTurn(t *testing.T) {
	a := duelist(0, 6, 500).Wearing(WornRelic{Relic: blockRelic(t, "thorned.kill", DoReflectDamage, 50)})
	goblin := creature(Ice, 10, 6, 1)

	events, a1, g1 := resolve(a, goblin, []Card{Of(Brace, Fire)}, []Card{Of(Bash, Ice), Of(Bash, Ice)}, 1)

	if g1.Alive() {
		t.Fatal("the reflection did not kill a one-life goblin")
	}
	if count(events, KindDefeated) != 1 {
		t.Errorf("wanted one fall, got %d", count(events, KindDefeated))
	}
	if a1.CurrentLife != a.CurrentLife {
		t.Errorf("the dead goblin's second hit landed: the duelist is on %d", a1.CurrentLife)
	}
}

// TestAMendingShieldHealsPerBlockAndATitheShieldPays. Two blocks are two heals and two payments.
func TestAMendingShieldHealsPerBlockAndATitheShieldPays(t *testing.T) {
	a := duelist(10, 6, 500).
		Wearing(WornRelic{Relic: blockRelic(t, "mending", DoHealOnBlock, 3)}).
		Wearing(WornRelic{Relic: blockRelic(t, "tithe", DoVitaeOnBlock, 1)})
	a.CurrentLife = 100

	events, a1, _ := resolve(a, creature(Ice, 10, 6, 5000),
		[]Card{Of(Block, Fire)}, []Card{Of(Bash, Ice), Of(Bash, Ice)}, 1)

	if got := regained(events, SideA); got != 6 {
		t.Errorf("two blocks healed %d, want 6", got)
	}
	if a1.Vitae != a.Vitae+2 {
		t.Errorf("two blocks paid %d vitae, want 2", a1.Vitae-a.Vitae)
	}
	if count(events, KindTithed) != 2 {
		t.Errorf("wanted two tithe announcements, got %d", count(events, KindTithed))
	}
}
