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
// **Every setting here may only change what the player sees or hears.** The speed bar scales
// `clock.go`'s one beat, and a whole round is resolved before playback begins, so it moves
// pictures and nothing else. That is the same constraint the debug flags and `internal/trace` are
// under, and it is the reason a game-speed control is safe to offer at all.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
)

// settingsTitleSize and settingsTitle are the heading.
const (
	settingsTitleSize = 40
	settingsTitle     = "Settings"
)

// SettingsScene is the settings screen.
type SettingsScene struct {
	music *models.Slider
	speed *models.Slider
	back  *models.Button
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
		s.music = models.NewSlider(settingsSliderWidth, settingsSliderHeight, "Music", 0)
		s.music.Ink = groundInk
		s.music.OnChange = func(v float64) { music.SetLevel(v) }
		s.music.OnCommit = func(v float64) { s.commit(gs) }

		s.speed = models.NewSlider(settingsSliderWidth, settingsSliderHeight, "Game speed", 0)
		s.speed.Ink = groundInk
		s.speed.OnChange = func(v float64) { SetSpeed(speedFor(v)) }
		s.speed.OnCommit = func(v float64) { s.commit(gs) }

		s.back = models.NewButton(200, 60, "Back", func() { s.leave(gs) })
	}

	// Re-read every visit. The level can have moved since the last one — a fresh profile is
	// silent and the game may have been played for an hour since — and Init runs on every entry.
	s.music.Value = music.Level()
	s.speed.Value = speedValue(Speed())

	// **The bar is only live if there is anything to hear.** Opening the audio device is allowed
	// to fail, and a volume control that silently did nothing would be worse than one that says
	// it cannot — the same rule the chrome's mute button was under before it became a cog.
	s.music.Disabled = !music.Available()

	centre := gs.PctX(50)
	top := gs.PctY(38)

	s.music.ScreenX, s.music.ScreenY = centre, top
	s.speed.ScreenX, s.speed.ScreenY = centre, top+settingsRowGap
	s.back.ScreenX, s.back.ScreenY = centre, top+settingsRowGap*2+60
}

func (s *SettingsScene) Update(gs *state.GlobalState) error {
	systems.UpdateSlider(gs, s.music)
	systems.UpdateSlider(gs, s.speed)
	systems.UpdateButton(gs, s.back)

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
}
