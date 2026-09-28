package screens

// The title screen's ground: three painted rooms side by side, cut apart by two inked slashes, and a
// different three every launch.

import (
	"image/color"
	"math/rand"
	"time"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// titlePanels is how many rooms the title shows, in equal-width panels.
	titlePanels = 3

	// titleSlashLean is how far a divider's top sits right of its bottom, so each cut reads as a
	// slash rather than a seam. The divider crosses its panel boundary at mid-height, which is what
	// keeps the three panels the same width.
	titleSlashLean = 140

	// titleSlashWidth is the inked cut between two rooms: the same heavy black the button and bar
	// art carry their contour in.
	titleSlashWidth = 14
)

var titleSlashInk = color.RGBA{A: 255}

// pickTitleBackdrops chooses the rooms the title shows, left to right.
//
// **Only a painted room is a candidate**, so a room whose picture is still to be generated never
// puts the default backdrop on the front screen. Different rooms are preferred over one room in two
// elements, which would read as the same picture twice; a catalog with fewer rooms than panels
// repeats rooms before it shows fewer panels.
//
// **The clock seeds this, deliberately, and nothing else reads it.** The title is not a station of a
// run and decides nothing: the run seed is not used because a continued run would then show the
// same three rooms every launch, and a pinned seed would freeze them. It is its own *rand.Rand, never
// the package-level functions, so it advances no stream a run owns.
func pickTitleBackdrops(gs *state.GlobalState) []string {
	type candidate struct{ room, key string }
	var all []candidate
	for _, motif := range data.MotifOrder(gs.Motifs) {
		for _, b := range gs.Motifs[motif].Backdrops {
			for _, e := range b.Affinities {
				if key := b.ArtKey(e); len(gs.ImageData[key]) > 0 {
					all = append(all, candidate{b.Backdrop, key})
				}
			}
		}
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	var picked []string
	rooms := map[string]bool{}
	for _, c := range all {
		if len(picked) == titlePanels {
			return picked
		}
		if !rooms[c.room] {
			rooms[c.room] = true
			picked = append(picked, c.key)
		}
	}
	for _, c := range all {
		if len(picked) == titlePanels {
			break
		}
		if !contains(picked, c.key) {
			picked = append(picked, c.key)
		}
	}
	return picked
}

// drawTitleBackdrops paints the chosen rooms as slanted panels, or the plain ground when no room has
// been painted at all.
//
// Each panel is its room drawn at the screen's own scale and centered on the panel, so what shows is
// the middle third of the picture — where a room's door and its subject are painted. The panels are
// two triangles apiece textured straight from the picture, so the slanted edge is a clip rather than
// a mask.
func drawTitleBackdrops(gs *state.GlobalState, screen *ebiten.Image, keys []string) {
	ui.FillGround(screen)
	if len(keys) == 0 {
		return
	}
	sw, sh := float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())
	n := len(keys)
	step := sw / float32(n)
	lean := float32(titleSlashLean) / 2

	// edge is where divider i crosses row y; the outer edges are the screen's own, and sit past it
	// so the slant of a neighbouring divider never leaves a gap at a corner.
	edge := func(i int, top bool) float32 {
		switch i {
		case 0:
			return -lean
		case n:
			return sw + lean
		}
		x := step * float32(i)
		if top {
			return x + lean
		}
		return x - lean
	}

	for i, key := range keys {
		picture := ui.Backdrop(gs, key)
		if picture == nil {
			continue
		}
		pw, ph := float32(picture.Bounds().Dx()), float32(picture.Bounds().Dy())
		scale := max(sw/pw, sh/ph)
		center := step*float32(i) + step/2
		src := func(x, y float32) (float32, float32) {
			return (x-center)/scale + pw/2, (y-sh/2)/scale + ph/2
		}
		corners := [4][2]float32{
			{edge(i, true), 0}, {edge(i+1, true), 0},
			{edge(i+1, false), sh}, {edge(i, false), sh},
		}
		vs := make([]ebiten.Vertex, 4)
		for k, c := range corners {
			sx, sy := src(c[0], c[1])
			vs[k] = ebiten.Vertex{DstX: c[0], DstY: c[1], SrcX: sx, SrcY: sy,
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
		}
		screen.DrawTriangles(vs, []uint16{0, 1, 2, 0, 2, 3}, picture,
			&ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear, Address: ebiten.AddressClampToZero})
	}

	for i := 1; i < n; i++ {
		vector.StrokeLine(screen, edge(i, true), 0, edge(i, false), sh,
			titleSlashWidth, titleSlashInk, true)
	}
}
