package screens

// **The Compendium's filter: a column of chips per facet** — the element and the motif in the bestiary
// and the realm book, the rarity in the relic book.
//
// Values within a facet are an OR and facets are an AND, and **a facet with nothing latched lets
// everything through** — the rule `tools/sheetfilter` keeps on the review pages, so the book and the
// sheet narrow the roster the same way. **A facet's values are read off the book's own entries**, never
// listed here, so a motif authored tomorrow puts its chip up with nothing edited.
//
// A chip is a button that latches: one press is one discrete step, which a focus ring can walk.

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The chips' shape: one tier of button, one width, each facet in columns of its own.
const (
	chipWidth = 170
	chipGapX  = 10
	chipGapY  = 8
	chipText  = systems.TextMedium
)

// compendiumFacet is one axis a book can be narrowed on, and its chips in the order they stand.
// of is an entry's one value on it, or many its several — a relic carries as many affinities as its
// rules name. under stands it beneath the facet before it, in that facet's column, rather than in
// columns of its own — for a short facet that would otherwise take a whole column of width — and
// heading is a word set over its chips there, taking a chip's slot.
type compendiumFacet struct {
	of      func(e compendiumEntry) string
	many    func(e compendiumEntry) []string
	chips   []*compendiumChip
	under   bool
	heading string

	// headX and headY are where the heading is set, placed with the chips.
	headX, headY int
}

// holds reports whether an entry carries a value on the facet.
func (f *compendiumFacet) holds(e compendiumEntry, value string) bool {
	if f.many != nil {
		return slices.Contains(f.many(e), value)
	}
	return f.of(e) == value
}

// compendiumChip is one value of a facet and the button that latches it.
type compendiumChip struct {
	value string
	on    bool
	btn   *models.Button
}

// rosterTags is the tag lines one entry carries, given whether it has been found and the words for
// its element and its motif — each the real word or its mask.
type rosterTags func(found bool, element, motif string) []models.TipLine

// roster finishes the bestiary or the realm book: its two facets — the element an entry was dealt
// as, in the affinity order, and its motif, in the book's own order — every entry's tags, and the
// caption room they take.
//
// **A value nothing found carries is masked**, on its chip and in every tag that would name it: the
// chip is disabled and reads a question mark per letter, so the shelf says how many motifs there
// are and how long each name is, and nothing more. A found entry is always both its values found.
func (d *compendiumDialog) roster(gs *state.GlobalState, tags rosterTags) {
	var elements, motifs []string
	seen, known := map[string]bool{}, map[string]bool{}
	for _, e := range d.entries {
		if e.motif != "" && !seen["m:"+e.motif] {
			seen["m:"+e.motif] = true
			motifs = append(motifs, e.motif)
		}
		seen["e:"+e.element] = true
		if e.found {
			known["e:"+e.element], known["m:"+e.motif] = true, true
		}
	}
	for _, el := range data.AffinityElements {
		if seen["e:"+el] {
			elements = append(elements, el)
		}
	}
	word := func(kind, value, name string) string {
		if known[kind+value] {
			return name
		}
		return masked(name)
	}

	element := &compendiumFacet{of: func(e compendiumEntry) string { return e.element }}
	for _, el := range elements {
		element.chips = append(element.chips, newChip(d, el, strings.ToUpper(word("e:", el, el)), known["e:"+el]))
	}
	motif := &compendiumFacet{of: func(e compendiumEntry) string { return e.motif }}
	for _, m := range motifs {
		name := motifName(gs.Motifs[m])
		motif.chips = append(motif.chips, newChip(d, m, strings.ToUpper(word("m:", m, name)), known["m:"+m]))
	}
	d.facets = []*compendiumFacet{element, motif}
	d.perCol = chipsPerColumn(gs)
	d.refilter()

	for i := range d.entries {
		e := &d.entries[i]
		e.tags = tags(e.found, word("e:", e.element, elementTitle(e.element)), word("m:", e.motif, motifName(gs.Motifs[e.motif])))
	}
	d.tagged()
}

// rooms adds the bestiary's and the realm book's third facet under the elements: which of a realm's
// rooms an entry is, or stands in — a boss is the portal room's.
// **A closed vocabulary**, so none of its chips is masked.
func (d *compendiumDialog) rooms(gs *state.GlobalState) {
	room := &compendiumFacet{of: func(e compendiumEntry) string { return e.room }, under: true}
	for _, r := range realmRooms {
		room.chips = append(room.chips, newChip(d, r.label, r.label, true))
	}
	d.facets = []*compendiumFacet{d.facets[0], room, d.facets[1]}
	d.refilter()
}

// rarities finishes the relic book: its two facets, the tier in the tiers' own order and under it the
// affinities (compendium_affinity.go). **Both are closed vocabularies**, so every value has a chip whether
// or not the catalog holds a relic of it, and none is masked — a tier is printed on every relic the
// shop offers and an affinity is in every relic's own sentence.
func (d *compendiumDialog) rarities(gs *state.GlobalState) {
	tier := &compendiumFacet{of: func(e compendiumEntry) string { return string(e.rarity) }}
	for _, r := range data.Rarities() {
		tier.chips = append(tier.chips, newChip(d, string(r), strings.ToUpper(string(r)), true))
	}
	affinity := &compendiumFacet{many: func(e compendiumEntry) []string { return e.affinities },
		under: true, heading: "AFFINITY"}
	for _, a := range affinityOrder() {
		affinity.chips = append(affinity.chips, newChip(d, a, strings.ToUpper(a), true))
	}
	d.facets = []*compendiumFacet{tier, affinity}
	d.perCol = chipsPerColumn(gs)
	d.refilter()
}

