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
// move about between launches is a menu that has to be re-read every time; a grayed row says both
// "this exists" and "not for you yet", which is the same rule the settings screen's music bar is
// under.
//
// **New Run opens the new-run dialog**: six wheels holding a rolled run code, which START takes as
// it is or after the player has turned them to a code of their own. With a journey in progress the
// same dialog says it will be lost and START takes the destructive red. See ui/seeddialog.go.
//
// **Achievements and Credits hang off here rather than off the run**, because neither is a station
// of a journey: they read the profile and a list of names respectively, and both are things a player
// looks at between runs. They are `actions` calls for the reason Settings is — each records where
// the player came from, so Back works from anywhere.

import (
	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

// The menu's shape. **Six rows now rather than three**, so both the buttons and the step are
// smaller than the 84-tall rows spaced 150 apart that three fitted comfortably. At the old figures
// the sixth row ended 200 pixels below the bottom of the screen.
const (
	titleButtonWidth  = 460
	titleButtonHeight = ui.ButtonLarge
	titleRowGap       = 88

	// The swirl that stands where the menu will be while the run's faces are painted: how big it
	// is, the least time it turns so a fast machine does not flash it, and how long it takes to
	// fade out as the buttons fade in.
	titleSwirlRadius = 200
	titleSwirlMin    = 45
	titleSwirlFade   = 24
)

// TitleScene is the front screen: the logo and the menu.
type TitleScene struct {
	newRunButton       *models.Button
	continueButton     *models.Button
	achievementsButton *models.Button
	creditsButton      *models.Button
	settingsButton     *models.Button
	exitButton         *models.Button

	// newRun is the dialog New Run opens: the run code the journey starts on, and the warning when
	// there is a journey in progress to lose.
	newRun ui.SeedDialog

	// backdrops is the rooms behind the menu, chosen once a launch — see pickTitleBackdrops. Nil
	// until the first Init, and kept through a return to the title so the picture does not change
	// under a player who only went to the settings.
	backdrops []string

	// **The menu waits for the run's faces.** Until warm has painted them, a swirl turns where the
	// buttons will be; then the swirl fades out as the menu fades in, over titleSwirlFade ticks. See
	// warm.go. ready is kept through a return to the title, so only the first visit of a launch
	// waits.
	warm  faceWarmer
	swirl models.Swirl
	ticks int
	ready bool
	fade  int

	// layer is what each half of the cross-fade is drawn into before it goes on the screen at its
	// own strength — a button and a swirl carry no alpha of their own.
	layer *ebiten.Image
}

// Init builds the buttons on first entry and positions them every time.
//
// Positioning belongs here rather than in Draw: Update hit-tests against ScreenX/Y,
// so a Draw-time assignment leaves the first frame testing against zeroes and goes
// stale any frame Ebiten chooses to skip Draw. The internal resolution is fixed, so
// these coordinates only need computing once per visit.
func (s *TitleScene) Init(gs *state.GlobalState) {
	if s.backdrops == nil {
		s.backdrops = pickTitleBackdrops(gs)
	}
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
	s.newRun.Close()

	// The percentage anchors the menu; the fixed steps space it. Giving each button its own
	// percentage would let the spacing drift apart the next time the menu moves.
	menuTop := gs.PctY(33)

	for i, b := range s.menu() {
		b.ScreenX = gs.PctX(50)
		b.ScreenY = menuTop + i*titleRowGap
	}

	if !s.ready {
		s.warm.start(gs)
		if s.layer == nil {
			s.layer = ebiten.NewImage(state.ScreenWidth, state.ScreenHeight)
		}
		// **The buttons are painted during the wait too**, as the last job, so the frame the menu
		// starts to appear is not also the frame every button paints its face for the first time.
		s.warm.jobs = append(s.warm.jobs, func() {
			for _, b := range s.menu() {
				systems.DrawButton(gs, s.layer, b)
			}
			s.layer.Clear()
		})
		s.ticks, s.fade = 0, 0
		s.swirl = models.Swirl{
			ScreenX: gs.PctX(50), ScreenY: menuTop + (len(s.menu())-1)*titleRowGap/2,
			Radius: titleSwirlRadius, State: models.ButtonStateDisabled,
		}
	}
}

// updateSwirl turns the swirl and paints faces, and reports whether the menu may be shown yet.
func (s *TitleScene) updateSwirl(gs *state.GlobalState) bool {
	if s.ready {
		return true
	}
	s.ticks++
	systems.UpdateSwirl(gs, &s.swirl)
	if s.warm.step() || s.ticks < titleSwirlMin {
		return false
	}
	s.fade++
	if s.fade >= titleSwirlFade {
		s.ready = true
	}
	return s.ready
}

func (s *TitleScene) Update(gs *state.GlobalState) error {
	// **The question owns the screen while it is up.** The menu underneath is still where it was,
	// and a click reaching New Run through the dialog asking about New Run would start two runs.
	if s.newRun.IsOpen() {
		s.newRun.Update(gs)
		return nil
	}

	// **Dead with nothing to go back to.** A run only exists here if BootRun resumed one off disk
	// or the player started one and came back to the title; either way the test is the same.
	ui.SetEnabled(s.continueButton, gs.Run != nil && gs.Resumed)

	if !s.updateSwirl(gs) {
		return nil
	}

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

// startNewRun opens the new-run dialog.
//
// **The warning is about the *saved* run rather than about whatever gs.Run happens to hold.** A
// fresh launch has a run standing already — BootRun builds one so the first press of New Run is
// instant — and warning "your journey will be lost" about a journey nobody has entered would be a
// sentence that means nothing.
func (s *TitleScene) startNewRun(gs *state.GlobalState) {
	inProgress := gs.Run != nil && gs.Resumed
	s.newRun.Open(
		func() int64 { return RollSeed(gs) },
		inProgress,
		func(seed int64, chosen bool) { NewRunOn(gs, seed, chosen) },
	)
}

func (s *TitleScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	drawTitleBackdrops(gs, screen, s.backdrops)

	// The logo, committed at the size it is drawn so nothing resamples it every frame.
	title := gs.Assets["title_png"]
	bounds := title.Bounds()
	imageCenterX := float64(bounds.Dx()) / 2
	imageCenterY := float64(bounds.Dy()) / 2
	// Drawn as authored: the logo's colors are the art's, with no color matrix over them.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(gs.PctX(50))-imageCenterX, titleLogoCenterY-imageCenterY)
	screen.DrawImage(title, op)

	//BUTTONS
	//
	// Positions are set in Init; Draw only draws. **The swirl stands in their place until the run's
	// faces are painted** — see updateSwirl.
	switch {
	case s.ready:
		for _, b := range s.menu() {
			systems.DrawButton(gs, screen, b)
		}
	case s.fade == 0:
		systems.DrawSwirl(screen, &s.swirl, portalSwirl(gs))
	default:
		t := float32(s.fade) / titleSwirlFade
		s.layer.Clear()
		systems.DrawSwirl(s.layer, &s.swirl, portalSwirl(gs))
		s.drawLayer(screen, 1-t)
		s.layer.Clear()
		for _, b := range s.menu() {
			systems.DrawButton(gs, s.layer, b)
		}
		s.drawLayer(screen, t)
	}

	// The build, bottom right. Small on purpose — it is a thing to be *found* when someone is asked
	// "which version are you on", not a thing to be read every time the title screen is looked at.
	// **In the prose glyphs, white under their outline**, because it sits on whichever painted room
	// the launch drew and no one ink reads on all of them.
	if systems.ProseCovers(gs.Version) {
		w := systems.MeasureProse(gs.Version, versionCap)
		systems.DrawProse(screen, gs.Version, color.RGBA{},
			float64(gs.PctX(100)-versionInset)-w, float64(gs.PctY(100)-versionInset), versionCap)
	} else {
		versionOp := &text.DrawOptions{}
		versionOp.GeoM.Translate(float64(gs.PctX(100)-versionInset), float64(gs.PctY(100)-versionInset))
		versionOp.PrimaryAlign = text.AlignEnd
		versionOp.SecondaryAlign = text.AlignEnd
		versionOp.ColorScale.ScaleWithColor(versionColor)
		text.Draw(screen, gs.Version,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextSmall}, versionOp)
	}

	// Over the menu it is asking about.
	s.newRun.Draw(gs, screen)
}

// Where the build string sits on the title screen, and how loud it is.
// titleLogoCenterY is where the logo is centered, clear of the menu below it.
const titleLogoCenterY = 165

const versionInset = 14

// versionCap is the build string's capital height, the prose set's ten-pixel floor.
const versionCap = 10

var versionColor = color.RGBA{R: 60, G: 80, B: 78, A: 255}

// drawLayer puts the cross-fade's layer on the screen at a strength between 0 and 1.
func (s *TitleScene) drawLayer(screen *ebiten.Image, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(alpha)
	screen.DrawImage(s.layer, op)
}
