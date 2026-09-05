package screens

// The front screen: the logo, and the six things a player can do from a cold start.
//
// **It is the door to a run rather than a door to a screen** *(owner's call, 2026-09-03)*. It
// carried a "Combat" button until then, which navigated — it put the game on the combat screen and
// the run had already been built by `main` before anybody saw the menu. Now the menu is the place
// the run is decided: **New Run** builds one, **Continue** walks back into the one on disk, and
// `main` only rolls a seed and calls `BootRun`. See run.go.
//
// **Continue is dead when there is nothing to continue**, rather than absent. A menu whose entries
// move about between launches is a menu that has to be re-read every time; a greyed row says both
// "this exists" and "not for you yet", which is the same rule the settings screen's music bar is
// under.
//
// **New Run asks first, and only when it would destroy something.** A fresh player pressing it has
// nothing to lose and gets no dialog; a player forty rooms up gets the question. See confirm.go.
//
// **Achievements and Credits hang off here rather than off the run**, because neither is a station
// of a climb: they read the profile and a list of names respectively, and both are things a player
// looks at between runs. They are `actions` calls for the reason Settings is — each records where
// the player came from, so Back works from anywhere.

import (
	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

// The menu's shape. **Six rows now rather than three**, so both the buttons and the step are
// smaller than the 84-tall rows spaced 150 apart that three fitted comfortably. At the old figures
// the sixth row ended 200 pixels below the bottom of the screen.
const (
	titleButtonWidth  = 460
	titleButtonHeight = 84
	titleRowGap       = 88
)

// TitleScene is the front screen: the logo and the menu.
type TitleScene struct {
	newRunButton       *models.Button
	continueButton     *models.Button
	achievementsButton *models.Button
	creditsButton      *models.Button
	settingsButton     *models.Button
	exitButton         *models.Button

	// confirm is the "are you sure" in front of New Run, and it is only ever raised when a run is
	// actually in progress.
	confirm confirmDialog
}

// Init builds the buttons on first entry and positions them every time.
//
// Positioning belongs here rather than in Draw: Update hit-tests against ScreenX/Y,
// so a Draw-time assignment leaves the first frame testing against zeroes and goes
// stale any frame Ebiten chooses to skip Draw. The internal resolution is fixed, so
// these coordinates only need computing once per visit.
func (s *TitleScene) Init(gs *state.GlobalState) {
	if s.newRunButton == nil {
		s.newRunButton = models.NewButton(titleButtonWidth, titleButtonHeight, "NEW RUN",
			func() { s.startNewRun(gs) })
		s.continueButton = models.NewButton(titleButtonWidth, titleButtonHeight, "CONTINUE",
			func() { ContinueRun(gs) })

		// **The two menu screens go through actions, like Settings does.** They are not stations of
		// a run — nothing in flow.go names them — so they record where the player was and Back puts
		// them there, which is the one thing every screen reachable from anywhere has to do.
		s.achievementsButton = models.NewButton(titleButtonWidth, titleButtonHeight, "ACHIEVEMENTS",
			func() { actions.OpenAchievements(gs) })
		s.achievementsButton.TextSize = 40

		s.creditsButton = models.NewButton(titleButtonWidth, titleButtonHeight, "CREDITS",
			func() { actions.OpenCredits(gs) })

		s.settingsButton = models.NewButton(titleButtonWidth, titleButtonHeight, "SETTINGS",
			func() { actions.OpenSettings(gs) })
		s.exitButton = models.NewButton(titleButtonWidth, titleButtonHeight, "EXIT",
			func() { actions.QuitGame(gs) })
	}

	// **The dialog does not survive a visit.** A scene's Init runs again every time it is entered,
	// and arriving at the title with a question already up would be a dialog nobody asked.
	s.confirm.close()

	// The percentage anchors the menu; the fixed steps space it. Giving each button its own
	// percentage would let the spacing drift apart the next time the menu moves.
	menuTop := gs.PctY(33)

	for i, b := range s.menu() {
		b.ScreenX = gs.PctX(50)
		b.ScreenY = menuTop + i*titleRowGap
	}
}

func (s *TitleScene) Update(gs *state.GlobalState) error {
	// **The question owns the screen while it is up.** The menu underneath is still where it was,
	// and a click reaching New Run through the dialog asking about New Run would start two runs.
	if s.confirm.isOpen() {
		s.confirm.update(gs)
		return nil
	}

	// **Dead with nothing to go back to.** A run only exists here if BootRun resumed one off disk
	// or the player started one and came back to the title; either way the test is the same.
	setEnabled(s.continueButton, gs.Run != nil && gs.Resumed)

	for _, b := range s.menu() {
		systems.UpdateButton(gs, b)
	}
	return nil
}

// menu is the rows, top to bottom. **One list read by Init, Update and Draw**, because three
// hand-written orders are three places a new entry can be forgotten — which is exactly how a
// button ends up drawn and not clickable.
func (s *TitleScene) menu() []*models.Button {
	return []*models.Button{
		s.newRunButton, s.continueButton,
		s.achievementsButton, s.creditsButton,
		s.settingsButton, s.exitButton,
	}
}

// startNewRun begins a new climb, asking first if that would throw one away.
//
// **The question is asked about the *saved* run rather than about whatever `gs.Run` happens to
// hold.** A fresh launch has a run standing already — BootRun builds one so the first press of New
// Run is instant — and asking "abandon your run?" about a tower nobody has entered would be a
// dialog that means nothing.
func (s *TitleScene) startNewRun(gs *state.GlobalState) {
	if gs.Run == nil || !gs.Resumed {
		NewRun(gs)
		return
	}
	s.confirm.ask(
		"START A NEW RUN?",
		"The climb in progress will be lost. This cannot be undone.",
		"NEW RUN",
		func() { NewRun(gs) },
	)
}

func (s *TitleScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 109, G: 141, B: 138, A: 255})

	//TITLE
	//
	var title *ebiten.Image
	if gs.CountSecond > 9 && gs.CountSecond%10 == 0 {
		title = gs.Assets["titleEaster_png"]
	} else {
		title = gs.Assets["title_png"]
	}
	bounds := title.Bounds()
	imageCenterX := float64(bounds.Dx()) / 2
	imageCenterY := float64(bounds.Dy()) / 2
	op := &colorm.DrawImageOptions{}
	scaleFactor := 0.75
	op.GeoM.Scale(scaleFactor, scaleFactor)
	op.GeoM.Translate(float64(gs.PctX(50))-imageCenterX*scaleFactor, 150-imageCenterY*scaleFactor)
	hue := float64(1)
	saturation := float64(1)
	value := float64(1)
	var c colorm.ColorM
	c.ChangeHSV(hue, saturation, value)
	colorm.DrawImage(screen, title, c, op)

	//BUTTONS
	//
	// Positions are set in Init; Draw only draws.
	for _, b := range s.menu() {
		systems.DrawButton(gs, screen, b)
	}

	// The build, bottom right. Small and dim on purpose — it is a thing to be *found* when
	// someone is asked "which version are you on", not a thing to be read every time the
	// title screen is looked at.
	versionOp := &text.DrawOptions{}
	versionOp.GeoM.Translate(float64(gs.PctX(100)-versionInset), float64(gs.PctY(100)-versionInset))
	versionOp.PrimaryAlign = text.AlignEnd
	versionOp.SecondaryAlign = text.AlignEnd
	versionOp.ColorScale.ScaleWithColor(versionColor)
	text.Draw(screen, gs.Version,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, versionOp)

	// Over the menu it is asking about.
	s.confirm.draw(gs, screen)
}

// Where the build string sits on the title screen, and how loud it is.
const versionInset = 14

var versionColor = color.RGBA{R: 60, G: 80, B: 78, A: 255}
