package data

// The screen art: **the backdrop behind every screen that is not a duel.**
//
// A duel is fought in front of a motif's room, and those rooms are `motifs/<motif>/backdrops.json`.
// Every other screen — the essence choice after a win, the shop, the pages around them — is reached
// after a fight in any realm, so its picture belongs to no motif and no element, and it lives here.
//
// **One record per picture, not per screen.** Several screens share one picture, and `Screens` is
// the list that says which. A screen named by two records is refused, because it would be one
// question with two answers.
//
// **`Draw` is the brief, and `docs/art/screen_art_prompt.MD` is the prompt it is pasted after.**
// The prompt says how every screen backdrop is painted; the record says what this one is of. A
// brief kept here rather than beside the picture is a brief that can be revisited after the picture
// is filed.
//
// **`Anchors` is a copy of the layout, and a test keeps it current.** It is the rectangles a brief
// places something against — a card, a button, or a place the picture puts something — and the
// numbers come from the screen's own layout code, which `data` may not import. So the check is the
// other way round: `internal/screens` holds every anchored record against the function that lays
// the screen out, and a layout that moves fails there rather than leaving a picture painted for
// where things used to be.
//
// **The screen names are checked in `internal/state`**, against `ActiveScreen.String`, for the
// reason `upgrade_art.json`'s keys are checked in `internal/cards`: `data` is the bottom of the graph
// and cannot see the vocabulary it is naming.

import (
	_ "embed"
	"fmt"
	"slices"
)

//go:embed screen_art.json
var screenArtJSON []byte

// ScreenArtData is one screen backdrop.
type ScreenArtData struct {
	// Backdrop is the key.
	Backdrop string `json:"Backdrop"`

	// Name is a label for a person reading the file. Nothing draws it.
	Name string `json:"Name"`

	// Art is the picture's filename stem: `screen-fog` is `screen-fog.jpg`.
	Art string `json:"Art"`

	// Screens are the screens this picture stands behind, each named as `state.ActiveScreen.String`
	// writes it.
	Screens []string `json:"Screens"`

	// Draw is the brief an art generator is given after `docs/art/screen_art_prompt.MD`. **Nothing
	// in the game reads it.**
	Draw string `json:"Draw"`

	// Anchors are the rectangles a brief places something against. Optional.
	Anchors []ScreenAnchor `json:"Anchors"`
}

// ScreenAnchor is one rectangle on the 1920x1080 canvas, in pixels from the top-left.
type ScreenAnchor struct {
	// Name is what a brief calls the anchor.
	Name string `json:"Name"`

	// Kind is what stands there, one of ScreenAnchorKinds: a `card` or a `button` the game lays
	// over the picture and covers it, or a `place` that is the picture's own — a shape painted into
	// that rectangle with nothing over it but what the brief says.
	Kind string `json:"Kind"`

	X int `json:"X"`
	Y int `json:"Y"`
	W int `json:"W"`
	H int `json:"H"`
}

// ScreenAnchorKinds is the closed vocabulary an anchor's Kind is written in.
var ScreenAnchorKinds = []string{"card", "button", "place"}

// The canvas every screen backdrop is painted at, and every ScreenCard is measured on.
const (
	ScreenArtWidth  = 1920
	ScreenArtHeight = 1080
)

// LoadScreenArt parses the embedded catalog into a map keyed by Backdrop.
//
// **It refuses what would quietly mislead a generator or a screen**: a record with no picture or
// no screen, a screen claimed twice, an anchor with no name or two sharing one — a brief names an
// anchor by it — an anchor of no known kind, and one that does not fit on the canvas.
func LoadScreenArt() map[string]ScreenArtData {
	art := keyed(screenArtJSON, "screen_art.json", func(s ScreenArtData) string { return s.Backdrop })
	claimed := map[string]string{}
	for _, key := range sortedKeys(art) {
		rec := art[key]
		if rec.Art == "" {
			panic("screen_art.json: " + key + " has no Art")
		}
		if len(rec.Screens) == 0 {
			panic("screen_art.json: " + key + " names no screen")
		}
		for _, screen := range rec.Screens {
			if other, ok := claimed[screen]; ok {
				panic(fmt.Sprintf("screen_art.json: %s is claimed by both %s and %s", screen, other, key))
			}
			claimed[screen] = key
		}
		named := map[string]bool{}
		for _, a := range rec.Anchors {
			if a.Name == "" || named[a.Name] {
				panic(fmt.Sprintf("screen_art.json: %s has an anchor with no name or a repeated one: %q", key, a.Name))
			}
			named[a.Name] = true
			if !slices.Contains(ScreenAnchorKinds, a.Kind) {
				panic(fmt.Sprintf("screen_art.json: %s anchor %q is of kind %q, which is none of %v",
					key, a.Name, a.Kind, ScreenAnchorKinds))
			}
			if a.W <= 0 || a.H <= 0 || a.X < 0 || a.Y < 0 ||
				a.X+a.W > ScreenArtWidth || a.Y+a.H > ScreenArtHeight {
				panic(fmt.Sprintf("screen_art.json: %s anchor %q does not fit a %dx%d canvas",
					key, a.Name, ScreenArtWidth, ScreenArtHeight))
			}
		}
	}
	return art
}

// ScreenArtOrder is every record, sorted by key.
func ScreenArtOrder(art map[string]ScreenArtData) []string {
	return sortedKeys(art)
}
