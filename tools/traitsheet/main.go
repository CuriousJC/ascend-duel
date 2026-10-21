// Command traitsheet draws every trait — the relic traits and the card traits — beside what each one
// says, what it does, and how often a run meets it.
//
//	go run ./tools/traitsheet
//
// It exists for the reason tools/goodsheet does. A trait turns up on one shelf relic in twenty or one
// bundled card in twenty, so meeting them all in a launched game is a great many shops; this puts the
// whole of both catalogs on one page.
//
// # It is a report, not a drawing-board
//
// It reads data/relic_traits.json and data/card_traits.json through internal/session and
// internal/combat, so both are *validated* before anything is drawn — a trait this page refuses is a
// trait the game refuses to start with.
//
// # What to look at
//
// **The odds, as a player meets them.** A trait's Percent is per shelf seat or per bundled card, which
// is not the question a player asks; the page also prints how often one shop shows at least one, and
// how often a bundle of each size holds at least one.
//
// **What it does, decoded.** A relic trait's Slot, Lasts and Rules and a card trait's Counted and
// Landings, as the file writes them — the sentence it carries can then be read against the rule.
//
// **What a mirrored card looks like.** The echo is drawn into the face, so it is drawn here: the same
// card plain and mirrored. An effortless card is drawn aloft in the hand, which is motion and needs a
// window; the animation gallery is where that is looked at.
//
// **No picture per trait, and no backlog.** A trait is a property of a wearing, drawn on the relic or
// card carrying it, so there is nothing of its own to paint; the Compendium's trait book shows each one
// on a sample.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/traitsheet/` and **committed**, on the same
// terms as every other sheet.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue the cards are actually drawn on.
const ground = "#a8bcd4"

// shelfSize is screens.shelfSize, how many relics stand on one shelf — restated, since a tool cannot
// import a package that links a window.
const shelfSize = 3

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "traitsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()
	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

// plate is one trait on the page.
type plate struct {
	Key, Name, Text, Kind string
	Percent               int
	Odds                  []string
	Does                  []string
}

// page is everything the template reads.
type page struct {
	Ground       string
	Relic, Card  []plate
	Mirror       []cell
	ShelfAnyText string
}

// cell is one drawn card with a caption.
type cell struct{ Image, Label string }

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	p := page{Ground: ground}
	none := 1.0
	for _, t := range session.ShelfTraits() {
		r := t.Record
		pl := plate{
			Key: r.RelicTrait, Name: r.Name, Text: r.Text, Kind: "relic", Percent: t.Percent,
		}
		if t.Percent > 0 {
			pl.Odds = []string{
				fmt.Sprintf("%d%% of shelf relics", t.Percent),
				fmt.Sprintf("%s of shops show at least one", atLeastOne(t.Percent, shelfSize)),
			}
			none *= 1 - float64(t.Percent)/100
		} else {
			pl.Odds = []string{"never sold"}
		}
		pl.Does = relicDoes(r)
		p.Relic = append(p.Relic, pl)
	}
	p.ShelfAnyText = fmt.Sprintf("%.1f%%", (1-none)*100)

	for _, t := range session.CardTraitCatalog() {
		r := t.Record
		pl := plate{
			Key: r.CardTrait, Name: r.Name, Text: r.Text, Kind: "card", Percent: t.Percent,
		}
		if t.Percent > 0 {
			pl.Odds = []string{fmt.Sprintf("%d%% of cards in an action bundle", t.Percent)}
			for _, g := range session.Goods() {
				if g.Contains == session.ContentsCards {
					pl.Odds = append(pl.Odds, fmt.Sprintf("%s of %s hold at least one",
						atLeastOne(t.Percent, g.Size), strings.ToLower(g.Name)+"s"))
				}
			}
		} else {
			pl.Odds = []string{"never in a bundle"}
		}
		pl.Does = cardDoes(r)
		p.Card = append(p.Card, pl)
	}

	if p.Mirror, err = mirrorPair(dir, faces); err != nil {
		return err
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, p); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Printf("wrote %s — %d relic traits and %d card traits; %s of shelf relics carry a trait\n",
		out, len(p.Relic), len(p.Card), p.ShelfAnyText)
	return nil
}

