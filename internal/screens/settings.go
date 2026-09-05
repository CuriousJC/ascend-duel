package screens

// The settings screen: how loud the score is, how fast the game moves, and the way back.
//
// **It is the program's screen, not a run's.** Every other scene in this package is a station of
// a climb — a duel, a reward, a shop — and is reached by `advance` walking the run forward. This
// one is reached from the cog in the game's chrome, from any screen, and it puts the player back
// exactly where they were: see `state.ReturnScreen`. The run's phase is never touched, which is
// what makes opening it mid-climb a look at a dialog rather than a decision.
//
// **Two controls today, and the third is expected.** Music and speed are the two things the game
// already has; combat sounds are not built, so there is no bar for them — a slider setting a
// number nothing reads would be a control that lies about what it does.
//
// **Every *setting* here may only change what the player sees or hears.** The speed bar scales
// `clock.go`'s one beat, and a whole round is resolved before playback begins, so it moves
// pictures and nothing else. That is the same constraint the debug flags and `internal/trace` are
// under, and it is the reason a game-speed control is safe to offer at all.
//
// **Abandon Run is the one thing on this screen that is not a setting, and it is deliberate**
// *(owner's call, 2026-09-03)*. The game had no way to give up a climb and start over — quitting
// meant the window's X, and the next launch resumed exactly where it left off — and this is the
// one screen reachable from everywhere, which is what a "give up" control has to be.
//
// The objection was raised and overruled: it does file a run decision under the program's screen,
// which is the distinction the paragraph above draws. What that costs is paid in the layout rather
// than pretended away — the button stands below a rule, in the destructive red, with its own
// confirm in front of it, so it reads as a different kind of thing from the two bars. It is also
// the only thing on this screen that touches `gs.Run`, and it does so by calling `AbandonRun`
// rather than by knowing anything about a climb. See run.go.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The screen's layout. A title, a column of bars under it, and one button at the bottom.
const (
	// settingsSliderWidth is how long a bar is. Wide, because the whole value of a bar over a row
	// of buttons is that the travel is fine enough to find a level you like — a short bar is a
	// row of buttons that is harder to hit.
	settingsSliderWidth = 460

	// settingsSliderHeight is the label band plus the track plus the knob's overhang.
	settingsSliderHeight = 56

	// settingsRowGap is the space between one bar and the next, measured from centre to centre.
	settingsRowGap = 100

	// settingsToggleHeight is the fullscreen button. **Shorter than a bar's row**, because a
	// latched button has no label band above it — the word is on the face.
	settingsToggleHeight = 52
)

// settingsTitleSize and settingsTitle are the heading.
const (
	settingsTitleSize = 40
	settingsTitle     = "SETTINGS"

	// **It names the state it puts you in, not the state you are in**, which is the rule a latched
	// control follows: the latch says whether it is on, so a label that also said so would be the
	// same fact twice and would read as wrong in one of the two states.
	settingsFullscreenLabel = "FULL SCREEN"
)

// The abandon band: the rule that separates it from the settings, and the button under it.
const (
	// settingsAbandonGap is how far below the last bar the rule sits. Wide, because the whole job
	// of the gap is to say that what is under it is not another setting.
	settingsAbandonGap = 76

	// settingsAbandonRuleWidth matches the bars, so the rule reads as the end of that column
	// rather than as a decoration of its own.
	settingsAbandonRuleWidth = settingsSliderWidth

	settingsAbandonLabel = "ABANDON RUN"

	// settingsExitLabel leaves the program, under the button that leaves the run. settingsExitGap
	// clears the note belonging to the button above it.
	settingsExitLabel = "EXIT"
	settingsExitGap   = 92
)

// The run code rides on the abandon button *(owner's call, 2026-09-05)*.
//
// **It is here because this is the screen a player can always reach**, and because it names the run
// the button is about to throw away — the two facts belong on one face. It was a quiet caption in
// the bottom-right corner until the button took it. The whole point of a six-character code is
// being able to write down the run you are *in*, to report a bug against it or to hand it to
// somebody, and the cog is on every screen, so this is the one place the answer is always two
// clicks away.
const settingsSeedCaption = "SEED"

