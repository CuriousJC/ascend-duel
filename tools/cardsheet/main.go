// Command cardsheet renders every card variation to a PNG and writes an HTML page that
// shows them together.
//
//	go run ./tools/cardsheet
//
// It exists for the same reason tools/glyphsheet does: **the art can be reviewed by
// opening a file instead of by launching the game**, and in the card's case launching
// the game is worse than usual — seeing a lightning Cleave you cannot afford means dealing
// yourself the hand that contains one. This renders that card directly.
//
// It needs no graphics context, because internal/cards draws into a plain Go image. That
// is the whole reason the package was split out of internal/screens, and it is the same
// trick systems.RenderGlyph already used.
//
// **The sheet cannot lie about the game.** Both this tool and the combat screen call
// cards.Render and blit the result, so there is no second drawing path to drift. An
// earlier version of the glyph sheet previewed art at a scale the cards did not use, and
// the glyphs duly looked fine in review and clunky in play.
//
// # Output
//
// Loose PNGs plus an index.html beside them, all written into this directory and all
// gitignored. Loose files rather than one composite because the border colours and the
// card's parts are still moving — the expectation is that some of these become sprites
// and some stay generated, and that sorting is easier when each piece is its own file.
//
// This differs from glyphsheet, which commits a single glyphs.png so GitHub renders an
// image diff in review. Twenty-odd regenerated binaries would not produce a diff anyone
// wants, which is the same reason /demo/ is ignored. If a committed review artefact
// turns out to be wanted, the fix is one composite PNG alongside these, not committing
// all of them.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/internal/cards"
)

// maxCost is how many action points the sheet sweeps up to. Costs run 1 to 4 in
// internal/combat today; the sheet goes to that and no further, because a row of
// impossible cards would be reviewing a card the game cannot deal.
const maxCost = 4

// deckStackPitch mirrors the constant of the same name in internal/screens: how far apart
// the deck overlay lays its cards, and so how much of each one shows. The sheet has to use
// the game's number or its deck section would be previewing an overlap the game does not
// draw — the same failure the old glyph sheet had when it previewed at the wrong scale.
const deckStackPitch = 75

// ground is the combat screen's background, so a card is judged against what it actually
// sits on rather than against a white browser page. It is the literal value from
// combat.go's screen.Fill.
const ground = "#323232"

