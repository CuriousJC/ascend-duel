package session

// Cantrips: the run's opinion about the duelist, for one fight.
//
// **A potion changes the duelist for the rest of the run; a cantrip changes them for the fight it is
// cast in**. It is bought sealed, in a bundle of scrolls, carried in the
// consumables pane beside the runes, and cast between the turns of a duel onto the fighter standing
// in the room. Nothing it does is written to the run: the fighter is rebuilt from the run at the top
// of every fight, so what a cantrip added is gone by the next one without anything having to undo it.
//
// **The one figure that crosses the seam is life**, because the wound a fight leaves is carried into
// the next room — see life.go. A duelist who finishes above the ceiling the run gives them walks out
// at that ceiling: the life a cantrip added is a heal as long as it lasts and never a debt afterwards.
// ShedCantrips is that rule.
//
// **Stacking is allowed and each cast is contained by itself.** A cast reads the fighter as it
// stands and moves it; it knows nothing about any other cantrip. Ten Mights are +100 DMG, and two
// Endurances double a doubled body.
//
// **This file is where a record becomes something usable and where a bad one is refused**, the job
// `potion.go` does for its catalog, and here for its reason: a cantrip is carried by a *run*, and the
// rules have never heard of one.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// CantripEffect is what a cantrip does to the fighter. **A closed vocabulary**, in Go rather than in
// the file: a new effect is a rule the fighter has to learn how to take, never something
// `cantrips.json` can assert into existence. Same posture as a potion's effect.
type CantripEffect int

const (
	// CantripAddDMG adds its amount to the fighter's DMG.
	CantripAddDMG CantripEffect = iota

	// CantripScaleLife scales the fighter's life ceiling and the life under it by its amount, as a
	// percentage — so 200 turns 30 of 60 into 60 of 120.
	CantripScaleLife
)

// String is the effect as the file spells it.
func (e CantripEffect) String() string {
	switch e {
	case CantripScaleLife:
		return "scale-life"
	default:
		return "add-dmg"
	}
}

// Cantrip is one scroll, resolved against the vocabulary above.
//
// Comparable, so a screen can hold one by value — exactly as Potion and Stone are.
type Cantrip struct {
	Record string
	Name   string

	// Family is the block the record was authored beside, and Art is the face the card draws —
	// **already resolved through data.CantripData.ArtKey**, so a record with no picture of its own
	// carries the catalog's default. Neither is read by anything that resolves a fight.
	Family string
	Art    string

	// Draw is the art brief, carried for the review sheet; nothing that plays the game reads it.
	Draw string

	Effect CantripEffect
	Amount int

	// Text is the record's authored line, which the tooltip says.
	Text string
}

// cantrips is the validated catalog, keyed by record, and cantripOrder its keys sorted — the walk
// anything deciding an outcome must use, since a map's own order is randomized.
//
// **A bad record panics at init**, for the potion catalog's reason: a scroll naming an effect this
// build has not got is a card that takes vitae and does nothing.
var cantrips, cantripOrder = loadCantrips()

// Cantrips is every cantrip in the catalog, in a fixed sorted order.
func Cantrips() []Cantrip {
	out := make([]Cantrip, 0, len(cantripOrder))
	for _, key := range cantripOrder {
		out = append(out, cantrips[key])
	}
	return out
}

// CantripByKey finds one by its record key.
func CantripByKey(key string) (Cantrip, bool) {
	c, ok := cantrips[key]
	return c, ok
}

func loadCantrips() (map[string]Cantrip, []string) {
	recs := data.LoadCantrips()
	if len(recs) == 0 {
		panic("cantrips.json: the catalog is empty, and a bundle of scrolls has nothing to hold")
	}

	order := data.CantripOrder(recs)
	out := make(map[string]Cantrip, len(recs))
	for _, key := range order {
		c, err := resolveCantrip(recs[key])
		if err != nil {
			panic("cantrips.json: " + err.Error())
		}
		out[key] = c
	}
	return out, order
}

// resolveCantrip turns one record into a Cantrip, or says why it cannot.
//
// **Each effect refuses the amount that would do nothing**: a Might adding no DMG, or an Endurance
// scaling life by 100% or less. The second is also the drawback that would make a "cantrip" a curse,
// and a curse is a design decision rather than a record.
func resolveCantrip(rec data.CantripData) (Cantrip, error) {
	if rec.CantripRecord == "" {
		return Cantrip{}, fmt.Errorf("a record with no CantripRecord")
	}

	effect, err := ParseCantripEffect(rec.Effect)
	if err != nil {
		return Cantrip{}, fmt.Errorf("%s: %w", rec.CantripRecord, err)
	}
	switch effect {
	case CantripAddDMG:
		if rec.Amount <= 0 {
			return Cantrip{}, fmt.Errorf("%s: adding %d DMG does nothing", rec.CantripRecord, rec.Amount)
		}
	case CantripScaleLife:
		if rec.Amount <= 100 {
			return Cantrip{}, fmt.Errorf("%s: scaling life to %d%% is not a strengthening",
				rec.CantripRecord, rec.Amount)
		}
	}

	return Cantrip{
		Record: rec.CantripRecord,
		Name:   rec.Name,
		Family: rec.Family,
		Art:    rec.ArtKey(),
		Draw:   rec.Draw,
		Effect: effect,
		Amount: rec.Amount,
		Text:   rec.Text,
	}, nil
}