// SettingsScene is the settings screen.
type SettingsScene struct {
	music *models.Slider
	speed *models.Slider
	back  *models.Button

	// abandon gives up the run, and confirm is the question in front of it. **Never one without
	// the other**: this is the only irreversible control in the game reachable from every screen.
	// full is the fullscreen toggle: **a latched button rather than a bar**, because unlike the
	// two settings above it this is genuinely a pair of states rather than a level. `Latched` is
	// already how the game draws a mode that is on — sunken and darker — so this needed no new
	// widget.
	full *models.Button

	abandon *models.Button
	confirm confirmDialog

	// exit closes the game. Live whether or not a run is standing, unlike abandon.
	exit *models.Button
}

// Init builds the controls on first entry and positions them every time.
//
// **The bars are set from what is in force rather than from the profile**, which is the same
// question asked of the thing that answers it: `music.Level()` is what the audio device is
// actually at and `screens.Speed()` is what the clock is actually running at. Reading the profile
// would let the screen show a level the game is not playing at, on a machine where the profile
// could not be written.
func (s *SettingsScene) Init(gs *state.GlobalState) {
	if s.music == nil {
		s.music = models.NewSlider(settingsSliderWidth, settingsSliderHeight, "MUSIC", 0)
		s.music.Ink = groundInk
		s.music.OnChange = func(v float64) { music.SetLevel(v) }
		s.music.OnCommit = func(v float64) { s.commit(gs) }

		s.speed = models.NewSlider(settingsSliderWidth, settingsSliderHeight, "GAME SPEED", 0)
		s.speed.Ink = groundInk
		s.speed.OnChange = func(v float64) { SetSpeed(speedFor(v)) }
		s.speed.OnCommit = func(v float64) { s.commit(gs) }

		s.full = models.NewButton(settingsSliderWidth, settingsToggleHeight,
			settingsFullscreenLabel, func() { s.toggleFullscreen(gs) })
		s.full.TextSize = 40

		s.back = models.NewButton(320, 80, "BACK", func() { s.leave(gs) })

		s.abandon = models.NewButton(760, 76, settingsAbandonLabel, func() { s.askAbandon(gs) })

		// **The modal X's red, which is the only red in the game.** It is already the colour of
		// the one control that gets you out of somewhere, and this is the largest version of that
		// there is. Nothing else on this screen is anything but slate.
		s.abandon.BaseColor = modalCloseColor
		s.abandon.TextSize = 40

		s.exit = models.NewButton(760, 76, settingsExitLabel, func() { actions.QuitGame(gs) })
		s.exit.TextSize = 40
	}

	// **The question does not survive a visit**, for the reason the title screen's does not: Init
	// runs again on every entry and arriving with a dialog up is a dialog nobody asked for.
	s.confirm.close()

	// Re-read every visit. The level can have moved since the last one — a fresh profile is
	// silent and the game may have been played for an hour since — and Init runs on every entry.
	s.music.Value = music.Level()
	s.speed.Value = speedValue(Speed())
	s.full.Latched = Fullscreen()

	// **The bar is only live if there is anything to hear.** Opening the audio device is allowed
	// to fail, and a volume control that silently did nothing would be worse than one that says
	// it cannot — the same rule the chrome's mute button was under before it became a cog.
	s.music.Disabled = !music.Available()

	centre := gs.PctX(50)
	top := gs.PctY(38)

	s.music.ScreenX, s.music.ScreenY = centre, top
	s.speed.ScreenX, s.speed.ScreenY = centre, top+settingsRowGap
	s.full.ScreenX, s.full.ScreenY = centre, top+2*settingsRowGap

	// The abandon band, below the rule; Back stays last, at the bottom of the screen, because the
	// way out of a screen is the last thing on it.
	s.abandon.ScreenX, s.abandon.ScreenY = centre, s.abandonRuleY(gs)+56
	s.exit.ScreenX, s.exit.ScreenY = centre, s.abandon.ScreenY+settingsExitGap
	s.back.ScreenX, s.back.ScreenY = centre, gs.PctY(88)
}

