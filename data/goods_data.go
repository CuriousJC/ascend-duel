package data

// The sealed goods: **what the shop sells without letting you read it first.**
//
// A relic on the shelf is a thing you read and then pay for. A good is paid for and then read: it
// holds four of one catalog, it is opened in a dialog, and the player keeps exactly one of the
// four. What five vitae buys is the *choice*, which is the whole design — see
// internal/session/good.go, where a record becomes something the shop can stand on a seat.
//
// **They were three pairs of constants in Go until 2026-09-14** — a name in `internal/screens`, a
// price and a size in `internal/session`, and a picture chosen by a switch. Nothing about any of
// that is a rule: a good is a shelf item with a face, a price and a paragraph saying what to draw,
// which is what every other catalog in this directory already is. The one thing that *is* a rule is
// `Contains`, and it is a closed vocabulary checked in `internal/session` exactly as a potion's
// `Effect` is.
//
// **The rules do not read this file**, for the reason `potions.json` is not read by them either: a
// good is bought by a run, and `internal/combat` has never heard of a shop.

import (
	_ "embed"
	"encoding/json"
)

//go:embed goods.json
var goodsJSON []byte

// GoodData is one sealed good as written in the file.
type GoodData struct {
	// GoodRecord is the key, and what the shelf names a seat by. Kebab-case, like every other
	// catalog's.
	GoodRecord string `json:"GoodRecord"`

	// Name is what is written across the top of the card. **The vessel rather than the contents**
	// — "BAG OF ROCKS" — because the line under it already says what is inside and how many.
	Name string `json:"Name"`

	// Family is the motif the record was authored beside, read by a review sheet and by nothing
	// else. See the data skill on the three fields no catalog's rules read.
	Family string `json:"Family"`

	// Art is an assets.LoadImageData key, in assets/other — the family the potions share, since
	// the two catalogs share one art prompt. **Empty means the good borrows the picture of whatever
	// is inside it** — the bag the boulder every stone card draws, the vial the essence catalog's
	// default face, the sack the rune catalog's — which is what a good drew before the three were
	// painted and what a newly authored one draws until it is. See screens.goodArt.
	Art string `json:"Art"`

	// Draw is the subject paragraph an art generator is given. The generic prompt is
	// docs/art/other_card_art_prompt.MD; this says what the object is. Ignored by the engine.
	Draw string `json:"Draw"`

	// Contains is which catalog is inside, by name. **A closed vocabulary, checked in
	// `internal/session`** — `stones`, `essences`, `runes` — because it decides which stream the
	// contents are drawn from and what happens when one is chosen. It is also the word the card's
	// own line is written with, so the face and the behavior cannot name different things.
	Contains string `json:"Contains"`

	// Size is how many are drawn from inside, of which the player keeps one.
	Size int `json:"Size"`

	// Price is what the shop charges.
	Price int `json:"Price"`

	// Title is what the dialog is headed when the good is opened. **Empty means the Name**, which
	// is what the bag and the sack want; the vial heads itself with an instruction instead,
	// because the four essences on the table are the whole of what the dialog is.
	Title string `json:"Title"`

	// Hint is the line under that title. **Empty means the computed one** — "take one of the four,
	// the rest are gone" — so the figure on the screen is the Size above rather than a number
	// authored twice and free to drift. A good whose dialog has something else to say writes it
	// here.
	Hint string `json:"Hint"`

	// Tip is what resting on the shelf card says about what is inside, in the player's words. **The
	// count and the price are not here**: both are computed from the fields above, for Hint's
	// reason.
	Tip []string `json:"Tip"`
}

// LoadGoods parses the catalog **as a slice, in file order**, which is the order the goods are
// shuffled from when a visit picks which two stand on the shelf. Nothing here is a map, so there is
// no sorted walk to protect.
func LoadGoods() []GoodData {
	var list []GoodData
	if err := json.Unmarshal(goodsJSON, &list); err != nil {
		panic("Failed to unmarshal goods.json: " + err.Error())
	}
	return list
}