// ParseCantripEffect resolves the file's spelling of an effect.
func ParseCantripEffect(name string) (CantripEffect, error) {
	switch name {
	case "add-dmg":
		return CantripAddDMG, nil
	case "scale-life":
		return CantripScaleLife, nil
	default:
		return 0, fmt.Errorf("%q is not a cantrip effect the rules have", name)
	}
}

// Cast is the cantrip taken by a fighter: the duelist as it stands, moved.
//
// **It reads nothing but the duelist it is handed**, which is what makes each cast contained by
// itself — a second Endurance doubles whatever the first left, and a Might adds to whatever DMG is
// standing, relics and earlier Mights included.
//
// **Life never scales below one**, the rule Equip keeps for a stack of drawbacks: a fighter a cast
// had killed would be a death nobody's blow caused.
func (c Cantrip) Cast(d combat.Duelist) combat.Duelist {
	switch c.Effect {
	case CantripAddDMG:
		d.DMG += c.Amount
	case CantripScaleLife:
		d.MaxLife = d.MaxLife * c.Amount / 100
		d.CurrentLife = d.CurrentLife * c.Amount / 100
		if d.MaxLife < 1 {
			d.MaxLife = 1
		}
		if d.CurrentLife < 1 {
			d.CurrentLife = 1
		}
		if d.CurrentLife > d.MaxLife {
			d.CurrentLife = d.MaxLife
		}
	}
	return d
}

// ShedCantrips is the fighter's life as the run takes it back at the end of a fight: the ceiling
// without what the fight's cantrips added, and the life under it clamped to that ceiling.
//
// `added` is how much ceiling the casts raised over the fight, which the screen that cast them
// tallies. **Above the ceiling is simply the ceiling** *(owner's call, 2026-09-28)* — whatever the
// cantrip bought is a heal, never a wound taken back.
func ShedCantrips(life, maxLife, added int) (lifeLeft, ceiling int) {
	ceiling = maxLife - added
	if ceiling < 1 {
		ceiling = 1
	}
	if life > ceiling {
		life = ceiling
	}
	return life, ceiling
}

// The scroll case: cantrips the run is carrying, uncast.
//
// **The sack's shape exactly**, for the same argument: two of the same cantrip are two cards to draw
// and two separate decisions to make, so it is a list of record keys in acquisition order rather
// than a count per key.

// StartingCantrips is what a run opens carrying in its scroll case, by record key.
//
// **Empty as shipped, and it is a debug seat** — the counterpart of StartingRunes, written only by
// `internal/scenario`, which is compiled out of every normal build.
var StartingCantrips []string

// Scrolls is every cantrip the run is carrying, by record key, in the order they were acquired.
func (s *Session) Scrolls() []string {
	out := make([]string, len(s.scrolls))
	copy(out, s.scrolls)
	return out
}

// ScrollCount is how many cantrips the run is carrying.
func (s *Session) ScrollCount() int { return len(s.scrolls) }

// HoldCantrip puts a cantrip in the scroll case, and reports whether it went in.
//
// **A full pane refuses**, on `Hold`'s terms: the cap is on the consumables pane as a whole, and the
// shop is where it is actually stopped — a bundle of scrolls goes dim rather than taking vitae for a
// scroll there is no room for. A cantrip the catalog does not have is refused too.
func (s *Session) HoldCantrip(key string) bool {
	if s.ConsumablesFull() {
		return false
	}
	return s.holdCantrip(key)
}

// holdCantrip is HoldCantrip without the cap: a fixture planting a case, or a save being resumed.
func (s *Session) holdCantrip(key string) bool {
	if _, ok := cantrips[key]; !ok {
		return false
	}
	s.scrolls = append(s.scrolls, key)
	return true
}

// DropScroll takes one out of the scroll case by position, and reports whether it was there.
//
// **By position rather than by key**, because the case may hold two of the same cantrip and
// casting one must not be ambiguous about which.
func (s *Session) DropScroll(i int) bool {
	if i < 0 || i >= len(s.scrolls) {
		return false
	}
	s.scrolls = append(s.scrolls[:i], s.scrolls[i+1:]...)
	return true
}
