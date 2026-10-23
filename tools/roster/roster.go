// Package roster renders one opponent-catalog review sheet: every record in a pool, drawn
// as the card the game draws, beside the deck it fights with and the stats it fights on.
//
// # Why this is a library and the other sheets are not
//
// `tools/sheets` says the other sheets share nothing but the words `png.Encode`, and that a
// library between them would exist to be a seam that command already is. That argument is still
// right, and this package is a `Pool` plus a page rather than a library between sheets:
// `tools/motifsheet` is a pool and a `main`. The shape is kept because the pool is the one thing
// a second roster view would vary, and a second view that had not learned about a new column
// would show the game as it was.
//
// # It is a report, like the relic and essence sheets
//
// It reads the motif files under `data/motifs` rather than writing its own contents out.
// It also imports `internal/decks`, which registers every creature concept at init — so a
// card naming a verb the rules do not have fails the sheet exactly as it fails the launch, and
// the deck sizes printed here are the real expanded piles rather than a `Copies` column added up
// in a template.
//
// # What to look at
//
// **The stat line against the realm.** The whole roster is hand-assigned, so the review question
// is whether realm 5's creatures are actually dearer than realm 4's — which an alphabetical list
// of ninety-six records cannot answer and a page grouped by realm can.
//
// **The deck beside the stats.** An enemy's personality is what it holds; `Copies` is the
// difficulty dial and it is sharper than it looks, since four copies of a 1 AP card in one turn
// is a Barrage at 5x. The strip shows one card per concept and the table under it says how many
// of each the pile holds.
//
// # Output
//
// One composite JPEG per record per colour it can be dealt as — the opponent's card followed by its
// deck — plus an index.html, written under `docs/sheets/`. **One strip rather than one file per
// card**: a file per card would be several times the binaries for the same pixels, and the sheets
// are committed with their weight charged to git history. **One strip per colour rather than one
// stacked picture**, because the element chips hide a colour by hiding its strip.
package roster

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/curiousjc/ascend-duel/assets"
	_ "github.com/curiousjc/ascend-duel/assets/motifs" // the portraits this sheet is made of
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

// Ground is the one an opponent card is actually drawn on — `screens.screenGround`, the fill
// behind both corners of the combat screen. Judging a card against a white browser page would be
// the same failure as previewing art at a scale the game does not use.
const Ground = "#a8bcd4"

// groundRGBA is Ground as pixels, for the strip the cards are composited onto. Said twice in two
// forms rather than parsed, because a hex parser here would be four lines to avoid one constant.
var groundRGBA = color.RGBA{R: 0xe2, G: 0xd0, B: 0xb0, A: 255}

// The space between cards on a strip. The gap after the opponent's own card is wider: the card on
// the left is the creature and the rest are what it holds, and a single pitch would read as one
// row of six equals.
const (
	stripGap   = 14
	stripSplit = 34
)

// Pool is one catalog: what it is called, and how to read it.
//
// **The pool is the only thing the two commands differ by.** Everything a page shows is derived
// from the entries it hands back.
type Pool struct {
	// Name is the sheet's own noun, used in the run's report — "enemy", "boss".
	Name string

	// Title is the page's heading.
	Title string

	// Blurb is the paragraph under the heading, saying what this page is for.
	Blurb string

	// GroupLabel is what a section of the page is a group of — "realm band", "realm".
	GroupLabel string

	// Entries reads the catalog, in the order the page should show it.
	Entries func() []Entry
}