// masked is a word as its question marks, one per letter.
func masked(s string) string {
	return strings.Repeat("?", utf8.RuneCountInString(s))
}

// newChip is one latching chip. Pressing it flips its value in or out of the filter and asks the
// book to filter again. A chip for a value not yet found is disabled.
func newChip(d *compendiumDialog, value, label string, found bool) *compendiumChip {
	c := &compendiumChip{value: value}
	c.btn = models.NewButton(chipWidth, ui.ButtonTiny, label, func() {
		c.on = !c.on
		c.btn.Latched = c.on
		d.refilter()
		d.refiltered = true
	})
	c.btn.TextSize = chipText
	if !found {
		c.btn.State = models.ButtonStateDisabled
	}
	return c
}

// passes reports whether an entry gets through every facet.
func (d *compendiumDialog) passes(e compendiumEntry) bool {
	for _, f := range d.facets {
		picked, hit := false, false
		for _, c := range f.chips {
			if c.on {
				picked = true
				hit = hit || f.holds(e, c.value)
			}
		}
		if picked && !hit {
			return false
		}
	}
	return true
}

// refilter works out which entries are shown, and the tally after the title: "(N/M)" found over
// the whole book, and on a book with a filter "(K SHOWN)" for what its chips let through.
func (d *compendiumDialog) refilter() {
	d.shown = d.shown[:0]
	found := 0
	for i, e := range d.entries {
		if e.found {
			found++
		}
		if d.passes(e) {
			d.shown = append(d.shown, i)
		}
	}
	d.tally = "(" + strconv.Itoa(found) + "/" + strconv.Itoa(len(d.entries)) + ")"
	if len(d.facets) > 0 {
		d.tally += " (" + strconv.Itoa(len(d.shown)) + " SHOWN)"
	}
}

// chipsPerColumn is how many chips stand down the body's height.
func chipsPerColumn(gs *state.GlobalState) int {
	h := compendiumBody(gs, &compendiumDialog{}).Dy()
	return max((h+chipGapY)/(ui.ButtonTiny+chipGapY), 1)
}

// columns is how many columns a facet's chips take, wrapping into another once one is full.
func (f *compendiumFacet) columns(perCol int) int {
	return (len(f.chips) + perCol - 1) / perCol
}

// chipSides is how wide the chips stand on either side of the grid: **the first facet down the left
// and every other down the right**, so the grid keeps the whole height of the panel — a facet standing
// under another taking that one's side and no width of its own. Nothing for a book with no facets.
func (d *compendiumDialog) chipSides() (left, right int) {
	for i, f := range d.facets {
		if f.under {
			continue
		}
		w := f.columns(d.perCol)*(chipWidth+chipGapX) - chipGapX + compendiumScrollGap
		if i == 0 {
			left += w
		} else {
			right += w
		}
	}
	return left, right
}

// placeChips stands every chip in its side's columns, top-aligned with the grid.
func (d *compendiumDialog) placeChips(gs *state.GlobalState) {
	r := ui.ModalPanelRect(gs)
	top := r.Min.Y + compendiumCatalogTop + ui.ButtonTiny/2
	leftX := r.Min.X + compendiumBodyInset + chipWidth/2
	_, rightW := d.chipSides()
	rightX := r.Max.X - compendiumBodyInset - rightW + compendiumScrollGap + chipWidth/2
	// A facet standing under another starts a chip's height below the last of it, in its column.
	lastX, lastEnd := leftX, 0
	for i, f := range d.facets {
		if f.under {
			start := lastEnd + 1
			if f.heading != "" {
				f.headX, f.headY = lastX, top+start*(ui.ButtonTiny+chipGapY)
				start++
			}
			for j, c := range f.chips {
				c.btn.ScreenX = lastX
				c.btn.ScreenY = top + (start+j)*(ui.ButtonTiny+chipGapY)
			}
			lastEnd = start + len(f.chips) - 1
			continue
		}
		x := &rightX
		if i == 0 {
			x = &leftX
		}
		for j, c := range f.chips {
			c.btn.ScreenX = *x + (j/d.perCol)*(chipWidth+chipGapX)
			c.btn.ScreenY = top + (j%d.perCol)*(ui.ButtonTiny+chipGapY)
		}
		lastX, lastEnd = *x, min(len(f.chips), d.perCol)
		*x += f.columns(d.perCol)*(chipWidth+chipGapX) + compendiumScrollGap - chipGapX
	}
}

// updateChips places and presses the open book's chips, and goes back to the top of the book when
// a press has changed what it shows.
func (s *CompendiumScene) updateChips(gs *state.GlobalState) {
	d := s.open
	if len(d.facets) == 0 {
		return
	}
	d.placeChips(gs)
	for _, f := range d.facets {
		for _, c := range f.chips {
			systems.UpdateButton(gs, c.btn)
		}
	}
	if d.refiltered {
		d.refiltered = false
		s.scroll.Offset = 0
	}
}

// drawChips draws the open book's chips.
func (s *CompendiumScene) drawChips(gs *state.GlobalState, screen *ebiten.Image) {
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: chipText}
	for _, f := range s.open.facets {
		if f.heading != "" {
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(f.headX), float64(f.headY))
			op.PrimaryAlign = text.AlignCenter
			op.SecondaryAlign = text.AlignCenter
			op.ColorScale.ScaleWithColor(compendiumQuietInk)
			systems.DrawText(screen, f.heading, face, op)
		}
		for _, c := range f.chips {
			systems.DrawButton(gs, screen, c.btn)
		}
	}
}
