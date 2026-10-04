package data

// The decks: **what a duelist plays from, and how you tell whose deck is on the table.**
//
// A deck is a back and the rules that come with holding it. A duelist names one in
// `data/duelists.json`, and the back is drawn on every face-down card that duelist owns — the
// draw pile, and every card on its way out of it.
//
// **The back is authored art, one full-bleed picture per deck**, at the card's own 200x280 under
// `assets/deck/`, and the game reduces it for the smaller pile the between-fights screens draw.
// `Draw` is the brief, pasted after `docs/art/deck_art_prompt.MD`. A deck with no `Art` draws the
// plain dark back `internal/cards` paints in code, so a deck can be authored before its picture.
//
// **`Discards` is the first rule a deck carries**: how many times a round the hand can be thrown
// back. Tonics add to it — see `session.Session.Discards`, which takes it as the base.
//
// **Read by the screens and the session, never by `internal/combat`.** Discarding is the combat
// screen's mechanic rather than the resolver's, and the back is a picture.

import (
	_ "embed"
	"fmt"
)

//go:embed decks.json
var decksJSON []byte

// DeckData is one deck a duelist can play from.
type DeckData struct {
	// DeckRecord is the key a duelist names in its `Deck` field.
	DeckRecord string `json:"DeckRecord"`

	// Name is what the deck is called on screen.
	Name string `json:"Name"`

	// Discards is how many discards a round of a fight allows before any tonic, refilled at the
	// end of each round. One press costs one discard however many cards were selected.
	Discards int `json:"Discards"`

	// Art is the back's filename stem under `assets/deck/`. **Empty means undrawn**, and the back is
	// the plain one drawn in code — see ArtKey.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given, beside the generic prompt in
	// `docs/art/deck_art_prompt.MD`. **Nothing in the game reads it.**
	Draw string `json:"Draw"`
}

// DeckArtPrefix is what the image map files this family under: `assets/deck/rainbow-gem.png` is
// the key `deck-rainbow-gem`. The prefix is the asset map's, not the record's, for
// UpgradeArtPrefix's reason — the map is flat, and a deck named after a gem is a name a relic or an
// essence could also take.
const DeckArtPrefix = "deck-"

// ArtKey is the image map key this deck's back draws, or "" for a deck with no picture yet.
//
// **There is no default picture**, for the playing cards' reason: a back with no art still has a
// back to draw — the plain one — and one shared placeholder would put every undrawn deck on the
// table looking alike while claiming to be a picture.
func (d DeckData) ArtKey() string {
	if d.Art == "" {
		return ""
	}
	return DeckArtPrefix + d.Art
}

// LoadDecks parses the embedded catalog into a map keyed by DeckRecord.
//
// **A negative discard count is refused**, because it is a number nothing can mean. Zero is legal
// and is a deck that cannot steer its hand at all — a design, not a typo.
func LoadDecks() map[string]DeckData {
	decks := keyed(decksJSON, "decks.json", func(d DeckData) string { return d.DeckRecord })
	for _, k := range DeckOrder(decks) {
		if d := decks[k]; d.Discards < 0 {
			panic(fmt.Sprintf("decks.json: %s has %d discards", k, d.Discards))
		}
	}
	return decks
}

// DeckOrder is every deck, sorted by key.
func DeckOrder(decks map[string]DeckData) []string {
	return sortedKeys(decks)
}
