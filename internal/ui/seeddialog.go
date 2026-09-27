package ui

// **The new-run dialog: which run code a journey starts on.**
//
// It is the confirm box's shape grown by one row — same scrim, same bevelled panel, same blue stroke,
// the same answer row along the bottom — because it is still a question rather than a page. What it
// adds is six **wheels**, one per character of the code, each turned by an arrow above and an arrow
// below it.
//
// **Wheels rather than a text field**, so a code is entered by clicking and a focus ring can walk
// every control on it: the game ships on Steam Deck and nothing may require the keyboard. A wheel
// only ever lands on a character `seeds.Code` emits, so there is no code to refuse — the six
// characters on the wheels are always a run.
//
// **It opens on a rolled code**, so START on its own is the ordinary new run and choosing a seed is
// the thing that costs clicks. Whether the code the run starts on is the one rolled is what
// "chosen" means: turning a wheel and turning it back is the rolled run, and RANDOM rolls again
// and is not a choice either.
//
// **START is the destructive answer only when there is something to destroy.** With a journey in
// progress it takes the modal red and the body says what is lost, which is the confirm box's whole
// question folded into this one — two dialogs in a row for one button is one too many.
//
// **A checkbox at the top says whether the run will earn achievements**, and it is a reading rather
// than a control: it is ticked while the wheels hold the rolled code and clears the moment a wheel
// makes the code a chosen one. RANDOM ticks it again. Clicking it does nothing, because what it
// reports is decided by the wheels.
//
// **Every word on it is set in the figure glyphs the buttons wear**, so the dialog reads as one
// piece with its own answer row. A line the set does not cover falls back to the font.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The box and the wheels in it. Fixed sizes rather than percentages, for the confirm box's reason.
const (
	seedDialogWidth  = 800
	seedDialogHeight = 580

	// Offsets down from the box's top edge, and the figure height each line is set at.
	seedTitleTop    = 52
	seedTitleHeight = 34
	seedCheckTop    = 108
	seedCheckSize   = 34
	seedCheckHeight = 20
	seedBodyTop     = 162
	seedWarnTop     = 194
	seedLineHeight  = 16

	// The wheel row: a character cell with an arrow button above and below it.
	seedWheelCenter = 334 // from the box's top edge to the middle of the character cells
	seedCellWidth   = 80
	seedCellHeight  = 92
	seedCellGap     = 20
	seedArrowHeight = 44
	seedArrowGap    = 8
	seedCharHeight  = 50

	// The answer row: three buttons, centered, off the bottom edge.
	seedButtonWidth  = 230
	seedButtonGap    = 20
	seedButtonHeight = confirmButtonHeight
	seedButtonBottom = 40
)

var (
	seedCellColor = color.RGBA{R: 18, G: 18, B: 24, A: 255}
	seedBodyColor = color.RGBA{R: 198, G: 198, B: 208, A: 255}
	seedWarnColor = color.RGBA{R: 240, G: 120, B: 124, A: 255}
)

// SeedDialog is the new-run dialog. A scene owns one and opens it with Open.
type SeedDialog struct {
	open   bool
	code   string // what the wheels read
	rolled string // what they opened on, or were last rolled to
	warn   bool   // a journey is in progress and START throws it away
	roll   func() int64
	start  func(seed int64, chosen bool)

	up, down                    [seeds.CodeLen]*models.Button
	cancel, random, startButton *models.Button
}

// Open puts the dialog up on a rolled code. roll is how RANDOM rolls another; start is called with
// the seed on the wheels and whether the player chose it. warn says a journey is in progress.
func (d *SeedDialog) Open(roll func() int64, warn bool, start func(seed int64, chosen bool)) {
	d.roll, d.warn, d.start = roll, warn, start
	d.rolled = seeds.Code(roll())
	d.code = d.rolled
	d.open = true
}

// IsOpen reports whether the dialog is up; Close puts it away without starting anything.
func (d *SeedDialog) IsOpen() bool { return d.open }
func (d *SeedDialog) Close()       { d.open = false }

// Code is what the wheels currently read.
func (d *SeedDialog) Code() string { return d.code }

// Turn steps wheel i by delta characters. It is what the arrows do.
func (d *SeedDialog) Turn(i, delta int) { d.code = seeds.Step(d.code, i, delta) }

// Chosen reports whether the wheels read something other than the code they were rolled to, which
// is what START records as a seed the player chose.
func (d *SeedDialog) Chosen() bool { return d.code != d.rolled }

// Warns reports whether the dialog is saying a journey in progress will be lost.
func (d *SeedDialog) Warns() bool { return d.warn }

