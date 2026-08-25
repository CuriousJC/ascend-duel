package tutorial

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The shipped script has to parse, or the game does not start. This is the test that turns a
// misspelled anchor into a red `go test` instead of a panic in front of a player.
func TestTheShippedScriptLoads(t *testing.T) {
	s, err := Parse(data.LoadTutorial())
	if err != nil {
		t.Fatalf("data/tutorial.json does not parse: %v", err)
	}
	if len(s) == 0 {
		t.Fatal("the shipped script is empty")
	}
}

// **Every step has to be reachable and every step has to be escapable.** A condition nothing can
// satisfy is a tutorial that hangs, and the whole reason this package is free of Ebitengine is so
// that walking the script end to end costs a test rather than a play session.
//
// The walk feeds each step exactly what its own condition asks for, which is also an assertion
// that the condition vocabulary is complete: a Condition added without an arm here fails at the
// default rather than passing quietly.
func TestEveryStepCanBeSatisfied(t *testing.T) {
	run := NewRun(Load())

	for guard := 0; run.Active(); guard++ {
		if guard > 200 {
			t.Fatal("the script did not finish; a step is unsatisfiable")
		}
		step, _ := run.Current()

		before := step
		f, next := satisfying(t, step.Until, run.baseRounds)
		run.Update(f, next)

		if cur, ok := run.Current(); ok && cur.Key == before.Key {
			t.Fatalf("step %q did not advance on %v", before.Key, before.Until)
		}
	}
}

// satisfying is the Facts and the button press that make one condition true.
func satisfying(t *testing.T, c Condition, baseRounds int) (Facts, bool) {
	t.Helper()
	switch c {
	case CondNext:
		return Facts{}, true
	case CondCardsQueued:
		return Facts{Queued: 1}, false
	case CondHandEmptied:
		return Facts{Queued: 5, Unqueued: 0}, false
	case CondMatchQueued:
		return Facts{Queued: 5, Matching: 5, MatchingQueued: 5}, false
	case CondDuelPressed:
		return Facts{Resolving: true}, false
	case CondRoundDone:
		return Facts{RoundsPlayed: baseRounds + 1}, false
	case CondPhaseFight:
		return Facts{Phase: "fight"}, false
	case CondPhaseReward:
		return Facts{Phase: "reward"}, false
	case CondPhaseShop:
		return Facts{Phase: "shop"}, false
	}
	t.Fatalf("condition %v has no arm in this test; add one with the condition", c)
	return Facts{}, false
}

// The zero Facts is a scene that has published nothing, and it must stall rather than sail
// through. A screen that forgets to publish should be obvious immediately.
func TestNothingPublishedSatisfiesNothingButNext(t *testing.T) {
	for c, name := range conditionNames {
		if c == CondNext {
			continue
		}
		run := &Run{script: Script{{Key: "x", Text: "x", Until: c}}}
		run.Update(Facts{}, false)
		if !run.Active() {
			t.Errorf("condition %q advanced on an empty Facts", name)
		}
	}
}

// A step asking for a click with nothing named to click would leave the player no legal click at
// all and a condition only they could satisfy. Refused rather than quietly downgraded.
func TestAnActionStepWithNothingToClickIsRefused(t *testing.T) {
	_, err := Parse([]data.TutorialStepData{
		{StepRecord: "bad", Text: "click it", Until: "cards-queued"},
	})
	if err == nil {
		t.Fatal("an action step with no anchor was accepted")
	}
	if !strings.Contains(err.Error(), "nothing to click") {
		t.Errorf("unhelpful error: %v", err)
	}
}

// Both vocabularies are closed, and a word the file invents does not exist.
func TestAnInventedWordIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name string
		rec  data.TutorialStepData
	}{
		{"anchor", data.TutorialStepData{StepRecord: "a", Text: "t", Until: "next", Anchor: "the-kitchen"}},
		{"condition", data.TutorialStepData{StepRecord: "a", Text: "t", Until: "whenever"}},
	} {
		if _, err := Parse([]data.TutorialStepData{tc.rec}); err == nil {
			t.Errorf("an invented %s was accepted", tc.name)
		}
	}
}

