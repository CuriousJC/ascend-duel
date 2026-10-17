package screens

// **The Compendium's catalog books: every record in a catalog, and what the player has had of it.**
//
// Each is a grid of the catalog's own cards — the face the shop, the pane and the review sheet draw —
// in the catalog's own order. **A record the player has never had is drawn as the placeholder**, with
// nothing of its own picture showing, and its tooltip keeps its name and its rule to itself: a
// compendium that gave the whole catalog away on the first launch would leave nothing to find. One
// that has been had carries its count under it and its full tooltip, and **one that has been had but
// has no art of its own is its real card darkened**, so a found record waiting for a picture is not
// mistaken for one still to find.
//
// **Which figure counts as "had" is the catalog's own verb**: a relic bought, a potion drunk, a
// cantrip cast, a creature beaten or fallen to. See the `Stat…` families in internal/session/stats.go.

import (
	"image"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The grid. **Every catalog card is 200x280 and is drawn here at three fifths**, the same reduction
// for all of them so the books read as one shelf.
const (
	catalogScaleNum = 3
	catalogScaleDen = 5

	catalogGapX     = 22
	catalogGapY     = 14
	catalogCaptionH = 34

	// catalogUndrawnScale is how far a found record with no art of its own is darkened, so the
	// placeholder on a found card is told apart from the placeholder standing in for an unfound one.
	catalogUndrawnScale = 0.45

	// undiscoveredTitle is what a record the player has never had says when rested on.
	undiscoveredTitle = "Undiscovered"
)

// compendiumEntry is one card in a catalog book.
type compendiumEntry struct {
	spec  cards.Spec
	style cards.Style

	// found is whether the player has had it, and count the figure under it. title and lines are the
	// tooltip a found record shows.
	found bool
	count int
	verb  string
	title string
	lines []string
	extra []string

	// creature is set on the bestiary's entries, and is what a found one shows in place of a
	// tooltip — see compendium_creature.go.
	creature *creaturePeek
}

// catalogCell is the size one entry takes: the reduced card and the caption under it.
func catalogCell() (w, h int) {
	st := cards.EssenceStyle.Scaled(catalogScaleNum, catalogScaleDen)
	return st.Width, st.Height + catalogCaptionH
}

// catalogGrid is how many entries fit across the body and how many rows down it.
func catalogGrid(body image.Rectangle) (cols, rows int) {
	w, h := catalogCell()
	cols = (body.Dx() + catalogGapX) / (w + catalogGapX)
	rows = (body.Dy() + catalogGapY) / (h + catalogGapY)
	return max(cols, 1), max(rows, 1)
}

// catalogSeat is where the entry at index i is drawn, given the scrolled-to row, and whether it is on
// the page at all.
func catalogSeat(body image.Rectangle, i, offset int) (image.Rectangle, bool) {
	cols, rows := catalogGrid(body)
	row := i/cols - offset
	if row < 0 || row >= rows {
		return image.Rectangle{}, false
	}
	w, h := catalogCell()
	used := cols*w + (cols-1)*catalogGapX
	left := body.Min.X + (body.Dx()-used)/2
	x := left + (i%cols)*(w+catalogGapX)
	y := body.Min.Y + row*(h+catalogGapY)
	return image.Rect(x, y, x+w, y+h), true
}

// pointAtEntry puts the tooltip on whichever card is under the cursor.
func (s *CompendiumScene) pointAtEntry(gs *state.GlobalState, body image.Rectangle) {
	if !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)
	for i, e := range s.open.entries {
		seat, ok := catalogSeat(body, i, s.scroll.Offset)
		if !ok || !at.In(seat) {
			continue
		}
		if !e.found {
			s.tip.Point(seat, ui.TipLine(undiscoveredTitle), ui.TipLines([]string{"Not yet found"}))
			return
		}
		if e.creature != nil {
			s.peek, s.peekSeat = e.creature, seat
			return
		}
		lines := append(append([]string{}, e.lines...), e.extra...)
		s.tip.Point(seat, ui.TipLine(e.title), ui.TipLines(lines))
		return
	}
}

