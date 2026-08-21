package pyramid

// The ascent curve: how much harder each room is than the one below it.

// AscentGrowthPct is how much tougher each fight is than the one before it, compounding.
//
// **The ascent curve** *(2026-08-17, owner's call)*: floor 1's outer room is the base and every
// room after it grows the opponent's HP and DMG by this much, so winning is what makes the next
// fight harder. A floor is three rooms — see the tower section of MECHANICS.md — so a floor costs
// about a third more than the one below it.
//
// **It compounds per *room*, not per floor**, which is what "with each win" means: the stairway
// boss is harder than the inner room on its own floor, not equal to it.
const AscentGrowthPct = 10

// The multiplier's fixed-point scale, and the ceiling it stops growing at.
//
// **The scale is what makes the curve work on small stats, and it was found by a test.** The
// obvious implementation compounds the stat itself — `v = v * 110 / 100` once per room — and it is
// wrong in a way that is invisible on the numbers you check first: integer division truncates
// `5 * 110 / 100` straight back to 5, so **every stat below 10 is frozen forever**. Half the roster
// opens on DMG 5 or 6, which is exactly the band the curve was added to lift. Compounding the
// *multiplier* at a scale of a million and truncating once at the end is what fixes it.
//
// The ceiling exists so a fight index far outside the tower cannot overflow the multiply. A
// million-fold is already past any number the game can use — the roster is 96 fights and the tower
// is 24 — so it is a guard rather than a balance decision.
const (
	ascentScale    = 1_000_000
	ascentMaxScale = 1_000_000 * 1_000_000
)

// ScaleToFight grows one base stat to the fight it is met at. Fight 0 is floor 1's outer room and
// is the base, unscaled.
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
// **Nothing caps the *fight*,** deliberately and with the same reasoning as the floor label on the
// combat screen: the fight order is the whole 96-record roster standing in for a generator, so
// playing far enough produces numbers the 8-floor tower would never ask for. Capping here would be
// the hydration layer quietly disagreeing with the counter it was handed.
func ScaleToFight(base, fight int) int {
	mul := ascentScale
	for i := 0; i < fight && mul < ascentMaxScale; i++ {
		mul = mul * (100 + AscentGrowthPct) / 100
	}
	return base * mul / ascentScale
}
