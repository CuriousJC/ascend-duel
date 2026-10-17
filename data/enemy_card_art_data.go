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
// than `card_art.json`'s: the five pictures of a flavor share one brief, and what each element
// changes about the vapor is said once for all of them in `docs/art/enemy_card_art_prompt.MD`.

import (
	_ "embed"
	"image"
)

// EnemyCardArtField is the open part of a creature card a flavor's picture is kept inside, in the
// card's own 200x280 pixels: **centred on the card left to right**, as the player's own figures
// are, between the cost ticks (which end at x 28) and the same distance in from the right edge;
// below the form mark, and above the strip holding the badge and the name. `tools/relicart -kind
// enemy-card` fits every picture into it on the way in, and the prompt asks the generator for the
// same box.
var EnemyCardArtField = image.Rect(30, 52, 170, 230)

// EnemyCardWeightShare is how much of EnemyCardArtField a picture of each weight fills, as a
// percentage of its width and height, centred in it. **The install step scales every figure to
// fill its weight's box, up or down**, because a generator's sense of "about half the field" varies
// batch to batch and a ladder read across four batches has to step evenly: light is visibly the
// smallest and medium visibly between. **Heavy and crushing share the whole field**, so what tells
// them apart is what the prompt asks for — mass, density and number — and never size.
var EnemyCardWeightShare = map[string]int{"light": 65, "medium": 82, "heavy": 100, "crushing": 100}

// EnemyCardFieldFor is the box a picture of this weight is fitted into: the field shrunk to the
// weight's share about its own centre. An unknown weight answers the whole field.
func EnemyCardFieldFor(weight string) image.Rectangle {
	pct, ok := EnemyCardWeightShare[weight]
	if !ok {
		return EnemyCardArtField
	}
	f := EnemyCardArtField
	w, h := f.Dx()*pct/100, f.Dy()*pct/100
	x, y := f.Min.X+(f.Dx()-w)/2, f.Min.Y+(f.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h)
}

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

	// Art is the picture family stem, and the face drawn is `<Art>-<weight>-<element>`, one per
	// weight per element. **Empty means no picture yet**, and the card draws as it always has — see
	// DefaultCardArt, whose argument holds here too.
	Art string `json:"Art"`

	// Weights is every weight of this flavor drawn in all five elements, in EnemyCardWeights
	// order. **Written by `tools/relicart -kind enemy-card`**, which adds a weight only once its
	// fifth element is on disk; a weight not listed draws its cards with no picture, so a realm
	// never shows a hit at one weight in four elements and blank in the fifth.
	Weights []string `json:"Weights"`

	// Draw is the subject paragraph the generator is given, pasted after
	// docs/art/enemy_card_art_prompt.MD — the blow at its medium weight. **Nothing in the game reads
	// it.** Empty means nobody has written one.
	Draw string `json:"Draw"`

	// Ladder is what changes about the subject at the other three weights, one sentence each, keyed
	// by weight and read beside Draw. **Nothing in the game reads it.**
	Ladder map[string]string `json:"Ladder"`
}

// EnemyCardWeights is the ladder a creature card's picture is chosen on, lightest first. **A
// closed list**: a fifth weight is five pictures per flavor, a design decision, never an edit.
var EnemyCardWeights = []string{"light", "medium", "heavy", "crushing"}

// EnemyCardWeight is the weight of a creature card dealing amount — its multiplier on the
// creature's DMG, as a percentage. **Derived, never authored**, so a card retuned across a line
// changes picture with nothing edited.
//
// The lines fall in the gaps the roster leaves: a 1 AP card deals about half, a 2 AP card about
// one, a 3 AP card two or more — and a creature's finisher at three or more is crushing, so two
// 3 AP hits on one creature still look different.
func EnemyCardWeight(amount int) string {
	switch {
	case amount < 75:
		return "light"
	case amount < 150:
		return "medium"
	case amount < 275:
		return "heavy"
	}
	return "crushing"
}

// HasWeight reports whether this flavor's pictures are drawn at a weight.
func (f EnemyCardArtData) HasWeight(weight string) bool {
	for _, w := range f.Weights {
		if w == weight {
			return true
		}
	}
	return false
}

// ArtKey is the picture this flavor draws at one weight in one element, or "" for none yet.
func (f EnemyCardArtData) ArtKey(weight, element string) string {
	if f.Art == "" || !f.HasWeight(weight) {
		return ""
	}
	return f.Art + "-" + weight + "-" + element
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
		for _, w := range f.Weights {
			if !isEnemyWeight(w) {
				panic("enemy_card_art.json: " + key + " lists the weight " + w + ", which is not one of light, medium, heavy, crushing")
			}
		}
		for w := range f.Ladder {
			if !isEnemyWeight(w) || w == "medium" {
				panic("enemy_card_art.json: " + key + " has a Ladder line for " + w + "; Draw is the medium, and the others are light, heavy and crushing")
			}
		}
	}
	return out
}

// EnemyCardArtOrder is every flavor, sorted by key. Sorted because LoadEnemyCardArt returns a map.
func EnemyCardArtOrder(art map[string]EnemyCardArtData) []string {
	return sortedKeys(art)
}

func isEnemyWeight(weight string) bool {
	for _, w := range EnemyCardWeights {
		if w == weight {
			return true
		}
	}
	return false
}

func isEnemyForm(form string) bool {
	for _, f := range EnemyForms {
		if f == form {
			return true
		}
	}
	return false
}
