package tutorial

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
)

// The shipped script has to parse, or the game does not start. This is the test that turns a
// misspelled anchor into a red `go test` instead of a panic in front of a player.
func TestTheShippedScriptLoads(t *testing.T) {
	parkTutorial(t)
	s, err := Parse(data.LoadTutorial())
	if err != nil {
		t.Fatalf("data/tutorial.json does not parse: %v", err)
	}
	if s.Len() == 0 {
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
	parkTutorial(t)
	run := NewRun(Load())

	for guard := 0; run.Active(); guard++ {
		if guard > 200 {
			t.Fatal("the script did not finish; a step is unsatisfiable")
		}
		step, _ := run.Current()

		before := step
		f, next := satisfying(t, step, run.baseRounds, run.baseLedger, run.baseDMG, run.baseBreaks)
		run.Update(f, next)

		if cur, ok := run.Current(); ok && cur.Key == before.Key {
			t.Fatalf("step %q did not advance on %v", before.Key, before.Until)
		}
	}
}

// satisfying is the Facts and the button press that make one condition true.
func satisfying(t *testing.T, step Step, baseRounds, baseLedger, baseDMG, baseBreaks int) (Facts, bool) {
	t.Helper()
	c := step.Until
	switch c {
	case CondNext:
		return Facts{}, true
	case CondCardsQueued:
		return Facts{Queued: max(1, step.Count)}, false
	case CondHandEmptied:
		return Facts{Queued: 5, Unqueued: 0}, false
	case CondMatchQueued:
		return Facts{Queued: 5, Matching: 5, MatchingQueued: 5}, false
	case CondDuelPressed:
		return Facts{Resolving: true}, false
	case CondRoundDone:
		return Facts{RoundsPlayed: baseRounds + 1}, false
	case CondShieldBroke:
		return Facts{Resolving: true, ShieldBreaks: baseBreaks + 1}, false
	case CondLedgerOpened:
		return Facts{LedgerOpens: baseLedger + 1}, false
	case CondRelicsWorn:
		return Facts{RelicsWorn: step.Count}, false
	case CondDMGBought:
		return Facts{DMGBonus: baseDMG + 1}, false
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
	parkTutorial(t)
	for c, name := range conditionNames {
		if c == CondNext {
			continue
		}
		run := &Run{script: Script{Steps: []Step{{Key: "x", Text: "x", Until: c}}}}
		run.Update(Facts{}, false)
		if !run.Active() {
			t.Errorf("condition %q advanced on an empty Facts", name)
		}
	}
}

// A step asking for a click with nothing named to click would leave the player no legal click at
// all and a condition only they could satisfy. Refused rather than quietly downgraded.
func TestAnActionStepWithNothingToClickIsRefused(t *testing.T) {
	parkTutorial(t)
	_, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
		{StepRecord: "bad", Text: "click it", Until: "cards-queued"},
	}})
	if err == nil {
		t.Fatal("an action step with no anchor was accepted")
	}
	if !strings.Contains(err.Error(), "nothing to click") {
		t.Errorf("unhelpful error: %v", err)
	}
}