// Entry is one opponent, whole: what the file says about it and what its deck expands to.
type Entry struct {
	Record string

	// Name is what goes on the card, which for a boss is the bare first name — see BossData.Name
	// for why the title is not on it.
	Name string

	// Title is the rest of what an opponent is called, drawn under the name on the page and never
	// on the card. Empty for every creature in the roster: a creature has no title.
	Title string

	// Family is the tier this record stands in — outer, inner or the portal room's boss — and Draw is
	// the subject paragraph an art generator would be given. **Draw is authored and ignored by
	// everything that plays the game**, exactly as a relic's is.
	Family string
	Draw   string

	// Portrait is the art key for the record's first colour.
	Portrait string

	// Portraits is that key for every colour the record can be dealt as, keyed by element, so the
	// page draws a strip per colour without learning how a record builds its own art key.
	Portraits map[string]string

	// Element is the record's first colour — what its deck size is read at — and Elements is every
	// colour it could be dealt as, each of which the page draws as a strip of its own.
	Element  string
	Elements []string

	DMG     int
	Actions int
	HP      int

	// Group is the sort key the page's sections are cut on, and Realms is how that section is
	// written. A section is one motif, ordered by the realm its band opens on.
	Group  int
	Realms string

	// Band is the same band as a pair of realms, which is what the realm chips are matched
	// against. Written out rather than parsed back off Realms, which is prose.
	Band [2]int

	// Motif is which file this record came from, and Tier is which of a realm's three rooms it can
	// stand in.
	Motif string
	Tier  string

	Cards []data.MotifCard
}

// MotifPool is every record in every motif file, motif by motif, shallowest band first.
//
// **Cut by motif rather than by realm**, because a motif is the unit a realm is built from: a
// realm takes one whole motif and one element, so the question this page answers is whether one
// motif's three rooms read as a journey and whether its creatures look like each other.
var MotifPool = Pool{
	Name:       "motif",
	Title:      "Motif sheet",
	GroupLabel: "motif",
	Blurb: "The whole roster, motif by motif: every creature's card as the game draws it, its " +
		"stat line, the colours it can be dealt as, and the deck it fights with. A realm takes " +
		"one motif and one element and holds three fights, so the questions this page answers " +
		"are whether a motif's outer, inner and portal records read as a journey, and whether " +
		"every one of its fights can be dealt at least two ways.",
	Entries: func() []Entry {
		motifs := data.LoadMotifs()
		var out []Entry
		for _, key := range data.MotifOrder(motifs) {
			m := motifs[key]
			for _, r := range m.Records {
				element := ""
				if len(r.Affinities) > 0 {
					element = r.Affinities[0]
				}
				portraits := make(map[string]string, len(r.Affinities))
				for _, el := range r.Affinities {
					portraits[el] = r.ArtKey(el)
				}
				out = append(out, Entry{
					Record: r.Record, Name: r.Name, Title: r.Title,
					Family: r.Tier, Draw: r.Draw,
					Portrait: r.ArtKey(element), Portraits: portraits,
					Element: element, Elements: r.Affinities,
					DMG: r.DMG, Actions: r.Actions, HP: r.HP,
					Group: m.ValidRealms[0], Realms: m.Name + " — " + realmBand(m.ValidRealms),
					Band:  m.ValidRealms,
					Motif: m.Motif, Tier: r.Tier,
					Cards: r.Cards,
				})
			}
		}
		return out
	},
}

// realmBand writes a [2]int range the way the page reads it. A zero range means every realm —
// see EnemyData.AllowsRealm, where a record written without the field is fightable rather than
// unreachable — and the page has to say so rather than printing "Realms 0-0".
func realmBand(f [2]int) string {
	switch {
	case f == [2]int{}:
		return "Any realm"
	case f[0] == f[1]:
		return "Realm " + strconv.Itoa(f[0])
	default:
		return "Realms " + strconv.Itoa(f[0]) + "–" + strconv.Itoa(f[1])
	}
}

