package screens

// **The Compendium's catalog books: every record in a catalog, and what the player has had of it.**
//
// Each is a grid of the catalog's own cards — the face the shop, the pane and the review sheet draw —
// in the catalog's own order. **A record the player has never had is drawn as the placeholder**, with
// nothing of its own picture showing, and its tooltip keeps its name and its rule to itself: a
// compendium that gave the whole catalog away on the first launch would leave nothing to find. One
// that has been had carries its full tooltip, ending on its count, and **one that has been had but
// has no art of its own is its real card darkened**, so a found record waiting for a picture is not
// mistaken for one still to find.
//
// **Which figure counts as "had" is the catalog's own verb**: a relic bought, a potion drunk, a
// cantrip cast, a creature beaten or fallen to. See the `Stat…` families in internal/session/stats.go.

import (
	"image"
	"slices"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The grid. **Every catalog card is 200x280 and is drawn here at its own size**, except the
// bestiary's, which is drawn at three quarters: a creature in every colour is many times any other
// catalog, and a page of them at full size is a handful of cards. Three quarters is the size that
// stands four rows down the body.
const (
	catalogScaleNum = 3
	catalogScaleDen = 4

	catalogGapX = 14
	catalogGapY = 14

	// catalogTagH is one line of an entry's tags — the element and the motif a creature or a realm
	// carries — set under the picture.
	catalogTagH = 22

	// catalogTagGap is what a book with tags keeps under them.
	catalogTagGap = 6

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
	// tooltip — see compendium_creature.go. A found realm-book entry shows its room the same way —
	// see compendium_room.go.
	creature *creaturePeek

	// element and motif are what the book's filter reads — a creature's colour and its motif key, or
	// a realm's — and tags are the lines set under the picture saying so. **Shown found or not**:
	// they say where an entry sits on the roster, which the filter's chips have already named.
	element string
	motif   string
	tags    []models.TipLine

	// rarity is a relic's tier, which the relic book's one facet reads, and room which of a realm's
	// three rooms a realm-book entry is.
	rarity     data.Rarity
	room       string
	affinities []string

	// backdrop is a realm's: the room drawn in place of a card — see compendium_realm.go.
	backdrop string

	// aloft and ephemeral are how the trait book draws its sample, the way the game draws a card or
	// relic carrying the trait: lifted off the row, or faded with its border broken. See
	// compendium_trait.go.
	aloft, ephemeral, edge bool
}

// cell is the size one entry takes: its picture and the caption under it.
func (d *compendiumDialog) cell() (w, h int) {
	return d.pic.X, d.pic.Y + d.captionH
}

// catalogGrid is how many entries fit across the body and how many rows down it.
func (d *compendiumDialog) catalogGrid(body image.Rectangle) (cols, rows int) {
	w, h := d.cell()
	cols = (body.Dx() + catalogGapX) / (w + catalogGapX)
	rows = (body.Dy() + catalogGapY) / (h + catalogGapY)
	return max(cols, 1), max(rows, 1)
}

// catalogSeat is where the i-th shown entry is drawn, given the scrolled-to row, and whether it is
// on the page at all.
func (d *compendiumDialog) catalogSeat(body image.Rectangle, i, offset int) (image.Rectangle, bool) {
	cols, rows := d.catalogGrid(body)
	row := i/cols - offset
	if row < 0 || row >= rows {
		return image.Rectangle{}, false
	}
	w, h := d.cell()
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
	for i, idx := range s.open.shown {
		e := s.open.entries[idx]
		seat, ok := s.open.catalogSeat(body, i, s.scroll.Offset)
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
		if e.backdrop != "" {
			s.roomPeek, s.peekSeat = &e, seat
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
	decodes := realmThumbsPerFrame
	for i, idx := range s.open.shown {
		e := s.open.entries[idx]
		seat, ok := s.open.catalogSeat(body, i, s.scroll.Offset)
		if !ok {
			continue
		}
		if e.backdrop != "" {
			drawRealmPicture(gs, screen, e, image.Rectangle{Min: seat.Min, Max: seat.Min.Add(s.open.pic)}, &decodes)
		} else {
			spec := e.spec
			if !e.found {
				spec = undiscoveredSpec(gs)
			}
			img := ui.CardImage(gs, spec, e.style)
			if e.found && e.ephemeral {
				img = ui.EphemeralFace(img)
			}
			if e.found && e.aloft {
				ui.DrawAloft(screen, img, seat.Min, 1, float64(gs.Count), float64(i)*2.1)
			} else if img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(seat.Min.X), float64(seat.Min.Y))
				if e.found && ui.IsPlaceholder(gs, e.spec.Art) {
					op.ColorScale.Scale(catalogUndrawnScale, catalogUndrawnScale, catalogUndrawnScale, 1)
				}
				screen.DrawImage(img, op)
				if e.found && e.edge {
					ui.DrawEdgeLight(screen, op.GeoM, e.style.Width, e.style.Height, e.style.CornerRadius, gs.Count, i)
				}
			}
		}
		ty := seat.Min.Y + s.open.pic.Y + 4
		for _, tag := range e.tags {
			x := seat.Min.X + (seat.Dx()-int(tagWidth(tag, face)))/2
			systems.DrawLine(screen, tag, face, x, ty, compendiumQuietInk)
			ty += catalogTagH
		}
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

// tagWidth is how wide a tag line is set, for centering it under its picture.
func tagWidth(line models.TipLine, face *text.GoTextFace) float64 {
	w := 0.0
	for _, run := range line {
		w += systems.MeasureText(run.Text, face)
	}
	return w
}

// catalogDialog finishes a book: the count line every found record's tooltip ends on, every entry
// shown, and its "(N/M)" tally.
//
// **A revealing scenario or the reveal dev tool finds everything here**, so every catalog — and the bestiary's peek — reads
// as found without each book asking. See scenario.Reveals: a roster entry is keyed by its realm, so a
// partly revealed bestiary and realm book agree about where the player has been.
func catalogDialog(gs *state.GlobalState, title string, entries []compendiumEntry) *compendiumDialog {
	for i := range entries {
		e := &entries[i]
		key := e.title
		if e.motif != "" {
			key = e.motif + ":" + e.element
		}
		if reveals(gs, key) {
			e.found = true
		}
		if e.found {
			e.extra = append(e.extra, e.verb+" "+times(e.count))
		}
	}
	// The cell is the book's own card, which every entry in one book shares.
	card := cards.EssenceStyle
	if len(entries) > 0 {
		card = entries[0].style
	}
	d := &compendiumDialog{
		title:   title,
		entries: entries,
		pic:     image.Pt(card.Width, card.Height),
	}
	d.refilter()
	return d
}

// tagged makes room under a book's pictures for the tag lines its entries carry.
func (d *compendiumDialog) tagged() {
	most := 0
	for _, e := range d.entries {
		most = max(most, len(e.tags))
	}
	if most > 0 {
		d.captionH = catalogTagGap + most*catalogTagH
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

// bleedStyle is the full-bleed card every catalog is drawn in — the relic's pink-bordered one for
// relics, the essence's for everything else.
func bleedStyle(relic bool) cards.Style {
	if relic {
		return cards.RelicStyle
	}
	return cards.EssenceStyle
}

// creatureStyle is the bestiary's card: the essence's, reduced.
func creatureStyle() cards.Style {
	return cards.EssenceStyle.Scaled(catalogScaleNum, catalogScaleDen)
}

// entry is one record's card with its count against a stat family, and the verb its tooltip uses.
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
		e := entry(ui.RelicSpec(gs, record, "", true, false), bleedStyle(true),
			all.Total(session.StatRelicBought+key), "BOUGHT", title, lines)
		e.rarity, e.affinities = record.Rarity, relicAffinities(record)
		out = append(out, e)
	}
	d := catalogDialog(gs, "RELICS", out)
	d.rarities(gs)
	return d
}

func cantripsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, c := range session.Cantrips() {
		out = append(out, entry(ui.CantripSpec(gs, c, true), bleedStyle(false),
			all.Total(session.StatCantripCast+c.Record), "CAST", c.Name, cantripTipLines(c)))
	}
	return catalogDialog(gs, "CANTRIPS", out)
}

func potionsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, p := range session.Potions() {
		title, lines := potionTip(p, p.Price)
		out = append(out, entry(potionSpec(gs, p, true), bleedStyle(false),
			all.Total(session.StatPotionDrunk+p.Record), "DRUNK", title, lines))
	}
	return catalogDialog(gs, "POTIONS", out)
}

func tonicsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, t := range session.Tonics() {
		title, lines := tonicTip(t, t.Price)
		out = append(out, entry(tonicSpec(gs, t, true), bleedStyle(false),
			all.Total(session.StatTonicDrunk+t.Record), "DRUNK", title, lines))
	}
	return catalogDialog(gs, "TONICS", out)
}