func main() {
	dir := flag.String("dir", filepath.Join("tools", "cardsheet"),
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

	page := page{Ground: ground, Style: styleFacts(cards.Hand)}

	// Section one, as specified: every border colour against every AP count. The card
	// underneath is held constant so the only things varying are the two axes.
	//
	// **The constant card is a real one**, built from the concept table rather than written out
	// here, so it carries its effect text like every other card on the sheet. A row of cards
	// with an empty right-hand side would be showing a layout the game never draws.
	for _, e := range cards.Elements() {
		row := row{Label: e.String()}
		for cost := 1; cost <= maxCost; cost++ {
			spec := specFor("Strike", e)
			spec.Cost = cost
			cell, err := write(dir, faces, spec, cards.Hand,
				fmt.Sprintf("card-%s-ap%d.png", e, cost),
				fmt.Sprintf("%s · %d AP", e, cost))
			if err != nil {
				return err
			}
			row.Cells = append(row.Cells, cell)
		}
		page.Borders = append(page.Borders, row)
	}

	// Section two: the four family marks, which is the only axis where the *mark* is
	// what varies rather than a colour or a count.
	//
	// **They are letters rather than art** *(2026-08-15)*, so this row is where the placeholder
	// gets judged: four capitals have to be told apart in a corner at a glance, and S against D
	// against C is the pairing that decides whether they can be lived with until glyphs land.
	famRow := row{Label: "family mark"}
	for _, fam := range cards.Families() {
		spec := specFor("Strike", cards.Fire)
		spec.Family = fam
		cell, err := write(dir, faces, spec, cards.Hand,
			fmt.Sprintf("family-%s.png", fam),
			fmt.Sprintf("%s — %s", fam, familyNotes[fam]))
		if err != nil {
			return err
		}
		famRow.Cells = append(famRow.Cells, cell)
	}
	page.Families = append(page.Families, famRow)

	// Section three: the states. These are here because the redesign changed how they have
	// to work — the surface used to be the element colour and dimming it was the whole
	// signal, and on a constant off-white face that signal is gone. What replaced it is
	// the border moving toward the surface, and this is where that gets judged.
	states := []struct {
		name  string
		label string
		spec  cards.Spec
	}{
		// **Built from the concept table rather than written out**, unlike the two sections
		// above: those vary one axis on a card that is deliberately not real, and this one has
		// to be a real card because the effect text is part of what a state has to do —
		// disabled dims the ink and there is now a paragraph of it to get wrong.
		{"enabled", "enabled — affordable, not queued", specFor("Strike", cards.Fire)},
		{"selected", "selected — queued this round", selected(specFor("Strike", cards.Fire))},
		{"disabled", "disabled — cannot afford it", disabled(specFor("Strike", cards.Fire))},
	}
	stateRow := row{Label: "card state"}
	for _, st := range states {
		cell, err := write(dir, faces, st.spec, cards.Hand, "state-"+st.name+".png", st.label)
		if err != nil {
			return err
		}
		stateRow.Cells = append(stateRow.Cells, cell)
	}
	page.States = append(page.States, stateRow)

	// Section four: the shapes a real deck actually contains. The grids above hold the card
	// constant, which is exactly what hides a name that overruns its width or an effect text
	// that wraps a line too far.
	shapeRow := row{Label: "real cards"}
	for _, spec := range realCards() {
		cell, err := write(dir, faces, spec, cards.Hand,
			fmt.Sprintf("shape-%s.png", spec.Name),
			fmt.Sprintf("%s · %s · %d AP", spec.Name, spec.Family, spec.Cost))
		if err != nil {
			return err
		}
		shapeRow.Cells = append(shapeRow.Cells, cell)
	}
	page.Shapes = append(page.Shapes, shapeRow)

	// Section five: the back, at both sizes, with a face beside it. The face is the point of
	// the row — the back has to share its silhouette exactly, or a flip would change outline
	// halfway through and a stack of backs would read as a different object next to the hand.
	// Two pictures side by side is the only way to check that.
	backs := []struct {
		name  string
		label string
		spec  cards.Spec
		style cards.Style
	}{
		// Every mark, at both sizes that show one. **A duelist and a card back go together**
		// — `data/duelists.json` names one of these — so the row is a picture of the choice
		// that file offers, which is otherwise only visible by editing it and relaunching.
		{"back-triangle-hand", "triangle — hand size", cards.Spec{FaceDown: true, Back: cards.MarkTriangle}, cards.Hand},
		{"back-diamond-hand", "diamond — hand size", cards.Spec{FaceDown: true, Back: cards.MarkDiamond}, cards.Hand},
		{"back-chevron-hand", "chevron — hand size", cards.Spec{FaceDown: true, Back: cards.MarkChevron}, cards.Hand},
		{"back-triangle-stack", "triangle — draw pile", cards.Spec{FaceDown: true, Back: cards.MarkTriangle}, cards.Stack},
		{"back-diamond-stack", "diamond — draw pile", cards.Spec{FaceDown: true, Back: cards.MarkDiamond}, cards.Stack},
		{"back-chevron-stack", "chevron — draw pile", cards.Spec{FaceDown: true, Back: cards.MarkChevron}, cards.Stack},
		{"back-face", "a face, for the silhouette", specFor("Strike", cards.Fire), cards.Hand},
	}
	backRow := row{Label: "card back"}
	for _, b := range backs {
		cell, err := write(dir, faces, b.spec, b.style, b.name+".png", b.label)
		if err != nil {
			return err
		}
		backRow.Cells = append(backRow.Cells, cell)
	}
	page.Backs = append(page.Backs, backRow)

	// Section six: the deck overlay. One row per element, cards at a third size and
	// overlapped so only the left half of each shows. Twelve concepts per element is what
	// the real deck holds, so these rows are the length they will actually be in game.
	for _, e := range cards.Elements() {
		stack := stack{Label: e.String(), Overlap: deckStackPitch}
		for _, spec := range realDeckRow(e) {
			cell, err := write(dir, faces, spec, cards.Mini,
				fmt.Sprintf("mini-%s-%s.png", e, spec.Name), "")
			if err != nil {
				return err
			}
			stack.Cells = append(stack.Cells, cell)
		}
		page.Deck = append(page.Deck, stack)
	}

	// Section six: the first ring. Same format, pink border, artwork instead of glyphs,
	// and nothing about cost or phase because a ring is not played from a hand.
	rings, err := ringSpecs()
	if err != nil {
		return err
	}
	ringRow := row{Label: "rings — same format, pink border"}
	for _, spec := range rings {
		state := "at rest"
		if spec.Dragging {
			state = "being dragged"
		}
		cell, err := write(dir, faces, spec, cards.RingStyle,
			fmt.Sprintf("ring-%s-%t.png", spec.Name, spec.Dragging),
			fmt.Sprintf("%s — %s", spec.Name, state))
		if err != nil {
			return err
		}
		ringRow.Cells = append(ringRow.Cells, cell)
	}
	page.Rings = append(page.Rings, ringRow)

	// Section seven: the enemy. Same format again, a portrait instead of glyphs, and a health
	// bar and a life fraction where an action card puts its cost and damage.
	enemies, err := enemySpecs()
	if err != nil {
		return err
	}
	enemyRow := row{Label: "enemies — portrait, name, health"}
	for _, spec := range enemies {
		cell, err := write(dir, faces, spec, cards.EnemyStyle,
			fmt.Sprintf("enemy-%s.png", spec.Name),
			fmt.Sprintf("%s — %d/%d", spec.Name, spec.Life, spec.MaxLife))
		if err != nil {
			return err
		}
		enemyRow.Cells = append(enemyRow.Cells, cell)
	}
	page.Rings = append(page.Rings, enemyRow)

	// Section eight: the player. The enemy card's twin — same format, same health bar at the
	// same offsets, three stat rows where the opponent has a portrait. Four states, because
	// the figures are the only thing on it that moves.
	duelistRow := row{Label: "duelists — name, figures, health"}
	for _, spec := range duelistSpecs() {
		cell, err := write(dir, faces, spec, cards.DuelistStyle,
			fmt.Sprintf("duelist-%s-%d.png", spec.Name, spec.Life),
			fmt.Sprintf("%s — %d/%d", spec.Name, spec.Life, spec.MaxLife))
		if err != nil {
			return err
		}
		duelistRow.Cells = append(duelistRow.Cells, cell)
	}
	page.Rings = append(page.Rings, duelistRow)

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	fmt.Printf("wrote %s and %d PNGs — open it and refresh after each re-run\n", out, page.count())
	return nil
}

// write renders one card, saves it, and returns what the page needs to show it.
func write(dir string, f *cards.Faces, s cards.Spec, st cards.Style, name, label string) (cell, error) {
	img, err := cards.Render(s, st, f)
	if err != nil {
		return cell{}, fmt.Errorf("rendering %s: %w", name, err)
	}
	if err := savePNG(filepath.Join(dir, name), img); err != nil {
		return cell{}, err
	}
	return cell{
		File:   name,
		Label:  label,
		Width:  st.Width,
		Height: st.Height,
	}, nil
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return nil
}

// styleFacts is the numbers the page prints under its heading, read off the style rather
// than typed into the template. A sheet quoting a corner radius the cards do not use
// would be the same failure as a preview at the wrong scale.
func styleFacts(st cards.Style) map[string]int {
	return map[string]int{
		"width":        st.Width,
		"height":       st.Height,
		"cornerRadius": st.CornerRadius,
		"borderWidth":  st.BorderWidth,
		"dashWidth":    st.DashWidth,
		"dashHeight":   st.DashHeight,
		"dashGap":      st.DashGap,
	}
}

type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

type row struct {
	Label string
	Cells []cell
}

// stack is a row drawn overlapping: each card sits Overlap pixels left of where it would
// otherwise, so only that much of it shows. The deck overlay uses it to fit sixty cards
// on one panel.
type stack struct {
	Label   string
	Overlap int
	Cells   []cell
}

type page struct {
	Ground   string
	Style    map[string]int
	Borders  []row
	Families []row
	States   []row
	Shapes   []row
	Backs    []row
	Deck     []stack
	Rings    []row
}

func (p page) count() int {
	n := 0
	for _, rs := range [][]row{p.Borders, p.Families, p.States, p.Shapes, p.Backs, p.Rings} {
		for _, r := range rs {
			n += len(r.Cells)
		}
	}
	for _, st := range p.Deck {
		n += len(st.Cells)
	}
	return n
}