// Run writes the sheet.
func Run(p Pool, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	entries := p.Entries()
	lo, hi := span(entries)
	pg := page{
		SpanLo:     lo,
		SpanHi:     hi,
		Ground:     Ground,
		Title:      p.Title,
		Blurb:      p.Blurb,
		GroupLabel: p.GroupLabel,
		Count:      len(entries),
		Style:      styleFacts(cards.EnemyStyle),
		Filters:    sheetfilter.Bar(facetsFor(entries)),
	}

	// **Every picture here is this run's**, so the last run's are cleared first: a record renamed,
	// or a colour taken off one, would otherwise leave a strip on disk that nothing links to and that
	// is still committed.
	var old []string
	for _, pattern := range []string{"*.png", "*.jpg"} {
		found, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return err
		}
		old = append(old, found...)
	}
	for _, f := range old {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	var written int64
	var strips int
	for _, e := range entries {
		pl := plate{
			Entry:   e,
			Affixes: strings.Join(e.Elements, ", "),
			Deck:    len(decks.EnemyCards(e.Record, e.Element)),
			Rows:    deckRows(e.Cards),
		}

		// One strip per colour the record can be dealt as, every one shown: a creature's portrait
		// and every attack's picture are drawn per element, so each colour is its own thing to
		// review, and the element chips are how a reader narrows to the ones they care about.
		colours := e.Elements
		if len(colours) == 0 {
			colours = []string{e.Element}
		}
		for _, el := range colours {
			strip, err := stripFor(faces, e, el)
			if err != nil {
				return err
			}
			name := "strip-" + strings.ToLower(e.Record)
			if el != "" {
				name += "-" + el
			}
			name += ".jpg"
			n, err := writeJPEG(filepath.Join(dir, name), strip)
			if err != nil {
				return err
			}
			written += n
			strips++
			pl.Strips = append(pl.Strips, elementStrip{
				Element: el, Label: label(el),
				Cell: cell{File: name, Width: strip.Bounds().Dx(), Height: strip.Bounds().Dy()},
			})
		}
		pg.add(pl)
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, pg); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s, %d strips — %d %s records in every colour each can be dealt as, %.1f MB of JPEG\n",
		out, strips, pg.Count, p.Name, float64(written)/(1<<20))
	for _, g := range pg.Groups {
		plural := "s"
		if len(g.Plates) == 1 {
			plural = " "
		}
		fmt.Printf("  %-14s %2d record%s — HP %d-%d, DMG %d-%d, AP %d-%d — %s\n",
			g.Label, len(g.Plates), plural, g.MinHP, g.MaxHP, g.MinDMG, g.MaxDMG,
			g.MinAP, g.MaxAP, g.Mix)
	}
	return nil
}

// stripFor composites one opponent's card and its deck onto a single picture, as the record is
// dealt in one element: its portrait in that element, and every attack wearing its flavor's
// picture in that element.
//
// **One card per concept, not per copy.** A pile of six Spores is six of the same picture, and
// the count is a number the table beside the strip prints — showing it as six cards would make a
// swarm's row six times as wide as a brute's while saying nothing a figure does not.
func stripFor(f *cards.Faces, e Entry, element string) (*image.RGBA, error) {
	art, err := artwork(e.Portraits[element])
	if err != nil {
		return nil, err
	}

	face, err := cards.Render(opponentSpec(e, art), cards.EnemyStyle, f)
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", e.Record, err)
	}

	deck := make([]*image.RGBA, 0, len(e.Cards))
	for _, c := range e.Cards {
		spec, err := cardSpec(c, element)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", e.Record, c.Label, err)
		}
		img, err := cards.Render(spec, cards.Hand, f)
		if err != nil {
			return nil, fmt.Errorf("rendering %s.%s: %w", e.Record, c.Label, err)
		}
		deck = append(deck, img)
	}

	w, h := face.Bounds().Dx(), face.Bounds().Dy()
	for i, img := range deck {
		w += img.Bounds().Dx()
		if i == 0 {
			w += stripSplit
		} else {
			w += stripGap
		}
		if d := img.Bounds().Dy(); d > h {
			h = d
		}
	}

	strip := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(strip, strip.Bounds(), &image.Uniform{C: groundRGBA}, image.Point{}, draw.Src)

	x := 0
	place := func(img *image.RGBA) {
		r := image.Rect(x, 0, x+img.Bounds().Dx(), img.Bounds().Dy())
		draw.Draw(strip, r, img, img.Bounds().Min, draw.Over)
		x = r.Max.X
	}
	place(face)
	for i, img := range deck {
		if i == 0 {
			x += stripSplit
		} else {
			x += stripGap
		}
		place(img)
	}
	return strip, nil
}

