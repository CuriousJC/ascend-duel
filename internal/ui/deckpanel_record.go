package ui

// **The run's record, under the deck: the tonics it drank and what it has come to so far.**
//
// The grid is the deck the run has built, and the band under it is the rest of what the run chose
// and did — so the panel a player opens to read their build answers "and what else is this run" in
// the same look. The tonics are cards, rested on for what each one does; the figures are the run's
// own tally, the round being fought included.
//
// **It is under the grid rather than beside it**, because the grid's rows already take the panel's
// width and the five of them stop short of its bottom. `deckRecordRect` is the one answer to where
// it is, read by the drawing and the hover alike.
//
// **A panel with no run behind it draws no band.** There is nothing to have drunk and nothing to
// have dealt.

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// The band. **The tonic cards are two fifths of their shop size**, which keeps a whole picture on
// each one inside the room under the grid.
const (
	deckRecordGap      = 18 // between the grid's last row and the band
	deckRecordTonicNum = 2
	deckRecordTonicDen = 5
	deckRecordTonicGap = 10
	deckRecordTonicsW  = 520 // the tonics' share of the band; the figures take the rest
	deckRecordStatColW = 300
	deckRecordStatGap  = 48
	deckRecordStatRows = 5
	deckRecordLineSize = systems.TextMedium
	deckRecordLinePit  = 26
)

// deckRecordStyle is a drunk tonic's card in the band.
func deckRecordStyle() cards.Style {
	return cards.EssenceStyle.Scaled(deckRecordTonicNum, deckRecordTonicDen)
}

// deckRecordRect is the band: the grid's width, from under its last row to the panel's floor.
func deckRecordRect(gs *state.GlobalState) image.Rectangle {
	r := ModalPanelRect(gs)
	left, right := deckGridSpan(r.Min.X, r.Max.X)
	_, _, top := deckGridRegion(gs)
	gridBottom := int(top) + deckRowCount*(cards.Mini.Height+deckRowGap)
	return image.Rect(left, gridBottom+deckRecordGap, right, r.Max.Y-modalBodyBottom)
}

// drunkTonics is the run's tonics in the order they were drunk, the ones the catalog no longer holds
// left out.
func drunkTonics(run *session.Session) []session.Tonic {
	if run == nil {
		return nil
	}
	var out []session.Tonic
	for _, key := range run.DrunkTonics() {
		if t, ok := session.TonicByKey(key); ok {
			out = append(out, t)
		}
	}
	return out
}

// tonicSeats is where each drunk tonic's card sits. **They overlap rather than run off the band** —
// the deck rows' rule — once there are more than the tonics' share holds side by side.
func tonicSeats(band image.Rectangle, n int) []image.Rectangle {
	st := deckRecordStyle()
	pitch := st.Width + deckRecordTonicGap
	if n > 1 {
		if fit := (deckRecordTonicsW - st.Width) / (n - 1); fit < pitch {
			pitch = max(fit, 1)
		}
	}
	top := band.Min.Y + deckColumnHeadDrop
	out := make([]image.Rectangle, n)
	for i := range out {
		x := band.Min.X + i*pitch
		out[i] = image.Rect(x, top, x+st.Width, top+st.Height)
	}
	return out
}

// deckRecordStat is one figure in the band: what it is and how much.
type deckRecordStat struct {
	label string
	n     int
}

// deckRecordStats is the run's figures, in the order the band writes them: down the first column,
// then the second. A panel with no run behind it reads every figure as nothing.
func deckRecordStats(run *session.Session) []deckRecordStat {
	total, most := func(string) int { return 0 }, func(string) int { return 0 }
	if run != nil {
		t := run.StatsSoFar()
		total, most = t.Total, t.MostOf
	}
	return []deckRecordStat{
		{"Fights won", total(session.StatFightsWon)},
		{"Rounds", total(session.StatRounds)},
		{"Damage dealt", total(session.StatDealt)},
		{"Damage taken", total(session.StatTaken)},
		{"Healed", total(session.StatHealed)},
		{"Hits blocked", total(session.StatBlocked)},
		{"Biggest hit", most(session.StatBestHit)},
		{"Cards played", total(session.StatPlayed)},
		{"Vitae earned", total(session.StatVitaeEarned)},
		{"Vitae spent", total(session.StatVitaeSpent)},
	}
}

// hoverDeckRecord points tip at the drunk tonic under the cursor, and reports whether there was one.
// **The last card drawn is the one on top**, so an overlapped row is walked from the right.
func hoverDeckRecord(gs *state.GlobalState, at image.Point, d DeckContents, tip *models.Tooltip) bool {
	tonics := drunkTonics(d.Run)
	seats := tonicSeats(deckRecordRect(gs), len(tonics))
	for i := len(seats) - 1; i >= 0; i-- {
		if at.In(seats[i]) {
			tip.Point(seats[i], TipLine(tonics[i].Name), TipLines(TonicEffect(tonics[i])))
			return true
		}
	}
	return false
}

// drawDeckRecord draws the band.
func drawDeckRecord(gs *state.GlobalState, screen *ebiten.Image, d DeckContents) {
	if d.Run == nil {
		return
	}
	band := deckRecordRect(gs)
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckRecordLineSize}
	head := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: deckColumnHeadSize}

	// The tonics, or a word saying there are none — a blank band reads as a band that failed to draw.
	drawRecordText(screen, "TONICS", head, band.Min.X, band.Min.Y, deckColumnFadedInk)
	tonics := drunkTonics(d.Run)
	if len(tonics) == 0 {
		drawRecordText(screen, "None drunk yet", face, band.Min.X, band.Min.Y+deckColumnHeadDrop, deckColumnFadedInk)
	}
	for i, at := range tonicSeats(band, len(tonics)) {
		if img := CardImage(gs, TonicSpec(gs, tonics[i], true), deckRecordStyle()); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(at.Min.X), float64(at.Min.Y))
			screen.DrawImage(img, op)
		}
	}

	// The figures, in columns of five: a label, and its figure against the column's right edge.
	left := band.Min.X + deckRecordTonicsW + deckRecordStatGap
	drawRecordText(screen, "THIS RUN", head, left, band.Min.Y, deckColumnFadedInk)
	for i, st := range deckRecordStats(d.Run) {
		col, row := i/deckRecordStatRows, i%deckRecordStatRows
		x := left + col*(deckRecordStatColW+deckRecordStatGap)
		y := band.Min.Y + deckColumnHeadDrop + row*deckRecordLinePit
		ink := deckColumnInk
		if st.n == 0 {
			ink = deckColumnFadedInk
		}
		drawRecordText(screen, st.label, face, x, y, deckColumnFadedInk)

		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+deckRecordStatColW), float64(y))
		op.PrimaryAlign = text.AlignEnd
		op.ColorScale.ScaleWithColor(ink)
		systems.DrawText(screen, strconv.Itoa(st.n), face, op)
	}
}

// drawRecordText writes one line of the band with its top-left at (x, y).
func drawRecordText(screen *ebiten.Image, s string, face *text.GoTextFace, x, y int, ink color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(ink)
	systems.DrawText(screen, s, face, op)
}
