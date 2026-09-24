package main

import (
	"fmt"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
)

// The catalogs this tool can review, and how each one becomes a row on the page.
//
// **One table rather than one tool per family.** The six art families differ in three things —
// which JSON names the pictures, which card style draws them, and what else is on the face — and
// nothing else, so a tool per family would be six copies of the page and the picking. A new
// family is one entry here.
//
// **A subject carries the card without its picture.** The Art field is the one thing the caller
// varies, which is what makes a row a fair comparison: everything else is held constant across
// the columns, so anything the eye catches is the art.

// subject is one record of a catalog: the picture it names, the card it is drawn into, and the
// words the page prints beside it.
type subject struct {
	// Key is the record id in its own catalog — `jab-fire`, `emberring`. What the page prints
	// so a decision can be written down against the file it is about.
	Key string

	// Stem is the picture's filename stem, which is the name the file has in every set's
	// directory. Usually the key and deliberately not assumed to be: a record names its own
	// picture, and two records may share one.
	Stem string

	// Group is the heading this record is filed under — the catalog's own Family, in the file's
	// own order. A page of ninety-five cards with no headings is a page nobody can find a card in.
	Group string

	// Caption is the stat line under the row: what the card is, in the terms its catalog uses.
	Caption string

	// Spec is the card with no Art. Style is what draws it.
	Spec  cards.Spec
	Style cards.Style
}

// catalog is one art family: where the installed pictures live, and what is in it.
type catalog struct {
	// Name is the word the -catalog flag takes, and the prefix a candidate directory under
	// artreview/ is found by — `relic` finds `artreview/relic-warm` as the set `warm`.
	Name string

	// Dir is the directory holding the pictures the game is shipping today, offered as the
	// `current` set so a review always has something to compare against.
	Dir string

	// File is the catalog this reads, named for the error message and the page header.
	File string

	// Subjects is the records, in the order the page should walk them.
	Subjects func() ([]subject, error)
}

// catalogs is every family that has art, keyed by the word the flag takes.
//
// **assets/enemy and assets/boss are deliberately absent.** Those are licensed creature
// portraits rather than generated pictures — they arrive once, with a license, and are not a
// thing anybody produces three candidate versions of. Add them the day that stops being true.
var catalogs = map[string]*catalog{
	"card":    {Name: "card", Dir: "assets/card", File: "data/card_art.json", Subjects: cardSubjects},
	"relic":   {Name: "relic", Dir: "assets/relic", File: "data/relics.json", Subjects: relicSubjects},
	"essence": {Name: "essence", Dir: "assets/essence", File: "data/essences.json", Subjects: essenceSubjects},
	"rune":    {Name: "rune", Dir: "assets/rune", File: "data/runes.json", Subjects: runeSubjects},
	"stone":   {Name: "stone", Dir: "assets/stone", File: "data/stones.json", Subjects: stoneSubjects},
	"other":   {Name: "other", Dir: "assets/other", File: "data/potions.json and data/goods.json", Subjects: otherSubjects},
}

