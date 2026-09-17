package data

// The stones: **what the player may do to the hand ladder.**
//
// An essence record says what a stone's sibling changes about a *card*; a stone record says what it
// changes about a *rung*. One stone raises one hand's multiplier by a tenth of the figure
// `hands.json` writes down, for the rest of the run.
//
// **A stone names a hand by key, and by nothing else.** The keys are `hands.json`'s own, and the
// arithmetic — how much a tenth is, and how two stones on one rung stack — is
// `internal/combat`'s. This file holds the shape; the rules hold the truth, which is the same
// division every list here is under.
//
// **What is deliberately not here: the bump.** A record could carry `"Percent": 10` and it would
// be the `CostTier` mistake again — a rules vocabulary declared in JSON ahead of the rules. The
// tenth is one decision about the whole mechanic and it lives in `internal/combat/stone.go`. It
// becomes a field here the day two stones want to be worth different amounts, and not before.
//
// **The rules do not read this file.** A stone is bought by the run and applied to the run's
// fighter, so the parsing and the validation live in `internal/session`, exactly as an essence's do.

import (
	_ "embed"
)

//go:embed stones.json
var stonesJSON []byte

// StoneData is one rung-raiser as written in the file.
type StoneData struct {
	// StoneRecord is the key, and what anything holding a stone stores. Kebab-case, like a relic's
	// and an essence's.
	StoneRecord string `json:"StoneRecord"`

	// Name is what is written across the top of the card. **A mineral rather than the rung**, so
	// the name and the text are not the same sentence twice: the text says which hand it raises.
	Name string `json:"Name"`

	// Hand is the rung this stone raises, by `hands.json` key. Resolved against the catalog the
	// rules loaded — a stone naming a hand this build has not got fails the launch rather than
	// landing on whichever rung happens to sit first.
	Hand string `json:"Hand"`

	// Text is what the card says it does, in the same clipped register the essences use. A `\n` is an
	// authored line break, honored by `cards.WrapText` — every stone carries one, because the
	// eighteen differ only in the rung they name and left to the measurer they would read as
	// eighteen layouts of one card.
	//
	// **The figure is not in it.** What a stone is worth depends on the rung's own multiplier, so
	// writing `+11` here would be a number that goes stale the moment `hands.json` is tuned. The
	// card face computes it; see `internal/screens/card_art.go`.
	Text string `json:"Text"`

	// Art is the assets key of this stone's picture — the filename stem under `assets/stone/`, so
	// `stone/jasper.png` is `jasper`. **Empty means the default face**, which a stone borrows from
	// the relics; see DefaultStoneArt.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this stone, pasted under
	// `docs/art/stone_art_prompt.MD` as this record's own JSON. **The engine ignores it**, exactly
	// as it ignores a status's Badge.
	//
	// **The material is the prompt's rule, not this field's.** Which axis a stone raises decides
	// its material class — silica on concept, plain rock on form, gem on element — and where it
	// sits in that ladder decides how refined the specimen looks. So a brief says what this
	// particular mineral is and leaves the family to the prompt; a brief that argued with the
	// ladder would be a record overruling the catalog.
	Draw string `json:"Draw"`
}

// LoadStones parses the catalog into a map keyed by StoneRecord.
func LoadStones() map[string]StoneData {
	return keyed(stonesJSON, "stones.json", func(s StoneData) string { return s.StoneRecord })
}

// DefaultStoneArt is the face a stone with no Art of its own draws.
//
// **It borrows the relics' default**, exactly as DefaultPotionArt does and for the same reason:
// what a default face says is "nobody has painted this yet", and that sentence does not need to be
// said in a second picture per catalog.
//
// **It replaced a generated boulder on 2026-09-16** *(owner's call)*. Every stone drew
// a generated boulder before the catalog was painted, and the fallback stayed one after — so a
// silhouette nothing had drawn in months was among the last reasons `internal/systems` generated
// anything at all. All eighteen stones carry their own art, so this fires only for a record
// somebody has just authored, which is precisely the case a default face is for.
const DefaultStoneArt = DefaultRelicArt

// ArtKey is the picture this stone actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: a stone is drawn by
// the shop and by tools/stonesheet, and a fallback living in internal/screens is one the review
// tool does not have — which is exactly how a sheet comes to disagree with the game.
func (st StoneData) ArtKey() string {
	if st.Art == "" {
		return DefaultStoneArt
	}
	return st.Art
}

// StoneOrder is every record, sorted by key.
//
// **Sorted for the reason EssenceOrder is**: LoadStones returns a map, Go randomizes map order, and
// the bag of rocks is a shuffle of this list — so an unsorted walk would make which stones a run
// is offered depend on map iteration and take the run's reproducibility with it. See the
// `randomness` skill.
func StoneOrder(stones map[string]StoneData) []string {
	return sortedKeys(stones)
}