// **The three-way split, which is the whole rule.** A step asking for a click opens its anchor; a
// step asking to be read locks everything; a step waiting on an outcome locks nothing, because it
// cannot know which controls reaching that outcome will need.
//
// This is derived rather than authored precisely so it cannot disagree with the condition — see
// Lock. The bug it replaced was a step about which room you are standing in leaving the screen
// live while the player queued cards it had not mentioned.
func TestTheLockIsDerivedFromTheCondition(t *testing.T) {
	s, err := Parse([]data.TutorialStepData{
		{StepRecord: "read", Text: "t", Until: "next", Anchor: "duel-button"},
		{StepRecord: "bare-read", Text: "t", Until: "next"},
		{StepRecord: "click", Text: "t", Until: "duel-pressed", Anchor: "duel-button"},
		{StepRecord: "wait", Text: "t", Until: "phase-shop", Anchor: "duel-button"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []Lock{LockAll, LockAll, LockToAnchor, LockNone} {
		if s[i].Lock != want {
			t.Errorf("step %q locks %v, wanted %v", s[i].Key, s[i].Lock, want)
		}
	}
}

// A read step locks the screen even though it points at something. That is this turn's bug stated
// as a test: pointing and permitting are different, and only Bob's buttons are live.
func TestAReadStepLocksTheScreen(t *testing.T) {
	s, err := Parse([]data.TutorialStepData{
		{StepRecord: "rooms", Text: "eight floors", Until: "next", Anchor: "tower-place"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s[0].Lock != LockAll {
		t.Errorf("a step that only wants reading locks %v", s[0].Lock)
	}
}

// One step per frame. `phase-shop` is true for every frame the shop is up, so a script whose next
// step also waited on it would never be seen at all if a satisfied step fell straight through.
func TestOnlyOneStepAdvancesPerFrame(t *testing.T) {
	run := &Run{script: Script{
		{Key: "a", Text: "a", Until: CondPhaseShop},
		{Key: "b", Text: "b", Until: CondPhaseShop},
		{Key: "c", Text: "c", Until: CondPhaseShop},
	}}
	run.Update(Facts{Phase: "shop"}, false)

	step, ok := run.Current()
	if !ok || step.Key != "b" {
		t.Fatalf("expected to be on step b, got %q (active %v)", step.Key, ok)
	}
}

// round-done must measure a round the step actually watched. A step that arrives mid-playback
// would otherwise count the round it did not start and vanish on its first frame.
func TestRoundDoneIgnoresTheRoundItArrivedDuring(t *testing.T) {
	run := &Run{script: Script{
		{Key: "watch", Text: "w", Until: CondRoundDone},
		{Key: "after", Text: "a", Until: CondNext},
	}}
	// Arriving with three rounds already played and one under way.
	run.Advance(Facts{RoundsPlayed: 3, Resolving: true})
	run.step = 0 // Advance moved the cursor; the baseline is what is being tested.

	run.Update(Facts{RoundsPlayed: 3, Resolving: false}, false)
	if step, _ := run.Current(); step.Key != "watch" {
		t.Fatal("round-done fired on a round that had already been played")
	}

	run.Update(Facts{RoundsPlayed: 4, Resolving: false}, false)
	if step, _ := run.Current(); step.Key != "after" {
		t.Fatal("round-done did not fire on the round the step watched")
	}
}

// Skip ends it, and nothing brings it back.
func TestSkipEndsTheRun(t *testing.T) {
	run := NewRun(Load())
	run.Skip()
	if run.Active() {
		t.Fatal("Skip left the tutorial running")
	}
	run.Update(Facts{Phase: "fight"}, true)
	if run.Active() {
		t.Fatal("a skipped tutorial came back")
	}
}

// Every anchor needs a name, or a step cannot write it down and String prints "unknown". This is
// the append-only enum's tripwire: a kind added without a table entry fails here.
func TestEveryAnchorAndConditionHasAName(t *testing.T) {
	for _, a := range Anchors() {
		if a.String() == "unknown" {
			t.Errorf("anchor %d has no name", a)
		}
	}
	for c := CondNext; int(c) < len(conditionNames); c++ {
		if c.String() == "unknown" {
			t.Errorf("condition %d has no name", c)
		}
	}
}

// Every action condition locks to its anchor. A script cannot ask for a click and leave the rest of
// the screen live, because that is not a thing the file can say any more.
func TestEveryActionConditionLocksToItsAnchor(t *testing.T) {
	for _, until := range []string{"cards-queued", "hand-emptied", "matching-queued", "duel-pressed"} {
		s, err := Parse([]data.TutorialStepData{
			{StepRecord: "do-it", Text: "do it", Until: until, Anchor: "hand"},
		})
		if err != nil {
			t.Fatalf("%q: %v", until, err)
		}
		if s[0].Lock != LockToAnchor {
			t.Errorf("%q locks %v, wanted to-anchor", until, s[0].Lock)
		}
	}
}

// And the outcome conditions must leave the screen alone, since winning a fight takes as many
// clicks as it takes on controls no step should be naming. Locking one would deadlock the tutorial
// against its own condition.
func TestAnOutcomeStepMayLeaveTheScreenAlone(t *testing.T) {
	for _, until := range []string{"round-done", "phase-fight", "phase-reward", "phase-shop"} {
		if _, err := Parse([]data.TutorialStepData{
			{StepRecord: "waiting", Text: "hold on", Until: until},
		}); err != nil {
			t.Errorf("an ungated %q step was refused: %v", until, err)
		}
	}
}

// The shipped script has to obey its own rule. Parse enforces it, so this is really a guard against
// the rule being loosened later without the script being re-read.
func TestEveryActionStepInTheScriptGates(t *testing.T) {
	for _, step := range Load() {
		if step.Until.isAction() && step.Lock != LockToAnchor {
			t.Errorf("step %q waits on %q but locks %v", step.Key, step.Until, step.Lock)
		}
	}
}

// The step asking for one card must be anchored on one card, not on the band it sits in. This is
// the specific regression: `hand` is the whole row and permitted five clicks where one was asked
// for.
func TestTheFirstCardStepPointsAtOneCard(t *testing.T) {
	var found bool
	for _, step := range Load() {
		if step.Until != CondCardsQueued {
			continue
		}
		found = true
		if step.Anchor == AnchorHand {
			t.Errorf("step %q asks for one card but unlocks the whole hand band; "+
				"it wants %q", step.Key, AnchorFirstCard)
		}
	}
	if !found {
		t.Skip("no step in the script waits on cards-queued")
	}
}

// **Queueing a card does not take it out of the hand.** The combat screen marks it selected and
// leaves it in the row, so a hand of five with five queued still has five cards in it — which is
// why the fact is named for what is *unqueued* rather than for the hand's length. A step waiting
// on the hand's length waits forever.
func TestHandEmptiedCountsWhatIsUnqueuedNotWhatIsHeld(t *testing.T) {
	run := &Run{script: Script{
		{Key: "all", Text: "take them all", Anchor: AnchorHand,
			Lock: LockToAnchor, Until: CondHandEmptied},
		{Key: "after", Text: "done", Until: CondNext},
	}}

	// Four of five picked: not yet.
	run.Update(Facts{Queued: 4, Unqueued: 1}, false)
	if step, _ := run.Current(); step.Key != "all" {
		t.Fatal("hand-emptied fired with a card still unqueued")
	}

	// All five picked. The player is still holding five cards; none of them are unqueued.
	run.Update(Facts{Queued: 5, Unqueued: 0}, false)
	if step, _ := run.Current(); step.Key != "after" {
		t.Fatal("hand-emptied did not fire once every card was queued")
	}
}
