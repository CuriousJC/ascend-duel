package screens

// The settings screen: how loud the score is, how fast the game moves, and the way back.
//
// **It is the program's screen, not a run's.** Every other scene in this package is a station of
// a journey — a duel, a reward, a shop — and is reached by `advance` walking the run forward. This
// one is reached from the cog in the game's chrome, from any screen, and it puts the player back
// exactly where they were: see `state.GlobalState.Returns`. The run's phase is never touched, which is
// what makes opening it mid-journey a look at a dialog rather than a decision.
//
// **Three bars: music, sounds and speed.** Music and sounds are two levels rather than one because
// a player can want either down while the other is up *(owner's call)*.
//
// **Every *setting* here may only change what the player sees or hears.** The speed bar scales
// `clock.go`'s one beat, and a whole round is resolved before playback begins, so it moves
// pictures and nothing else. That is the same constraint the debug flags and `internal/trace` are
// under, and it is the reason a game-speed control is safe to offer at all.
//
// **Abandon Run is the one thing on this screen that is not a setting, and it is deliberate**
// *(owner's call, 2026-09-03)*. The game had no way to give up a journey and start over — quitting
// meant the window's X, and the next launch resumed exactly where it left off — and this is the
// one screen reachable from everywhere, which is what a "give up" control has to be.
//
// The objection was raised and overruled: it does file a run decision under the program's screen,
// which is the distinction the paragraph above draws. What that costs is paid in the layout rather
// than pretended away — the button stands below a rule, in the destructive red, with its own
// confirm in front of it, so it reads as a different kind of thing from the two bars. It is also
// the only thing on this screen that touches `gs.Run`, and it does so by calling `AbandonRun`
// rather than by knowing anything about a journey. See run.go.

import (
	"fmt"
	"image"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/actions"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
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

	// settingsRowGap is the space between one bar and the next, measured from center to center.
	settingsRowGap = 90

	// settingsStripGap is the air above and below the speed strip. It is the one row on this screen
	// that is not a control, so it needs to read as belonging to the bar above it without looking
	// like a third thing to click.
	settingsStripGap = 30

	// settingsTopPct is where the column starts. **The column is full**: three bars, the speed strip,
	// the toggle and the abandon band fill it to just above the bottom row, and
	// TestTheSettingsColumnClearsTheBottomRow fails when a change lands EXIT on BACK. The column is
	// measured sequentially in settingsRows so one insertion cannot silently land a button on another.
	settingsTopPct = 20

	// settingsTitlePct is where the heading sits, above the column.
	settingsTitlePct = 15

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
	settingsAbandonGap = 50

	// settingsAbandonRuleWidth matches the bars, so the rule reads as the end of that column
	// rather than as a decoration of its own.
	settingsAbandonRuleWidth = settingsSliderWidth

	settingsAbandonLabel = "ABANDON RUN"

	// settingsExitLabel leaves the program, under the button that leaves the run. settingsExitGap
	// clears the note belonging to the button above it.
	settingsExitLabel = "EXIT"
	settingsExitGap   = 92
)

// The dev column, on the screen's right: a heading and one latched button per tool. Small buttons,
// because these are instrumentation rather than settings.
const (
	settingsDevPct   = 82
	settingsDevWidth = 360
	settingsDevGap   = 64
	settingsDevTitle = "DEV TOOLS"
)

