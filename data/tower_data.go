package data

// The tower's own numbers: how tall it is, and how much harder each fight is than the one below.

import _ "embed"

//go:embed tower.json
var towerJSON []byte

// TowerData is the shape of the climb, and it is the only tuning file in this package.
//
// **The two growth rates are the whole difficulty curve.** Every creature in every motif writes
// a base stat line that says what it is worth at the very bottom of the tower, and these say what
// the bottom of the tower turns into by the time it is met. Tuning a floor means moving one of
// these, not editing a record.
type TowerData struct {
	// Floors is how tall the tower is, and it is the number MustBeClimbable is asked about — a
	// roster that cannot give this many floors a motif of its own fails the launch.
	//
	// **It is a configured stop rather than a baked constant**, so climbing further is a number
	// here rather than a rewrite.
	Floors int `json:"Floors"`

	// HPGrowth and DMGGrowth are how much tougher each *fight* is than the one before it, in
	// basis points — 1000 is 10.00% and 600 is 6.00%, compounding.
	//
	// **Basis points rather than a percentage**, because a whole percent is a coarse dial on
	// something that compounds twenty-three times between the first fight and the last: 10% and
	// 11% are a factor of 9.8 and a factor of 12.2 at the summit, with nothing expressible
	// between them.
	//
	// **Two rates rather than one**, so how fast a creature's life outruns the player's damage is
	// a separate question from how fast its blows outrun the player's life.
	//
	// **They compound per fight, not per floor**, which is what makes a floor's boss harder than
	// its inner chamber, and the next floor's outer chamber harder than that boss. See
	// pyramid.ScaleToFight, which is the arithmetic and is integer on purpose.
	HPGrowth  int `json:"HPGrowth"`
	DMGGrowth int `json:"DMGGrowth"`
}

// LoadTower reads the tower's numbers, refusing any that would stop the climb being a climb.
//
// A growth below zero shrinks the tower as it is climbed and a zero-floor tower has nowhere to
// go; both are authoring mistakes rather than tuning, so they fail the launch.
func LoadTower() TowerData {
	t := parseOne[TowerData](towerJSON, "tower.json")
	if t.Floors <= 0 {
		panic("tower.json: a tower needs at least one floor")
	}
	if t.HPGrowth < 0 || t.DMGGrowth < 0 {
		panic("tower.json: a growth rate below zero makes the climb easier as it is climbed")
	}
	return t
}
