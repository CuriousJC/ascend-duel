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

	if got := s.LifeAtFightStart(maxLife); got != maxLife-30 {
		t.Fatalf("a win on %d leaves %d, wanted %d", maxLife-30, got, maxLife-30)
	}

	salve, ok := PotionByKey("salve")
	if !ok {
		t.Fatal("no salve in the catalogue")
	}
	before := s.LifeLeft()
	if !s.Drink(salve.Record) {
		t.Fatal("the run could not drink a salve it could afford")
	}

	if got, want := s.LifeAtFightStart(maxLife), maxLife-30+salve.Amount; got != want {
		t.Errorf("life reads %d after a salve of %d, wanted %d", got, salve.Amount, want)
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

	salve, _ := PotionByKey("salve")
	if !s.Drink(salve.Record) {
		t.Fatal("could not drink a salve")
	}
	if s.Hurt() != 0 {
		t.Errorf("a %d heal on a wound of 2 left %d; a heal cannot overshoot", salve.Amount, s.Hurt())
	}

	draught, _ := PotionByKey("draught")
	tonic, _ := PotionByKey("tonic")
	if !s.Drink(draught.Record) || !s.Drink(tonic.Record) {
		t.Fatal("could not drink the two boosts")
	}
	if s.DMGBonus() != draught.Amount {
		t.Errorf("DMG bonus reads %d, wanted %d", s.DMGBonus(), draught.Amount)
	}
	if s.LifeBonus() != tonic.Amount {
		t.Errorf("life bonus reads %d, wanted %d", s.LifeBonus(), tonic.Amount)
	}
}
