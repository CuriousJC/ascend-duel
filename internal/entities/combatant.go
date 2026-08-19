package entities

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Combatant is a duelist that can be drawn. The stats live in the embedded
// combat.Duelist so the rules engine can take them without ever seeing a sprite;
// the promoted fields mean gs.Fighter.DMG and friends still read the same.
type Combatant struct {
	combat.Duelist

	// Record is which roster entry this combatant was built from, and it is what names its deck
	// in `internal/decks` *(2026-08-16)*. It replaced `Style combat.PlanStyle`: an enemy's
	// behaviour used to be a string picking one of four planners and is now the cards it holds,
	// so what a caller needs from this struct is the key to those cards.
	//
	// Empty for the player, whose deck is built by the screen.
	Record string

	// **There is no sprite here any more** *(2026-08-11)*. The fighter's went when the
	// character block replaced it, and the enemy's went when the enemy became a card — so
	// the field, the sheet slicing and the *ebiten.Image import all went with them. This
	// struct now holds nothing Ebitengine defines, which is worth keeping: it is one of the
	// two things standing between `entities` and needing a window.
	//
	// Portrait is the assets key for the picture on this combatant's card, carried through
	// from the data record rather than resolved here — entities cannot reach the asset map,
	// and the card is drawn by internal/cards from raw bytes rather than from an
	// *ebiten.Image anyway.
	//
	// Empty for the fighter, like Sprite is nil for it.
	Portrait string

	// Name is what this combatant is called on screen. Set for a duelist, whose record
	// carries one; an enemy's name comes from the roster in internal/screens rather than
	// from its record, and moving that is a separate change.
	Name string

	// CardBack is the mark on the back of this duelist's cards, by name. A string rather
	// than a cards.BackMark because entities must not import the drawing package — the
	// screen parses it with cards.ParseBackMark, exactly as it parses an element.
	//
	// Empty for an enemy: enemies do not have a deck the player ever sees the back of.
	CardBack string
}

// FightsPerFloor is how many fights a floor holds — outer room, inner room, stairway — and the
// third of them is its boss. See the tower section of MECHANICS.md.
//
// **It lives here rather than in `internal/screens` because two things outside that screen need
// it**: the combat screen names the room under the duelist card, and `tools/balance` maps an
// enemy's lowest valid floor onto the fight it would first be met at. A tool that cannot import a
// screen still has to agree with it about how deep a floor is.
const FightsPerFloor = 3

// FirstFightOnFloor is the fight index of a floor's outer room, counting floors from one.
func FirstFightOnFloor(floor int) int {
	if floor < 1 {
		return 0
	}
	return (floor - 1) * FightsPerFloor
}

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

// NewEnemyFrom builds an opponent from an enemy record, **grown to the fight it is met at** — see
// ScaleToFight. Fight 0 is the first fight of a run and takes the record's stats unchanged.
//
// **The fight index is a parameter rather than something read later**, so an unscaled opponent
// cannot be built by accident: every caller has to say where in the ascent this one stands.
//
// **It takes no sprite sheet since 2026-08-11.** It used to slice a west-facing idle frame
// out of one, which is why the caller had to resolve an asset out of global state and pass
// it in; the enemy is a card now, so all that is left is the portrait's key.
func NewEnemyFrom(d data.EnemyData, fight int) *Combatant {
	c := &Combatant{
		Duelist: combat.Duelist{
			// **Two of the three stats climb and one does not.** HP and DMG are what the curve is
			// made of; `Actions` is left alone because it is the budget a *deck* is spent out of,
			// and growing it would hand a floor-eight opponent more cards rather than a harder
			// version of its own. It is the dial to reach for on purpose, per enemy, not one to
			// move by arithmetic.
			DMG:     ScaleToFight(d.DMG, fight),
			Actions: d.Actions,
			MaxLife: ScaleToFight(d.HP, fight),

			// **Enemies do not form hands** *(2026-08-17)*. Their cards resolve one at a time, in the
			// order the planner chose them. It is set here because this is the one place an
			// opponent is built from a record — the same seat `Rings` deliberately leaves at its
			// zero value for the mirror-image reason.
			SoloAttacks: true,
		},
		Record:   d.EnemyRecord,
		Name:     d.Name,
		Portrait: d.Portrait,
	}
	c.CurrentLife = c.MaxLife
	return c
}

// NewDuelistFrom builds the player from a duelist record.
//
// **No sprite, no plan style, and neither is a gap.** The character block replaced the
// fighter's sprite on the combat screen, and a duelist is planned by whoever is holding the
// mouse — which is exactly why the two records split.
func NewDuelistFrom(d data.DuelistData) *Combatant {
	c := &Combatant{
		Duelist: combat.Duelist{
			DMG:     d.DMG,
			Actions: d.Actions,
			MaxLife: d.HP,
		},
		Name:     d.Name,
		CardBack: d.CardBack,
	}
	c.CurrentLife = c.MaxLife
	return c
}
