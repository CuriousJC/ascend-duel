package data

// The card art: **a picture per playing card per element.**
//
// Every other catalog here is one record per *thing*. This one is one record per *pairing*, and
// that is the whole reason it is a file rather than a field: a Jab and a Cut want different
// pictures, and so do a fire Jab and an ice Jab. Nineteen concepts across five colors is 95
// records, which is more than `duelist_cards.json` could carry as a column without the deck
// becoming unreadable — and none of it is a rule.
//
// **Nothing in the game's rules reads any of it.** This is `Art`, `Draw` and `Family` — the three
// fields every catalog carries and the engine ignores — with no fourth field beside them. A card's
// cost, verb, amount and form all still live in `duelist_cards.json`, which is where a rule
// belongs; this file is consulted by whatever draws a card and by nothing else. Same
// who-consumes-it test every file in this package answers.
//
// **It is keyed by the pairing rather than nested**, so a record is one flat line that can be
// searched for by the name a person would type. `jab-fire` is the key, `Jab` and `fire` are the
// two halves of it written out, and `CardArtKey` is the one place the two are joined — a second
// join somewhere else is how a lookup comes to disagree with a filename.

import (
	_ "embed"
	"strings"
)

//go:embed card_art.json
var cardArtJSON []byte

// CardArtData is the picture for one card in one element.
type CardArtData struct {
	// CardArtRecord is the key: the card's label lowercased, a hyphen, and the element —
	// `jab-fire`, `pulverize-arcane`. Built by CardArtKey rather than by hand at a call site.
	CardArtRecord string `json:"CardArtRecord"`

	// Card is the concept's Label, exactly as duelist_cards.json writes it, and Element is one of
	// the five. **Both are written out beside the key rather than being parsed back out of it**:
	// the key is a string a person types, and a record that can only say what it is about by
	// having its key split on a hyphen is a record nobody can grep for.
	Card    string `json:"Card"`
	Element string `json:"Element"`

	// Family is the motif this picture was authored beside, and the heading it is reviewed under.
	//
	// **The engine ignores it, exactly as it ignores Art and Draw**, and it is authored rather
	// than derived on the terms RelicData.Family records. Here the obvious value is the form and
	// the color — "Stab in fire" — which is what the catalog ships with, and the reason to change
	// one is that a set of cards turned out to share a subject rather than a category.
	Family string `json:"Family"`

	// Art is the assets.LoadImageData key for the picture. **Empty means no picture at all**, and
	// that is deliberate: see DefaultCardArt.
	Art string `json:"Art"`

	// Draw is the subject paragraph the art generator is given for this pairing — what the picture
	// *is*, in a sentence or two. **Nothing in the game reads it**, exactly like a relic's.
	//
	// **Empty means nobody has written one yet.** Read against an empty Art, that is the backlog,
	// and with 95 records it is the largest one in the repo.
	Draw string `json:"Draw"`
}

// DefaultCardArt is what a record with no Art of its own draws, and it is **deliberately empty**
// — which makes this the one catalog in `data/` with no fallback face.
//
// Every other catalog has one because a card with no picture would be a blank rectangle nobody
// could identify. A playing card is the opposite case: it already says everything it is through
// the form mark, the cost ticks, the name and its text, and it has been drawn that way since the
// game had cards at all. So an unauthored record draws **the card exactly as it looks today**,
// which is a working card rather than a placeholder — and a shared placeholder across 95 records
// would be 95 copies of one picture on the table at once.
const DefaultCardArt = ""

// ArtKey is the picture this pairing draws, or "" for none.
//
// **It is here rather than at the call sites** for RelicData.ArtKey's reason: a card is drawn by
// the hand, by the deck overlay and by the review tools, and a fallback living in a screen is a
// fallback the review tools do not have.
func (c CardArtData) ArtKey() string {
	if c.Art == "" {
		return DefaultCardArt
	}
	return c.Art
}

// CardArtKey builds the record key for one card in one element. **The one place the two halves
// are joined**, so a lookup and a filing cannot spell the pairing differently.
func CardArtKey(card, element string) string {
	return strings.ToLower(card) + "-" + strings.ToLower(element)
}

// CardArtFileOrder is every record id in the order data/card_art.json writes them, which is the
// deck's own grid order — three attack forms of five rungs, then the defenses, each walked
// through the five colors.
//
// **Nothing that decides an outcome may walk this.** It is a layout, exactly as RuneFileOrder is;
// a picture never decides anything, so there is no outcome here to protect, and the rule is
// restated so the next reader does not have to work that out.
func CardArtFileOrder() []string {
	return fileOrder(cardArtJSON, "card_art.json", func(c CardArtData) string { return c.CardArtRecord })
}

// LoadCardArt parses the catalog into a map keyed by CardArtRecord.
func LoadCardArt() map[string]CardArtData {
	return keyed(cardArtJSON, "card_art.json", func(c CardArtData) string { return c.CardArtRecord })
}

// CardArtOrder is every record, sorted by key.
//
// **Sorted because LoadCardArt returns a map and Go randomizes that order.** Nothing here decides
// an outcome, so this is weaker than RuneOrder's requirement — but a review sheet that listed its
// rows in a different order every run would be a page nobody could diff against the last one.
func CardArtOrder(art map[string]CardArtData) []string {
	return sortedKeys(art)
}
