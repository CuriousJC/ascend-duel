package pyramid

// The ascent curve: how much harder each room is than the one below it.

// The multiplier's fixed-point scale, and the ceiling it stops growing at.
//
// **The scale is what makes the curve work on small stats, and it was found by a test.** The
// obvious implementation compounds the stat itself — `v = v * 110 / 100` once per room — and it is
// wrong in a way that is invisible on the numbers you check first: integer division truncates
// `5 * 110 / 100` straight back to 5, so **every stat below 10 is frozen forever**. Half the roster
// opens on DMG 5 or 6, which is exactly the band the curve exists to lift. Compounding the
// *multiplier* at a scale of a million and truncating once at the end is what fixes it.
//
// The ceiling exists so a fight index far outside the tower cannot overflow the multiply. A
// million-fold is already past any number the game can use, so it is a guard rather than a balance
// decision.
const (
	ascentScale    = 1_000_000
	ascentMaxScale = 1_000_000 * 1_000_000
)

// BasisPoints is the unit the two growth rates in data/tower.json are written in: a hundredth of
// a percent, so 1000 is 10.00% per fight.
//
// **A whole percent is too coarse a dial on something that compounds twenty-three times** between
// the first fight of the tower and the last — 10% and 11% are a factor of 9.8 and a factor of 12.2
// at the summit, with nothing expressible between them.
const BasisPoints = 10_000

// ScaleToFight grows one base stat to the fight it is met at, at a given growth rate in basis
// points. Fight 0 is floor 1's outer room and is the base, unscaled.
//
// **The step is the fight, not the floor**, which is what makes a floor's boss harder than its own
// inner chamber and the next floor's outer chamber harder than that boss. A creature's base says
// what it is worth at the very bottom of the tower whatever floor it is actually met on, so the
// tier is already in this number and must not be written into the base as well.
//
// **Integer arithmetic rather than one `math.Pow`.** A float power is deterministic on one machine
// and not reliably identical across two, and a stat feeds a duel that is meant to be replayable
// from a seed — so the same rule that keeps `math/rand` out of the game keeps `math.Pow` out of
// this.
//
// **Truncating, like every other percentage in the game** — `blunt`, the defend reductions and
// `scaleDamage` all round toward zero, and a curve that rounded the other way would be the one
// number a player could not work out from the others.
//
// **Nothing caps the fight.** The tower has a configured height and the climb wraps past it; the
// curve does not, so a player who keeps going keeps meeting bigger numbers. That is the endless
// tower, and capping here would be the hydration layer quietly disagreeing with the counter it was
// handed.
func ScaleToFight(base, fight, growthBP int) int {
	if growthBP <= 0 {
		return base
	}
	mul := ascentScale
	for i := 0; i < fight && mul < ascentMaxScale; i++ {
		mul = mul * (BasisPoints + growthBP) / BasisPoints
	}
	return base * mul / ascentScale
}
