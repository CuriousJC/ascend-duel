// Command upgradesheet renders every *visible upgrade* a card can carry, against every form mark
// and in every way the mark and the upgrade's ink can be combined, and writes an HTML page.
//
//	go run ./tools/upgradesheet
//
// # Why it exists
//
// Sixteen parasites attach seven kinds of rider and, until 2026-09-07, not one of them reached the
// drawing: a ridden card looked exactly like an unridden one and the only place a rider was
// visible was the tooltip prose. That is a hand of altered cards the player cannot read. See
// TODO.md, which is where the owner asked for it to be tracked.
//
// The wildcard is the first upgrade that says so on the face, and it will not be the last — so
// the review question is not "does this one look right" but "do these read as a *set*, and can a
// player tell two of them apart at 162 pixels". A page per upgrade could never answer that. This
// is the ring sheet's argument applied to a catalogue that has one entry in it, deliberately
// early: the shape of the page is what the second upgrade will be drawn against.
//
// # What to look at
//
// **The three tint modes, side by side, at card scale.** Which of TintWeave, TintRamp and
// TintProject reads best at 32 pixels on an off-white card is a question to answer by looking,
// which is the whole reason all three exist rather than one being chosen in the source. The
// game draws `cards.DefaultTintMode`; this page is what changes it.
//
// **The actual-size row before the enlarged one.** The same rule the glyph sheet is under:
// reviewing only the blown-up row is how a mark comes to look acceptable in review and clunky in
// play. The enlargement here is CSS with `image-rendering: pixelated`, so it is a nearest-neighbour
// blow-up of the real pixels rather than a second, kinder rendering.
//
// **The plain card in every row.** An upgrade is only legible against what an ordinary card looks
// like, and the ordinary card is what the player has fifty-five of.
//
// **All four form marks.** The marks have different margins and different amounts of interior
// detail — a shield is wide-shouldered where a spear is thin and vertical — so an ink that is
// projected across the ink bounds lands differently on each. A mode that works on the sword and
// mud on the shield is a mode that does not work.
//
// **The three card states.** A wildcard's column has to fade with the rest of its card when it
// cannot be afforded, exactly as an element's does, and the ticks and the mark have to move
// together. That is one switch in `Spec.atState`, and this is where it is visible.
//
// **Which parasite grants it.** An upgrade nobody can acquire is invisible in the other
// direction, so the page names the rider and the record that attaches it, and says so loudly when
// nothing in `data/parasites.json` does.
//
// # It is a report, not a drawing-board
//
// Same split as ringsheet against cardsheet. The upgrades come from `systems.Upgrades()`, the
// riders from `combat.RiderKinds()` and the parasites from `internal/session`, which validates the
// catalogue at init — so an upgrade this page cannot draw is one the game cannot draw either.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/upgradesheet/` and **committed**, on
// the same terms as every other sheet. A clone opens `docs/sheets/index.html`.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/assets"
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
	{cards.FormStab, "Lunge", cards.Fire},
	{cards.FormSlash, "Cleave", cards.Ice},
	{cards.FormCrush, "Smash", cards.Lightning},
	{cards.FormDefend, "Guard", cards.Earth},
}

// demoCost is the cost every demonstration card is drawn at. See demoCards.
const demoCost = 3

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
		Ground:  ground,
		Zoom:    zoom,
		Style:   styleFacts(cards.Hand),
		Default: cards.DefaultTintMode.String(),
		InkSize: systems.UpgradeInkSize,
	}

	// The control row: the same four cards with no upgrade at all. It goes first because every
	// judgement below it is a comparison against this.
	plain, err := renderRow(dir, faces, systems.UpgradeNone, cards.DefaultTintMode, "plain")
	if err != nil {
		return err
	}
	page.Plain = plain

	for _, u := range systems.Upgrades() {
		p := plate{
			Upgrade: u.String(),
			Riders:  ridersFor(u),
			Grants:  grantsFor(u),
		}

		for _, mode := range cards.TintModes() {
			row, err := renderRow(dir, faces, u, mode, u.String()+"-"+mode.String())
			if err != nil {
				return err
			}
			p.Modes = append(p.Modes, modeRow{
				Mode:    mode.String(),
				Default: mode == cards.DefaultTintMode,
				Cells:   row,
			})
		}

		// The three states, in the default mode only. Which mode is chosen has nothing to do with
		// whether state works — that is one switch in Spec.atState — so drawing nine cards here
		// would be nine pictures of one fact.
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
			spec := specFor(d.Form, d.Name, d.Element, u, cards.DefaultTintMode)
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

	fmt.Printf("wrote %s — %d upgrades, %d tint modes, %d forms\n",
		out, len(page.Plates), len(cards.TintModes()), len(demoCards))
	for _, p := range page.Plates {
		grants := p.Grants
		if grants == "" {
			grants = "NOTHING GRANTS IT"
		}
		fmt.Printf("  %-8s %-14s %s\n", p.Upgrade, p.Riders, grants)
	}
	return nil
}