// The bottom row: the Compendium beside Back. **Beside rather than above**, because the column is
// full down to Back and a row inserted into it would land EXIT on BACK. Both are ways off this
// screen, so they share the row the way off a screen is on.
const (
	settingsCompendiumLabel = "COMPENDIUM"
	settingsBottomWidth     = 320
	settingsBottomGap       = 40
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
	music  *models.Slider
	sounds *models.Slider
	speed  *models.Slider
	back   *models.Button

	// compendium opens the Compendium over this screen; its Back comes back here.
	compendium *models.Button

	// abandon gives up the run, and confirm is the question in front of it. **Never one without
	// the other**: this is the only irreversible control in the game reachable from every screen.
	// full is the fullscreen toggle: **a latched button rather than a bar**, because unlike the
	// two settings above it this is genuinely a pair of states rather than a level. `Latched` is
	// already how the game draws a mode that is on — sunken and darker — so this needed no new
	// widget.
	full *models.Button

	abandon *models.Button
	confirm ui.ConfirmDialog

	// exit closes the game. Live whether or not a run is standing, unlike abandon.
	exit *models.Button

	// strip is the hand of relic cards under the speed bar: what the game-speed setting looks
	// like, rather than what it reads as. See settings_speed.go.
	strip speedStrip

	// dev is the dev column, one latched button per row of devRows. **Built always, drawn and live
	// only while the profile says devTools** — see devtools.go.
	dev []*models.Button
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
		s.music.Ink = ui.GroundInk
		s.music.OnChange = func(v float64) { music.SetLevel(v) }
		s.music.OnCommit = func(v float64) { s.commit(gs) }

		s.sounds = models.NewSlider(settingsSliderWidth, settingsSliderHeight, "SOUNDS", 0)
		s.sounds.Ink = ui.GroundInk
		s.sounds.OnChange = func(v float64) { music.SetSoundLevel(v) }

		// **A toast on release, so the bar says what it set.** An effect only plays when a relic fires,
		// and a level chosen in silence is a level chosen blind.
		s.sounds.OnCommit = func(v float64) { music.PlayCue(data.CueRelicToast); s.commit(gs) }

		s.speed = models.NewSlider(settingsSliderWidth, settingsSliderHeight, "GAME SPEED", 0)
		s.speed.Ink = ui.GroundInk
		s.speed.OnChange = func(v float64) { ui.SetSpeed(speedFor(v)) }
		s.speed.OnCommit = func(v float64) { s.commit(gs) }

		s.full = models.NewButton(settingsSliderWidth, settingsToggleHeight,
			settingsFullscreenLabel, func() { s.toggleFullscreen(gs) })
		s.full.TextSize = 40

		s.back = models.NewButton(settingsBottomWidth, ui.ButtonLarge, "BACK", func() { s.leave(gs) })

		s.compendium = models.NewButton(settingsBottomWidth, ui.ButtonLarge, settingsCompendiumLabel,
			func() { actions.OpenCompendium(gs) })
		s.compendium.TextSize = 40

		s.abandon = models.NewButton(760, ui.ButtonLarge, settingsAbandonLabel, func() { s.askAbandon(gs) })

		// **The modal X's red, which is the only red in the game.** It is already the color of
		// the one control that gets you out of somewhere, and this is the largest version of that
		// there is. Nothing else on this screen is anything but slate.
		s.abandon.BaseColor = ui.ModalCloseColor
		s.abandon.TextSize = 40

		s.exit = models.NewButton(760, ui.ButtonLarge, settingsExitLabel, func() { askExit(gs, &s.confirm) })
		s.exit.TextSize = 40

		for _, row := range devRows {
			row := row
			b := models.NewButton(settingsDevWidth, ui.ButtonSmall, row.label, func() { s.flipDev(gs, row) })
			s.dev = append(s.dev, b)
		}
	}

	// **The question does not survive a visit**, for the reason the title screen's does not: Init
	// runs again on every entry and arriving with a dialog up is a dialog nobody asked for.
	s.confirm.Close()

	// Re-read every visit. The level can have moved since the last one — a fresh profile is
	// silent and the game may have been played for an hour since — and Init runs on every entry.
	s.music.Value = music.Level()
	s.sounds.Value = music.SoundLevel()
	s.speed.Value = speedValue(ui.Speed())
	s.full.Latched = Fullscreen()

	// **The bar is only live if there is anything to hear.** Opening the audio device is allowed
	// to fail, and a volume control that silently did nothing would be worse than one that says
	// it cannot — the same rule the chrome's mute button was under before it became a cog.
	s.music.Disabled = !music.Available()
	s.sounds.Disabled = s.music.Disabled

	center := gs.PctX(50)
	rows := settingsRows(gs)

	s.music.ScreenX, s.music.ScreenY = center, rows.music
	s.sounds.ScreenX, s.sounds.ScreenY = center, rows.sounds
	s.speed.ScreenX, s.speed.ScreenY = center, rows.speed
	s.full.ScreenX, s.full.ScreenY = center, rows.full

	// The abandon band, below the rule; Back stays last, at the bottom of the screen, because the
	// way out of a screen is the last thing on it.
	s.abandon.ScreenX, s.abandon.ScreenY = center, rows.abandon
	s.exit.ScreenX, s.exit.ScreenY = center, rows.exit
	half := (settingsBottomWidth + settingsBottomGap) / 2
	s.compendium.ScreenX, s.compendium.ScreenY = center-half, rows.bottom
	s.back.ScreenX, s.back.ScreenY = center+half, rows.bottom

	// **The dev column stands beside the settings rather than in them**: the column is full, and a
	// tool a player never sees has no business being read as one of their settings.
	for i, b := range s.dev {
		b.ScreenX, b.ScreenY = gs.PctX(settingsDevPct), rows.music+i*settingsDevGap
	}
	s.latchDev(gs)
}

