package screens

import (
	"strconv"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
)

// **The lesson now says what the shield did, so what it did has to stay true.**
//
// The taught round is a wound rather than a kill precisely so the creature gets a turn, and the
// point of that turn is the one shield in the taught hand eating one of its blows. Since 2026-09-08
// the shield eats the *heaviest* blow rather than the first one queued — see combat.shieldedSlots —
// and the tutorial step that explains it names that. Four files can break the promise between them:
// the creature's card amounts, its DMG, the taught set, and the ladder's multiplier.
//
// **It is the whole round, resolved**, rather than an assertion about the mask. What the step
// claims is a thing the player watches happen, so what is checked is the log they will watch.
func TestTheTutorialsShieldEatsTheCreaturesHeaviestBlow(t *testing.T) {
	script := tutorial.Load()
	runSeed, err := seeds.Parse(script.Seed)
	if err != nil {
		t.Fatalf("tutorial.json seed %q: %v", script.Seed, err)
	}

	me, ok := data.LoadDuelists()["Fighter1"]
	if !ok {
		t.Fatal("no duelist record Fighter1")
	}
	foe, ok := data.LoadEnemies()[script.Enemy]
	if !ok {
		t.Fatalf("tutorial.json names enemy %q, which is in no roster", script.Enemy)
	}

	// The taught turn: the largest elemental set in the dealt hand, which is what the lesson points
	// at and what the lock leaves clickable. Read off the seed rather than written down, for
	// TestTheTutorialsSeedDealsTheHandTheLessonDescribes's reason.
	hand := OpeningCards(seeds.ForFight(runSeed, seeds.PlayerDeck, 0))
	taught := taughtSet(hand)
	if len(taught) == 0 {
		t.Fatal("the tutorial's seed deals no matching set at all")
	}

	// The creature's own turn, planned exactly as the screen plans it.
	pile := decks.NewEnemyPile(script.Enemy, seeds.ForFight(runSeed, seeds.EnemyDeck, 0), decks.EnemyHandSize)
	fighter := combat.Duelist{
		DMG: me.DMG, MaxLife: me.HP, CurrentLife: me.HP, Actions: me.Actions,
	}
	creature := combat.Duelist{
		DMG: foe.DMG, MaxLife: foe.HP, CurrentLife: foe.HP,
		Actions: foe.Actions, SoloAttacks: true,
	}
	theirs := pile.Plan(creature)

	log, after, _ := combat.ResolveRound(fighter, creature, taught, theirs, 1, combat.Sources{})

	// **The block has to happen at all.** A taught set that stopped holding a shield, or a blow that
	// started killing, would leave the step explaining something the player never sees.
	var blocked []combat.Event
	for _, e := range log {
		if e.Kind == combat.KindBlocked {
			blocked = append(blocked, e)
		}
	}
	if len(blocked) != 1 {
		t.Fatalf("the taught round produced %d blocks and the lesson describes exactly one\n%s",
			len(blocked), describeTurn(theirs, creature))
	}

	// **And it has to be the heaviest**, which is the thing the step actually teaches. A creature
	// whose deck flattened out — every card worth the same — would make the sentence true and
	// meaningless, so the heaviest also has to be the only one of its size.
	eaten, heaviest, ties := blocked[0].Slot, 0, 0
	for i, c := range theirs {
		if c.Category() != combat.CategoryAttack {
			continue
		}
		switch dmg := creature.CardDamage(c); {
		case dmg > creature.CardDamage(theirs[heaviest]):
			heaviest, ties = i, 1
		case dmg == creature.CardDamage(theirs[heaviest]):
			ties++
		}
	}
	if eaten != heaviest {
		t.Errorf("the shield ate slot %d (%s) and the heaviest blow is slot %d (%s)\n%s",
			eaten, combat.ConceptOf(theirs[eaten].Concept).Label,
			heaviest, combat.ConceptOf(theirs[heaviest].Concept).Label,
			describeTurn(theirs, creature))
	}
	if ties > 1 {
		t.Errorf("the creature's turn holds %d blows of equal weight, so the lesson's "+
			"'it ate the biggest one' is true and says nothing\n%s", ties, describeTurn(theirs, creature))
	}

	// **The player has to survive it and still be visibly hurt**, since the next step sends them to
	// the ledger to read what happened and then back in to finish the fight.
	took := me.HP - after.CurrentLife
	t.Logf("the taught round: the shield eats %s, the player takes %d and stands at %d of %d\n%s",
		combat.ConceptOf(theirs[eaten].Concept).Label, took, after.CurrentLife, me.HP,
		describeTurn(theirs, creature))

	// **The step names the card by name, so the name has to be right.** `the-shield-bit` points at
	// the break and says which blow it took; a retuned creature deck would leave Bob naming a card
	// the player is not looking at, and nothing else would catch it.
	eatenLabel := combat.ConceptOf(theirs[eaten].Concept).Label
	for _, step := range script.Steps {
		if step.Key != "the-shield-bit" {
			continue
		}
		if !strings.Contains(step.Text, eatenLabel) {
			t.Errorf("the shield eats %s and the step says %q: Bob is naming the wrong card",
				eatenLabel, step.Text)
		}
	}

	if took <= 0 {
		t.Error("the creature's turn cost the player nothing, so the round has nothing to read about")
	}
	if after.CurrentLife <= 0 {
		t.Error("the creature's turn killed the player, and the lesson has three more steps")
	}
}

// taughtSet is the largest elemental set in a dealt hand — the cards the lesson points at.
func taughtSet(hand []combat.Card) []combat.Card {
	counts := map[combat.Element]int{}
	for _, c := range hand {
		counts[c.Element]++
	}
	best, bestN := combat.Basic, 0
	for _, c := range hand {
		if n := counts[c.Element]; n > bestN {
			best, bestN = c.Element, n
		}
	}
	var out []combat.Card
	for _, c := range hand {
		if c.Element == best {
			out = append(out, c)
		}
	}
	return out
}

// describeTurn writes a creature's queue out with what each card is worth, so a failure above says
// which blow the shield should have taken rather than which index.
func describeTurn(turn []combat.Card, actor combat.Duelist) string {
	out := "creature's turn:"
	for i, c := range turn {
		out += "\n  " + strconv.Itoa(i) + " " + combat.ConceptOf(c.Concept).Label +
			" for " + strconv.Itoa(actor.CardDamage(c))
	}
	return out
}