func (d *SeedDialog) build() {
	for i := 0; i < seeds.CodeLen; i++ {
		i := i
		d.up[i] = models.NewButton(seedCellWidth, seedArrowHeight, "", func() { d.Turn(i, 1) })
		d.up[i].BaseColor = confirmCancelColor
		d.down[i] = models.NewButton(seedCellWidth, seedArrowHeight, "", func() { d.Turn(i, -1) })
		d.down[i].BaseColor = confirmCancelColor
	}
	d.cancel = models.NewButton(seedButtonWidth, seedButtonHeight, "CANCEL", func() { d.Close() })
	d.cancel.BaseColor = confirmCancelColor
	d.random = models.NewButton(seedButtonWidth, seedButtonHeight, "RANDOM", func() {
		d.rolled = seeds.Code(d.roll())
		d.code = d.rolled
	})
	d.random.BaseColor = confirmCancelColor
	d.startButton = models.NewButton(seedButtonWidth, seedButtonHeight, "START", nil)
}

// Update runs every control on the dialog. It sets gs.ModalOpen while it is up, so the frame's
// controls stand down as they do under every other dialog.
func (d *SeedDialog) Update(gs *state.GlobalState) {
	if !d.open {
		return
	}
	if d.cancel == nil {
		d.build()
	}

	// Rebound every frame, for the confirm box's reason: the answer is the caller's of the moment.
	d.startButton.BaseColor = color.RGBA{}
	if d.warn {
		d.startButton.BaseColor = ModalCloseColor
	}
	start := d.start
	d.startButton.OnClick = func() {
		seed, err := seeds.Parse(d.code)
		if err != nil {
			return // unreachable: a wheel only lands on characters Code emits
		}
		chosen := d.Chosen()
		d.Close()
		if start != nil {
			start(seed, chosen)
		}
	}

	d.place(gs)
	gs.ModalOpen = true
	for _, b := range d.controls() {
		systems.UpdateButton(gs, b)
	}
}

// controls is every button on the dialog, in the order a focus ring would walk them.
func (d *SeedDialog) controls() []*models.Button {
	out := make([]*models.Button, 0, 2*seeds.CodeLen+3)
	out = append(out, d.up[:]...)
	out = append(out, d.down[:]...)
	return append(out, d.cancel, d.random, d.startButton)
}

// place positions every control against the box. The resolution is fixed, so this is the same
// answer every frame; it is here rather than in Open so a control is never hit-tested at zero.
func (d *SeedDialog) place(gs *state.GlobalState) {
	box := seedRect(gs)
	for i := 0; i < seeds.CodeLen; i++ {
		x, y := seedCellCenter(box, i)
		d.up[i].ScreenX, d.up[i].ScreenY = x, y-seedCellHeight/2-seedArrowGap-seedArrowHeight/2
		d.down[i].ScreenX, d.down[i].ScreenY = x, y+seedCellHeight/2+seedArrowGap+seedArrowHeight/2
	}
	row := box.Max.Y - seedButtonBottom - seedButtonHeight/2
	left := box.Min.X + box.Dx()/2 - (3*seedButtonWidth+2*seedButtonGap)/2
	for i, b := range []*models.Button{d.cancel, d.random, d.startButton} {
		b.ScreenX = left + i*(seedButtonWidth+seedButtonGap) + seedButtonWidth/2
		b.ScreenY = row
	}
}

// Draw puts the scrim, the box, the words, the wheels and the answers over the scene.
func (d *SeedDialog) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	if !d.open || d.cancel == nil {
		return
	}
	ModalScrim(screen)

	box := seedRect(gs)
	systems.BevelRect(screen, box.Min.X, box.Min.Y, box.Dx(), box.Dy(),
		systems.PaneBevelWidth, color.RGBA{R: 30, G: 30, B: 38, A: 255}, false)
	vector.StrokeRect(screen, float32(box.Min.X), float32(box.Min.Y),
		float32(box.Dx()), float32(box.Dy()), 2, panelBlue, false)

	center := float64(box.Min.X + box.Dx()/2)
	lettering(gs, screen, "NEW RUN", center, float64(box.Min.Y+seedTitleTop), seedTitleHeight, color.RGBA{})
	d.drawCheck(gs, screen, box)
	lettering(gs, screen, "START ON THIS CODE, OR TURN THE WHEELS TO CHOOSE ONE.",
		center, float64(box.Min.Y+seedBodyTop), seedLineHeight, seedBodyColor)
	if d.warn {
		lettering(gs, screen, "THE JOURNEY IN PROGRESS WILL BE LOST.",
			center, float64(box.Min.Y+seedWarnTop), seedLineHeight, seedWarnColor)
	}

	for i := 0; i < seeds.CodeLen; i++ {
		x, y := seedCellCenter(box, i)
		systems.BevelRect(screen, x-seedCellWidth/2, y-seedCellHeight/2, seedCellWidth, seedCellHeight,
			systems.PaneBevelWidth, seedCellColor, true)
		lettering(gs, screen, d.code[i:i+1], float64(x), float64(y), seedCharHeight, color.RGBA{})

		systems.DrawButton(gs, screen, d.up[i])
		systems.DrawButton(gs, screen, d.down[i])
		drawChevron(screen, d.up[i], true)
		drawChevron(screen, d.down[i], false)
	}

	systems.DrawButton(gs, screen, d.cancel)
	systems.DrawButton(gs, screen, d.random)
	systems.DrawButton(gs, screen, d.startButton)
}