// abandonRuleY is where the rule between the settings and the abandon band is drawn. One function
// so the rule and the button under it cannot drift apart.
func (s *SettingsScene) abandonRuleY(gs *state.GlobalState) int {
	return gs.PctY(38) + 2*settingsRowGap + settingsAbandonGap
}

// toggleFullscreen flips the display and records it.
//
// **It reads the window back rather than trusting the flip**, so a request the platform refuses
// leaves the button saying what is actually true instead of what was asked for.
func (s *SettingsScene) toggleFullscreen(gs *state.GlobalState) {
	SetFullscreen(!Fullscreen())
	s.full.Latched = Fullscreen()
	s.commit(gs)
}

func (s *SettingsScene) Update(gs *state.GlobalState) error {
	// **The question owns the screen while it is up**, exactly as it does on the title. A drag
	// reaching a bar through the dialog would be a volume changed while being asked about a climb.
	if s.confirm.isOpen() {
		s.confirm.update(gs)
		return nil
	}

	systems.UpdateSlider(gs, s.music)
	systems.UpdateSlider(gs, s.speed)
	systems.UpdateButton(gs, s.full)
	systems.UpdateButton(gs, s.back)

	// **Re-read every tick, not only on entry.** Nothing else in the game changes it today, but a
	// window manager can drop a game out of fullscreen on its own and a latch that then disagreed
	// with the display would be the control lying about the thing it controls.
	s.full.Latched = Fullscreen()

	// **Dead with no run to give up.** Settings is reachable from the title screen, where there may
	// be no climb at all — and a control that works and does nothing is worse than one that says it
	// has nothing to do.
	setEnabled(s.abandon, gs.Run != nil)
	s.abandon.Text = abandonLabel(gs)
	systems.UpdateButton(gs, s.abandon)
	systems.UpdateButton(gs, s.exit)

	// The readouts follow the bars rather than being written once, so the number under the
	// cursor is the number the game is at.
	s.music.Readout = fmt.Sprintf("%d%%", int(s.music.Value*100+0.5))
	s.speed.Readout = fmt.Sprintf("%.2fx", speedFor(s.speed.Value))
	return nil
}

func (s *SettingsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(20)))
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, settingsTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: settingsTitleSize}, op)

	systems.DrawSlider(gs, screen, s.music)
	systems.DrawSlider(gs, screen, s.speed)
	systems.DrawButton(gs, screen, s.full)

	// The rule, then the two things that are not settings: the way out of the run and the way out
	// of the program.
	ruleY := s.abandonRuleY(gs)
	ruleLeft := gs.PctX(50) - settingsAbandonRuleWidth/2
	vector.StrokeLine(screen,
		float32(ruleLeft), float32(ruleY),
		float32(ruleLeft+settingsAbandonRuleWidth), float32(ruleY),
		1, systems.ColorToward(groundInk, screenGround, 60), false)

	systems.DrawButton(gs, screen, s.abandon)

	systems.DrawButton(gs, screen, s.exit)

	systems.DrawButton(gs, screen, s.back)

	// **The note under a dead music bar, and nothing when it is live.** A control that has gone
	// grey for a reason outside the game has to say what the reason was, or it reads as a bug.
	if s.music.Disabled {
		r := systems.SliderRect(s.music)
		note := &text.DrawOptions{}
		note.GeoM.Translate(float64(r.Min.X), float64(r.Max.Y+8))
		note.ColorScale.ScaleWithColor(systems.ColorToward(groundInk, screenGround, 40))
		text.Draw(screen, "no audio device on this machine",
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, note)
	}

	// Over the screen it is asking about.
	s.confirm.draw(gs, screen)
}

// abandonLabel is what the button says: the deed, and the run it would end. **With no run standing
// there is no code to name**, and the button is dead anyway — the settings are reachable from the
// title screen, where the next New Run has not been rolled yet.
func abandonLabel(gs *state.GlobalState) string {
	if gs.Run == nil {
		return settingsAbandonLabel
	}
	return fmt.Sprintf("%s, %s: %s", settingsAbandonLabel, settingsSeedCaption, seeds.Code(gs.RunSeed))
}