// Both vocabularies are closed, and a word the file invents does not exist.
func TestAnInventedWordIsRefused(t *testing.T) {
	parkTutorial(t)
	for _, tc := range []struct {
		name string
		rec  data.TutorialStepData
	}{
		{"anchor", data.TutorialStepData{StepRecord: "a", Text: "t", Until: "next", Anchor: "the-kitchen"}},
		{"condition", data.TutorialStepData{StepRecord: "a", Text: "t", Until: "whenever"}},
	} {
		if _, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{tc.rec}}); err == nil {
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
	parkTutorial(t)
	s, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
		{StepRecord: "read", Text: "t", Until: "next", Anchor: "duel-button"},
		{StepRecord: "bare-read", Text: "t", Until: "next"},
		{StepRecord: "click", Text: "t", Until: "duel-pressed", Anchor: "duel-button"},
		{StepRecord: "wait", Text: "t", Until: "phase-shop", Anchor: "duel-button"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []Lock{LockAll, LockAll, LockToAnchor, LockNone} {
		if s.Steps[i].Lock != want {
			t.Errorf("step %q locks %v, wanted %v", s.Steps[i].Key, s.Steps[i].Lock, want)
		}
	}
}

// A read step locks the screen even though it points at something. That is this turn's bug stated
// as a test: pointing and permitting are different, and only Bob's buttons are live.
func TestAReadStepLocksTheScreen(t *testing.T) {
	parkTutorial(t)
	s, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
		{StepRecord: "rooms", Text: "eight realms", Until: "next", Anchor: "journey-place"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Steps[0].Lock != LockAll {
		t.Errorf("a step that only wants reading locks %v", s.Steps[0].Lock)
	}
}

// One step per frame. `phase-shop` is true for every frame the shop is up, so a script whose next
// step also waited on it would never be seen at all if a satisfied step fell straight through.
func TestOnlyOneStepAdvancesPerFrame(t *testing.T) {
	parkTutorial(t)
	run := &Run{script: Script{Steps: []Step{
		{Key: "a", Text: "a", Until: CondPhaseShop},
		{Key: "b", Text: "b", Until: CondPhaseShop},
		{Key: "c", Text: "c", Until: CondPhaseShop},
	}}}
	run.Update(Facts{Phase: "shop"}, false)

	step, ok := run.Current()
	if !ok || step.Key != "b" {
		t.Fatalf("expected to be on step b, got %q (active %v)", step.Key, ok)
	}
}

// round-done must measure a round the step actually watched. A step that arrives mid-playback
// would otherwise count the round it did not start and vanish on its first frame.
func TestRoundDoneIgnoresTheRoundItArrivedDuring(t *testing.T) {
	parkTutorial(t)
	run := &Run{script: Script{Steps: []Step{
		{Key: "watch", Text: "w", Until: CondRoundDone},
		{Key: "after", Text: "a", Until: CondNext},
	}}}
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
	parkTutorial(t)
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
	parkTutorial(t)
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
	parkTutorial(t)
	for _, until := range []string{"cards-queued", "hand-emptied", "matching-queued", "duel-pressed"} {
		s, err := Parse(data.TutorialData{Match: "concept", Steps: []data.TutorialStepData{
			{StepRecord: "do-it", Text: "do it", Until: until, Anchor: "hand"},
		}})
		if err != nil {
			t.Fatalf("%q: %v", until, err)
		}
		if s.Steps[0].Lock != LockToAnchor {
			t.Errorf("%q locks %v, wanted to-anchor", until, s.Steps[0].Lock)
		}
	}
}

// And the outcome conditions must leave the screen alone, since winning a fight takes as many
// clicks as it takes on controls no step should be naming. Locking one would deadlock the tutorial
// against its own condition.
func TestAnOutcomeStepMayLeaveTheScreenAlone(t *testing.T) {
	parkTutorial(t)
	for _, until := range []string{"round-done", "phase-fight", "phase-reward", "phase-shop"} {
		if _, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
			{StepRecord: "waiting", Text: "hold on", Until: until},
		}}); err != nil {
			t.Errorf("an ungated %q step was refused: %v", until, err)
		}
	}
}

// The shipped script has to obey its own rule. Parse enforces it, so this is really a guard against
// the rule being loosened later without the script being re-read.
func TestEveryActionStepInTheScriptGates(t *testing.T) {
	parkTutorial(t)
	for _, step := range Load().Steps {
		if step.Until.isAction() && step.Lock != LockToAnchor {
			t.Errorf("step %q waits on %q but locks %v", step.Key, step.Until, step.Lock)
		}
	}
}