// latchDev sets each dev toggle's latch from the profile.
func (s *SettingsScene) latchDev(gs *state.GlobalState) {
	if gs.Profile == nil {
		return
	}
	for i, row := range devRows {
		s.dev[i].Latched = *row.on(&gs.Profile.Settings.Dev)
	}
}

// flipDev turns one dev tool over, puts the set in force and saves it.
func (s *SettingsScene) flipDev(gs *state.GlobalState, row devRow) {
	if gs.Profile == nil {
		return
	}
	on := row.on(&gs.Profile.Settings.Dev)
	*on = !*on
	ApplyDevTools(gs)
	s.latchDev(gs)
	saveProfile(gs)
}

// settingsLayout is where every row on this screen sits, in screen pixels.
type settingsLayout struct {
	music, sounds, speed int
	strip                image.Rectangle
	full                 int
	rule                 int

	// abandon and exit are the two buttons under the rule, and bottom is the row Compendium and
	// Back share. All three are centers, as a button stores them.
	abandon, exit, bottom int
}

// settingsRows measures the column **sequentially, each row off the one above it**.
//
// **It exists because the strip was inserted into the middle of it** *(2026-09-15)*. Every position
// used to be `top + n*settingsRowGap` with the rule and the buttons carrying their own arithmetic
// off the same `top`, which works exactly until something of a different height goes in between
// them: then five expressions have to be updated together and the one that is missed puts a button
// under another button, on a screen with no test that can see it. Measured in order, inserting a
// row is one addition here and nothing else moves by accident.
func settingsRows(gs *state.GlobalState) settingsLayout {
	var l settingsLayout

	l.music = gs.PctY(settingsTopPct)
	l.sounds = l.music + settingsRowGap
	l.speed = l.sounds + settingsRowGap

	stripTop := l.speed + settingsSliderHeight/2 + settingsStripGap
	half := settingsSliderWidth / 2
	l.strip = image.Rect(
		gs.PctX(50)-half, stripTop,
		gs.PctX(50)+half, stripTop+speedStripHeight,
	)

	l.full = l.strip.Max.Y + settingsStripGap + settingsToggleHeight/2
	l.rule = l.full + settingsToggleHeight/2 + settingsAbandonGap
	l.abandon = l.rule + 56
	l.exit = l.abandon + settingsExitGap
	l.bottom = gs.PctY(88)
	return l
}

// abandonRuleY is where the rule between the settings and the abandon band is drawn. One function
// so the rule and the button under it cannot drift apart.
func (s *SettingsScene) abandonRuleY(gs *state.GlobalState) int {
	return settingsRows(gs).rule
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
	// reaching a bar through the dialog would be a volume changed while being asked about a journey.
	if s.confirm.IsOpen() {
		s.confirm.Update(gs)
		return nil
	}

	// **The strip runs whether or not anything is being dragged.** It is a loop rather than a
	// response to a change: a player who has not touched the bar still gets to see what the speed
	// they are already on looks like, which is what makes the bar comparable at all.
	s.strip.update(gs)

	systems.UpdateSlider(gs, s.music)
	systems.UpdateSlider(gs, s.sounds)
	systems.UpdateSlider(gs, s.speed)
	systems.UpdateButton(gs, s.full)
	systems.UpdateButton(gs, s.compendium)
	systems.UpdateButton(gs, s.back)
	if devToolsOn(gs) {
		for _, b := range s.dev {
			systems.UpdateButton(gs, b)
		}
	}

	// **Re-read every tick, not only on entry.** Nothing else in the game changes it today, but a
	// window manager can drop a game out of fullscreen on its own and a latch that then disagreed
	// with the display would be the control lying about the thing it controls.
	s.full.Latched = Fullscreen()

	// **Dead with no run to give up.** Settings is reachable from the title screen, where there may
	// be no journey at all — and a control that works and does nothing is worse than one that says it
	// has nothing to do.
	ui.SetEnabled(s.abandon, gs.Run != nil)
	s.abandon.Text = abandonLabel(gs)
	systems.UpdateButton(gs, s.abandon)
	systems.UpdateButton(gs, s.exit)

	// The readouts follow the bars rather than being written once, so the number under the
	// cursor is the number the game is at.
	s.music.Readout = fmt.Sprintf("%d%%", int(s.music.Value*100+0.5))
	s.sounds.Readout = fmt.Sprintf("%d%%", int(s.sounds.Value*100+0.5))
	s.speed.Readout = fmt.Sprintf("%.2fx", speedFor(s.speed.Value))
	return nil
}