// label writes an element key the way a reader reads it. **Written here rather than taken from
// internal/ui**, which holds the game's own ElementWord and links Ebitengine: the same wall every
// sheet in this repo is on the far side of.
func label(el string) string {
	if el == "" {
		return "Basic"
	}
	return strings.ToUpper(el[:1]) + el[1:]
}

// opponentSpec is the opponent as the game's own card: name, portrait, and a full health bar.
//
// **Full life rather than a sample wound**, because the figure a reviewer is reading is the
// record's HP and a bar drawn at some fraction of it would be inviting the question of which
// fraction.
func opponentSpec(e Entry, art image.Image) cards.Spec {
	spec := cards.Spec{
		Name:    e.Name,
		Element: cards.Basic,
		Art:     art,
		Life:    e.HP,
		MaxLife: e.HP,
		Enabled: true,
	}

	// **The two orbs, as the game writes them.** A card here missing a figure the combat screen
	// draws is a page about a card that does not exist.
	spec.Stats[0] = cards.StatLine{Label: "DMG", Value: strconv.Itoa(e.DMG)}
	spec.Stats[1] = cards.StatLine{Label: "AP", Value: strconv.Itoa(e.Actions)}
	return spec
}

// cardSpec is one of an opponent's cards as the combat screen draws it when the creature plays it:
// name, form mark, cost ticks, the multiplier badge, and the flavor's picture at the card's weight
// in the element the realm dealt it as. **The fields `ui.CardSpec` sets, and no others** — a face carrying effect text
// the game never prints would be a page about a card that does not exist; what a card does is the
// table under the strip.
//
// **A flavor with no Art draws no picture**, as in the game. **A flavor whose Art names a file that
// is not there is an error**, where the game would log and draw a hole: the review tool is where a
// missing element file is meant to be caught.
func cardSpec(c data.MotifCard, el string) (cards.Spec, error) {
	spec := cards.Spec{
		Name:    c.Label,
		Form:    form(c.Form),
		Cost:    c.Cost,
		Element: element([]string{el}),
		Enabled: true,
	}
	if verb, ok := combat.ParseVerb(c.Verb); ok && verb == combat.VerbShield {
		spec.Shields = c.Amount
	} else {
		spec.Badge, spec.BadgePct = badge(c.Amount), c.Amount
	}

	key := enemyCardArt()[c.Flavor].ArtKey(data.EnemyCardWeight(c.Amount), el)
	if key == "" {
		return spec, nil
	}
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return spec, fmt.Errorf("the flavor %s names the picture %q, which is not embedded", c.Flavor, key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return spec, fmt.Errorf("decoding %s: %w", key, err)
	}
	spec.Art = img
	return spec, nil
}

// enemyCardArt is the flavor catalog, read once.
var enemyCardArt = sync.OnceValue(data.LoadEnemyCardArt)

// badge writes an attack's amount the way the card's badge does: a fraction under one, a whole
// number at or above it. A snapshot of `ui.damageMultiplier`, as tools/cardsheet's copy is, for the
// wall every sheet is on the far side of.
func badge(pct int) string {
	switch pct {
	case 25:
		return "1/4"
	case 50:
		return "1/2"
	case 75:
		return "3/4"
	}
	if pct%100 == 0 {
		return strconv.Itoa(pct / 100)
	}
	return strconv.FormatFloat(float64(pct)/100, 'g', -1, 64)
}

// effectText is what the card face says it does.
//
// **A copy of `screens.cardEffect`'s unheld branch**, and knowingly so: that function reads the
// holder's relics and an essence's scaling, neither of which an enemy card has — an enemy wears no
// relics and no essence reaches its deck. Importing `internal/screens` to reach the real one would
// pull Ebitengine into a review tool, which is the thing every sheet here is built to avoid.
// `tools/cardsheet` keeps its own snapshot of the player's wording for the same reason. If the
// two ever disagree, prose.go is the one that is right.
func effectText(c data.CardData) string {
	verb, ok := combat.ParseVerb(c.Verb)
	if !ok {
		// Unreachable in practice: importing internal/decks registers every card at init and a bad
		// verb panics there. Said anyway, so a future pool that skips registration cannot draw a
		// card claiming to hit for 0x.
		return "unknown verb " + strconv.Quote(c.Verb)
	}

	switch verb {
	case combat.VerbShield:
		return shieldText(c.Amount)
	default:
		return attackVerb(c.Form) + " for " + multiplier(c.Amount) + " DMG"
	}
}

