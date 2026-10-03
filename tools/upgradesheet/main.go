// Command upgradesheet renders every *visible upgrade* a card can carry, against every form mark,
// and writes an HTML page.
//
//	go run ./tools/upgradesheet
//
// # Why it exists
//
// Sixteen runes attach seven kinds of rider and, until 2026-09-07, not one of them reached the
// drawing: a ridden card looked exactly like an unridden one and the only place a rider was
// visible was the tooltip prose. That is a hand of altered cards the player cannot read. See
// TODO.md, which is where the owner asked for it to be tracked.
//
// Every rider draws, so the review question is not "does this one look right" but "do these read as
// a *set*". A page per upgrade could never answer that, which is the relic sheet's argument.
//
// # What to look at
//
// **The art each upgrade lays over the face.** `data/upgrade_art.json` names a picture per upgrade,
// plus a default for every upgrade without one, and the page prints which beside each plate.
//
// **The stacked rows.** The art has to read in the deck panel, which overlaps every card over the
// one before it so only the left edge shows. The page draws every upgrade as that panel does — the
// half-size card at the panel's resting pitch and again packed tight — because art that reads
// on one card standing alone and vanishes in a pile has missed its one job.
//
// **The tooltip each upgrade produces, printed beside the card.** Those are the *same strings the
// game shows* — `internal/carddesc` is windowless precisely so this page can call it rather than
// keeping a snapshot — so a line that reads badly here reads badly under the cursor.
//
// **The actual-size row before the enlarged one.** The same rule the glyph sheet is under:
// reviewing only the blown-up row is how a mark comes to look acceptable in review and clunky in
// play. The enlargement here is CSS with `image-rendering: pixelated`, so it is a nearest-neighbor
// blow-up of the real pixels rather than a second, kinder rendering.
//
// **The plain card in every row.** An upgrade is only legible against what an ordinary card looks
// like, and the ordinary card is what the player has fifty-five of.
//
// **All four form marks.** The marks sit in the left column the art runs under, and have
// different margins — a shield is wide-shouldered where a spear is thin — so art that crowds
// one may clear another.
//
// **The three card states.** An upgraded card has to fade with the rest of its row when it cannot
// be afforded, exactly as an ordinary one does; the art walks the same distance toward the
// surface the cost ticks do, and this is where it is visible.
//
// **Which rune grants it.** An upgrade nobody can acquire is invisible in the other
// direction, so the page names the rider and the record that attaches it, and says so loudly when
// nothing in `data/runes.json` does.
//
// # It is a report, not a drawing-board
//
// Same split as relicsheet against cardsheet. The upgrades come from `systems.Upgrades()`, the
// riders from `combat.RiderKinds()` and the runes from `internal/session`, which validates the
// catalog at init — so an upgrade this page cannot draw is one the game cannot draw either.
//
// **It also walks the riders the other way round**, and says so when a rider kind draws nothing: an
// upgrade nobody can acquire and a rider nobody can see are the same failure from opposite ends.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/upgradesheet/` and **committed**, on
// the same terms as every other sheet. A clone opens `docs/sheets/index.html`.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// ground is screens.screenGround, the light slate blue a card is actually held on. Judging a card against a
// white browser page would be the same failure as previewing art at a scale the game does not use.
const ground = "#a8bcd4"

// zoom is what the enlarged row is blown up by, in the page's CSS. Whole-number, because the point
// of the row is to see the real pixels bigger and a fractional scale resamples them.
const zoom = 3

// The card an upgrade is demonstrated on. **One concept per form**, drawn at the same cost so the
// tick column is the same height in every cell and the eye is comparing the ink rather than the
// arithmetic — and at a cost of three, which is the tallest stack the shipped deck actually uses
// and therefore the most ink there is to look at.
var demoCards = []struct {
	Form    cards.Form
	Name    string
	Element cards.Element
}{
	{cards.FormStab, "Skewer", cards.Fire},
	{cards.FormSlash, "Cleave", cards.Ice},
	{cards.FormCrush, "Smash", cards.Lightning},
	{cards.FormDefend, "Block", cards.Earth},
}

// demoCost is the cost every demonstration card is drawn at. See demoCards.
const demoCost = 3

