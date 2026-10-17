package session

import "testing"

// **A heal moves the wound, and the wound is what a screen has to draw.** The bug this exists for
// shipped: the build band wrote `LifeLeft` — the figure the last fight ended on — so a Salve bought
// in the shop changed nothing anybody could see, while the two potions that move the *ceiling* were
// visible immediately. The two numbers agree the moment a fight is won and only ever come apart
// between fights, which is exactly where a potion is drunk.
func TestASalveMovesTheWoundAndNotTheFightsLastFigure(t *testing.T) {
	s := New(testDeck())
	s.AddVitae(50)

	const maxLife = 100
	s.WonFight(maxLife-30, maxLife)
	won := maxLife - 30 + PercentOfLife(maxLife, winHealPercent)

	if got := s.LifeAtFightStart(maxLife); got != won {
		t.Fatalf("a win on %d leaves %d, wanted %d", maxLife-30, got, won)
	}

	salve, ok := PotionByKey("heal-life")
	if !ok {
		t.Fatal("no salve in the catalog")
	}
	before := s.LifeLeft()
	if !s.Drink(salve.Record, maxLife) {
		t.Fatal("the run could not drink a salve it could afford")
	}

	if got, want := s.LifeAtFightStart(maxLife), won+PercentOfLife(maxLife, salve.Amount); got != want {
		t.Errorf("life reads %d after a salve of %d%%, wanted %d", got, salve.Amount, want)
	}
	if s.LifeLeft() != before {
		t.Errorf("the salve moved LifeLeft to %d; it is what the last fight ended on", s.LifeLeft())
	}
}

// A heal never takes the wound past healthy, and the two boosts land on the duelist's own figures.
func TestTheThreePotionsMoveWhatTheySay(t *testing.T) {
	s := New(testDeck())
	s.AddVitae(100)

	const maxLife = 100
	s.WonFight(maxLife-2, maxLife)

	salve, _ := PotionByKey("heal-life")
	if !s.Drink(salve.Record, maxLife) {
		t.Fatal("could not drink a salve")
	}
	if s.Hurt() != 0 {
		t.Errorf("a %d heal on a wound of 2 left %d; a heal cannot overshoot", salve.Amount, s.Hurt())
	}

	draught, _ := PotionByKey("dmg-buff")
	elixir, _ := PotionByKey("hp-buff")
	if !s.Drink(draught.Record, maxLife) || !s.Drink(elixir.Record, maxLife) {
		t.Fatal("could not drink the two boosts")
	}
	if s.DMGBonus() != draught.Amount {
		t.Errorf("DMG bonus reads %d, wanted %d", s.DMGBonus(), draught.Amount)
	}
	if s.LifeBonus() != elixir.Amount {
		t.Errorf("life bonus reads %d, wanted %d", s.LifeBonus(), elixir.Amount)
	}
}

// **Every win gives back a share of max life**, off the wound the fight left, after the payout read
// the life the fight ended on — and never past healthy.
func TestAWinHealsAShareOfMaxLife(t *testing.T) {
	s := New(testDeck())
	const maxLife = 200
	s.WonFight(maxLife-50, maxLife)
	if got, want := s.Hurt(), 50-PercentOfLife(maxLife, winHealPercent); got != want {
		t.Errorf("a win down 50 of %d left a wound of %d, want %d", maxLife, got, want)
	}
	s.WonFight(maxLife-1, maxLife)
	if s.Hurt() != 0 {
		t.Errorf("a win down 1 left a wound of %d; the heal cannot overshoot", s.Hurt())
	}
}

// A salve is a share of the ceiling it is drunk under, rounded up.
func TestASalveHealsAShareOfMaxLife(t *testing.T) {
	if got := PercentOfLife(130, 25); got != 33 {
		t.Errorf("25%% of 130 is %d, want 33, rounded up", got)
	}
	if got := PercentOfLife(10, 5); got != 1 {
		t.Errorf("5%% of 10 is %d, want 1: any share is worth at least one life", got)
	}
}
