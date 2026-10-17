package session

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// aRound is a two-card turn and the log a resolver might hand back for it: both cards land, one is
// a pair, and the creature lands one blow in return.
func aRound(t *testing.T) ([]combat.Event, []combat.Card) {
	t.Helper()
	pair, ok := combat.HandIDForKey("pair")
	if !ok {
		t.Fatal("the ladder has no pair")
	}
	turn := []combat.Card{
		{Concept: combat.Bash, Element: combat.Fire},
		{Concept: combat.Bash, Element: combat.Ice},
	}
	log := []combat.Event{
		{Kind: combat.KindHand, Side: combat.SideA, Hand: pair},
		{Kind: combat.KindDamage, Side: combat.SideA, Target: combat.SideB, Slot: 0, Amount: 7},
		{Kind: combat.KindDamage, Side: combat.SideA, Target: combat.SideB, Slot: 1, Amount: 12},
		{Kind: combat.KindDamage, Side: combat.SideB, Target: combat.SideA, Amount: 5},
	}
	return log, turn
}

// TestARoundIsCountedWhenItsDuelSettles is the whole arithmetic: what was dealt, by element and by
// form, what was taken, the hand, and the records â€” and none of it before the duel settles.
func TestARoundIsCountedWhenItsDuelSettles(t *testing.T) {
	run := bare(t)
	log, turn := aRound(t)
	run.TallyRound(log, turn)

	if got := run.Stats().Total(StatDealt); got != 0 {
		t.Fatalf("a duel still being fought is not in the run's figures yet, got %d dealt", got)
	}

	run.TallyFight("goblins-outer-serf", "fire", true, 1)
	st := run.Stats()
	for name, want := range map[string]int{
		StatDealt:                 19,
		StatTaken:                 5,
		StatHits:                  2,
		StatRounds:                1,
		StatPlayed:                2,
		StatFightsWon:             1,
		StatDealtElement + "fire": 7,
		StatDealtElement + "ice":  12,
		StatDealtForm + "crush":   19,
		StatPlayedForm + "crush":  2,
		StatHandFormed + "pair":   1,
		StatCreatureDefeated + "goblins-outer-serf":                      1,
		CreatureAs(StatCreatureDefeatedAs, "goblins-outer-serf", "fire"): 1,
		CreatureAs(StatCreatureDefeatedAs, "goblins-outer-serf", "ice"):  0,
	} {
		if got := st.Total(name); got != want {
			t.Errorf("%s: got %d want %d", name, got, want)
		}
	}
	if got := st.MostOf(StatBestHit); got != 12 {
		t.Errorf("the biggest hit is 12, got %d", got)
	}
	if got := st.MostOf(StatBestFight); got != 19 {
		t.Errorf("the biggest fight is 19, got %d", got)
	}
	if got, ok := st.LeastOf(StatFastestWin); !ok || got != 1 {
		t.Errorf("the fastest win is one round, got %d (%v)", got, ok)
	}
}

// TestALostDuelNamesWhatKilledThePlayer holds the other branch.
func TestALostDuelNamesWhatKilledThePlayer(t *testing.T) {
	run := bare(t)
	run.TallyFight("goblins-outer-serf", "ice", false, 4)

	st := run.Stats()
	if st.Total(StatFightsLost) != 1 || st.Total(StatCreatureKiller+"goblins-outer-serf") != 1 ||
		st.Total(CreatureAs(StatCreatureKillerAs, "goblins-outer-serf", "ice")) != 1 {
		t.Errorf("a lost duel is a loss against its creature: %+v", st.Totals)
	}
	if _, ok := st.LeastOf(StatFastestWin); ok {
		t.Error("a loss is not a win in any number of rounds")
	}
}

