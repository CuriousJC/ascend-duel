package screens

// **The Compendium's catalog books: every record in a catalog, and what the player has had of it.**
//
// Each is a grid of the catalog's own cards — the face the shop, the pane and the review sheet draw —
// in the catalog's own order. **A record the player has never had is drawn dark**, its picture down
// to a silhouette, and its tooltip keeps its name and its rule to itself: a compendium that gave the
// whole catalog away on the first launch would leave nothing to find. One that has been had carries
// its count under it and its full tooltip.
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

	catalogGapX      = 22
	catalogGapY      = 14
	catalogCaptionH  = 34
	catalogDarkScale = 0.14

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
		img := ui.CardImage(gs, e.spec, e.style)
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(seat.Min.X), float64(seat.Min.Y))
			if !e.found {
				op.ColorScale.Scale(catalogDarkScale, catalogDarkScale, catalogDarkScale, 1)
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
			all.Total(session.StatRuneUsed+p.Record), "USED", p.Name, runeTipLines(gs, p)))
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
// motif file's own order.
//
// **A creature is found once it has been fought**, won or lost — a player who fell to the Toll-Taker
// has met it. Its caption is the wins; its tooltip says both.
//
// **It wears the picture of the first element it can be dealt as.** A creature is painted once per
// element and the page has one seat for it, so the choice is the record's own first affinity rather
// than a roll.
func creaturesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, key := range data.MotifOrder(gs.Motifs) {
		motif := gs.Motifs[key]
		for _, r := range motif.Records {
			element := ""
			if len(r.Affinities) > 0 {
				element = r.Affinities[0]
			}
			spec := cards.Spec{
				Name: r.FullName(), Form: cards.FormNone, Element: ui.ArtFor(combat.Basic),
				Art:     ui.ArtworkOr(gs, r.ArtKey(element), data.DefaultEnemyArt),
				Enabled: true,
			}
			beaten := all.Total(session.StatCreatureDefeated + r.Record)
			fell := all.Total(session.StatCreatureKiller + r.Record)

			e := entry(spec, bleedStyle(false), beaten, "BEATEN", r.FullName(),
				[]string{motif.Name + ", " + strings.ToLower(r.Tier)})
			e.found = beaten+fell > 0
			if fell > 0 {
				e.lines = append(e.lines, "Fell to it "+times(fell))
			}
			out = append(out, e)
		}
	}
	return catalogDialog("CREATURES", out)
}