func attackVerb(f string) string {
	switch f {
	case combat.FormStab.String():
		return "Stabs"
	case combat.FormSlash.String():
		return "Slashes"
	case combat.FormCrush.String():
		return "Crushes"
	default:
		return "Hits"
	}
}

// multiplier writes an Amount the way the card face does: percent of the wielder's DMG, so 100
// is 1x and 250 is 2.5x.
func multiplier(amount int) string {
	whole, frac := amount/100, amount%100
	switch {
	case frac == 0:
		return strconv.Itoa(whole) + "x"
	case frac%10 == 0:
		return fmt.Sprintf("%d.%dx", whole, frac/10)
	default:
		return fmt.Sprintf("%d.%02dx", whole, frac)
	}
}

// form maps a card's form name onto the drawing package's enum. Every creature card carries stab,
// slash or crush, so this is what puts its corner mark on.
func form(name string) cards.Form {
	for _, f := range cards.Forms() {
		if f != cards.FormNone && f.String() == name {
			return f
		}
	}
	return cards.FormNone
}

// element is which color to draw the card in. **Empty means basic**, exactly as the deck builder
// reads it — see internal/decks. Only the first is drawn: a concept shipping in several colors is
// several cards in the pile, and a strip showing one of each would say a swarm was a rainbow.
func element(names []string) cards.Element {
	if len(names) == 0 {
		return cards.Basic
	}
	for _, e := range cards.Elements() {
		if e.String() == names[0] {
			return e
		}
	}
	return cards.Basic
}

// deckRows is the table under a strip: one row per concept, in the order the strip draws them.
func deckRows(list []data.MotifCard) []row {
	out := make([]row, 0, len(list))
	for _, c := range list {
		out = append(out, row{
			Label:  c.Label,
			Verb:   c.Verb,
			Effect: effectText(c.CardData),
			Cost:   c.Cost,
			Copies: c.Copies,
		})
	}
	return out
}

// artwork decodes one embedded portrait. **A key that is in no embed is an error rather than a
// blank face**, unlike the game, which logs and draws the hole: the game must not refuse to start
// over a picture, and a review tool that quietly drew nothing would be hiding exactly what it is
// for. The portraits are keyed by filename stem — see assets/embed.go — so this is also the check
// that a renamed file and its JSON are still in step.
func artwork(key string) (image.Image, error) {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		// **The placeholder rather than an error.** Nearly every record's own picture is still to
		// be generated, and a sheet that refused to build until the last one existed would be a
		// sheet nobody could use while the art was being made.
		raw = assets.LoadImageData()[data.DefaultEnemyArt]
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no embedded picture called %q and no placeholder either", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

// writeJPEG writes one strip. **JPEG rather than PNG**, unlike every other sheet: a strip is
// mostly a photographic portrait and painted vapor, a creature is drawn once per colour it can be
// dealt as, and the sheet is committed — so lossless strips would cost the history several times
// what the review gains. The quality is high enough that a card's rim and its type stay crisp.
func writeJPEG(path string, img image.Image) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		return 0, fmt.Errorf("encoding %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":     st.Width,
		"height":    st.Height,
		"artTop":    st.ArtTop,
		"artMaxH":   st.ArtMaxH,
		"nameSize":  int(st.NameSize),
		"healthTop": st.HealthBarTop,
	}
}

// shieldText is what a shield card's face says, and it is shared with internal/screens through
// nothing at all — a roster card and a played card are drawn by two packages that may not import
// each other, so the wording is written twice on purpose.
func shieldText(n int) string {
	if n == 1 {
		return "1 shield"
	}
	return strconv.Itoa(n) + " shields"
}
