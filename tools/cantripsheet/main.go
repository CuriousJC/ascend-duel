// Command cantripsheet renders every cantrip in data/cantrips.json to a PNG — the scroll and the
// cantrip-relic it casts, side by side — and writes an HTML page that shows each one beside the
// relic's rules and what wearing it does to a real duelist.
//
//	go run ./tools/cantripsheet
//
// It is the rune sheet's idea on the catalog beside it. A cantrip is bought sealed in a bundle of
// scrolls and cast between the turns of a fight, so the whole catalog is several shop visits and a
// lot of luck away in a launched game.
//
// # It is a report, not a drawing-board
//
// This reads the real file through internal/session, so the catalog is *validated* before anything
// is drawn: a relic rule the grammar refuses, or one waking at a moment a cantrip-relic is never
// worn for, panics at init exactly as it would in the game.
//
// # What to look at
//
// **The lines against the rules.** The scroll's `Text` and the relic's are what a player reads, and
// nothing checks either against the rules. "+10 DMG" over a relic carrying `Amount: 5` is the
// failure this page exists to make visible.
//
// **The worked example.** Every plate equips the shipped duelist with the cantrip-relic — at full
// life, wounded to half, and cast twice — through `session.Session.EquipWearing`, the function the
// combat screen's refit calls, and keeps the wound the way a refit does. So the figures are the
// game's, not a second arithmetic.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/cantripsheet/` and committed on the
// terms every other sheet is. A clone opens `docs/sheets/index.html`.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// ground is screens.screenGround, the light slate blue a cantrip is offered on.
const ground = "#a8bcd4"

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "cantripsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}

	body, err := duelist()
	if err != nil {
		return err
	}

	// **The page walks the file's own order and the bundle walks the sorted one**, the split every
	// catalog sheet makes: nothing on this page decides an outcome.
	order := data.CantripFileOrder()
	records := data.LoadCantrips()

	page := page{
		Ground:  ground,
		Count:   len(order),
		Bundles: goodsPhrase(session.ContentsCantrips),
		Cap:     session.MaxConsumables,
		Body:    fmt.Sprintf("%d DMG, %d life", body.DMG, body.MaxLife),
	}

	var plates []plate
	for _, key := range order {
		c, ok := session.CantripByKey(key)
		if !ok {
			return fmt.Errorf("cantrips.json writes %q and internal/session resolved no such cantrip", key)
		}

		art, err := artwork(c.Art)
		if err != nil {
			return err
		}
		cell, err := write(dir, faces, specFor(c, art, true), cards.EssenceStyle,
			"cantrip-"+c.Record+".png", c.Name)
		if err != nil {
			return err
		}
		relicArt, err := artwork(c.RelicArt)
		if err != nil {
			return err
		}
		relicCell, err := write(dir, faces, relicSpecFor(c, relicArt), cards.RelicStyle,
			"relic-"+c.Record+".png", c.RelicName)
		if err != nil {
			return err
		}
		plates = append(plates, plate{
			Cell:         cell,
			RelicCell:    relicCell,
			Record:       c.Record,
			Name:         c.Name,
			Text:         c.Text,
			RelicName:    c.RelicName,
			RelicText:    c.RelicText,
			Rules:        ruleLines(records[key].Relic.Rules),
			Examples:     examples(c, body),
			Family:       c.Family,
			Draw:         c.Draw,
			Art:          c.Art,
			Default:      c.Art == data.DefaultCantripArt,
			RelicDraw:    c.RelicDraw,
			RelicArt:     c.RelicArt,
			RelicDefault: c.RelicArt == data.DefaultCantripRelicArt,
		})
		if c.Art == data.DefaultCantripArt {
			page.Undrawn++
		}
		if c.RelicArt == data.DefaultCantripRelicArt {
			page.RelicUndrawn++
		}
		if c.Draw == "" {
			page.Unwritten++
		}
		if c.RelicDraw == "" {
			page.RelicUnwritten++
		}
	}
	page.Families = groupByFamily(plates)
	page.Effects = effectCounts(records, order)

	// The two states a carried cantrip is drawn in: lit while the player plans, dim while a round
	// plays or on a screen where nothing can be cast.
	if len(order) > 0 {
		first, _ := session.CantripByKey(order[0])
		art, err := artwork(first.Art)
		if err != nil {
			return err
		}
		for _, s := range []struct {
			name, label string
			enabled     bool
		}{
			{"rest", first.Name + " — castable, while planning", true},
			{"disabled", first.Name + " — dim, mid-round or out of a fight", false},
		} {
			cell, err := write(dir, faces, specFor(first, art, s.enabled), cards.EssenceStyle,
				"state-"+s.name+".png", s.label)
			if err != nil {
				return err
			}
			page.States = append(page.States, cell)
		}
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s and %d PNGs — %d cantrips, %d scrolls and %d relics with art of their own; "+
		"bundles at %s\n",
		out, 2*len(plates)+len(page.States), page.Count,
		page.Count-page.Undrawn, page.Count-page.RelicUndrawn, page.Bundles)
	return nil
}