func (s *SettingsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillScreenBackdrop(gs, screen)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(gs.PctX(50)), float64(gs.PctY(settingsTitlePct)))
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ui.GroundInk)
	systems.DrawText(screen, settingsTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: settingsTitleSize}, op)

	systems.DrawSlider(gs, screen, s.music)
	systems.DrawSlider(gs, screen, s.sounds)
	systems.DrawSlider(gs, screen, s.speed)
	s.strip.draw(gs, screen, settingsRows(gs).strip)
	systems.DrawButton(gs, screen, s.full)

	// The rule, then the two things that are not settings: the way out of the run and the way out
	// of the program.
	ruleY := s.abandonRuleY(gs)
	ruleLeft := gs.PctX(50) - settingsAbandonRuleWidth/2
	vector.StrokeLine(screen,
		float32(ruleLeft), float32(ruleY),
		float32(ruleLeft+settingsAbandonRuleWidth), float32(ruleY),
		1, systems.ColorToward(ui.GroundInk, ui.ScreenGround, 60), false)

	systems.DrawButton(gs, screen, s.abandon)

	systems.DrawButton(gs, screen, s.exit)

	systems.DrawButton(gs, screen, s.compendium)
	systems.DrawButton(gs, screen, s.back)
	if devToolsOn(gs) {
		s.drawDev(gs, screen)
	}

	// **The note under a dead music bar, and nothing when it is live.** A control that has gone
	// gray for a reason outside the game has to say what the reason was, or it reads as a bug —
	// and there are two reasons: a device that refused to open, and a build carrying no audio,
	// which is every clean clone. See music.DeviceFailed.
	if s.music.Disabled {
		r := systems.SliderRect(s.music)
		note := &text.DrawOptions{}
		note.GeoM.Translate(float64(r.Min.X), float64(r.Max.Y+8))
		note.ColorScale.ScaleWithColor(systems.ColorToward(ui.GroundInk, ui.ScreenGround, 40))
		why := "no music or sounds in this build"
		if music.DeviceFailed() {
			why = "no audio device on this machine"
		}
		systems.DrawText(screen, why,
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextSmall}, note)
	}

	// Over the screen it is asking about.
	s.confirm.Draw(gs, screen)
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
// go through here** — a journey is thrown away by this and by nothing else in the game, so a misclick
// on a screen the player opened to turn the music down must not be able to end their run.
func (s *SettingsScene) askAbandon(gs *state.GlobalState) {
	if gs.Run == nil {
		return
	}
	s.confirm.Ask(
		"ABANDON THIS RUN?",
		"The journey will be lost. You will see what it came to first.",
		settingsAbandonLabel,
		func() { AbandonRun(gs) },
	)
}

// leave goes back to whichever screen opened this one.
//
// **The title screen is the fallback**, for the case nothing set a return: an ActiveScreen of zero
// is Title, so this is what the zero value already means rather than a rule invented here.
func (s *SettingsScene) leave(gs *state.GlobalState) {
	gs.PopReturn()
}

// commit writes every setting to the profile, once, at the end of a drag.
//
// **It writes all of them rather than the one that moved.** The profile is one file and a save rewrites
// all of it anyway, so a per-setting path would be two functions that can disagree about which
// fields are current.
func (s *SettingsScene) commit(gs *state.GlobalState) {
	if gs.Profile == nil {
		return
	}
	gs.Profile.Settings.MusicVolume = music.Level()
	gs.Profile.Settings.SoundVolume = music.SoundLevel()
	gs.Profile.Settings.Speed = ui.Speed()
	gs.Profile.Settings.Fullscreen = Fullscreen()
	saveProfile(gs)
}

// speedFor maps a bar position onto a game-speed multiplier, and speedValue maps one back.
//
// **The scale is linear between the two bounds and 1 is not the middle of it**: the range is
// 0.5x to 2x, so the tuned speed sits a third of the way along. A geometric scale would center it,
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
	music.SetSoundLevel(s.SoundVolume)

	// A zero speed is ignored by SetSpeed rather than applied, and profile.LoadProfile has
	// already normalized one off disk — so an older profile with no settings block at all lands
	// on the tuned speed rather than on a stopped clock.
	ui.SetSpeed(s.Speed)

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

// drawDev draws the dev column: its heading, then the toggles.
func (s *SettingsScene) drawDev(gs *state.GlobalState, screen *ebiten.Image) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(gs.PctX(settingsDevPct)), float64(settingsRows(gs).music-settingsDevGap))
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ui.GroundInk)
	systems.DrawText(screen, settingsDevTitle,
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: systems.TextMedium}, op)
	for _, b := range s.dev {
		systems.DrawButton(gs, screen, b)
	}
}
