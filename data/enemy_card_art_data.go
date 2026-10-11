package data

// The enemy card art: **a picture per attack flavor per element**, shared by every creature card
// that names the flavor.
//
// A creature's cards are authored per record — two creatures of one motif are two fights — but
// their *pictures* are not. A wolf's Bite and a lion's Bite are different cards at different
// figures, and they are one picture: `stab-bite`, drawn once in each of the five elements. So
// every card in `motifs/*/motif.json` names a `Flavor` from this file, the way a player card's
// picture is found in `card_art.json`, and the roster's hundreds of cards come down to a short
// list of pictures.
//
// **A flavor belongs to exactly one form**, and a card naming it must carry that form — refused at
// load in checkRecord. So the corner mark and the picture cannot disagree about whether a bite is
// a stab.
//
// **Nothing in the game's rules reads any of it.** A card's form is a rule and lives on the card in
// the motif file; this catalog is `Name`, `Art` and `Draw`, consulted by whatever draws an enemy
// card and by the review sheet. Same who-consumes-it test every file in this package answers.
//
// **One record per flavor, with the element in the filename**, the motif records' shape rather
// than `card_art.json`'s: the five pictures of a flavor share one brief and differ only in the
// color of the vapor, which `docs/art/enemy_card_art_prompt.MD` says once for all of them.

import (
	_ "embed"
)

//go:embed enemy_card_art.json
var enemyCardArtJSON []byte

// EnemyForms is every form a creature card may carry. **Defend is not one**: every creature deck
// is pure attack, and a creature raises no shields.
var EnemyForms = []string{"stab", "slash", "crush"}

// EnemyCardArtData is one flavor: the picture every creature card naming it wears.
type EnemyCardArtData struct {
	// Flavor is the key, `<form>-<flavor>` — `stab-bite`, `crush-grip`. A motif card's `Flavor`
	// field names one.
	Flavor string `json:"Flavor"`

	// Form is the form every card wearing this picture carries. One of EnemyForms.
	Form string `json:"Form"`

	// Name is what the review sheet calls the flavor. **Never drawn on a card**: a card shows its
	// own label, and two creatures' Bites keep their two names over one picture.
	Name string `json:"Name"`

	// Art is the picture family stem, and the face drawn is `<Art>-<element>`, one per element.
	// **Empty means no picture yet**, and the card draws as it always has — see DefaultCardArt,
	// whose argument holds here too.
	Art string `json:"Art"`

	// Draw is the subject paragraph the generator is given, pasted after
	// docs/art/enemy_card_art_prompt.MD. **Nothing in the game reads it.** Empty means nobody has
	// written one.
	Draw string `json:"Draw"`
}

// ArtKey is the picture this flavor draws in one element, or "" for none yet.
func (f EnemyCardArtData) ArtKey(element string) string {
	if f.Art == "" {
		return ""
	}
	return f.Art + "-" + element
}

// EnemyCardArtFileOrder is every flavor in the order the file writes them: stab, slash, crush.
//
// **Nothing that decides an outcome may walk this.** It is a layout for the review sheet.
func EnemyCardArtFileOrder() []string {
	return fileOrder(enemyCardArtJSON, "enemy_card_art.json", func(f EnemyCardArtData) string { return f.Flavor })
}

// LoadEnemyCardArt parses the catalog into a map keyed by Flavor, refusing a form that is not one
// of EnemyForms and a key that does not begin with its own form.
func LoadEnemyCardArt() map[string]EnemyCardArtData {
	out := keyed(enemyCardArtJSON, "enemy_card_art.json", func(f EnemyCardArtData) string { return f.Flavor })
	for key, f := range out {
		if !isEnemyForm(f.Form) {
			panic("enemy_card_art.json: " + key + " has the form " + f.Form + ", and a creature card is stab, slash or crush")
		}
		if want := f.Form + "-"; len(key) <= len(want) || key[:len(want)] != want {
			panic("enemy_card_art.json: " + key + " is a " + f.Form + " flavor and should be keyed " + want + "<flavor>")
		}
	}
	return out
}

// EnemyCardArtOrder is every flavor, sorted by key. Sorted because LoadEnemyCardArt returns a map.
func EnemyCardArtOrder(art map[string]EnemyCardArtData) []string {
	return sortedKeys(art)
}

func isEnemyForm(form string) bool {
	for _, f := range EnemyForms {
		if f == form {
			return true
		}
	}
	return false
}