// duelist is the shipped duelist's own figures, off data/duelists.json — the body every example on
// the page is cast onto. **The record rather than an equipped fighter**: this page is drawn against
// no run, so a relic or a potion would be a run's opinion the page does not have.
func duelist() (combat.Duelist, error) {
	recs := data.LoadDuelists()
	if len(recs) == 0 {
		return combat.Duelist{}, fmt.Errorf("duelists.json is empty")
	}
	keys := make([]string, 0, len(recs))
	for k := range recs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	r := recs[keys[0]]
	return combat.Duelist{DMG: r.DMG, MaxLife: r.HP, CurrentLife: r.HP}, nil
}

// examples is the cantrip-relic worn by the duelist at full life, wounded to half, and twice over,
// **through EquipWearing and a refit's wound**, so the page prints the game's figures rather than
// working them out a second time. A figure the cast did not move is left out of the line, and a
// card the relic changes is priced as an attack of each element, which is what makes Chill's line
// say something.
func examples(c session.Cantrip, body combat.Duelist) []string {
	run := session.New(nil)
	bare := run.Equip(body)
	once := run.EquipWearing(body, session.CantripRelics([]session.Cantrip{c}))
	twice := run.EquipWearing(body, session.CantripRelics([]session.Cantrip{c, c}))

	var out []string
	for _, wound := range []int{0, bare.MaxLife / 2} {
		out = append(out, figures(bare, once, wound))
	}
	out = append(out, "cast twice at full: "+figures(bare, twice, 0))

	// **Every element's attack, priced bare and worn.** A relic changing what a card deals moves no
	// figure on the duelist, so without this a Chill would read as a cantrip that does nothing. The
	// worn side is priced at the bare DMG, so a Might's line — already said above — is not repeated
	// once per element.
	at := once
	at.DMG = bare.DMG
	for _, e := range combat.AllElements {
		card := combat.Card{Concept: combat.Bash, Element: e}
		if was, now := bare.CardDamage(card), at.CardDamage(card); was != now {
			out = append(out, fmt.Sprintf("%s Bash: %d → %d", e, was, now))
		}
	}
	return out
}

// figures is what a cast moved, read the way a refit carries it: the wound stays a wound.
func figures(bare, worn combat.Duelist, wound int) string {
	var parts []string
	if worn.DMG != bare.DMG {
		parts = append(parts, fmt.Sprintf("DMG %d → %d", bare.DMG, worn.DMG))
	}
	if worn.MaxLife != bare.MaxLife {
		parts = append(parts, fmt.Sprintf("life %d/%d → %d/%d",
			bare.MaxLife-wound, bare.MaxLife, max(worn.MaxLife-wound, 1), worn.MaxLife))
	}
	if len(parts) == 0 {
		return "moves no figure on the duelist"
	}
	return strings.Join(parts, ", ")
}

// specFor is a cantrip as the card the pane draws — the fields ui.DrawCantripCard fills: a name and
// a picture, no form, no cost, no sentence.
func specFor(c session.Cantrip, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    c.Name,
		Form:    cards.FormNone,
		Element: cards.Basic,
		Art:     art,
		Enabled: enabled,
	}
}

// relicSpecFor is the cantrip-relic as the relic row draws it: ui.RelicSpec's fields, with no rarity
// because a cantrip-relic is never sold.
func relicSpecFor(c session.Cantrip, art image.Image) cards.Spec {
	return cards.Spec{
		Name:    c.RelicName,
		Element: cards.Relic,
		Art:     art,
		Enabled: true,
	}
}

// ruleLines turns a cantrip-relic's rules into one line each, in the file's own vocabulary — the
// relic sheet's layout, so a cantrip-relic and a relic read alike. **Deliberately not prose**: the
// sentences a player reads are the two Texts, printed beside these.
func ruleLines(rules []data.RelicRuleData) []string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		line := "when " + rule.When
		if cond := condition(rule.If); cond != "" {
			line += ", if " + cond
		}
		effects := make([]string, 0, len(rule.Then))
		for _, e := range rule.Then {
			effects = append(effects, effect(e))
		}
		out = append(out, line+" → "+strings.Join(effects, " and "))
	}
	return out
}

func condition(in *data.RelicIfData) string {
	if in == nil {
		return ""
	}
	var parts []string
	if in.Element != "" {
		parts = append(parts, in.Element)
	}
	if in.Form != "" {
		parts = append(parts, in.Form)
	}
	if in.Concept != "" {
		parts = append(parts, in.Concept)
	}
	if in.Tier != 0 {
		parts = append(parts, fmt.Sprintf("tier %d", in.Tier))
	}
	if in.Lead {
		parts = append(parts, "lead")
	}
	if in.Hand != "" {
		parts = append(parts, in.Hand)
	}
	if len(in.Hands) > 0 {
		parts = append(parts, "any of "+strings.Join(in.Hands, " / "))
	}
	if in.MinForms > 0 {
		parts = append(parts, fmt.Sprintf("%d+ forms", in.MinForms))
	}
	return strings.Join(parts, " and ")
}