// atLeastOne is the chance that n independent rolls at pct hit at least once, as a percent.
func atLeastOne(pct, n int) string {
	return fmt.Sprintf("%.1f%%", (1-math.Pow(1-float64(pct)/100, float64(n)))*100)
}

// relicDoes is a relic trait's properties and rules as the file writes them.
func relicDoes(r data.RelicTraitData) []string {
	var out []string
	if r.Slot != nil && !*r.Slot {
		out = append(out, "Slot: false — takes no relic slot")
	}
	if r.LastsFight() {
		out = append(out, "Lasts: fight — gone when the fight ends")
	}
	for _, rule := range r.Rules {
		for _, e := range rule.Then {
			line := fmt.Sprintf("When %s: %s", rule.When, e.Do)
			if e.Element != "" {
				line += " " + e.Element
			}
			if e.Amount != 0 {
				line += fmt.Sprintf(" x%d", e.Amount)
			}
			out = append(out, line)
		}
	}
	return out
}

// cardDoes is a card trait's properties as the file writes them.
func cardDoes(r data.CardTraitData) []string {
	var out []string
	if r.Counted != nil && !*r.Counted {
		out = append(out, "Counted: false — not one of the turn's five")
	}
	if r.Landings > 1 {
		out = append(out, fmt.Sprintf("Landings: %d — lands %d times", r.Landings, r.Landings))
	}
	return out
}

// mirrorPair is one real playing card drawn plain and mirrored, so the echo is judged on a face the
// game deals rather than on a mock-up.
func mirrorPair(dir string, f *cards.Faces) ([]cell, error) {
	pool := session.ActionPool()
	if len(pool) == 0 {
		return nil, nil
	}
	c := pool[0]
	art := data.LoadCardArt()[data.CardArtKey(c.Label(), c.Element.String())]
	img, err := decode(art.ArtKey())
	if err != nil {
		img = nil
	}
	var out []cell
	for _, mirrored := range []bool{false, true} {
		spec := cards.Spec{
			Name: c.Label(), Form: formOf(c), Cost: c.Cost(), Element: elementOf(c),
			Art: img, Enabled: true, Mirrored: mirrored,
		}
		name, label := "mirror-plain.png", c.Label()+" — plain"
		if mirrored {
			name, label = "mirror-mirrored.png", c.Label()+" — mirrored"
		}
		file, err := write(dir, f, spec, cards.Hand, name)
		if err != nil {
			return nil, err
		}
		out = append(out, cell{Image: file, Label: label})
	}
	return out, nil
}

// formOf and elementOf are ui.form and ui.ArtFor, restated: internal/ui links a window.
func formOf(c combat.Card) cards.Form {
	switch c.Form() {
	case combat.FormStab:
		return cards.FormStab
	case combat.FormSlash:
		return cards.FormSlash
	case combat.FormCrush:
		return cards.FormCrush
	case combat.FormDefend:
		return cards.FormDefend
	}
	return cards.FormNone
}

func elementOf(c combat.Card) cards.Element {
	switch c.Element {
	case combat.Fire:
		return cards.Fire
	case combat.Ice:
		return cards.Ice
	case combat.Lightning:
		return cards.Lightning
	case combat.Earth:
		return cards.Earth
	case combat.Arcane:
		return cards.Arcane
	}
	return cards.Basic
}

func decode(key string) (image.Image, error) {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return nil, fmt.Errorf("%q is in no embed", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

func write(dir string, f *cards.Faces, s cards.Spec, st cards.Style, name string) (string, error) {
	img, err := cards.Render(s, st, f)
	if err != nil {
		return "", err
	}
	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer out.Close()
	return name, png.Encode(out, img)
}
