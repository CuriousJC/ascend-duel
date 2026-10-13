package screens

// combat_rung.go — the plates the cards that made the hand sit on.
//
// **A card on the table never lifts.** On the `KindHand` beat the cards of the rung are set on
// highlight plates drawn behind them, and the plates stay for the rest of that turn — through the
// lines, since they move nothing and can be read beside them. What is paying *now* is still the
// sideways shake; what made the hand is the plate.
//
// **One plate per group of the hand, and per run of neighbours within it.** A Two Pair is two
// pairs, so it is two plates even when the pairs sit side by side; a pair with a dead card between
// its two cards is two plates of one, so the dead card stands on nothing. Where two plates meet they
// leave a gap at the seam, so two groups next to each other never read as one group of four. The
// plate's art stretches to any width — see ui.DrawHandHighlight.

import (
	"image"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// rungPlatePad is how far a plate reaches past its cards, and rungPlateSeam the gap two plates
// leave between them where their cards are neighbours.
const (
	rungPlatePad  = 14
	rungPlateSeam = 10
)

// rungPlateTicks is the plate fading up behind its cards.
func rungPlateTicks() int { return ui.Beat(1, 2) }

// rungRuns cuts a group's places along its row into runs of neighbours, each as its first and last.
func rungRuns(places []int) [][2]int {
	if len(places) == 0 {
		return nil
	}
	sorted := append([]int(nil), places...)
	sort.Ints(sorted)
	runs := [][2]int{{sorted[0], sorted[0]}}
	for _, p := range sorted[1:] {
		last := &runs[len(runs)-1]
		if p == last[1]+1 {
			last[1] = p
			continue
		}
		if p != last[1] {
			runs = append(runs, [2]int{p, p})
		}
	}
	return runs
}

// rungPlates is every plate a hand needs, as runs of places: each group's runs, the groups in
// their own order. `groupOf` says which group a place is in.
func rungPlates(places []int, groupOf func(place int) int) [][2]int {
	byGroup := map[int][]int{}
	var order []int
	for _, p := range places {
		g := groupOf(p)
		if _, seen := byGroup[g]; !seen {
			order = append(order, g)
		}
		byGroup[g] = append(byGroup[g], p)
	}
	sort.Ints(order)
	var plates [][2]int
	for _, g := range order {
		plates = append(plates, rungRuns(byGroup[g])...)
	}
	return plates
}

// drawRungPlates sets the cards of the hand that was announced on their plates, on whichever row
// formed it. **Drawn before the rows**, so the cards stand on it.
func (s *CombatScene) drawRungPlates(gs *state.GlobalState, screen *ebiten.Image) {
	t := s.Theater.rungPlate
	if t.Ticks == 0 {
		return
	}
	alpha := float32(ui.EaseOut(t.Progress()))

	for _, side := range []combat.Side{combat.SideA, combat.SideB} {
		if s.soloAttacker(side) {
			continue
		}
		// **Neighbours on screen, not in the resolved turn**: the row is drawn in the order the
		// cards were picked, so the runs are cut over each card's place along it.
		place, n, at := s.playedPlace, len(s.Theater.resolved), playedSeatAt
		if side == combat.SideB {
			place, n, at = s.enemyPlace, len(s.Theater.enemyDealt), enemySeatAt
		}
		groupAt := map[int]int{}
		var places []int
		for _, seat := range s.firingSeatsOf(side) {
			if seat >= 0 && seat < n {
				p := place(seat)
				places = append(places, p)
				groupAt[p] = s.Theater.rungGroups[seat]
			}
		}
		plates := rungPlates(places, func(p int) int { return groupAt[p] })

		// A plate whose neighbour on either side is another plate's card meets it at a seam.
		owner := map[int]int{}
		for i, run := range plates {
			for p := run[0]; p <= run[1]; p++ {
				owner[p] = i
			}
		}
		for i, run := range plates {
			first, last := at(gs, run[0], n), at(gs, run[1], n)
			r := image.Rect(first.X, first.Y, last.X+cardWidth, last.Y+cardHeight).Inset(-rungPlatePad)
			if j, ok := owner[run[0]-1]; ok && j != i {
				prev := at(gs, run[0]-1, n)
				r.Min.X = (prev.X+cardWidth+first.X)/2 + rungPlateSeam/2
			}
			if j, ok := owner[run[1]+1]; ok && j != i {
				next := at(gs, run[1]+1, n)
				r.Max.X = (last.X+cardWidth+next.X)/2 - rungPlateSeam/2
			}
			ui.DrawHandHighlight(screen, r, alpha)
		}
	}
}
