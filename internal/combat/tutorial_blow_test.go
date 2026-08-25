package combat

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// **The tutorial promises a kill in one blow, and four separate files can quietly break it.**
//
// The lesson `data/tutorial.json` teaches is four cards of one colour — a Jab, a Cut, a Thrust and
// a Strike, all fire — swung at a GiantBat: an Elemental Four of a Kind against the gentlest
// opponent on floor one. Nothing about that is enforced by the things it depends on — those cards'
// Amounts in `duelist_cards.json`, the ladder's multiplier in `hands.json`, the duelist's DMG in
// `duelists.json`, and the bat's HP in `enemies.json` are each tuned for their own reasons — so a
// balance pass anywhere would leave Bob telling a new player to swing a killing blow that does not
// kill.
//
// **This checks the rules; `internal/screens` checks the deal.** The turn below is written out
// rather than read from the shuffle, because the shuffle needs a scene and this package must stay
// window-free. `TestTheTutorialsSeedDealsTheHandTheLessonDescribes` over there is the other half:
// it takes the script's own seed, works out what the hand actually holds, and fails if these four
// cards are not what the lesson deals. Neither test alone is enough.
//
// **It fails here rather than in front of the one player who cannot tell it is broken**, which is
// the whole argument: a tutorial is read by someone with no idea what the game is supposed to do.
//
// It lives in this package because this is where the blow is worked out and where a test can run
// with no window. It is deliberately *not* in `internal/scenario`, which is compiled out of every
// ordinary build and so would take the check with it.
func TestTheTutorialsBlowKillsTheTutorialsEnemy(t *testing.T) {
	const duelist = "Fighter1"

	// The taught turn, as `data/tutorial.json`'s seed deals it: four fire cards, 6 AP exactly.
	taught := []string{"Jab", "Cut", "Thrust", "Strike"}
	const taughtElement = Fire

	// **The opponent is read off the script rather than written here** *(2026-08-25)*, since
	// `data/tutorial.json` is where the lesson now pins the room it is fought in. Two copies of the
	// creature's name would let the test go on passing against a bat the tutorial no longer meets.
	enemyRecord := data.LoadTutorial().Enemy
	if enemyRecord == "" {
		t.Fatal("data/tutorial.json names no enemy, so the lesson promises a kill against nobody")
	}

	me, ok := data.LoadDuelists()[duelist]
	if !ok {
		t.Fatalf("no duelist record %q", duelist)
	}
	bat, ok := data.LoadEnemies()[enemyRecord]
	if !ok {
		t.Fatalf("no enemy record %q", enemyRecord)
	}
	turn := make([]Slot, len(taught))
	spent := 0
	for i, key := range taught {
		id, ok := ConceptByKey(key)
		if !ok {
			t.Fatalf("no concept %q", key)
		}
		turn[i] = Slot{Card: Of(id, taughtElement), Index: i}
		spent += ConceptOf(id).Cost
	}

	if spent > me.Actions {
		t.Fatalf("the taught turn costs %d AP and the duelist has %d", spent, me.Actions)
	}

	blow := BlowFor(turn)
	if len(blow.Cards) != len(taught) {
		t.Fatalf("the hand took %d of the %d cards; the lesson says all of them",
			len(blow.Cards), len(taught))
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