// drawCheck is the achievements reading: a sunken box, ticked while the run would earn them, and
// its label beside it. The pair is centered on the box as one object.
func (d *SeedDialog) drawCheck(gs *state.GlobalState, screen *ebiten.Image, box image.Rectangle) {
	const label, gap = "ACHIEVEMENTS ENABLED", 16
	width := float64(seedCheckSize + gap)
	if systems.FigureCovers(label) {
		width += systems.MeasureFigure(label, seedCheckHeight)
	}
	left := float64(box.Min.X+box.Dx()/2) - width/2
	cy := float64(box.Min.Y + seedCheckTop)
	top := int(cy) - seedCheckSize/2

	systems.BevelRect(screen, int(left), top, seedCheckSize, seedCheckSize,
		systems.PaneBevelWidth, seedCellColor, true)
	ink := seedBodyColor
	if !d.Chosen() {
		ink = color.RGBA{R: 235, G: 235, B: 240, A: 255}
		x, y, s := float32(left), float32(top), float32(seedCheckSize)
		var p vector.Path
		p.MoveTo(x+s*0.22, y+s*0.52)
		p.LineTo(x+s*0.42, y+s*0.72)
		p.LineTo(x+s*0.78, y+s*0.28)
		vector.StrokePath(screen, &p, &vector.StrokeOptions{Width: 4, LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound},
			&vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(ink)})
	}
	labelLeft := left + float64(seedCheckSize+gap)
	w := width - float64(seedCheckSize+gap)
	lettering(gs, screen, label, labelLeft+w/2, cy, seedCheckHeight, ink)
}

// lettering sets str centered on (cx, cy) in the figure glyphs, `height` pixels tall, multiplied by
// ink (zero draws the sheet as it is). A string the set does not cover is drawn in the font instead.
func lettering(gs *state.GlobalState, screen *ebiten.Image, str string, cx, cy, height float64, ink color.RGBA) {
	if systems.FigureCovers(str) {
		systems.DrawFigure(screen, str, systems.FigureNeutral, ink, cx, cy, height, 1, 1)
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy)
	op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter
	if ink.A != 0 {
		op.ColorScale.ScaleWithColor(ink)
	}
	text.Draw(screen, str, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: height * 1.4}, op)
}

// drawChevron puts a solid triangle on an arrow button, pointing up or down. **A stand-in until
// there is arrow art**, drawn in the button labels' near-white so it reads as the button's label.
func drawChevron(screen *ebiten.Image, b *models.Button, up bool) {
	const half, rise = 14, 10
	cx, cy := float32(b.ScreenX), float32(b.ScreenY)
	tip, base := cy-rise, cy+rise
	if !up {
		tip, base = base, tip
	}
	var p vector.Path
	p.MoveTo(cx, tip)
	p.LineTo(cx+half, base)
	p.LineTo(cx-half, base)
	p.Close()
	ink := color.RGBA{R: 235, G: 235, B: 240, A: 255}
	if b.State == models.ButtonStateDisabled {
		ink = color.RGBA{R: 110, G: 110, B: 110, A: 255}
	}
	vector.FillPath(screen, &p, &vector.FillOptions{}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(ink)})
}

func colorScale(c color.RGBA) ebiten.ColorScale {
	var s ebiten.ColorScale
	s.ScaleWithColor(c)
	return s
}

// seedRect is the box, centered at a fixed size.
func seedRect(gs *state.GlobalState) image.Rectangle {
	left := gs.PctX(50) - seedDialogWidth/2
	top := gs.PctY(50) - seedDialogHeight/2
	return image.Rect(left, top, left+seedDialogWidth, top+seedDialogHeight)
}

// seedCellCenter is the middle of wheel i's character cell.
func seedCellCenter(box image.Rectangle, i int) (int, int) {
	row := seeds.CodeLen*seedCellWidth + (seeds.CodeLen-1)*seedCellGap
	left := box.Min.X + box.Dx()/2 - row/2
	return left + i*(seedCellWidth+seedCellGap) + seedCellWidth/2, box.Min.Y + seedWheelCenter
}
