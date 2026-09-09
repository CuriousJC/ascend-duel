package session

// Potions: the run's own opinion about the duelist's body.
//
// **A ring is worn, a stone raises a rung, a worm eats a card — and a potion changes the duelist**
// *(owner's call, 2026-09-06)*. It is the only thing in the shop that moves one of the three
// figures the fighter is actually made of: the life it has left, the ceiling that life sits under,
// or the damage it hits for.
//
// **It is drunk on the spot.** There is no carrying and no cap, which is what keeps a potion out of
// the consumables pane the parasites live in and out of `MaxHeld`: a click pays for it and applies
// it in the same frame, so a purchase interrupted by a quit cannot leave a bottle in a bucket
// nobody can open.
//
// **Two of the three change the record and one changes the wound**, and that is the whole of the
// mechanic. `+DMG` and `+max life` are stored on the run and added to the duelist's own figures in
// `Equip`, *before* the boss bonus and before any ring — a potion is growth of the body, so a
// percentage ring scales the boosted duelist, exactly as it scales one that has climbed a stairway.
// Healing is not stored at all: it takes the wound down, which is the one number `life.go` keeps.
//
// **This file is where a record becomes something usable and where a bad one is refused**, the job
// `stone.go` and `worm.go` do for their catalogues. It lives here rather than in `internal/combat`
// for their reason too: a potion is bought by a *run*, and the rules have never heard of a shop.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// PotionEffect is which of the duelist's three figures a potion moves. **A closed vocabulary**, in
// Go rather than in the file: a fourth effect is a rule the run has to learn how to apply, never
// something `potions.json` can assert into existence. Same posture as a worm's target.
type PotionEffect int

const (
	// PotionHeal takes the run's wound down by its amount. It is the only one of the three that
	// does nothing to a duelist who is not hurt.
	PotionHeal PotionEffect = iota

	// PotionDMG adds to the damage the duelist deals, for the rest of the run.
	PotionDMG

	// PotionLife raises the ceiling the duelist's life sits under, for the rest of the run.
	PotionLife
)

// Potion is one bottle, resolved against the vocabulary above.
//
// Comparable, so a screen can hold one by value and compare two without reaching for the key —
// exactly as `Stone` and `Worm` are.
type Potion struct {
	Record string
	Name   string
	Text   string
	Effect PotionEffect
	Amount int
	Price  int
}

// potions is the validated catalogue, in file order, built once at package init.
//
// **A bad record panics at init.** A potion naming an effect this build has not got is a card that
// takes vitae and does nothing, and nothing else in the game would ever notice — the same failure
// an unearnable achievement is, and it takes the same exit.
var potions = loadPotions()

// Potions is the whole catalogue, in the order the file writes it, which is the order the shelf
// stands in. **Nothing is drawn and nothing is weighted**: every visit offers all three, so there
// is no roll here and no stream to own.
func Potions() []Potion {
	out := make([]Potion, len(potions))
	copy(out, potions)
	return out
}

// PotionByKey finds one by its record key.
func PotionByKey(key string) (Potion, bool) {
	for _, p := range potions {
		if p.Record == key {
			return p, true
		}
	}
	return Potion{}, false
}

func loadPotions() []Potion {
	recs := data.LoadPotions()
	if len(recs) == 0 {
		panic("potions.json: the catalogue is empty, and the shelf has a pane to fill")
	}

	seen := map[string]bool{}
	out := make([]Potion, 0, len(recs))
	for _, rec := range recs {
		p, err := resolvePotion(rec)
		if err != nil {
			panic("potions.json: " + err.Error())
		}
		if seen[p.Record] {
			panic("potions.json: two records keyed " + p.Record)
		}
		seen[p.Record] = true
		out = append(out, p)
	}
	return out
}

