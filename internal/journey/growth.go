package journey

// The growth curve: how much harder each room is than the one below it.

// The multiplier's fixed-point scale, and the ceiling it stops growing at.
//
// **The scale is what makes the curve work on small stats, and it was found by a test.** The
// obvious implementation compounds the stat itself — `v = v * 110 / 100` once per room — and it is
// wrong in a way that is invisible on the numbers you check first: integer division truncates
// `5 * 110 / 100` straight back to 5, so **every stat below 10 is frozen forever**. Half the roster
// opens on DMG 5 or 6, which is exactly the band the curve exists to lift. Compounding the
// *multiplier* at a scale of a million and truncating once at the end is what fixes it.
//
// The ceiling exists so a fight index far outside the journey cannot overflow the multiply. A
// million-fold is already past any number the game can use, so it is a guard rather than a balance
// decision.
const (
	growthScale    = 1_000_000
	growthMaxScale = 1_000_000 * 1_000_000
)

// BasisPoints is the unit the two growth rates in data/journey.json are written in: a hundredth of
// a percent, so 1000 is 10.00% per fight.
//
// **A whole percent is too coarse a dial on something that compounds twenty-three times** between
// the first fight of the journey and the last — 10% and 11% are a factor of 9.8 and a factor of 12.2
// at the last realm, with nothing expressible between them.
const BasisPoints = 10_000

// ScaleToFight grows one base stat to the fight it is met at, at a given growth rate in basis
// points, and then sets it at scaleBP of that — BasisPoints is the stat as authored, 8000 is four
// fifths of it. Fight 0 is realm 1's outer room, where the base is only scaled, not grown.
//
// **The scale starts the multiplier rather than being applied to the result**, so the stat is
// truncated once rather than twice. It is the difficulty dial over the whole roster: every record's
// base keeps its authored ratio to every other, and the curve's shape is untouched.
//
// **The step is the fight, not the realm**, which is what makes a realm's boss harder than its own
// inner chamber and the next realm's outer chamber harder than that boss. A creature's base says
// what it is worth at the very bottom of the journey whatever realm it is actually met on, so the
// tier is already in this number and must not be written into the base as well.
//
// **Integer arithmetic rather than one `math.Pow`.** A float power is deterministic on one machine
// and not reliably identical across two, and a stat feeds a duel that is meant to be replayable
// from a seed — so the same rule that keeps `math/rand` out of the game keeps `math.Pow` out of
// this.
//
// **Truncating, like every other percentage in the game** — `scaleDamage` and every relic
// percentage round toward zero, and a curve that rounded the other way would be the one
// number a player could not work out from the others.
//
// **Nothing caps the fight.** The journey has a configured height and the journey wraps past it; the
// curve does not, so a player who keeps going keeps meeting bigger numbers. That is the endless
// journey, and capping here would be the hydration layer quietly disagreeing with the counter it was
// handed.
func ScaleToFight(base, fight, growthBP, scaleBP int) int {
	mul := growthScale / BasisPoints * scaleBP
	for i := 0; growthBP > 0 && i < fight && mul < growthMaxScale; i++ {
		mul = mul * (BasisPoints + growthBP) / BasisPoints
	}
	return base * mul / growthScale
}
