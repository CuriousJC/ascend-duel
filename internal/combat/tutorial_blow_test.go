package combat

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// **The tutorial promises a kill in one blow, and four separate files can quietly break it.**
//
// The lesson `data/tutorial.json` teaches is a deck of five Jabs, one per element, swung at a
// GiantBat: a Card Five of a Kind against the gentlest opponent on floor one. Nothing about that
// is enforced by the things it depends on — Jab's Amount in `duelist_cards.json`, the ladder's
// multiplier in `hands.json`, the duelist's DMG in `duelists.json`, and the bat's HP in
// `enemies.json` are each tuned for their own reasons — so a balance pass anywhere would leave
// Bob telling a new player to swing a killing blow that does not kill.
//
// **It fails here rather than in front of the one player who cannot tell it is broken**, which is
// the whole argument: a tutorial is read by someone with no idea what the game is supposed to do.
//
// It lives in this package because this is where the blow is worked out and where a test can run
// with no window. It is deliberately *not* in `internal/scenario`, which is compiled out of every
// ordinary build and so would take the check with it.
func TestTheTutorialsBlowKillsTheTutorialsEnemy(t *testing.T) {
	const (
		enemyRecord = "GiantBat"
		conceptKey  = "Jab"
		duelist     = "Fighter1"
	)

	me, ok := data.LoadDuelists()[duelist]
	if !ok {
		t.Fatalf("no duelist record %q", duelist)
	}
	bat, ok := data.LoadEnemies()[enemyRecord]
	if !ok {
		t.Fatalf("no enemy record %q", enemyRecord)
	}
	id, ok := ConceptByKey(conceptKey)
	if !ok {
		t.Fatalf("no concept %q", conceptKey)
	}

	// The taught turn: one Jab of every colour, which is the whole of the tutorial's deck.
	els := []Element{Fire, Ice, Lightning, Earth, Arcane}
	turn := make([]Slot, len(els))
	spent := 0
	for i, e := range els {
		turn[i] = Slot{Card: Of(id, e), Index: i}
		spent += ConceptOf(id).Cost
	}

	if spent > me.Actions {
		t.Fatalf("the taught turn costs %d AP and the duelist has %d", spent, me.Actions)
	}

	blow := BlowFor(turn)
	if len(blow.Cards) != len(els) {
		t.Fatalf("the hand took %d of the %d cards; the lesson says all of them",
			len(blow.Cards), len(els))
	}

	base := 0
	for _, i := range blow.Cards {
		base += ConceptOf(turn[i].Card.Concept).Amount * me.DMG / 100
	}
	total := scaleDamage(base, blow.Multiplier)

	t.Logf("%s x%d: %d base x %d%% = %d against %d HP (%d of %d AP)",
		blow.Hand.Name, blow.Multiplier, base, blow.Multiplier, total, bat.HP, spent, me.Actions)

	if total < bat.HP {
		t.Errorf("the tutorial's blow deals %d and %s has %d HP: the taught round no longer "+
			"wins in one. Retune the lesson or the numbers it depends on.", total, enemyRecord, bat.HP)
	}
}
