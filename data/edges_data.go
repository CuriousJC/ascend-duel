package data

// The card edges: **how an upgraded card says so, as a gilded strip down each side of its face.**
//
// A card carries at most one upgrade, and the upgrade is drawn as a strip of authored art just
// inside the border ring on the left edge and, mirrored, on the right. The left edge is the one
// that survives a stacked row — the deck panel overlaps every card over the one before it — so a
// player reading a pile of cards by its left edges can still pick the gold ones out.
//
// **One record per upgrade, plus `default`**, which an upgrade without a record of its own draws
// and whose picture a record with no `Art` borrows. The key is the upgrade's name exactly as
// `systems.Upgrade.String` writes it; `internal/cards` refuses a key naming no upgrade at
// package init, because `data` may not import the vocabulary to check it here.
//
// **Nothing in the rules reads any of it**, the who-consumes-it test every file here answers: this
// is consulted by the card renderer and by `tools/upgradesheet`, and by nothing that resolves a
// round.
//
// **`Width`, `Inset`, `Opacity`, `FadeFrom`, `FadeTo` and `Sides` are draw properties, not rules.** They
// are measured on the hand card (`cards.Hand`) and scaled with it, so the deck panel's half-size
// card draws a half-width strip. They are in the file rather than in Go so the look can be tuned beside the art it is
// tuning, the way a picture and its placement are judged together.

import (
	_ "embed"
	"fmt"
)

//go:embed edges.json
var edgesJSON []byte

// EdgeData is the strip one upgrade draws down the sides of a card.
type EdgeData struct {
	// EdgeRecord is the key: an upgrade's name — `golden`, `wild` — or `default`.
	EdgeRecord string `json:"EdgeRecord"`

	// Art is the assets.LoadImageData key for the strip, filed under `assets/edge/`. The picture is
	// the **left** strip, its left side the card's outside edge; the right strip is the same
	// picture mirrored. **Empty means undrawn**, and the record draws DefaultEdgeArt at its own
	// draw properties — see ArtKey.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given, beside the generic prompt in
	// `docs/art/edge_art_prompt.MD`. **Nothing in the game reads it.**
	Draw string `json:"Draw"`

	// Width is the strip's width in pixels on the hand card. The picture is stretched to it and
	// to the face's full height, so art authored at the strip's own proportions is not distorted.
	Width int `json:"Width"`

	// Inset is how far in from the card's outer edge the strip starts, in pixels on the hand card.
	// **The border width puts it just inside the ring**, which keeps the ring free to say the
	// card's state; zero lays it over the ring.
	Inset int `json:"Inset"`

	// Opacity is how much of the strip covers the card under it, 1 to 100.
	Opacity int `json:"Opacity"`

	// FadeFrom and FadeTo are where the strip fades out across its width, in pixels on the hand
	// card measured from the strip's outer side: full Opacity up to FadeFrom, falling evenly to
	// nothing at FadeTo. **The fade is here rather than in the art** so it can be tuned, and so a
	// generator is never asked for a soft transparent edge. A strip that does not fade sets both
	// to its Width.
	FadeFrom int `json:"FadeFrom"`
	FadeTo   int `json:"FadeTo"`

	// Sides is which edges the strip is drawn on: `left`, or `both`, which mirrors it onto the
	// right. **The left edge is the one a stacked row shows**, so it is the one every record has.
	Sides string `json:"Sides"`
}

// The two values Sides may take. Anything else is refused at load rather than defaulted, since a
// misspelled side would be a strip quietly drawn on one edge where two were asked for.
const (
	EdgeLeft = "left"
	EdgeBoth = "both"
)

// DefaultEdge is the record every upgrade without one of its own draws.
const DefaultEdge = "default"

// DefaultEdgeArt is the picture a record with no Art of its own draws: the default record's.
const DefaultEdgeArt = "edge-default"

// ArtKey is the picture this record draws. **The fallback is here rather than in the renderer**
// for RelicData.ArtKey's reason: the review sheet draws edges too, and a fallback living in one
// drawing is a fallback the other does not have.
func (e EdgeData) ArtKey() string {
	if e.Art == "" {
		return DefaultEdgeArt
	}
	return e.Art
}

// LoadEdges parses the embedded edge catalog into a map keyed by EdgeRecord, refusing a record
// whose draw properties could not draw anything and a catalog with no default.
func LoadEdges() map[string]EdgeData {
	edges := keyed(edgesJSON, "edges.json", func(e EdgeData) string { return e.EdgeRecord })
	if _, ok := edges[DefaultEdge]; !ok {
		panic("edges.json: no " + DefaultEdge + " record")
	}
	for k, e := range edges {
		if e.Width <= 0 || e.Inset < 0 || e.Opacity < 1 || e.Opacity > 100 {
			panic(fmt.Sprintf("edges.json: %s has Width %d, Inset %d, Opacity %d — want a positive "+
				"width, an inset of zero or more and an opacity of 1 to 100", k, e.Width, e.Inset, e.Opacity))
		}
		if e.FadeFrom < 0 || e.FadeFrom > e.FadeTo || e.FadeTo > e.Width {
			panic(fmt.Sprintf("edges.json: %s fades from %d to %d across a strip %d wide — want "+
				"0 <= FadeFrom <= FadeTo <= Width", k, e.FadeFrom, e.FadeTo, e.Width))
		}
		if e.Sides != EdgeLeft && e.Sides != EdgeBoth {
			panic(fmt.Sprintf("edges.json: %s has Sides %q — want %q or %q", k, e.Sides, EdgeLeft, EdgeBoth))
		}
	}
	return edges
}

// EdgeOrder is every record, sorted by key.
func EdgeOrder(edges map[string]EdgeData) []string {
	return sortedKeys(edges)
}