// catalogNames is the flag's vocabulary, sorted, for the usage line and the error message.
func catalogNames() []string {
	out := make([]string, 0, len(catalogs))
	for name := range catalogs {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// cardSubjects is the playing cards: one record per concept per element, drawn at hand size with
// the picture under the type.
//
// **The costs, forms and amounts come from data/duelist_cards.json**, which makes this a report
// rather than the drawing-board tools/cardsheet is: a picture exists for a card that exists.
func cardSubjects() ([]subject, error) {
	art := data.LoadCardArt()
	concepts := map[string]data.CardData{}
	for _, c := range data.LoadDuelistCards() {
		concepts[c.Label] = c
	}

	var out []subject
	for _, key := range data.CardArtFileOrder() {
		rec, ok := art[key]
		if !ok {
			return nil, fmt.Errorf("card_art.json: %s is in the file order and not in the catalog", key)
		}
		c, ok := concepts[rec.Card]
		if !ok {
			return nil, fmt.Errorf("%s names the card %q, which duelist_cards.json does not have", key, rec.Card)
		}
		el, ok := cards.ParseElement(rec.Element)
		if !ok {
			return nil, fmt.Errorf("%s names the element %q, which is not one of the five", key, rec.Element)
		}

		spec := cards.Spec{
			Name: c.Label, Form: formOf(c.Form), Cost: c.Cost,
			Element: el, Enabled: true,
		}
		if c.Verb == "shield" {
			spec.Shields = c.Amount
		} else {
			spec.Badge, spec.BadgePct = multiplier(c.Amount), c.Amount
		}

		// **Grouped by the concept rather than by Family**, which is the one place this tool
		// departs from the catalog's own heading. A playing card's Family is its form *and* its
		// element — "Stab in fire" — and the file order walks each concept through the five
		// colors, so Family changes on every record and the page would be ninety-five headings
		// over one card each. The concept is the group the file order actually has: five colors
		// of one card, side by side, which is also the comparison worth making.
		out = append(out, subject{
			Key: key, Stem: rec.Art, Group: fmt.Sprintf("%s — %s, %d AP", c.Label, c.Form, c.Cost),
			Caption: fmt.Sprintf("%s · %s", c.Label, rec.Element),
			Spec:    spec, Style: cards.Hand,
		})
	}
	return out, nil
}

// relicSubjects is the relics, in relics.json's own motif order — which is what tools/relicsheet
// walks, so the two pages read the same way.
//
// **The counter disc is not drawn**, where the relic sheet draws one. A counter is a fact about a
// run rather than about the picture, and a figure invented for a review page would be the sheet
// lying about the game in the one direction that matters.
func relicSubjects() ([]subject, error) {
	relics := data.LoadRelics()
	var out []subject
	for _, key := range data.RelicFileOrder() {
		rec, ok := relics[key]
		if !ok {
			return nil, fmt.Errorf("relics.json: %s is in the file order and not in the catalog", key)
		}
		out = append(out, subject{
			Key: key, Stem: rec.Art, Group: rec.Family,
			Caption: fmt.Sprintf("%s · %s", rec.Name, rec.Rarity),
			Spec: cards.Spec{
				Name: rec.Name, Element: cards.Relic, Rarity: rec.Rarity, Enabled: true,
			},
			Style: cards.RelicStyle,
		})
	}
	return out, nil
}

// essenceSubjects is the essences, which print their sentence across the lower half of the
// picture — so the scrim and the type are part of what is being judged here more than anywhere
// else in the game.
func essenceSubjects() ([]subject, error) {
	essences := data.LoadEssences()
	var out []subject
	for _, key := range data.EssenceFileOrder() {
		rec, ok := essences[key]
		if !ok {
			return nil, fmt.Errorf("essences.json: %s is in the file order and not in the catalog", key)
		}

		// An essence that sets an element writes it in Value, and that is what the card's
		// border takes. Every other target is elementless and draws Basic, exactly as the rune
		// and stone cards do.
		el := cards.Basic
		if rec.Target == "element" {
			if parsed, ok := cards.ParseElement(rec.Value); ok {
				el = parsed
			}
		}

		out = append(out, subject{
			Key: key, Stem: rec.Art, Group: rec.Family,
			Caption: rec.Name,
			Spec: cards.Spec{
				Name: rec.Name, Element: el, Text: rec.Text,
				Highlights: cards.ElementHighlights(rec.Text), Enabled: true,
			},
			Style: cards.EssenceStyle,
		})
	}
	return out, nil
}

func runeSubjects() ([]subject, error) {
	runes := data.LoadRunes()
	var out []subject
	for _, key := range data.RuneFileOrder() {
		rec, ok := runes[key]
		if !ok {
			return nil, fmt.Errorf("runes.json: %s is in the file order and not in the catalog", key)
		}
		out = append(out, subject{
			Key: key, Stem: rec.Art, Group: rec.Family,
			Caption: rec.Name,
			Spec: cards.Spec{
				Name: rec.Name, Element: cards.Basic, Text: rec.Text,
				Highlights: cards.ElementHighlights(rec.Text), Enabled: true,
			},
			Style: cards.EssenceStyle,
		})
	}
	return out, nil
}

// stoneSubjects is the stones. **Sorted by key rather than walked in file order**, because
// stones.json has no file-order reader — one stone per shape is what the catalog is, and
// tools/stonesheet walks the hand ladder to show it. A picture review wants a stable order and
// nothing more.
//
// **Grouped by the shape it raises**, since this is the one catalog carrying no Family: a stone's
// finish is a rule about where its shape sits on the ladder rather than a motif it was authored
// beside, which is what docs/art/stone_art_prompt.MD says and what the heading should therefore
// say too.
func stoneSubjects() ([]subject, error) {
	stones := data.LoadStones()
	var out []subject
	for _, key := range data.StoneOrder(stones) {
		rec := stones[key]
		out = append(out, subject{
			Key: key, Stem: rec.Art, Group: "Shape — " + fmt.Sprint(rec.Groups),
			Caption: rec.Name,
			Spec: cards.Spec{
				Name: rec.Name, Element: cards.Basic, Text: rec.Text,
				Highlights: cards.ElementHighlights(rec.Text), Enabled: true,
			},
			Style: cards.EssenceStyle,
		})
	}
	return out, nil
}

// otherSubjects is the catch-all family in assets/other: the potions and the sealed goods, which
// share a directory because they share a prompt and are too few to be catalogs of their own.
//
// **Neither carries a Text the card prints**, so these draw as a picture and a name. That is what
// the shop shows, and it is why the group headings say which file a row came from.
func otherSubjects() ([]subject, error) {
	var out []subject
	for _, rec := range data.LoadPotions() {
		out = append(out, subject{
			Key: rec.PotionRecord, Stem: rec.Art, Group: "Potions — " + rec.Family,
			Caption: fmt.Sprintf("%s · %d vitae", rec.Name, rec.Price),
			Spec: cards.Spec{
				Name: rec.Name, Element: cards.Basic, Enabled: true,
			},
			Style: cards.EssenceStyle,
		})
	}
	for _, rec := range data.LoadGoods() {
		out = append(out, subject{
			Key: rec.GoodRecord, Stem: rec.Art, Group: "Sealed goods — " + rec.Family,
			Caption: fmt.Sprintf("%s · %d vitae", rec.Name, rec.Price),
			Spec: cards.Spec{
				Name: rec.Name, Element: cards.Basic, Enabled: true,
			},
			Style: cards.EssenceStyle,
		})
	}
	return out, nil
}

// formOf turns the word duelist_cards.json writes into the mark the card draws. internal/cards
// has no parser of its own, so the four-line table is here; an unknown word draws no mark, which
// is the honest failure rather than a guess.
func formOf(name string) cards.Form {
	switch name {
	case "stab":
		return cards.FormStab
	case "slash":
		return cards.FormSlash
	case "crush":
		return cards.FormCrush
	case "defend":
		return cards.FormDefend
	}
	return cards.FormNone
}
