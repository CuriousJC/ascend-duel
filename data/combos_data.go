package data

// The combo catalogue as data: which hands exist, which element mixes exist, and what each is
// worth. `internal/combat` reads this and turns it into rules.
//
// **This file is the one place `data` is read by the rules rather than by a layer above them**,
// and it is why `internal/combat` imports this package at all. Every other list here is consumed
// by `screens`, `decks` or `entities`; a combo is consumed by the resolver itself. That is the
// line worth holding if a seventh list is proposed — ask who reads it, not whether it is data.
//
// **Two lists, multiplied at load** *(2026-08-14)*. A turn forms one hand and that hand has one
// element mix, so the combos a player can meet are `hands x mixes` — 27 of them once the sizes
// that cannot hold four colours are struck. Authoring them as a grid would be 27 entries whose
// numbers nobody could keep consistent; authoring them as two axes is 11.
//
// **What is deliberately not here: any meaning.** The strings below are names, and joining them
// to `ActionKind`, `Element` and `Category` is `internal/combat`'s job, exactly as it is for
// `CardData.Concept`. A malformed catalogue is refused *there*, on the same principle
// CheckCostTiers already works on — data holds the shape, the rules hold the truth.

import (
	_ "embed"
	"encoding/json"
)

//go:embed combos.json
var combosJSON []byte

// comboFile is the shape of combos.json.
type comboFile struct {
	Hands []HandData `json:"hands"`
	Mixes []MixData  `json:"mixes"`
}

// HandData is one rung of the of-a-kind ladder as written in the file.
//
// **Every hand counts copies of a card.** There is no element axis here any more — the colours
// a hand happens to be made of are the *mix*, read separately off whatever the hand formed, so
// a hand and a mix are genuinely independent and multiply out rather than being enumerated.
type HandData struct {
	// Key is a stable slug shared by every expansion of this entry — `flurry`, `onslaught`. It
	// is how a caller asks for "the flurry built on Strike" without knowing the ID arithmetic,
	// and it is deliberately independent of Name so a hand can be renamed without breaking
	// lookups.
	Key string `json:"key"`

	// ID is the hand's number, and an expanded entry adds the card's enum value to it. Discovery
	// will persist on the profile, so this must not move under a player who has already found a
	// hand.
	ID int `json:"id"`

	// Name is what the hand is called on screen. `{card}` is replaced when an entry expands, so
	// `"{card} Flurry"` becomes `Strike Flurry`.
	Name string `json:"name"`

	// Groups is how many cards of each distinct concept the hand wants. `[3,2]` is a full house
	// and can never be satisfied by five of one card.
	Groups []int `json:"groups"`

	// Scope is which categories the hand counts, by name. Empty means every card in the turn;
	// every shipping hand names `attack`, because the ladder is an offence you build.
	Scope []string `json:"scope"`

	// Expand is `none` or `attack-cards`.
	Expand string `json:"expand"`

	// Multiplier is the hand's contribution to the turn's damage multiplier, in **percent** —
	// 150 is 1.5x. An integer because `internal/combat` is integer arithmetic throughout, and a
	// float here would be the one number in the game that rounds differently from the rest.
	Multiplier int `json:"multiplier"`

	Effect EffectData `json:"effect"`
}

// MixData is one element makeup: how many distinct non-basic colours the formed hand shows, and
// what that is worth.
//
// **Basic is not a colour and is never counted.** A hand of two basic Strikes and one ice Strike
// shows one colour and is mono; `Colours` is the count of *real* elements present, so drab is
// the hand that showed none at all.
type MixData struct {
	Key  string `json:"key"`
	ID   int    `json:"id"`
	Name string `json:"name"`

	// Colours is the exact number of distinct non-basic elements. Exact rather than "at least",
	// so the five mixes partition every hand and exactly one of them can ever apply — which is
	// what makes a turn produce one combo without any ranking machinery.
	Colours int `json:"colours"`

	// Multiplier is this mix's contribution, in percent, added to the hand's.
	Multiplier int `json:"multiplier"`
}

// EffectData is the non-damage reward vocabulary as written in the file. Damage is the
// Multiplier fields above; this is everything else.
//
// **StaggerAll is a bool here rather than the sentinel the rules use.** A file saying
// `"stagger": -1` is one nobody can read.
type EffectData struct {
	BankAP     int  `json:"bankAP"`
	Stagger    int  `json:"stagger"`
	StaggerAll bool `json:"staggerAll"`
}

// LoadCombos parses the catalogue, in file order.
//
// **File order is load-bearing rather than cosmetic** — a slice, never a map, per the
// determinism rules in CLAUDE.md.
//
// It panics on a file it cannot read, like the other loaders here. The rules do the rest of the
// checking: `internal/combat` refuses a catalogue that names a card or category it does not
// have, or that describes a shape it cannot match.
func LoadCombos() ([]HandData, []MixData) {
	var file comboFile
	if err := json.Unmarshal(combosJSON, &file); err != nil {
		panic("Failed to unmarshal combos.json: " + err.Error())
	}
	if len(file.Hands) == 0 {
		panic("combos.json declares no hands")
	}
	if len(file.Mixes) == 0 {
		panic("combos.json declares no element mixes")
	}
	return file.Hands, file.Mixes
}
