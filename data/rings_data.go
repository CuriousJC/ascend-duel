package data

// The rings: what the player can equip between fights.
//
// **Added 2026-08-11, and it is a layout sketch rather than a mechanic.** Nothing reads
// Element or Text yet — the combat screen draws these three as cards in the ring pane and
// that is all. The file exists first so the set can be *seen and extended* while the rules
// are still being decided, which is the same order the enemy roster arrived in.
//
// **What is deliberately not here.** MECHANICS.md decides what a ring does — a per-element
// discount, or a flip mapping one element onto another across the whole deck — and both need
// `element` to cross into `internal/combat` before they can be written down as anything a
// loader could check. So Element below says *which element this ring is about* and no more;
// when the rules land, the discount or the flip becomes a field beside it and this comment
// goes. Do not invent a rules vocabulary in JSON ahead of the rules, which is the mistake
// `CostTier` in the card lists is checked against rather than trusted.
//
// **Art is an assets key, not a path**, like every other named asset — see assets/embed.go.
// It has to be a key `LoadImageData` hands back rather than one `LoadAssets` does, because a
// ring's artwork is drawn *into* a card by internal/cards, which has no graphics context.

import (
	_ "embed"
	"encoding/json"
	"sort"
)

//go:embed rings.json
var ringsJSON []byte

// RingData is one ring.
type RingData struct {
	// RingRecord is the key, and what anything holding a ring stores. Kebab-case, matching
	// the art filenames rather than the display name — a record key that is a sentence is a
	// key nobody can type twice the same way.
	RingRecord string `json:"RingRecord"`

	// Name is what is written across the top of the card.
	Name string `json:"Name"`

	// Art is the assets.LoadImageData key for the picture on the face. An empty or unknown
	// name draws a ring with no artwork and logs once — the same choice the enemy portraits
	// make, and for the same reason: a card with a hole in it gets reported, a game that
	// refuses to start over a missing picture is worse.
	Art string `json:"Art"`

	// Element is which element this ring is about — the one it would discount, or the one a
	// flip would land on. It is documentation until elements reach internal/combat; nothing
	// reads it today.
	//
	// **It does not colour the card.** A ring's border is pink whatever its element, because
	// the one thing that must never happen is reaching for a ring thinking it is a card you
	// can play. See cards.Ring.
	Element string `json:"Element"`

	// Text is one line saying what the ring does, for the long press that does not exist yet.
	// Written now because it is the thing whoever adds a ring will want to write down, and a
	// field added later is a field every existing entry is missing.
	Text string `json:"Text"`
}

// LoadRings parses the embedded ring list into a map keyed by RingRecord.
func LoadRings() map[string]RingData {
	var list []RingData
	if err := json.Unmarshal(ringsJSON, &list); err != nil {
		panic("Failed to unmarshal our RingData: " + err.Error())
	}

	out := make(map[string]RingData, len(list))
	for _, r := range list {
		out[r.RingRecord] = r
	}
	return out
}

// RingOrder is every record, sorted by key.
//
// **Sorted because LoadRings returns a map and Go randomises that order**, exactly like
// EnemyOrder. The ring pane draws whatever this walks, so map order would deal a different
// row of rings every launch — a determinism breach that would look like a bug in the layout.
//
// By key rather than by name or element: it is the one field guaranteed unique, and a sort
// on something that can tie is a sort that can still shuffle.
func RingOrder(rings map[string]RingData) []string {
	keys := make([]string, 0, len(rings))
	for k := range rings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
