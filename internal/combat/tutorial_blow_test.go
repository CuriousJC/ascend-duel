package combat

import (
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// **The tutorial promises a blow that wounds and does not kill, and several files can quietly
// break it in either direction.**
//
// The lesson `data/tutorial.json` teaches is four cards of one colour — a Jab, a Ward, a Thrust
// and a Strike, all lightning — swung at a GiantBat: an Elemental Four of a Kind against the
// gentlest opponent on floor one. **One of the four is a shield**, which is the whole of the
// second lesson: it deals nothing, it counts toward the hand anyway, and it eats one of the
// three attacks the bat swings back. That is why the taught round no longer ends the fight —
// a creature that dies in one blow never gets a turn, and a shield that never blocks anything
// is a rule the player is told about rather than shown.
//
// **So this test is two-sided, and that is the point** *(2026-09-06)*. It fails if the blow
// stops killing... it never did; it fails if the blow *starts* killing, which would delete the
// bat's turn and with it the shield step, the ledger step and the second round; and it fails if
// the blow gets so weak that the fight cannot plausibly be finished on the round after. Nothing
// about any of that is enforced by the things it depends on — those cards' Amounts in
// `duelist_cards.json`, the ladder's multiplier in `hands.json`, the duelist's DMG in
// `duelists.json`, and the bat's HP in `enemies.json` are each tuned for their own reasons.
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
func TestTheTutorialsBlowWoundsTheTutorialsEnemyWithoutKillingIt(t *testing.T) {
	const duelist = "Fighter1"

	// The taught turn, as `data/tutorial.json`'s seed deals it: four lightning cards, 6 AP exactly,
	// one of them a shield.
	taught := []string{"Jab", "Ward", "Thrust", "Strike"}
	const taughtElement = Lightning

	// **The opponent is read off the script rather than written here** *(2026-08-25)*, since
	// `data/tutorial.json` is where the lesson now pins the room it is fought in. Two copies of the
	// creature's name would let the test go on passing against a bat the tutorial no longer meets.
	enemyRecord := data.LoadTutorial().Enemy
	if enemyRecord == "" {
		t.Fatal("data/tutorial.json names no enemy, so the lesson promises a wound against nobody")
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
	spent, shields := 0, 0
	for i, key := range taught {
		id, ok := ConceptByKey(key)
		if !ok {
			t.Fatalf("no concept %q", key)
		}
		turn[i] = Slot{Card: Of(id, taughtElement), Index: i}
		spent += ConceptOf(id).Cost
		if ConceptOf(id).Verb != VerbAttack {
			shields++
		}
	}

	// **Exactly one of the four is a shield.** The step that explains it says "one of those four",
	// and a taught set that became all-attack would leave Bob pointing at a card that is not there.
	if shields != 1 {
		t.Errorf("the taught set holds %d cards that are not attacks; the lesson describes one shield",
			shields)
	}

	if spent > me.Actions {
		t.Fatalf("the taught turn costs %d AP and the duelist has %d", spent, me.Actions)
	}

	blow := BlowFor(turn)
	if len(blow.Cards) != len(taught) {
		t.Fatalf("the hand took %d of the %d cards; the lesson says all of them — a shield joins a "+
			"hand like anything else, which is the lesson the step is teaching",
			len(blow.Cards), len(taught))
	}

	base := 0
	for _, i := range blow.Cards {
		base += ConceptOf(turn[i].Card.Concept).Amount * me.DMG / 100
	}
	total := scaleDamage(base, blow.Multiplier)

	t.Logf("%s x%d: %d base x %d%% = %d against %d HP (%d of %d AP), leaving %d",
		blow.Hand.Name, blow.Multiplier, base, blow.Multiplier, total, bat.HP, spent, me.Actions,
		bat.HP-total)

	if total >= bat.HP {
		t.Errorf("the tutorial's blow deals %d and %s has %d HP: the taught round now kills in one, "+
			"so the creature never takes a turn and the shield, the counter-attack and the ledger "+
			"steps all describe something the player cannot see. Retune the lesson or the numbers "+
			"it depends on.", total, enemyRecord, bat.HP)
	}

	// **And the round after has to be able to finish it.** The lesson says "it has little left,
	// take it down", and a bat left on most of its life turns a two-round lesson into a slog the
	// script has nothing more to say about. Half is the line: a second turn of the same order
	// clears it comfortably.
	if left := bat.HP - total; left*2 > bat.HP {
		t.Errorf("the taught blow leaves %s on %d of %d HP, which is more than half: the lesson "+
			"tells the player to finish it on the next round", enemyRecord, left, bat.HP)
	}
}