// demoDMG is what the duelist holding these cards hits for, for the tooltip figures.
//
// **Ten, because it makes every card on the page arithmetic somebody can check by eye**: a 2x Skewer
// held by a duelist on 10 is 20, and a reader who disagrees with a figure can say so without
// reaching for a calculator. It is not the game's starting DMG and does not pretend to be.
const demoDMG = 10

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "upgradesheet"),
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

	page := page{
		Ground: ground,
		Zoom:   zoom,
		Style:  styleFacts(cards.Hand),
	}

	// The control row: the same four cards with no upgrade at all. It goes first because every
	// judgment below it is a comparison against this.
	plain, err := renderRow(dir, faces, systems.UpgradeNone, "plain")
	if err != nil {
		return err
	}
	page.Plain = plain

	// The deck panel's view: every upgrade stacked as the panel stacks a row, at its resting
	// pitch and packed tight.
	for _, pitch := range []int{stackPitch, stackTight} {
		c, err := writeStack(dir, faces, pitch)
		if err != nil {
			return err
		}
		page.Stacks = append(page.Stacks, c)
	}

	for _, u := range systems.Upgrades() {
		e, _ := cards.UpgradeArtOf(u)
		p := plate{
			Upgrade: u.String(),
			Record:  e.UpgradeArtRecord,
			Art:     e.ArtKey(),
			Undrawn: e.Art == "",
			Riders:  ridersFor(u),
			Grants:  grantsFor(u),
			Tip:     tipFor(u),
		}

		row, err := renderRow(dir, faces, u, u.String())
		if err != nil {
			return err
		}
		p.Cells = row

		for _, st := range []struct {
			name              string
			label             string
			enabled, selected bool
		}{
			{"rest", "resting in the hand", true, false},
			{"selected", "queued for the turn", true, true},
			{"disabled", "more AP than the turn has left", false, false},
		} {
			d := demoCards[0]
			spec := specFor(d.Form, d.Name, d.Element, u)
			spec.Enabled, spec.Selected = st.enabled, st.selected
			c, err := write(dir, faces, spec, u.String()+"-state-"+st.name+".png", st.label)
			if err != nil {
				return err
			}
			p.States = append(p.States, c)
		}

		page.Plates = append(page.Plates, p)
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

	fmt.Printf("wrote %s — %d upgrades, %d forms\n", out, len(page.Plates), len(demoCards))
	for _, p := range page.Plates {
		grants := p.Grants
		if grants == "" {
			grants = "NOTHING GRANTS IT"
		}
		fmt.Printf("  %-11s art %-8s %-14s %s\n", p.Upgrade, p.Record, p.Riders, grants)
	}
	return nil
}

// renderRow draws the four demonstration cards for one upgrade.
func renderRow(dir string, f *cards.Faces, u systems.Upgrade, tag string) ([]cell, error) {
	out := make([]cell, 0, len(demoCards))
	for _, d := range demoCards {
		name := tag + "-" + d.Form.String() + ".png"
		c, err := write(dir, f, specFor(d.Form, d.Name, d.Element, u), name, d.Name)
		if err != nil {
			return nil, err
		}
		c.Note = d.Form.String() + " · " + d.Element.String()
		out = append(out, c)
	}
	return out, nil
}

// specFor is one demonstration card.
//
// **It carries no text**, because a hand card's face carries none: the upgrade is the art, and
// the words are the tooltip's.
func specFor(form cards.Form, name string, e cards.Element, u systems.Upgrade) cards.Spec {
	return cards.Spec{
		Name:    name,
		Form:    form,
		Cost:    demoCost,
		Element: e,
		Upgrade: u,
		Enabled: true,
	}
}

// The two pitches the stacked rows are drawn at. **stackPitch is the deck panel's resting
// `deckStackPitch`, restated** — `internal/ui` links Ebitengine and a command-line tool cannot
// reach it — and stackTight is a row that has had to tighten, which is where only the left edge and a
// sliver of the left column show.
const (
	stackPitch = 75
	stackTight = 30
)

// writeStack draws the plain card and then every upgrade as one deck-panel row: the half-size card,
// each overlapping the one before it at pitch, the last drawn on top — the panel's own order, so
// what shows of each card is its left edge.
func writeStack(dir string, f *cards.Faces, pitch int) (cell, error) {
	ups := append([]systems.Upgrade{systems.UpgradeNone}, systems.Upgrades()...)
	st := cards.Mini
	w := pitch*(len(ups)-1) + st.Width
	img := image.NewRGBA(image.Rect(0, 0, w, st.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: groundRGBA()}, image.Point{}, draw.Src)

	d := demoCards[0]
	for i, u := range ups {
		face, err := cards.Render(specFor(d.Form, d.Name, d.Element, u), st, f)
		if err != nil {
			return cell{}, fmt.Errorf("rendering the stack: %w", err)
		}
		at := image.Rect(i*pitch, 0, i*pitch+st.Width, st.Height)
		draw.Draw(img, at, face, image.Point{}, draw.Over)
	}

	name := fmt.Sprintf("stack-%d.png", pitch)
	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return cell{}, fmt.Errorf("creating %s: %w", name, err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		return cell{}, fmt.Errorf("encoding %s: %w", name, err)
	}

	label := "none"
	for _, u := range ups[1:] {
		label += " · " + u.String()
	}
	return cell{
		File: name, Label: label, Note: fmt.Sprintf("%dpx pitch", pitch),
		Width: w, Height: st.Height,
	}, nil
}