func effect(e data.RelicEffectData) string {
	switch {
	case e.Element != "":
		return e.Do + " " + e.Element
	case e.Amount != 0:
		return fmt.Sprintf("%s %+d", e.Do, e.Amount)
	default:
		return e.Do
	}
}

// effectCounts is how many cantrip-relics wake at each moment to do each verb, in the order they are
// first met in the file — what the catalog reaches for, read off the records.
func effectCounts(records map[string]data.CantripData, order []string) []effectCount {
	var out []effectCount
	seen := map[string]int{}
	for _, key := range order {
		for _, rule := range records[key].Relic.Rules {
			for _, e := range rule.Then {
				name := rule.When + " / " + e.Do
				if i, ok := seen[name]; ok {
					out[i].Count++
					continue
				}
				seen[name] = len(out)
				out = append(out, effectCount{Name: name, Count: 1})
			}
		}
	}
	return out
}

// groupByFamily splits the catalog into the blocks its records are authored in, in the file's own
// order. A record with no Family lands under "unfamilied" rather than being dropped.
func groupByFamily(plates []plate) []family {
	var order []string
	byName := map[string][]plate{}
	for _, p := range plates {
		name := p.Family
		if name == "" {
			name = "unfamilied"
		}
		if _, seen := byName[name]; !seen {
			order = append(order, name)
		}
		byName[name] = append(byName[name], p)
	}

	out := make([]family, 0, len(order))
	for _, name := range order {
		f := family{Name: name, Cantrips: byName[name], Count: len(byName[name]), Noun: "cantrips"}
		if f.Count == 1 {
			f.Noun = "cantrip"
		}
		out = append(out, f)
	}
	return out
}

// write renders one card at its own style, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, st cards.Style, name, label string) (cell, error) {
	img, err := cards.Render(s, st, f)
	if err != nil {
		return cell{}, fmt.Errorf("rendering %s: %w", name, err)
	}

	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return cell{}, fmt.Errorf("creating %s: %w", name, err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return cell{}, fmt.Errorf("encoding %s: %w", name, err)
	}
	return cell{File: name, Label: label, Width: st.Width, Height: st.Height}, nil
}

// artwork decodes one embedded picture. **A key that is in no embed is an error rather than a
// blank face**: a review tool that quietly drew nothing would be hiding exactly what it is for.
func artwork(key string) (image.Image, error) {
	raw := assets.LoadImageData()[key]
	if len(raw) == 0 {
		return nil, fmt.Errorf("no embedded image called %q", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", key, err)
	}
	return img, nil
}

// goodsPhrase is every sealed good holding this catalog, as one sentence: "3 for 3, 4 for 5 or 5
// for 6 vitae". **Asked of the catalog rather than written down**, so the page cannot quote a price
// the shop does not charge.
func goodsPhrase(c session.GoodContents) string {
	var held []session.Good
	for _, key := range session.GoodKeys() {
		if g, ok := session.GoodByKey(key); ok && g.Contains == c {
			held = append(held, g)
		}
	}
	sort.Slice(held, func(i, j int) bool { return held[i].Size < held[j].Size })

	parts := make([]string, 0, len(held))
	for _, g := range held {
		parts = append(parts, fmt.Sprintf("%d for %d", g.Size, g.Price))
	}
	switch len(parts) {
	case 0:
		return "nothing on the shelf holds them"
	case 1:
		return parts[0] + " vitae"
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " or " + parts[len(parts)-1] + " vitae"
	}
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// plate is one cantrip: the scroll and its relic, and everything the file says about both.
type plate struct {
	Cell      cell
	RelicCell cell
	Record    string
	Name      string
	Text      string
	RelicName string
	RelicText string
	Rules     []string
	Examples  []string

	// Family, Draw and Art are the fields the engine ignores, for the scroll and then the relic.
	// Default says the picture is the placeholder rather than one of its own.
	Family       string
	Draw         string
	Art          string
	Default      bool
	RelicDraw    string
	RelicArt     string
	RelicDefault bool
}

type effectCount struct {
	Name  string
	Count int
}

type family struct {
	Name     string
	Count    int
	Noun     string
	Cantrips []plate
}

type page struct {
	Ground         string
	Count          int
	Bundles        string
	Cap            int
	Body           string
	Undrawn        int
	Unwritten      int
	RelicUndrawn   int
	RelicUnwritten int
	Effects        []effectCount
	Families       []family
	States         []cell
}
