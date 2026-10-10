package session

// Cantrips: the run's opinion about the duelist, for one fight.
//
// **A potion changes the duelist for the rest of the run; a cantrip changes them for the fight it is
// cast in**. It is bought sealed, in a bundle of scrolls, carried in the consumables pane beside the
// runes, and cast between the turns of a duel.
//
// **Casting a cantrip puts its cantrip-relic on the fighter**, and that relic is the whole of what
// the cantrip does. It is worn after the run's own relics, weightless — it takes no slot — and
// ephemeral: the screen holds the fight's casts and hands them to EquipWearing, so the run is never
// written to and the next fight's fighter is built without them. Every effect a cantrip can have is
// therefore a relic effect, said in the relic grammar and answered by the relic machinery.
//
// **The one figure that crosses the seam is life**, because the wound a fight leaves is carried into
// the next room — see life.go. A duelist who finishes above the ceiling the run gives them walks out
// at that ceiling: the life a cantrip added is a heal as long as it lasts and never a debt afterwards.
// ShedCantrips is that rule.
//
// **Stacking is allowed, and two casts are two relics.** Ten Mights are ten +10 DMG relics, and two
// Endurances are two 2x life relics compounding left to right, as any two scaling relics do.
//
// **This file is where a record becomes something usable and where a bad one is refused**, the job
// `potion.go` does for its catalog, and here for its reason: a cantrip is carried by a *run*, and the
// rules have never heard of one.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Cantrip is one scroll, resolved, with the relic it casts.
//
// Comparable, so a screen can hold one by value — exactly as Potion and Stone are.
type Cantrip struct {
	Record string
	Name   string

	// Family is the block the record was authored beside, and Art is the scroll's face —
	// **already resolved through data.CantripData.ArtKey**, so a record with no picture of its own
	// carries the catalog's default. Neither is read by anything that resolves a fight.
	Family string
	Art    string

	// Draw is the scroll's art brief, carried for the review sheet; nothing that plays the game reads
	// it.
	Draw string

	// Text is the scroll's authored line, which the tooltip says.
	Text string

	// Relic is the cantrip-relic a cast puts on, registered with the rules under the cantrip's own
	// record key. RelicName, RelicArt, RelicDraw and RelicText are its face, as Name, Art, Draw and
	// Text are the scroll's — RelicArt already resolved through data.CantripRelicData.ArtKey.
	Relic     combat.RelicID
	RelicName string
	RelicArt  string
	RelicDraw string
	RelicText string
}

// Worn is the cantrip-relic as a duelist wears it: **weightless and ephemeral**, always. Those are
// properties of the wearing rather than of the relic, and a cast is the only way one is worn.
func (c Cantrip) Worn() combat.WornRelic {
	return combat.WornRelic{Relic: c.Relic, Weightless: true, Ephemeral: true}
}

// CantripRelics is the cantrip-relics a run of casts puts on, in cast order — what EquipWearing is
// handed.
func CantripRelics(cast []Cantrip) []combat.WornRelic {
	out := make([]combat.WornRelic, 0, len(cast))
	for _, c := range cast {
		out = append(out, c.Worn())
	}
	return out
}

// cantrips is the validated catalog, keyed by record, and cantripOrder its keys sorted — the walk
// anything deciding an outcome must use, since a map's own order is randomized.
//
// **A bad record panics at init**, for the potion catalog's reason: a scroll whose relic the rules
// refuse is a card that takes vitae and does nothing.
var cantrips, cantripOrder = loadCantrips()

// cantripRelics is the catalog by the relic each cantrip casts, for a screen holding a worn relic
// and wanting its face.
var cantripRelics = func() map[combat.RelicID]Cantrip {
	out := make(map[combat.RelicID]Cantrip, len(cantrips))
	for _, c := range cantrips {
		out[c.Relic] = c
	}
	return out
}()

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

// CantripByRelic finds the cantrip whose cantrip-relic this is.
func CantripByRelic(id combat.RelicID) (Cantrip, bool) {
	c, ok := cantripRelics[id]
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

// resolveCantrip turns one record into a Cantrip and registers its relic, or says why it cannot.
func resolveCantrip(rec data.CantripData) (Cantrip, error) {
	rules, err := checkCantripRecord(rec)
	if err != nil {
		return Cantrip{}, err
	}
	id, err := combat.RegisterRelic(rec.CantripRecord, rec.Relic.Name, rules)
	if err != nil {
		return Cantrip{}, err
	}

	return Cantrip{
		Record:    rec.CantripRecord,
		Name:      rec.Name,
		Family:    rec.Family,
		Art:       rec.ArtKey(),
		Draw:      rec.Draw,
		Text:      rec.Text,
		Relic:     id,
		RelicName: rec.Relic.Name,
		RelicArt:  rec.Relic.ArtKey(),
		RelicDraw: rec.Relic.Draw,
		RelicText: rec.Relic.Text,
	}, nil
}

// checkCantripRecord holds one record to everything registration would, and hands back its relic's
// rules.
//
// **A cantrip-relic may only wake at a moment the fight's own duelist answers.** The run's relics
// are also read off the run — the draw's flips, the win's growth and payout, the prizes, an essence
// spent — and a cantrip-relic is worn by the fighter alone, so a rule at one of those moments would
// load cleanly and never fire. It is refused here instead.
func checkCantripRecord(rec data.CantripData) ([]combat.RelicRule, error) {
	if rec.CantripRecord == "" {
		return nil, fmt.Errorf("a record with no CantripRecord")
	}
	if rec.Relic.Name == "" {
		return nil, fmt.Errorf("%s casts a relic with no Name", rec.CantripRecord)
	}
	rules, err := parseRelicRules(rec.CantripRecord, rec.Relic.Rules)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		if !fightMoment(rule.When) {
			return nil, fmt.Errorf("%s casts a relic that wakes at %s, which a cantrip-relic is never "+
				"worn for", rec.CantripRecord, rule.When)
		}
	}
	if err := combat.CheckRelic(rec.CantripRecord, rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// fightMoment is whether a moment is answered by the fight's duelist, and so reaches a cantrip-relic.
func fightMoment(m combat.Moment) bool {
	switch m {
	case combat.MomentCardDrawn, combat.MomentFightWon, combat.MomentPrizesDealt,
		combat.MomentEssenceSpent, combat.MomentFightBegun:
		return false
	}
	return true
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
	return s.scrolls.list()
}

// ScrollCount is how many cantrips the run is carrying.
func (s *Session) ScrollCount() int { return s.scrolls.len() }

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
	s.scrolls.add(key, false)
	return true
}

// DropScroll takes one out of the scroll case by position, and reports whether it was there.
//
// **By position rather than by key**, because the case may hold two of the same cantrip and
// casting one must not be ambiguous about which.
func (s *Session) DropScroll(i int) bool {
	return s.scrolls.drop(i)
}