// essencesDialog is the essence book, **in three runs**: the form essences in the forms' order, the
// elemental ones in the affinity order, then every other upgrade to a card in the catalog's own
// order. Sorted here rather than in data/essences.json, whose order is the family grouping the
// essence sheet reads.
func essencesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	var file []session.Essence
	for _, key := range data.EssenceFileOrder() {
		if w, ok := session.EssenceByKey(key); ok {
			file = append(file, w)
		}
	}
	for _, w := range essenceBookOrder(file) {
		title, lines := ui.EssenceTip(w, 1)
		out = append(out, entry(ui.EssenceSpec(gs, w, true), bleedStyle(false),
			all.Total(session.StatEssenceUsed+w.Record), "USED", title, lines))
	}
	return catalogDialog(gs, "ESSENCES", out)
}

// essenceForms is the order the form essences stand in.
var essenceForms = []string{"stab", "slash", "crush", "defend"}

// essenceBookOrder sorts the essences into the book's three runs, stably, so an essence the runs do
// not order keeps its place in the catalog.
func essenceBookOrder(ws []session.Essence) []session.Essence {
	rank := func(w session.Essence) (run, at int) {
		switch w.Target {
		case session.TargetForm:
			return 0, slices.Index(essenceForms, w.Form.String())
		case session.TargetElement:
			return 1, slices.Index(data.AffinityElements, w.Element.String())
		}
		return 2, 0
	}
	out := slices.Clone(ws)
	slices.SortStableFunc(out, func(a, b session.Essence) int {
		ra, ia := rank(a)
		rb, ib := rank(b)
		if ra != rb {
			return ra - rb
		}
		return ia - ib
	})
	return out
}

func runesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, p := range session.Runes() {
		out = append(out, entry(ui.RuneSpec(gs, p, true, false), bleedStyle(false),
			all.Total(session.StatRuneUsed+p.Record), "USED", runeTitle(p), runeTipLines(gs, p)))
	}
	return catalogDialog(gs, "RUNES", out)
}

func stonesDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, st := range session.Stones() {
		out = append(out, entry(ui.StoneSpec(gs, st, true), bleedStyle(false),
			all.Total(session.StatStoneUsed+st.Record), "USED", st.Name, strings.Split(st.Text, "\n")))
	}
	return catalogDialog(gs, "STONES", out)
}

func boostersDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, g := range session.Boosters() {
		title, lines := boosterTip(g, g.Price)
		spec := cards.Spec{
			Name: g.Name, Form: cards.FormNone, Element: ui.ArtFor(combat.Basic),
			Art: boosterArt(gs, g), Enabled: true,
		}
		out = append(out, entry(spec, bleedStyle(false),
			all.Total(session.StatBoosterBought+g.Record), "OPENED", title, lines))
	}
	return catalogDialog(gs, "BOOSTERS", out)
}

// traitOdds is how often a trait turns up, said where it turns up — or that it is never sold.
func traitOdds(pct int, where string) string {
	if pct <= 0 {
		return "never sold"
	}
	return strconv.Itoa(pct) + "% of " + where
}

// creaturesDialog is the bestiary: every creature and boss on the roster, motif by motif, in each
// motif file's own order — **once per element it can be dealt as**, in the record's own order of
// affinities, since a fire goblin and an ice goblin are two drawings and two decks.
//
// **Each element is found on its own, once it has been fought** in that colour, won or lost — a
// player who fell to the Toll-Taker has met it. The figures are the per-element families, which
// count from the build that introduced them, so a creature met before then is still to find in
// every colour. Resting on a found one opens its card and its deck.
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

				e := entry(spec, creatureStyle(), beaten, "BEATEN", r.FullName(), nil)
				e.found = beaten+fell > 0
				e.creature = &creaturePeek{
					record: r, element: element, motif: motif.Name, beaten: beaten, fell: fell,
				}
				e.element, e.motif, e.room = element, key, roomLabel(r.Tier)
				out = append(out, e)
			}
		}
	}
	// **No tags under a creature**: the chips already say which colours and motifs are on the page,
	// and the room they took is another row of cards.
	d := catalogDialog(gs, "CREATURES", out)
	d.roster(gs, func(bool, string, string) []models.TipLine { return nil })
	d.rooms(gs)
	return d
}

// motifName is what a motif is called, or its key while it has no name.
func motifName(m data.MotifData) string {
	if m.Name == "" {
		return m.Motif
	}
	return m.Name
}