// groundRGBA is ground as a color, for a picture rather than a stylesheet.
func groundRGBA() color.RGBA {
	var r, g, b uint8
	fmt.Sscanf(ground, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// demoRidden is the demonstration card carrying this upgrade's rider, or a bare one for an upgrade
// nothing grants. **One builder for the face and the tooltip**, so a figure on the picture and the
// same figure in the panel cannot come from two different amounts.
func demoRidden(u systems.Upgrade) combat.Card {
	c := combat.Of(demoConcept(), combat.Fire)
	if k := riderOf(u); k != combat.RiderNone {
		c = c.SetRider(combat.Rider{Kind: k, Amount: demoRiderAmount(k)})
	}
	return c
}

// ridersFor is the rider kinds that produce this upgrade, as the page names them.
//
// **The mapping is `internal/screens`' and cannot be imported**, for textFor's reason, so it is
// restated here — which is why the page prints it rather than hiding it: a drifted line is
// visible on the sheet instead of being a silent disagreement.
func ridersFor(u systems.Upgrade) string {
	for _, k := range combat.RiderKinds() {
		if riderUpgrade[k] == u {
			return k.String()
		}
	}
	return ""
}

// riderUpgrade is `screens.upgradeForRider`, restated. **A knowingly accepted duplicate**, for
// textFor's reason: `internal/screens` links Ebitengine and a command-line tool cannot reach it.
// What keeps it honest is that the page *prints* what it resolved, so a drift shows on the sheet
// rather than being a silent disagreement — and grantsFor turns a wrong entry into a loud
// "NOTHING GRANTS IT" rather than into a plausible-looking row.
var riderUpgrade = map[combat.RiderKind]systems.Upgrade{
	combat.RiderWildElement:  systems.UpgradeWild,
	combat.RiderGolden:       systems.UpgradeGolden,
	combat.RiderSilver:       systems.UpgradeSilver,
	combat.RiderHealOnPlay:   systems.UpgradeHeal,
	combat.RiderShieldOnPlay: systems.UpgradeShield,
	combat.RiderDamageOnPlay: systems.UpgradeDamage,
	combat.RiderScaleInCombo: systems.UpgradeCombo,
	combat.RiderDamageInHand: systems.UpgradeHeldDamage,
	combat.RiderScaleInHand:  systems.UpgradeHeldScale,
	combat.RiderVitaeInHand:  systems.UpgradeHeldVitae,
	combat.RiderWildForm:     systems.UpgradeVersatile,
}

// grantsFor names every rune in the catalog that attaches this upgrade's rider, or an empty
// string if nothing does.
//
// **An upgrade nothing grants is a drawing with no way into the game.** That is exactly as
// invisible as a rider with no drawing, in the other direction, so the page says so loudly rather
// than showing a card that can never be dealt.
func grantsFor(u systems.Upgrade) string {
	want := ridersFor(u)
	out := ""
	for _, p := range session.Runes() {
		if p.Target != session.RuneRider || p.Rider.String() != want {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += p.Name + " (" + p.Record + ")"
	}
	return out
}

// tipFor is the tooltip a card carrying this upgrade shows, exactly as the game builds it.
//
// **The demonstration card and the duelist behind it are the row's own** — a fire Skewer held by a
// duelist hitting for demoDMG — so the figures on the page are figures a real pairing produces
// rather than round numbers chosen to look tidy. The cost comes off the card rather than off
// demoCost: the pictures are all drawn at one cost so the tick columns line up, and a tooltip
// quoting that instead of the card's own price would be the sheet's layout leaking into its text.
//
// **No relics**, which is what makes the block the whole panel: `screens.cardTip` appends its damage
// chain only when a relic has moved something, and a page about upgrades is not the place to explain
// a relic.
func tipFor(u systems.Upgrade) tip {
	c := demoRidden(u)
	out := tip{Title: tipRuns(carddesc.Title(c))}
	for _, line := range carddesc.Lines(c, c.Cost(), demoDMG, 100, 100) {
		out.Lines = append(out.Lines, tipRuns(line))
	}
	return out
}

// tipRuns cuts one line into the runs the game draws it as, with each color written out as CSS.
//
// **It is `screens.tipLine` through the same vocabulary**, which is the whole reason `ElementSpans`
// lives in `internal/cards`: the page cannot import the screen, but it can import the table the
// screen reads. A word colored here is colored under the cursor.
func tipRuns(line string) []tipRun {
	var out []tipRun
	for _, seg := range washed(line) {
		r := tipRun{Text: seg.Text}
		if seg.Ink.A != 0 {
			r.Ink = fmt.Sprintf("#%02x%02x%02x", seg.Ink.R, seg.Ink.G, seg.Ink.B)
		}
		out = append(out, r)
	}
	return out
}

// washed is one line cut into its colored segments — the element words first, then CHROMATIC into
// the wildcard's own wash.
//
// **The order is a rule.** cards.SplitWash only touches a segment nothing has colored, so running
// the element vocabulary first is what lets a word be an element or the wildcard and never both.
// `screens.chromatic` is the same two lines against the same two functions; what stops the two
// drifting is that both cuts live in `internal/cards`.
func washed(line string) []cards.Segment {
	var out []cards.Segment
	for _, seg := range cards.SplitSpans(line, cards.ElementSpans(line)) {
		out = append(out, cards.SplitWash(seg, carddesc.Chromatic, systems.UpgradeWild)...)
	}
	return cards.SplitMetals(out)
}

// riderOf is the rider this upgrade is drawn for, or RiderNone. It is riderUpgrade read backwards.
func riderOf(u systems.Upgrade) combat.RiderKind {
	for _, k := range combat.RiderKinds() {
		if riderUpgrade[k] == u {
			return k
		}
	}
	return combat.RiderNone
}

// demoRiderAmount is the figure the page gives a rider, taken from **the rune that grants it**
// rather than made up here — so the tooltip on this page carries the number a run would actually
// see, and a retune in `data/runes.json` moves it.
func demoRiderAmount(k combat.RiderKind) int {
	for _, p := range session.Runes() {
		if p.Target == session.RuneRider && p.Rider == k {
			return p.Number
		}
	}
	return 0
}

// demoConcept is the card the tooltips are built on: the same Skewer the pictures use, found in the
// registry rather than named by hand so a rename fails the sheet instead of quietly changing it.
func demoConcept() combat.ConceptID {
	for _, id := range combat.AllConcepts() {
		if combat.ConceptOf(id).Label == demoCards[0].Name {
			return id
		}
	}
	log.Fatalf("no card is called %q", demoCards[0].Name)
	return combat.NoConcept
}

// write renders one card, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, name, label string) (cell, error) {
	img, err := cards.Render(s, cards.Hand, f)
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
	return cell{
		File: name, Label: label,
		Width: cards.Hand.Width, Height: cards.Hand.Height,
	}, nil
}

// styleFacts is the numbers the page prints, read off the style rather than typed into the
// template, so the page cannot quote a card it is not showing.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":        st.Width,
		"height":       st.Height,
		"cornerRadius": st.CornerRadius,
		"formSize":     st.FormSize,
		"formTop":      st.FormTop,
		"dashLeft":     st.DashLeft,
		"dashTop":      st.DashTop,
		"dashWidth":    st.DashWidth,
		"dashHeight":   st.DashHeight,
	}
}

type cell struct {
	File   string
	Label  string
	Note   string
	Width  int
	Height int
}

// tipRun is one stretch of a tooltip line in its own color, as CSS. An empty Ink is the panel's
// own. **Named for the tooltip rather than just `run`**, because `run` is this command's entry point.
type tipRun struct {
	Text string
	Ink  string
}

// tip is a card's tooltip as the page prints it: the title and its lines, both in runs.
type tip struct {
	Title []tipRun
	Lines [][]tipRun
}

// plate is one upgrade: how it is acquired, the art it draws, how it looks and how it states.
type plate struct {
	Upgrade string
	Record  string
	Art     string
	Undrawn bool
	Riders  string
	Grants  string
	Tip     tip
	Cells   []cell
	States  []cell
}

type page struct {
	Ground string
	Zoom   int
	Style  map[string]int
	Plain  []cell
	Stacks []cell
	Plates []plate
}