// askAbandon puts the question up. **There is no path from the button to AbandonRun that does not
// go through here** — a climb is thrown away by this and by nothing else in the game, so a misclick
// on a screen the player opened to turn the music down must not be able to end their run.
func (s *SettingsScene) askAbandon(gs *state.GlobalState) {
	if gs.Run == nil {
		return
	}
	s.confirm.ask(
		"ABANDON THIS RUN?",
		"The climb will be lost. You will see what it came to first.",
		settingsAbandonLabel,
		func() { AbandonRun(gs) },
	)
}

// leave goes back to whichever screen opened this one.
//
// **The title screen is the fallback**, for the case nothing set a return: an ActiveScreen of zero
// is Title, so this is what the zero value already means rather than a rule invented here.
func (s *SettingsScene) leave(gs *state.GlobalState) {
	back := gs.ReturnScreen
	if back == state.Settings {
		back = state.Title
	}
	gs.ActiveScreen = back
	gs.NewScreen = true
}

// commit writes both settings to the profile, once, at the end of a drag.
//
// **It writes both rather than the one that moved.** The profile is one file and a save rewrites
// all of it anyway, so a per-setting path would be two functions that can disagree about which
// fields are current.
func (s *SettingsScene) commit(gs *state.GlobalState) {
	if gs.Profile == nil {
		return
	}
	gs.Profile.Settings.MusicVolume = music.Level()
	gs.Profile.Settings.Speed = Speed()
	gs.Profile.Settings.Fullscreen = Fullscreen()
	saveProfile(gs)
}

// speedFor maps a bar position onto a game-speed multiplier, and speedValue maps one back.
//
// **The scale is linear between the two bounds and 1 is not the middle of it**: the range is
// 0.5x to 2x, so the tuned speed sits a third of the way along. A geometric scale would centre it,
// and was not taken — a bar whose left half covers a two-fold slowdown and whose right half covers
// a two-fold speed-up reads correctly but makes every position a different size of step, which is
// harder to describe than it is worth for a control with a readout on it.
func speedFor(v float64) float64 {
	return profile.SpeedMin + v*(profile.SpeedMax-profile.SpeedMin)
}

func speedValue(speed float64) float64 {
	v := (speed - profile.SpeedMin) / (profile.SpeedMax - profile.SpeedMin)
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// ApplySettings puts a loaded profile's settings into force.
//
// **This is the one place the saved numbers become the running ones**, called before the audio
// device is opened so a returning player hears the level they chose rather than a moment of
// silence or a moment of full volume.
func ApplySettings(s profile.Settings) {
	music.SetLevel(s.MusicVolume)

	// A zero speed is ignored by SetSpeed rather than applied, and profile.LoadProfile has
	// already normalised one off disk — so an older profile with no settings block at all lands
	// on the tuned speed rather than on a stopped clock.
	SetSpeed(s.Speed)

	// **Fullscreen is applied here rather than in main** for the reason the other two are: one
	// function puts a whole settings block into force, so a setting that is loaded and never
	// applied cannot exist. Ebitengine takes this before the window opens as happily as after.
	SetFullscreen(s.Fullscreen)
}

// SetFullscreen puts the game on the whole display, or back in its window.
//
// **A thin wrapper so the settings screen does not reach for Ebitengine itself**, which is the
// same shape SetSpeed and music.SetLevel have — the screen asks for a state and something else
// owns how it is reached. It also gives Fullscreen() one answer to read back.
func SetFullscreen(on bool) { ebiten.SetFullscreen(on) }

// Fullscreen is whether the game is on the whole display right now.
//
// **Asked of the window rather than of the profile**, exactly as the two bars ask the audio device
// and the clock. A screen that read the file would show a state the game is not in on a machine
// where the file could not be written.
func Fullscreen() bool { return ebiten.IsFullscreen() }
