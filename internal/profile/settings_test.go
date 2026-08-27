package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAFreshProfileIsSilentAndAtTheTunedSpeed(t *testing.T) {
	// **A new player boots to silence**, which predates there being anywhere to turn it up from:
	// music that begins on its own is the first thing anyone reaches for a control to stop.
	p, _, _ := LoadProfile(At(t.TempDir()))
	if p.Settings.MusicVolume != 0 {
		t.Errorf("a new player's music is at %v, want silence", p.Settings.MusicVolume)
	}
	if p.Settings.Speed != 1 {
		t.Errorf("a new player's speed is %v, want the tuned 1", p.Settings.Speed)
	}
}

func TestSettingsSurviveARoundTrip(t *testing.T) {
	s := At(t.TempDir())

	p, _, _ := LoadProfile(s)
	p.Settings.MusicVolume = 0.6
	p.Settings.Speed = 1.5
	if err := SaveProfile(s, p); err != nil {
		t.Fatalf("saving: %v", err)
	}

	back, writable, err := LoadProfile(s)
	if err != nil || !writable {
		t.Fatalf("reading back what this build wrote: %v, writable=%v", err, writable)
	}
	if back.Settings.MusicVolume != 0.6 || back.Settings.Speed != 1.5 {
		t.Errorf("read back %+v, want the pair that was written", back.Settings)
	}
}

func TestAProfileWithNoSettingsBlockGetsTheDefaults(t *testing.T) {
	// **This is the path every profile that already exists takes.** An older file has no settings
	// object at all, so both fields read as zero — and a speed of zero would stop every clock in
	// the game rather than merely being slow. Nothing about a save file may fail a launch, so it
	// is normalised rather than rejected.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, profileFile),
		[]byte(`{"version":1,"tutorialSeen":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	p, _, err := LoadProfile(At(dir))
	if err != nil {
		t.Fatalf("an older profile is not an error: %v", err)
	}
	if !p.TutorialSeen {
		t.Error("the fields the older file did have were lost")
	}
	if p.Settings.Speed != Defaults().Speed {
		t.Errorf("a profile with no settings runs at %v, want the tuned %v",
			p.Settings.Speed, Defaults().Speed)
	}
}

func TestAnOutOfRangeSettingIsClamped(t *testing.T) {
	// A hand-edited file is the case, and the bounds belong here rather than in whichever scene
	// draws the bar: a slider clamped only at the widget would let a file past it.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, profileFile),
		[]byte(`{"version":1,"settings":{"musicVolume":9,"speed":99}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	p, _, _ := LoadProfile(At(dir))
	if p.Settings.MusicVolume != 1 {
		t.Errorf("a music level of 9 loaded as %v, want it clamped to 1", p.Settings.MusicVolume)
	}
	if p.Settings.Speed != SpeedMax {
		t.Errorf("a speed of 99 loaded as %v, want it clamped to %v", p.Settings.Speed, SpeedMax)
	}
}
