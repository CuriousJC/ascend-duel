package data

// The worms: **what the player may do to their deck between fights.**
//
// It is the card language's shape applied to a different subject. A card record says what a card
// *does* when it is played; a worm record says what it *changes* about a card that already
// exists — a target and, where the target needs one, a new value.
//
// **What is deliberately not here.** Which worms are offered, how many, and what they cost are
// not fields on a record: the offer is two drawn at random and nothing is bought yet. Whichever
// of those becomes real is a field beside Target when it does, and not before — the mistake
// `CostTier` was, which declared a rules vocabulary in JSON ahead of the rules, is the one to
// keep not making.
//
// **The rules do not read this file**, and cannot: a worm is applied to the *run's* deck, which
// is `internal/session`. That is the same test every file here answers — who consumes it — and
// it is why the parsing and the validation live over there rather than in `internal/combat`.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed worms.json
var wormsJSON []byte

// WormData is one alteration the player can be offered.
type WormData struct {
	// WormRecord is the key, and what anything holding a worm stores. Kebab-case, like a relic's.
	WormRecord string `json:"WormRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Family is the motif this worm belongs to — the block of siblings it was authored beside,
	// and the heading it is reviewed under on the worm sheet.
	//
	// **The engine ignores it, exactly as it ignores Art and Draw.** It groups the review page and
	// nothing else reads it; a worm with no Family still loads and is still offered.
	//
	// **It is the relic catalogue's field brought over** *(owner's call, 2026-09-12)*. The same
	// argument holds and the same caveat does: nearly every value here is implied by the record's
	// own Target and Value — the five elemental worms are one block because they all recolour — so
	// this is legibility for whoever is authoring rather than a fact the file knows and the rules
	// do not. It can go quietly out of date when a worm is retargeted, and no test fails, so
	// re-read the block when you change what a worm does.
	Family string `json:"Family"`

	// Art is the assets.LoadImageData key for the picture on the face. **Empty means the default
	// worm face** — see ArtKey.
	//
	// **It became a field on 2026-09-12**, having been the one constant `screens.wormArtKey`. The
	// note on that constant said the day worms got art it should become a field appearing here
	// rather than a fallback being unpicked, and this is that day: the fallback is `ArtKey`, which
	// is the shape `RelicData.ArtKey` already had, so a worm with no art of its own draws the
	// placeholder and one with art draws it.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this worm — what the thing
	// *is* and what it is doing, in one sentence. **Nothing in the game reads it**, exactly like
	// Art's own key and a relic's Draw.
	//
	// **Empty means nobody has written one yet**, which — read against an empty Art — is what the
	// worm sheet reports as the backlog.
	Draw string `json:"Draw"`

	// Target is which aspect of a card this worm changes. A closed vocabulary, resolved by
	// `session.ParseWormTarget`: `element`, `remove`, `duplicate`.
	//
	// **Closing it is the point**, exactly as with a card's verb. The set is short because
	// `combat.Card` is a concept plus an element and the element is the only per-instance field —
	// a worm that changed a card's *cost* would be changing the concept, and so every copy of
	// that card in the deck. Making cost per-card is a field on `combat.Card` and a change at
	// every `Cost()` call site, which is a price worth charging deliberately rather than
	// discovering because a JSON file asked for it.
	Target string `json:"Target"`

	// Value is the new value, read against the target. An element name for `element`; empty for
	// the targets that need none, and refused if one is supplied anyway.
	Value string `json:"Value,omitempty"`

	// Text is what the card says it does, in the same clipped register the action cards use —
	// the column is about a dozen characters wide.
	//
	// **A `\n` is an authored line break** *(2026-08-23)*, honoured by `cards.WrapText` before the
	// width is measured and split back into lines by the tooltip. The four elemental worms carry
	// one, because they differ only in the element they name and FIRE sits comfortably on the line
	// where LIGHTNING all but fills it — so left to the measurer the four read as four layouts of
	// the same card. A break is the author saying where it goes; it can only ever add a line,
	// since a too-wide authored line still wraps.
	Text string `json:"Text"`
}

// DefaultWormArt is the face a record with no Art of its own draws: assets/worm/default-worm.png.
//
// **Keys are not file paths** — LoadImageData files that picture under this, which is what a
// lookup has to spell. Writing the filename instead is why the first version of the worm card drew
// nothing and logged `no artwork named "default-relic"`.
const DefaultWormArt = "default-worm"

// ArtKey is the picture this worm actually draws: its own if it has one, the default otherwise.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: a worm is drawn in
// the reward screen and in tools/wormsheet, and a fallback living in a screen is a fallback the
// review tool does not have — which is exactly how a sheet comes to disagree with the game.
func (w WormData) ArtKey() string {
	if w.Art == "" {
		return DefaultWormArt
	}
	return w.Art
}

// WormFileOrder is every record id in the order data/worms.json writes them.
//
// **File order rather than WormOrder's sorted keys**, and it is for the review page alone: the
// catalogue is authored in motif order — the five recolours together, the two that resize a card
// beside each other — and sorting by key throws exactly that away. It is as deterministic as
// sorted order and carries more.
//
// **Nothing that decides an outcome may walk this.** The offer is a shuffle of WormOrder, which is
// sorted for the reason the randomness skill gives; this is a layout.
func WormFileOrder() []string {
	var list []WormData
	if err := json.Unmarshal(wormsJSON, &list); err != nil {
		panic("Failed to unmarshal worms.json: " + err.Error())
	}
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, w.WormRecord)
	}
	return out
}

// LoadWorms parses the catalogue into a map keyed by WormRecord.
func LoadWorms() map[string]WormData {
	var list []WormData
	if err := json.Unmarshal(wormsJSON, &list); err != nil {
		panic("Failed to unmarshal worms.json: " + err.Error())
	}

	out := make(map[string]WormData, len(list))
	for _, w := range list {
		out[w.WormRecord] = w
	}
	return out
}

// WormOrder is every record, sorted by key.
//
// **Sorted because LoadWorms returns a map and Go randomises that order**, and this one decides
// an outcome rather than a layout: the offer is a shuffle of this list, so an unsorted walk would
// make which worms you are offered depend on map iteration and take the run's reproducibility
// with it. See the `randomness` skill.
func WormOrder(worms map[string]WormData) []string {
	names := make([]string, 0, len(worms))
	for n := range worms {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