// renderRow draws the four demonstration cards for one upgrade in one mode.
func renderRow(dir string, f *cards.Faces, u systems.Upgrade, mode cards.TintMode, tag string) ([]cell, error) {
	out := make([]cell, 0, len(demoCards))
	for _, d := range demoCards {
		name := tag + "-" + d.Form.String() + ".png"
		c, err := write(dir, f, specFor(d.Form, d.Name, d.Element, u, mode), name, d.Name)
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
// **The text is what a card carrying this upgrade actually prints**, taken from the same place the
// game takes it rather than written out here — a sheet quoting its own wording would be the one
// place a mismatch between the face and the rules is invisible.
func specFor(form cards.Form, name string, e cards.Element, u systems.Upgrade, mode cards.TintMode) cards.Spec {
	return cards.Spec{
		Name:        name,
		Form:        form,
		Cost:        demoCost,
		Element:     e,
		Upgrade:     u,
		UpgradeTint: mode,
		Text:        textFor(u),
		Enabled:     true,
	}
}

// textFor is the line an upgraded card carries under its name.
//
// **It is hand-written here and that is a knowingly accepted duplicate.** The real wording is
// `screens.riderText`, which sits above this tool's whole import set — `internal/screens` links
// Ebitengine, so a command-line tool cannot reach it. `tools/cardsheet` keeps its own snapshot of
// the card names and costs for exactly this reason and says so. What that costs is a line that can
// drift; what it buys is a sheet that runs with no window.
func textFor(u systems.Upgrade) string {
	switch u {
	case systems.UpgradeWild:
		return "2x DMG\nANY ELEMENT"
	default:
		return "2x DMG"
	}
}

// ridersFor is the rider kinds that produce this upgrade, as the page names them.
//
// **The mapping is `internal/screens`' and cannot be imported**, for textFor's reason, so it is
// restated here — which is why the page prints it rather than hiding it: a drifted line is
// visible on the sheet instead of being a silent disagreement.
func ridersFor(u systems.Upgrade) string {
	switch u {
	case systems.UpgradeWild:
		return combat.RiderWildElement.String()
	default:
		return ""
	}
}

// grantsFor names every parasite in the catalogue that attaches this upgrade's rider, or an empty
// string if nothing does.
//
// **An upgrade nothing grants is a drawing with no way into the game.** That is exactly as
// invisible as a rider with no drawing, in the other direction, so the page says so loudly rather
// than showing a card that can never be dealt.
func grantsFor(u systems.Upgrade) string {
	want := ridersFor(u)
	out := ""
	for _, p := range session.Parasites() {
		if p.Target != session.ParasiteRider || p.Rider.String() != want {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += p.Name + " (" + p.Record + ")"
	}
	return out
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

// modeRow is one way of combining the ink and the mark, across every form.
type modeRow struct {
	Mode    string
	Default bool
	Cells   []cell
}

// plate is one upgrade: how it is acquired, how it looks under each mode, and how it states.
type plate struct {
	Upgrade string
	Riders  string
	Grants  string
	Modes   []modeRow
	States  []cell
}

type page struct {
	Ground  string
	Zoom    int
	InkSize int
	Default string
	Style   map[string]int
	Plain   []cell
	Plates  []plate
}
