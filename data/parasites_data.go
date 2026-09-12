package data

// The parasites: **what the player may do to their deck during a fight.**
//
// A worm is won and spent between rooms; a parasite is bought, carried, and spent between the
// turns of a duel. They overlap on purpose — both alter the run's deck and some of them do the
// same thing — and they are separate things because *when* you spend one is most of what it is.
// See MECHANICS.md.
//
// **A record is a target, a count and a value**, which is the worm record's shape plus the one
// field a worm never needed: how many cards this one eats. A worm was always aimed at exactly
// one card; a parasite may name two.
//
// **The rules read one field of this and only one.** A rider has to be consulted while a round is
// resolving, so its vocabulary is a closed Go enum in `internal/combat` — everything else here is
// applied to the *run's* deck, which is `internal/session`, and that is where a record is parsed
// and refused. Same who-consumes-it test every file in this package answers.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed parasites.json
var parasitesJSON []byte

// ParasiteData is one consumable the player can be sold.
type ParasiteData struct {
	// ParasiteRecord is the key, and what anything holding a parasite stores. Kebab-case, like a
	// worm's and a relic's.
	ParasiteRecord string `json:"ParasiteRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Family is the motif this parasite belongs to — the block of siblings it was authored beside,
	// and the heading it is reviewed under on the parasite sheet.
	//
	// **The engine ignores it, exactly as it ignores Art and Draw.** It groups the review page and
	// nothing else reads it; a parasite with no Family still loads and is still sold.
	//
	// **It is the relic catalogue's field brought over** *(owner's call, 2026-09-12)*, with the
	// same argument and the same caveat: it is legibility for whoever is authoring rather than a
	// fact the file knows and the rules do not, and it can go quietly out of date when a record is
	// retargeted without anything failing. Re-read the block when you change what a parasite does.
	Family string `json:"Family"`

	// Art is the assets.LoadImageData key for the picture on the face. **Empty means the default
	// parasite face** — see ArtKey.
	//
	// **The parasite borrowed the worm's placeholder until 2026-09-12**, through one constant in
	// internal/screens whose own note said the day parasites got art it should become a field here
	// rather than a fallback being unpicked. This is that field, and the placeholder is a parasite's
	// own now: two catalogues wearing one picture is two backlogs that cannot be told apart.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this parasite — what the thing
	// *is* and what it is doing, in one sentence. **Nothing in the game reads it**, exactly like
	// Art's own key and a relic's Draw.
	//
	// **Empty means nobody has written one yet**, which — read against an empty Art — is what the
	// parasite sheet reports as the backlog.
	Draw string `json:"Draw"`

	// Change is what class of alteration this parasite makes, from a closed vocabulary resolved by
	// `session.ParseParasiteChange`: `normal` or `upgrade`. **Required on every record.**
	//
	// **A card has a form, an element and an action — and then one upgrade** *(owner's call,
	// 2026-09-09)*. A `normal` parasite moves one of the first three and leaves the upgrade alone:
	// Emberbore paints a card fire, Bulwark turns it into a Guard, and a gold card is still gold
	// afterwards. An `upgrade` parasite writes the one upgrade slot, and whatever was in it is gone.
	//
	// **It is authored rather than derived, and the loader refuses a record that disagrees with its
	// own target.** Every `rider` parasite is an upgrade and nothing else is, so this could have
	// been computed — and a computed field would say nothing, where an authored one is a claim the
	// record makes and the loader checks. It is the same posture `Match` takes in the tutorial
	// script: the thing the author meant, written down where the author is looking.
	Change string `json:"Change"`

	// Target is what this parasite does, from a closed vocabulary resolved by
	// `session.ParseParasiteTarget`: `rider`, `remove`, `swap`, `vitae`.
	//
	// **Closing it is the point**, exactly as with a card's verb and a worm's target. A new target
	// is a Go change plus one place applying it, never something a JSON file can assert into
	// existence.
	Target string `json:"Target"`

	// Rider names which rule a `rider` parasite attaches — `combat.ParseRiderKind` resolves it.
	// Empty for every other target, and refused if one is supplied anyway.
	Rider string `json:"Rider,omitempty"`

	// Value is read against the target: a figure for `rider` and `vitae`, a concept key for
	// `swap`, and nothing at all for `remove`.
	Value string `json:"Value,omitempty"`

	// Count is how many cards of the run this parasite takes. **Zero is a real answer** and means
	// it takes none — a parasite that fills the purse is aimed at the run rather than at a card,
	// and the board piece asks for no target at all.
	Count int `json:"Count"`

	// Text is what the card says it does, in the clipped register the action cards use — the
	// column is about a dozen characters wide, and a `\n` is an authored line break honoured by
	// `cards.WrapText`.
	Text string `json:"Text"`
}

// DefaultParasiteArt is the face a record with no Art of its own draws:
// assets/parasite/default-parasite.png.
//
// **A picture of its own rather than the worm's** *(2026-09-12)*. The two catalogues wore one
// placeholder while neither had art, and the moment either gets some that becomes a page where a
// drawn worm and an undrawn parasite are the same picture — so the backlogs are told apart by
// giving each its own seat. **Keys are not file paths**: LoadImageData files that picture under
// this.
const DefaultParasiteArt = "default-parasite"

// ArtKey is the picture this parasite actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: a parasite is drawn
// in the consumables pane, in the shop and in tools/parasitesheet, and a fallback living in a
// screen is a fallback the review tool does not have.
func (p ParasiteData) ArtKey() string {
	if p.Art == "" {
		return DefaultParasiteArt
	}
	return p.Art
}

// ParasiteFileOrder is every record id in the order data/parasites.json writes them.
//
// **File order rather than ParasiteOrder's sorted keys**, and it is for the review page alone: the
// catalogue is authored in motif order — the five bores together, the four grubs, the metals
// beside each other — and sorting by key throws exactly that away.
//
// **Nothing that decides an outcome may walk this.** What a bucket holds is a shuffle of
// ParasiteOrder, which is sorted for the reason the randomness skill gives; this is a layout.
func ParasiteFileOrder() []string {
	var list []ParasiteData
	if err := json.Unmarshal(parasitesJSON, &list); err != nil {
		panic("Failed to unmarshal parasites.json: " + err.Error())
	}
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, p.ParasiteRecord)
	}
	return out
}

// LoadParasites parses the catalogue into a map keyed by ParasiteRecord.
func LoadParasites() map[string]ParasiteData {
	var list []ParasiteData
	if err := json.Unmarshal(parasitesJSON, &list); err != nil {
		panic("Failed to unmarshal parasites.json: " + err.Error())
	}

	out := make(map[string]ParasiteData, len(list))
	for _, p := range list {
		out[p.ParasiteRecord] = p
	}
	return out
}

// ParasiteOrder is every record, sorted by key.
//
// **Sorted because LoadParasites returns a map and Go randomises that order**, and this one
// decides an outcome rather than a layout: what a bucket holds is a shuffle of this list, so an
// unsorted walk would make a purchase depend on map iteration and take the run's reproducibility
// with it. See the `randomness` skill.
func ParasiteOrder(parasites map[string]ParasiteData) []string {
	names := make([]string, 0, len(parasites))
	for n := range parasites {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