// resolvePotion turns one record into a Potion, or says why it cannot.
//
// **An amount of zero is refused whatever the effect**, because all three of them are a figure: a
// potion that heals nothing, adds no damage or raises no ceiling is a card the player pays for and
// cannot tell apart from a bug.
func resolvePotion(rec data.PotionData) (Potion, error) {
	if rec.PotionRecord == "" {
		return Potion{}, fmt.Errorf("a record with no PotionRecord")
	}

	effect, err := parsePotionEffect(rec.Effect)
	if err != nil {
		return Potion{}, fmt.Errorf("%s: %w", rec.PotionRecord, err)
	}
	if rec.Amount <= 0 {
		return Potion{}, fmt.Errorf("%s: an amount of %d does nothing", rec.PotionRecord, rec.Amount)
	}
	if rec.Price <= 0 {
		return Potion{}, fmt.Errorf("%s: a price of %d is not something the shop can charge",
			rec.PotionRecord, rec.Price)
	}

	return Potion{
		Record: rec.PotionRecord,
		Name:   rec.Name,
		Text:   rec.Text,
		Effect: effect,
		Amount: rec.Amount,
		Price:  rec.Price,
	}, nil
}

func parsePotionEffect(name string) (PotionEffect, error) {
	switch name {
	case "heal":
		return PotionHeal, nil
	case "dmg":
		return PotionDMG, nil
	case "life":
		return PotionLife, nil
	default:
		return 0, fmt.Errorf("%q is not an effect the rules have", name)
	}
}

// The run's half: what a potion costs, whether it can be had, and what drinking one does.

// CanDrink is whether the purse covers a potion. **The question, not the guard** — `Drink` checks
// the purse itself, the line `CanBuy` already draws beside `Buy`. It exists so a shelf can dim a
// card rather than swallow a click.
func (s *Session) CanDrink(key string) bool {
	p, ok := PotionByKey(key)
	return ok && s.vitae >= p.Price
}

// Drink pays for a potion and applies it, and reports whether it could.
//
// **The purse moves first**, exactly as `Buy` wears the ring after spending: `SpendVitae` is the
// one place a refusal happens, so nothing can be applied to a run that could not pay for it.
//
// **A heal cannot take the wound below zero**, which is the only clamp here: the other two grow
// figures that have no ceiling of their own.
func (s *Session) Drink(key string) bool {
	p, ok := PotionByKey(key)
	if !ok || !s.SpendVitae(p.Price) {
		return false
	}

	switch p.Effect {
	case PotionHeal:
		s.hurt -= p.Amount
		if s.hurt < 0 {
			s.hurt = 0
		}
	case PotionDMG:
		s.dmgBonus += p.Amount
	case PotionLife:
		s.lifeBonus += p.Amount
	}
	return true
}

// Grant adds a permanent bonus the rules rolled up during a fight — a golden card's gamble coming
// good. See combat.RiderGolden.
//
// **It lands on the same two figures a potion moves**, which is what makes it permanent: they ride
// on the run, are saved with it, and are applied to the fighter by `Equip` at the top of every duel.
// The rules moved the *fighting* duelist when they rolled it, so the bonus was already worth
// something for the rest of that fight; this is what makes it worth something for the rest of the
// run.
//
// **It is called off the resolved event log, never off the playback** — see
// screens.settleGrants, which is the rule payHeldVitae and recordHandsPlayed are both under. A
// bonus applied as an animation reached it would be a bonus the player could change by leaving the
// screen.
func (s *Session) Grant(dmg, life int) {
	if dmg > 0 {
		s.dmgBonus += dmg
	}
	if life > 0 {
		s.lifeBonus += life
	}
}

// DMGBonus and LifeBonus are what the run has drunk, read by `Equip` and by anything drawing the
// duelist as they actually are. **Two plain figures rather than a list of bottles**: a potion is
// gone the moment it is drunk, so what a run carries is the sum and not the receipts — the same
// argument that makes the stones a count per rung rather than a pile of rocks.
func (s *Session) DMGBonus() int  { return s.dmgBonus }
func (s *Session) LifeBonus() int { return s.lifeBonus }