// The step asking for one card must be anchored on one card, not on the band it sits in. This is
// the specific regression: `hand` is the whole row and permitted five clicks where one was asked
// for.
func TestTheFirstCardStepPointsAtOneCard(t *testing.T) {
	parkTutorial(t)
	var found bool
	for _, step := range Load().Steps {
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
	parkTutorial(t)
	run := &Run{script: Script{Steps: []Step{
		{Key: "all", Text: "take them all", Anchor: AnchorHand,
			Lock: LockToAnchor, Until: CondHandEmptied},
		{Key: "after", Text: "done", Until: CondNext},
	}}}

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

// **An axis is required by any script that points at a matching set, and refused when absent.** The
// lit square and the condition that lets the player past it are the same cards, and which cards
// those are depends entirely on the axis — so defaulting it would be a lesson pointing confidently
// at the wrong ones.
func TestAMatchingStepWithoutAnAxisIsRefused(t *testing.T) {
	parkTutorial(t)
	_, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
		{StepRecord: "take", Text: "take them", Until: "matching-queued", Anchor: "matching-cards"},
	}})
	if err == nil {
		t.Fatal("a matching step with no Match axis was accepted")
	}
	if !strings.Contains(err.Error(), "Match axis") {
		t.Errorf("unhelpful error: %v", err)
	}
}

// A script that never points at a set does not need one, which is what keeps the axis a property of
// the lesson rather than a field every script has to carry.
func TestAScriptWithNoMatchingStepNeedsNoAxis(t *testing.T) {
	parkTutorial(t)
	if _, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
		{StepRecord: "hello", Text: "hello", Until: "next"},
	}}); err != nil {
		t.Errorf("a script with no matching step should not need an axis: %v", err)
	}
}

// The axis vocabulary is closed like the other three, and an invented word does not exist.
func TestAnInventedAxisIsRefused(t *testing.T) {
	parkTutorial(t)
	if _, err := Parse(data.TutorialData{Match: "color", Steps: []data.TutorialStepData{
		{StepRecord: "hello", Text: "hello", Until: "next"},
	}}); err == nil {
		t.Error("an invented axis was accepted")
	}
}

// Card names belong to `named-cards` alone: required there and checked against the deck, refused
// everywhere else.
func TestCardNamesBelongToTheNamedCardsAnchor(t *testing.T) {
	parkTutorial(t)
	cases := []struct {
		why  string
		step data.TutorialStepData
		ok   bool
	}{
		{"a card the deck holds", data.TutorialStepData{Anchor: "named-cards", Cards: []string{"Smash"}}, true},
		{"an element and a card", data.TutorialStepData{Anchor: "named-cards", Cards: []string{"arcane Smash"}}, true},
		{"no card at all", data.TutorialStepData{Anchor: "named-cards"}, false},
		{"a card the deck does not hold", data.TutorialStepData{Anchor: "named-cards", Cards: []string{"Smack"}}, false},
		{"an element the card does not ship in", data.TutorialStepData{Anchor: "named-cards", Cards: []string{"stone Smash"}}, false},
		{"a card on another anchor", data.TutorialStepData{Anchor: "enemy-card", Cards: []string{"Smash"}}, false},
	}
	for _, c := range cases {
		c.step.StepRecord, c.step.Text, c.step.Until = "s", "s", "next"
		_, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{c.step}})
		if (err == nil) != c.ok {
			t.Errorf("%s: accepted %v, want %v (%v)", c.why, err == nil, c.ok, err)
		}
	}
}

// **A step waiting for NEXT holds the round, and nothing else does.** The shield step is the first
// in the lesson that lands inside a playing round, and the pause is the whole of what makes it
// readable — without it the break appears and the creature is already swinging.
//
// **The narrowness is the safety.** A step waiting on an *outcome* must never hold, or it stops the
// thing it is waiting for: `watch` waits for the shield to bite, and a `watch` that froze the round
// would deadlock the lesson on its own condition.
func TestOnlyANextStepHoldsTheRound(t *testing.T) {
	parkTutorial(t)
	for _, step := range Load().Steps {
		run := &Run{script: Script{Steps: []Step{step}}}

		holds := run.HoldsRound()
		if want := step.Until == CondNext; holds != want {
			t.Errorf("step %q waits on %v and reports HoldsRound %v", step.Key, step.Until, holds)
		}

		// The one that would deadlock: a step whose condition can only become true while the round
		// is playing must not be the thing stopping it.
		if holds && (step.Until == CondRoundDone || step.Until == CondShieldBroke) {
			t.Errorf("step %q holds the round and waits for the round to do something", step.Key)
		}
	}
}

