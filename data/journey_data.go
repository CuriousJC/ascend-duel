package data

// The journey's own numbers: how tall it is, and how much harder each fight is than the one below.

import _ "embed"

//go:embed journey.json
var journeyJSON []byte

// JourneyData is the shape of the journey, and it is the only tuning file in this package.
//
// **The two growth rates are the whole difficulty curve.** Every creature in every motif writes
// a base stat line that says what it is worth at the very bottom of the journey, and these say what
// the bottom of the journey turns into by the time it is met. Tuning a realm means moving one of
// these, not editing a record.
type JourneyData struct {
	// Realms is how tall the journey is, and it is the number MustFillJourney is asked about — a
	// roster that cannot give this many realms a motif of its own fails the launch.
	//
	// **It is a configured stop rather than a baked constant**, so going further is a number
	// here rather than a rewrite.
	Realms int `json:"Realms"`

	// HPGrowth and DMGGrowth are how much tougher each *fight* is than the one before it, in
	// basis points — 1000 is 10.00% and 600 is 6.00%, compounding.
	//
	// **Basis points rather than a percentage**, because a whole percent is a coarse dial on
	// something that compounds twenty-three times between the first fight and the last: 10% and
	// 11% are a factor of 9.8 and a factor of 12.2 at the last realm, with nothing expressible
	// between them.
	//
	// **Two rates rather than one**, so how fast a creature's life outruns the player's damage is
	// a separate question from how fast its blows outrun the player's life.
	//
	// **They compound per fight, not per realm**, which is what makes a realm's boss harder than
	// its inner chamber, and the next realm's outer chamber harder than that boss. See
	// journey.ScaleToFight, which is the arithmetic and is integer on purpose.
	HPGrowth  int `json:"HPGrowth"`
	DMGGrowth int `json:"DMGGrowth"`
}

// LoadJourney reads the journey's numbers, refusing any that would stop the journey being a journey.
//
// A growth below zero shrinks the journey as it goes and a zero-realm journey has nowhere to
// go; both are authoring mistakes rather than tuning, so they fail the launch.
func LoadJourney() JourneyData {
	t := parseOne[JourneyData](journeyJSON, "journey.json")
	if t.Realms <= 0 {
		panic("journey.json: a journey needs at least one realm")
	}
	if t.HPGrowth < 0 || t.DMGGrowth < 0 {
		panic("journey.json: a growth rate below zero makes the journey easier as it goes")
	}
	return t
}
