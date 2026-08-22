package data

// Rarity is how scarce a ring is: one word carrying both how often the shop offers it and what it
// costs.
//
// **Three tiers and nothing between them** *(owner's call, 2026-08-22)*. A per-ring price let the
// catalogue drift into seventeen numbers that could only be judged one at a time; a tier can be
// read against every other ring at a glance, and rebalancing a ring is moving it rather than
// picking a new figure. The cost of that is real and worth saying: two rings in the same tier are
// the same price even when one is plainly stronger, and the answer is the tier, not a fourth.
//
// **Weight and price are deliberately not the same curve.** A rare ring is roughly a tenth as
// likely to appear as a common one but only a bit over twice the price — scarcity is what makes it
// feel rare, and a price that tracked the odds would make it unbuyable on the visit it finally
// turns up. The ladder is 3 / 5 / 7 *(owner's call, 2026-08-22)*, which is the same span the
// hand-written prices used to cover, now with the whole catalogue landing on three of its rungs.
type Rarity string

// The three tiers. **Strings rather than an enum**, because this one is written in a data file and
// an ordinal in a file is the hazard every ID in this project is careful to avoid.
const (
	Common   Rarity = "common"
	Uncommon Rarity = "uncommon"
	Rare     Rarity = "rare"
)

// rarityTiers is the whole table: what each tier costs and how heavily it is drawn.
//
// **A word not in this map does not exist** — see Valid, and the load-time check in
// internal/session, which refuses a misspelled tier rather than shipping a ring that is free and
// never offered.
var rarityTiers = map[Rarity]struct {
	price  int
	sell   int
	weight int
}{
	Common:   {price: 3, sell: 1, weight: 10},
	Uncommon: {price: 5, sell: 2, weight: 4},
	Rare:     {price: 7, sell: 3, weight: 1},
}

// Valid reports whether this is one of the three tiers.
func (r Rarity) Valid() bool {
	_, ok := rarityTiers[r]
	return ok
}

// Price is what the shop charges for a ring of this tier, in vitae. Zero for a tier that does not
// exist, which is a ring that never loads.
func (r Rarity) Price() int { return rarityTiers[r].price }

// Sell is what taking a ring of this tier off pays back.
//
// **A written figure rather than a fraction of the price** *(owner's call, 2026-08-22)*. It was a
// quarter rounded up, and across three prices that arithmetic paid an uncommon and a rare the same
// 2 — a cleverness that produced a wrong answer and then had to be argued about. Three tiers is
// three numbers; write them.
//
// **The round trip still loses, and by more the dearer the ring**: 3→1, 5→2, 7→3. A shelf you could
// try on for free would be a rerolling of your hand every visit rather than a decision.
func (r Rarity) Sell() int { return rarityTiers[r].sell }

// Weight is how many tickets a ring of this tier holds in the shelf draw. Relative, not a
// percentage: a common ring is ten times as likely to be drawn as a rare one, whatever the
// catalogue's mix happens to be.
func (r Rarity) Weight() int { return rarityTiers[r].weight }

// Rarities is the three tiers, cheapest first. For a tool or a test that wants to walk them
// without writing the list down a second time.
func Rarities() []Rarity { return []Rarity{Common, Uncommon, Rare} }