// A run that is over holds nothing, or a finished lesson would freeze every round after it.
func TestASpentScriptHoldsNothing(t *testing.T) {
	parkTutorial(t)
	run := &Run{script: Script{Steps: []Step{{Key: "x", Text: "x", Until: CondNext}}}}
	run.Advance(Facts{})
	if run.HoldsRound() {
		t.Error("a script that has run out is still holding the round")
	}
}

// A counted `cards-queued` waits for the count, so a step asking for three named cards does not give
// way on the first.
func TestACountedQueueWaitsForTheCount(t *testing.T) {
	parkTutorial(t)
	run := &Run{script: Script{Steps: []Step{{Key: "x", Text: "x", Until: CondCardsQueued, Count: 3}}}}
	run.Update(Facts{Queued: 2}, false)
	if !run.Active() {
		t.Fatal("two queued cards satisfied a step waiting for three")
	}
	run.Update(Facts{Queued: 3}, false)
	if run.Active() {
		t.Error("three queued cards did not satisfy a step waiting for three")
	}
}

// A mark is `{part:phrase}`, it must name a part its step's anchor has, and the phrase is what the
// step reads as.
func TestMarksNameTheAnchorsParts(t *testing.T) {
	parkTutorial(t)
	parse := func(anchor, text string) error {
		_, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
			{StepRecord: "s", Text: text, Anchor: anchor, Until: "next"},
		}})
		return err
	}
	if err := parse("fight-frame", "{realm:Each realm} and {clock:your time}"); err != nil {
		t.Errorf("marks naming the anchor's parts were refused: %v", err)
	}
	if parse("fight-frame", "{shop:the shop}") == nil {
		t.Error("a mark naming a part the anchor does not have was accepted")
	}
	if parse("enemy-card", "{enemy:it}") == nil {
		t.Error("a mark on an anchor with no parts was accepted")
	}
	if parse("fight-frame", "{realm the realm}") == nil || parse("fight-frame", "{realm:open") == nil {
		t.Error("a malformed mark was accepted")
	}

	step := Step{Text: "{realm:Each realm} has {enemy:enemies}, and {realm:realms} end."}
	if got := step.Plain(); got != "Each realm has enemies, and realms end." {
		t.Errorf("Plain = %q", got)
	}
	if got := step.MarkedParts(); len(got) != 2 || got[0] != "realm" || got[1] != "enemy" {
		t.Errorf("MarkedParts = %v, want [realm enemy]", got)
	}
}

// An Also frame is a control on a reading step, and a mark may name it.
func TestAlsoFramesAControlOnAReadingStep(t *testing.T) {
	parkTutorial(t)
	parse := func(until string, also []string, text string) error {
		_, err := Parse(data.TutorialData{Steps: []data.TutorialStepData{
			{StepRecord: "s", Text: text, Anchor: "enemy-card", Until: until, Also: also},
		}})
		return err
	}
	if err := parse("next", []string{"ledger-button"}, "{ledger-button:the ledger}"); err != nil {
		t.Errorf("an Also control on a reading step, marked by name, was refused: %v", err)
	}
	if parse("duel-pressed", []string{"ledger-button"}, "s") == nil {
		t.Error("an Also frame on a step waiting for a click was accepted")
	}
	if parse("next", []string{"first-card"}, "s") == nil {
		t.Error("a card anchor in Also was accepted; cards are tinted, not framed")
	}
	if parse("next", []string{"ledgr-button"}, "s") == nil {
		t.Error("a misspelled Also was accepted")
	}
}