// TestAResumeDoesNotCountADuelTwice is why a duel's rounds are held apart: a run saved mid-duel â€”
// which spending a consumable does â€” and resumed is fought again from the top of the room.
func TestAResumeDoesNotCountADuelTwice(t *testing.T) {
	motifs, shape := rosters(t)
	seed, err := seeds.Parse(theSeed)
	if err != nil {
		t.Fatal(err)
	}
	s := Start(motifs, shape, seed)
	log, turn := aRound(t)

	// One duel settled, then a round of the next before a save.
	s.TallyRound(log, turn)
	s.TallyFight("goblins-outer-serf", "fire", true, 1)
	s.TallyRound(log, turn)

	back, _, err := Resume(motifs, shape, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	if got := back.Stats().Total(StatDealt); got != 19 {
		t.Errorf("the resumed run carries the settled duel and nothing of the unsettled one: got %d dealt", got)
	}
}

// TestPurchasesAndUsesAreCountedWhereTheyHappen holds that the session methods doing the buying are
// the ones counting it, and that a refused purchase counts nothing.
func TestPurchasesAndUsesAreCountedWhereTheyHappen(t *testing.T) {
	run := rich(t)
	if !run.Buy("dmg-all-slash") {
		t.Fatal("the purchase was refused")
	}
	if got := run.Stats().Total(StatRelicBought + "dmg-all-slash"); got != 1 {
		t.Errorf("a relic bought is counted once, got %d", got)
	}
	if got := run.Stats().Total(StatVitaeSpent); got != priceOf(t, "dmg-all-slash") {
		t.Errorf("what it cost is counted as spent, got %d", got)
	}

	poor := bare(t)
	poor.vitae = 0
	poor.Buy("dmg-all-slash")
	if !poor.Stats().Empty() {
		t.Errorf("a refused purchase counts nothing, got %+v", poor.Stats())
	}
}

// TestTheFinalTallyCarriesHowTheRunEnded is what the profile is handed.
func TestTheFinalTallyCarriesHowTheRunEnded(t *testing.T) {
	run := bare(t)
	st := run.FinalStats(RunSummary{Ended: EndedInDefeat, Realm: 3, Dealt: 400, Defeated: 7})
	if st.Total(StatRunsDefeat) != 1 || st.Total(StatRunsAbandoned) != 0 {
		t.Errorf("a defeat is one run lost: %+v", st.Totals)
	}
	if st.MostOf(StatBestRealm) != 3 || st.MostOf(StatBestRunWins) != 7 || st.MostOf(StatBestRunDealt) != 400 {
		t.Errorf("the run's records came out %+v", st.Most)
	}
}

// lastBoss is the fight index of the journey's last boss, for a run started on the test seed.
func lastBoss(t *testing.T) (*Session, int) {
	t.Helper()
	motifs, shape := rosters(t)
	seed, err := seeds.Parse(theSeed)
	if err != nil {
		t.Fatal(err)
	}
	s := Start(motifs, shape, seed)
	return s, shape.Realms*3 - 1
}

// TestBeatingTheLastBossIsAVictoryOnce is the whole rule: the victory is counted when the boss
// falls, the station is stood at once, and the fight after it walks straight past.
func TestBeatingTheLastBossIsAVictoryOnce(t *testing.T) {
	s, boss := lastBoss(t)
	s.fight = boss
	s.WonFight(10, 10)
	if !s.Victorious() || s.Stats().Total(StatVictories) != 1 {
		t.Fatalf("beating the last boss wins the journey: victorious %v, counted %d",
			s.Victorious(), s.Stats().Total(StatVictories))
	}

	s.Advance()
	if s.Phase() != PhaseVictory {
		t.Fatalf("the fight that won the journey goes to the victory, got %v", s.Phase())
	}
	s.Advance()
	if s.Phase() != PhaseReward {
		t.Fatalf("walking on goes to the boss's reward, got %v", s.Phase())
	}

	s.SetPhase(PhaseFight)
	s.WonFight(10, 10)
	s.Advance()
	if s.Phase() == PhaseVictory {
		t.Error("the victory is stood at once; the next fight goes straight to its reward")
	}
	if got := s.Stats().Total(StatVictories); got != 1 {
		t.Errorf("a journey is won once, got %d", got)
	}
}

// TestAnEarlierBossIsNoVictory holds that only the last realm's boss wins the journey.
func TestAnEarlierBossIsNoVictory(t *testing.T) {
	s, boss := lastBoss(t)
	s.fight = boss - 3
	s.WonFight(10, 10)
	s.Advance()
	if s.Victorious() || s.Phase() == PhaseVictory {
		t.Error("the second-to-last realm's boss does not win the journey")
	}
}

// TestTheVictoryLeavesTheBossRewardToBePaid holds that standing at the victory claims nothing, so
// the post-battle screen still has the win's payout to read out.
func TestTheVictoryLeavesTheBossRewardToBePaid(t *testing.T) {
	s, boss := lastBoss(t)
	s.fight = boss
	s.WonFight(10, 10)
	owed := s.spoils
	s.Advance()
	s.Advance()
	if s.spoils != owed {
		t.Errorf("the victory station paid out the reward early: %+v, was %+v", s.spoils, owed)
	}
}

// TestAVictorySurvivesAResume holds the flag and the summary it reaches.
func TestAVictorySurvivesAResume(t *testing.T) {
	motifs, shape := rosters(t)
	s, boss := lastBoss(t)
	seed, _ := seeds.Parse(theSeed)
	s.fight = boss
	s.WonFight(10, 10)

	back, _, err := Resume(motifs, shape, s.Snapshot(seed))
	if err != nil {
		t.Fatal(err)
	}
	if !back.Victorious() || !back.Summarize(seed, EndedInDefeat).Victorious {
		t.Error("a won journey is still won after a resume, and the summary says so")
	}
}
