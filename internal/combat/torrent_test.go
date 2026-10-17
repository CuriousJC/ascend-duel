package combat

import "testing"

func torrent(t *testing.T, key string, axis Axis) RelicID {
	return relic(t, key,
		RelicRule{When: MomentEquipped, Then: []RelicEffect{{Do: DoAddDMG, Amount: 0}}},
		RelicRule{When: MomentTurnTaken, If: RelicCondition{Favored: axis, HasFavored: true},
			Then: []RelicEffect{{Do: DoGrowOnStreak, Amount: 1}}})
}

// TestATorrentGrowsOnPureTurnsAndBreaksOnAnotherAttack. Fire is the favorite: a turn of fire
// attacks and an ice shield steps it, a turn of shields leaves it, and an ice attack puts it back to
// zero.
func TestATorrentGrowsOnPureTurnsAndBreaksOnAnotherAttack(t *testing.T) {
	id := torrent(t, "torrent-element", AxisElement)
	d := duelist(10, 6, 100).Wearing(WornRelic{Relic: id})
	d.FavoredElement = Fire

	d = d.TurnTaken([]Card{Of(Bash, Fire), Of(Jab, Fire), Of(Brace, Ice)})
	d = d.TurnTaken([]Card{Of(Bash, Fire)})
	if got := d.Relics[0].Grown; got != 2 {
		t.Fatalf("two pure fire turns grew it to %d, want 2", got)
	}
	d = d.TurnTaken([]Card{Of(Brace, Ice)})
	if got := d.Relics[0].Grown; got != 2 {
		t.Errorf("a turn of shields moved it to %d, want it left at 2", got)
	}
	d = d.TurnTaken([]Card{Of(Bash, Fire), Of(Jab, Ice)})
	if got := d.Relics[0].Grown; got != 0 {
		t.Errorf("an ice attack left it at %d, want 0", got)
	}
	if !Resets(id) || !Grows(id) {
		t.Error("a Torrent must read as a growing relic that resets")
	}
}

// TestATorrentWithNoFavoriteSleeps. Nothing grows and nothing resets.
func TestATorrentWithNoFavoriteSleeps(t *testing.T) {
	d := duelist(10, 6, 100).Wearing(WornRelic{Relic: torrent(t, "torrent-form", AxisForm), Grown: 3})
	d = d.TurnTaken([]Card{Of(Bash, Fire), Of(Jab, Ice)})
	if got := d.Relics[0].Grown; got != 3 {
		t.Errorf("a duelist with no favorite form moved it to %d, want it left at 3", got)
	}
}

// TestFavoredBelongsToAStreak. It is read only by grow-on-streak, and a streak needs something to
// keep to.
func TestFavoredBelongsToAStreak(t *testing.T) {
	for name, rule := range map[string]RelicRule{
		"favored-on-grow": {When: MomentTurnTaken, If: RelicCondition{Favored: AxisElement, HasFavored: true},
			Then: []RelicEffect{{Do: DoGrowOnTurn, Amount: 1}}},
		"streak-bare": {When: MomentTurnTaken, Then: []RelicEffect{{Do: DoGrowOnStreak, Amount: 1}}},
		"favored-concept": {When: MomentTurnTaken, If: RelicCondition{Favored: AxisConcept, HasFavored: true},
			Then: []RelicEffect{{Do: DoGrowOnStreak, Amount: 1}}},
	} {
		if _, err := RegisterRelic("relictest."+name, name, []RelicRule{rule}); err == nil {
			t.Errorf("%s registered", name)
		}
	}
}
