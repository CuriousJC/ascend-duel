package combat

import (
	"slices"
	"testing"
)

// formed is the key of the rung a turn is paid as, and whether the synergy rung of that size is
// among what it satisfied.
func formed(t *testing.T, cards []Card, synergyKey string) (string, bool) {
	t.Helper()
	b := BlowFor(slots(cards))
	id, ok := HandIDForKey(synergyKey)
	if !ok {
		t.Fatalf("the catalog has no %s", synergyKey)
	}
	return b.Hand.Key, slices.Contains(b.Satisfied, id)
}

// **A synergy hand is the same card in the same color**, and nothing less: three fire Jabs are
// one, and two fire Jabs beside an ice Jab are a Card Three of a Kind.
func TestASynergyHandIsTheSameCardInTheSameColor(t *testing.T) {
	if got, _ := formed(t, []Card{Of(Jab, Fire), Of(Jab, Fire), Of(Jab, Fire)}, "synergy-three-of-a-kind"); got != "synergy-three-of-a-kind" {
		t.Errorf("three fire Jabs were paid as %s", got)
	}
	if got, sat := formed(t, []Card{Of(Jab, Fire), Of(Jab, Fire), Of(Jab, Ice)}, "synergy-three-of-a-kind"); sat {
		t.Errorf("two fire Jabs and an ice Jab satisfied the synergy rung, paid as %s", got)
	}
	if got, sat := formed(t, []Card{Of(Jab, Fire), Of(Jab, Fire), Of(Cut, Fire)}, "synergy-three-of-a-kind"); sat {
		t.Errorf("two fire Jabs and a fire Cut satisfied the synergy rung, paid as %s", got)
	}
}

// **The four and the five are read the same way**, and the bigger one wins.
func TestASynergyFiveBeatsEverythingInsideIt(t *testing.T) {
	five := []Card{Of(Jab, Ice), Of(Jab, Ice), Of(Jab, Ice), Of(Jab, Ice), Of(Jab, Ice)}
	if got, _ := formed(t, five, "synergy-five-of-a-kind"); got != "synergy-five-of-a-kind" {
		t.Errorf("five ice Jabs were paid as %s", got)
	}
	four := append(five[:4:4], Of(Cut, Ice))
	if got, _ := formed(t, four, "synergy-four-of-a-kind"); got != "synergy-four-of-a-kind" {
		t.Errorf("four ice Jabs and an ice Cut were paid as %s", got)
	}
}

// **A wildcard joins its own card in any color, and no other card.** An every-element ice Jab
// beside two fire Jabs is a fire Jab; beside two fire Cuts it is nothing to them on this axis.
func TestAWildcardJoinsOnlyItsOwnCardOnSynergy(t *testing.T) {
	if got, _ := formed(t, []Card{Of(Jab, Fire), Of(Jab, Fire), wild(Jab, Ice)}, "synergy-three-of-a-kind"); got != "synergy-three-of-a-kind" {
		t.Errorf("two fire Jabs and a wild Jab were paid as %s", got)
	}
	if got, sat := formed(t, []Card{Of(Cut, Fire), Of(Cut, Fire), wild(Jab, Ice)}, "synergy-three-of-a-kind"); sat {
		t.Errorf("two fire Cuts and a wild Jab satisfied the synergy rung, paid as %s", got)
	}
}

// **A turn of nothing but wildcards groups by card.** Three every-element Jabs agree on
// everything; an every-element Jab, Cut and Thump agree on color and on nothing a synergy counts.
func TestWildcardsAloneGroupByCardOnSynergy(t *testing.T) {
	if got, _ := formed(t, []Card{wild(Jab, Ice), wild(Jab, Fire), wild(Jab, Earth)}, "synergy-three-of-a-kind"); got != "synergy-three-of-a-kind" {
		t.Errorf("three wild Jabs were paid as %s", got)
	}
	if got, sat := formed(t, []Card{wild(Jab, Ice), wild(Cut, Fire), wild(Thump, Earth)}, "synergy-three-of-a-kind"); sat {
		t.Errorf("a wild Jab, Cut and Thump satisfied the synergy rung, paid as %s", got)
	}
}