// drawCatalog draws the visible rows of a catalog book.
func (s *CompendiumScene) drawCatalog(gs *state.GlobalState, screen *ebiten.Image, body image.Rectangle) {
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: compendiumCaptionSize}
	for i, e := range s.open.entries {
		seat, ok := catalogSeat(body, i, s.scroll.Offset)
		if !ok {
			continue
		}
		spec := e.spec
		if !e.found {
			spec = undiscoveredSpec(gs)
		}
		img := ui.CardImage(gs, spec, e.style)
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(seat.Min.X), float64(seat.Min.Y))
			if e.found && ui.IsPlaceholder(gs, e.spec.Art) {
				op.ColorScale.Scale(catalogUndrawnScale, catalogUndrawnScale, catalogUndrawnScale, 1)
			}
			screen.DrawImage(img, op)
		}
		if !e.found {
			continue
		}
		cap := &text.DrawOptions{}
		cap.GeoM.Translate(float64(seat.Min.X+seat.Dx()/2), float64(seat.Max.Y-catalogCaptionH/2))
		cap.PrimaryAlign = text.AlignCenter
		cap.SecondaryAlign = text.AlignCenter
		cap.ColorScale.ScaleWithColor(compendiumInk)
		systems.DrawUI(screen, e.verb+" "+thousands(e.count), face, cap)
	}
}

// undiscoveredSpec is the card an unfound record is drawn as: the placeholder and nothing of the
// record, so neither its picture nor its outline is given away. It keeps the book's own style, which
// says only which catalog the page is.
func undiscoveredSpec(gs *state.GlobalState) cards.Spec {
	return cards.Spec{
		Form: cards.FormNone, Element: ui.ArtFor(combat.Basic),
		Art: ui.Placeholder(gs), Enabled: true,
	}
}

// catalogDialog finishes a book: its "N of M found" line, and the count line every found record's
// tooltip ends on.
func catalogDialog(title string, entries []compendiumEntry) *compendiumDialog {
	found := 0
	for i := range entries {
		e := &entries[i]
		if !e.found {
			continue
		}
		found++
		e.extra = append(e.extra, e.verb+" "+times(e.count))
	}
	return &compendiumDialog{
		title:    title,
		subtitle: strconv.Itoa(found) + " of " + strconv.Itoa(len(entries)) + " found",
		entries:  entries,
	}
}

// times is a count written as a phrase: "once", "twice", "7 times".
func times(n int) string {
	switch n {
	case 1:
		return "once"
	case 2:
		return "twice"
	default:
		return thousands(n) + " times"
	}
}

// bleedStyle is the reduced full-bleed card every catalog is drawn in — the relic's pink-bordered
// one for relics, the essence's for everything else.
func bleedStyle(relic bool) cards.Style {
	if relic {
		return cards.RelicStyle.Scaled(catalogScaleNum, catalogScaleDen)
	}
	return cards.EssenceStyle.Scaled(catalogScaleNum, catalogScaleDen)
}

// entry is one record's card with its count against a stat family, and the verb its caption uses.
func entry(spec cards.Spec, style cards.Style, tally int, verb, title string, lines []string) compendiumEntry {
	return compendiumEntry{
		spec: spec, style: style, found: tally > 0, count: tally,
		verb: verb, title: title, lines: lines,
	}
}

func relicsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, key := range data.RelicFileOrder() {
		record, ok := gs.Relics[key]
		if !ok {
			continue
		}
		title, lines := ui.RelicTip(record)
		out = append(out, entry(ui.RelicSpec(gs, record, "", true, false), bleedStyle(true),
			all.Total(session.StatRelicBought+key), "BOUGHT", title, lines))
	}
	return catalogDialog("RELICS", out)
}

func cantripsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, c := range session.Cantrips() {
		out = append(out, entry(ui.CantripSpec(gs, c, true), bleedStyle(false),
			all.Total(session.StatCantripCast+c.Record), "CAST", c.Name, cantripTipLines(c)))
	}
	return catalogDialog("CANTRIPS", out)
}

func potionsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, p := range session.Potions() {
		title, lines := potionTip(p, p.Price)
		out = append(out, entry(potionSpec(gs, p, true), bleedStyle(false),
			all.Total(session.StatPotionDrunk+p.Record), "DRUNK", title, lines))
	}
	return catalogDialog("POTIONS", out)
}

func tonicsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, t := range session.Tonics() {
		title, lines := tonicTip(t, t.Price)
		out = append(out, entry(tonicSpec(gs, t, true), bleedStyle(false),
			all.Total(session.StatTonicDrunk+t.Record), "DRUNK", title, lines))
	}
	return catalogDialog("TONICS", out)
}

func essencesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, w := range session.Essences() {
		title, lines := ui.EssenceTip(w, 1)
		out = append(out, entry(ui.EssenceSpec(gs, w, true), bleedStyle(false),
			all.Total(session.StatEssenceUsed+w.Record), "USED", title, lines))
	}
	return catalogDialog("ESSENCES", out)
}

func runesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, p := range session.Runes() {
		out = append(out, entry(ui.RuneSpec(gs, p, true, false), bleedStyle(false),
			all.Total(session.StatRuneUsed+p.Record), "USED", runeTitle(p), runeTipLines(gs, p)))
	}
	return catalogDialog("RUNES", out)
}

func stonesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, st := range session.Stones() {
		out = append(out, entry(ui.StoneSpec(gs, st, true), bleedStyle(false),
			all.Total(session.StatStoneUsed+st.Record), "USED", st.Name, strings.Split(st.Text, "\n")))
	}
	return catalogDialog("STONES", out)
}

func goodsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, g := range session.Goods() {
		title, lines := goodTip(g, g.Price)
		spec := cards.Spec{
			Name: g.Name, Form: cards.FormNone, Element: ui.ArtFor(combat.Basic),
			Art: goodArt(gs, g), Enabled: true,
		}
		out = append(out, entry(spec, bleedStyle(false),
			all.Total(session.StatGoodBought+g.Record), "OPENED", title, lines))
	}
	return catalogDialog("SEALED GOODS", out)
}

// creaturesDialog is the bestiary: every creature and boss on the roster, motif by motif, in each
// motif file's own order — **once per element it can be dealt as**, in the record's own order of
// affinities, since a fire goblin and an ice goblin are two drawings and two decks.
//
// **Each element is found on its own, once it has been fought** in that colour, won or lost — a
// player who fell to the Toll-Taker has met it. The figures are the per-element families, which
// count from the build that introduced them, so a creature met before then is still to find in
// every colour. Its caption is the wins; resting on a found one opens its card and its deck.
func creaturesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, key := range data.MotifOrder(gs.Motifs) {
		motif := gs.Motifs[key]
		for _, r := range motif.Records {
			for _, element := range r.Affinities {
				spec := cards.Spec{
					Name: r.FullName(), Form: cards.FormNone, Element: ui.ArtFor(combat.Basic),
					Art:     ui.ArtworkOr(gs, r.ArtKey(element), data.DefaultEnemyArt),
					Enabled: true,
				}
				beaten := all.Total(session.CreatureAs(session.StatCreatureDefeatedAs, r.Record, element))
				fell := all.Total(session.CreatureAs(session.StatCreatureKillerAs, r.Record, element))

				e := entry(spec, bleedStyle(false), beaten, "BEATEN", r.FullName(), nil)
				e.found = beaten+fell > 0
				e.creature = &creaturePeek{
					record: r, element: element, motif: motif.Name, beaten: beaten, fell: fell,
				}
				out = append(out, e)
			}
		}
	}
	return catalogDialog("CREATURES", out)
}
