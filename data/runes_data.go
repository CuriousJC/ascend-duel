package data

// The runes: **what the player may do to their deck during a fight.**
//
// An essence is won and spent between rooms; a rune is bought, carried, and spent between the
// turns of a duel. They overlap on purpose — both alter the run's deck and some of them do the
// same thing — and they are separate things because *when* you spend one is most of what it is.
// See MECHANICS.md.
//
// **A record is a target, a count and a value**, which is the essence record's shape plus the one
// field an essence never needed: how many cards this one eats. An essence was always aimed at exactly
// one card; a rune may name two.
//
// **The rules read one field of this and only one.** A rider has to be consulted while a round is
// resolving, so its vocabulary is a closed Go enum in `internal/combat` — everything else here is
// applied to the *run's* deck, which is `internal/session`, and that is where a record is parsed
// and refused. Same who-consumes-it test every file in this package answers.

import (
	_ "embed"
)

//go:embed runes.json
var runesJSON []byte

// RuneData is one consumable the player can be sold.
type RuneData struct {
	// RuneRecord is the key, and what anything holding a rune stores. Kebab-case, like a
	// essence's and a relic's.
	RuneRecord string `json:"RuneRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Family is the motif this rune belongs to — the block of siblings it was authored beside,
	// and the heading it is reviewed under on the rune sheet.
	//
	// **The engine ignores it, exactly as it ignores Art and Draw.** It groups the review page and
	// nothing else reads it; a rune with no Family still loads and is still sold.
	//
	// **It is the relic catalog's field brought over** *(owner's call, 2026-09-12)*, with the
	// same argument and the same caveat: it is legibility for whoever is authoring rather than a
	// fact the file knows and the rules do not, and it can go quietly out of date when a record is
	// retargeted without anything failing. Re-read the block when you change what a rune does.
	Family string `json:"Family"`

	// Art is the assets.LoadImageData key for the picture on the face. **Empty means the default
	// rune face** — see ArtKey.
	//
	// **The rune borrowed the essence's placeholder until 2026-09-12**, through one constant in
	// internal/screens whose own note said the day runes got art it should become a field here
	// rather than a fallback being unpicked. This is that field, and the placeholder is a rune's
	// own now: two catalogs wearing one picture is two backlogs that cannot be told apart.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this rune — what the thing
	// *is* and what it is doing, in one sentence. **Nothing in the game reads it**, exactly like
	// Art's own key and a relic's Draw.
	//
	// **Empty means nobody has written one yet**, which — read against an empty Art — is what the
	// rune sheet reports as the backlog.
	Draw string `json:"Draw"`

	// Change is what class of alteration this rune makes, from a closed vocabulary resolved by
	// `session.ParseRuneChange`: `normal` or `upgrade`. **Required on every record.**
	//
	// **A card has a form, an element and an action — and then one upgrade** *(owner's call,
	// 2026-09-09)*. A `normal` rune moves one of the first three and leaves the upgrade alone:
	// Embermark paints a card fire, Bulwark turns it into a Guard, and a gold card is still gold
	// afterwards. An `upgrade` rune writes the one upgrade slot, and whatever was in it is gone.
	//
	// **It is authored rather than derived, and the loader refuses a record that disagrees with its
	// own target.** Every `rider` rune is an upgrade and nothing else is, so this could have
	// been computed — and a computed field would say nothing, where an authored one is a claim the
	// record makes and the loader checks. It is the same posture `Match` takes in the tutorial
	// script: the thing the author meant, written down where the author is looking.
	Change string `json:"Change"`

	// Target is what this rune does, from a closed vocabulary resolved by
	// `session.ParseRuneTarget`: `rider`, `remove`, `swap`, `vitae`.
	//
	// **Closing it is the point**, exactly as with a card's verb and an essence's target. A new target
	// is a Go change plus one place applying it, never something a JSON file can assert into
	// existence.
	Target string `json:"Target"`

	// Rider names which rule a `rider` rune attaches — `combat.ParseRiderKind` resolves it.
	// Empty for every other target, and refused if one is supplied anyway.
	Rider string `json:"Rider,omitempty"`

	// Value is read against the target: a figure for `rider` and `vitae`, a concept key for
	// `swap`, and nothing at all for `remove`.
	Value string `json:"Value,omitempty"`

	// Count is how many cards of the run this rune takes. **Zero is a real answer** and means
	// it takes none — a rune that fills the purse is aimed at the run rather than at a card,
	// and the board piece asks for no target at all.
	Count int `json:"Count"`

	// Text is what the card says it does, in the clipped register the action cards use — the
	// column is about a dozen characters wide, and a `\n` is an authored line break honored by
	// `cards.WrapText`.
	Text string `json:"Text"`
}

// DefaultRuneArt is the face a record with no Art of its own draws:
// assets/rune/default-rune.png.
//
// **A picture of its own rather than the essence's** *(2026-09-12)*. The two catalogs wore one
// placeholder while neither had art, and the moment either gets some that becomes a page where a
// drawn essence and an undrawn rune are the same picture — so the backlogs are told apart by
// giving each its own seat. **Keys are not file paths**: LoadImageData files that picture under
// this.
const DefaultRuneArt = "default-rune"

// ArtKey is the picture this rune actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: a rune is drawn
// in the consumables pane, in the shop and in tools/runesheet, and a fallback living in a
// screen is a fallback the review tool does not have.
func (p RuneData) ArtKey() string {
	if p.Art == "" {
		return DefaultRuneArt
	}
	return p.Art
}

// RuneFileOrder is every record id in the order data/runes.json writes them.
//
// **File order rather than RuneOrder's sorted keys**, and it is for the review page alone: the
// catalog is authored in motif order — the five marks together, the four staves, the metals
// beside each other — and sorting by key throws exactly that away.
//
// **Nothing that decides an outcome may walk this.** What a sack holds is a shuffle of
// RuneOrder, which is sorted for the reason the randomness skill gives; this is a layout.
func RuneFileOrder() []string {
	return fileOrder(runesJSON, "runes.json", func(p RuneData) string { return p.RuneRecord })
}

// LoadRunes parses the catalog into a map keyed by RuneRecord.
func LoadRunes() map[string]RuneData {
	return keyed(runesJSON, "runes.json", func(p RuneData) string { return p.RuneRecord })
}

// RuneOrder is every record, sorted by key.
//
// **Sorted because LoadRunes returns a map and Go randomizes that order**, and this one
// decides an outcome rather than a layout: what a sack holds is a shuffle of this list, so an
// unsorted walk would make a purchase depend on map iteration and take the run's reproducibility
// with it. See the `randomness` skill.
func RuneOrder(runes map[string]RuneData) []string {
	return sortedKeys(runes)
}
