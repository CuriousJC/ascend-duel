package data

import "testing"

func TestTheAudioCatalogsLoad(t *testing.T) {
	cat := LoadAudio()
	if len(cat.Music) == 0 {
		t.Fatal("music.json names no track")
	}
	for _, cue := range SoundCues {
		if len(cat.CueSounds(cue)) == 0 {
			t.Logf("no sound plays on the cue %q", cue)
		}
	}
}

// TestTheAudioCatalogsRefuseWhatWouldShipWrong walks the refusals one record at a time, each a set
// of files that would otherwise build a game-ready folder nobody chose.
func TestTheAudioCatalogsRefuseWhatWouldShipWrong(t *testing.T) {
	const packs = `[{"Pack": "p", "Name": "A pack"}]`
	const okMusic = `[{"File": "m.ogg", "Pack": "p", "Source": "p/m.wav", "GainDB": 0, "Quality": 6}]`
	music := map[string]string{
		"no gain":         `{"File": "a.ogg", "Pack": "p", "Source": "p/a.wav", "Quality": 6}`,
		"no quality":      `{"File": "a.ogg", "Pack": "p", "Source": "p/a.wav", "GainDB": 0}`,
		"quality on copy": `{"File": "a.ogg", "Pack": "p", "Source": "p/a.ogg", "GainDB": 0, "Quality": 6}`,
		"gain on midi":    `{"File": "a.mid", "Pack": "p", "Source": "p/a.mid", "GainDB": 0}`,
		"encode to wav":   `{"File": "a.wav", "Pack": "p", "Source": "p/a.wav", "GainDB": 1.5, "Quality": 6}`,
		"off the scale":   `{"File": "a.ogg", "Pack": "p", "Source": "p/a.wav", "GainDB": 0, "Quality": 11}`,
		"in a folder":     `{"File": "x/a.ogg", "Pack": "p", "Source": "p/a.wav", "GainDB": 0, "Quality": 6}`,
		"outside pack":    `{"File": "a.ogg", "Pack": "p", "Source": "q/a.wav", "GainDB": 0, "Quality": 6}`,
		"unknown pack":    `{"File": "a.ogg", "Pack": "q", "Source": "q/a.wav", "GainDB": 0, "Quality": 6}`,
		"unknown field":   `{"File": "a.ogg", "Pack": "p", "Source": "p/a.wav", "GainDB": 0, "Quality": 6, "Screen": []}`,
		"unplayable":      `{"File": "a.flac", "Pack": "p", "Source": "p/a.flac", "GainDB": 0}`,
	}
	for name, track := range music {
		refused(t, name, packs, `[`+track+`]`, `[]`)
	}

	sound := func(rec string) string { return `[` + rec + `]` }
	refused(t, "a midi effect", packs, okMusic,
		sound(`{"File": "s.mid", "Pack": "p", "Source": "p/s.mid", "Cues": []}`))
	refused(t, "a cue the game does not raise", packs, okMusic,
		sound(`{"File": "s.ogg", "Pack": "p", "Source": "p/s.wav", "GainDB": 0, "Quality": 6, "Cues": ["relic-tost"]}`))
	refused(t, "a name in both catalogs", packs, okMusic,
		sound(`{"File": "m.ogg", "Pack": "p", "Source": "p/m2.wav", "GainDB": 0, "Quality": 6, "Cues": []}`))
	refused(t, "a credited pack nothing comes from",
		`[{"Pack": "p", "Name": "A"}, {"Pack": "q", "Name": "B"}]`, okMusic, `[]`)
}

func refused(t *testing.T, name, packs, music, sounds string) {
	t.Run(name, func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Errorf("loaded: music %s, sounds %s", music, sounds)
			}
		}()
		ParseAudio([]byte(packs), []byte(music), []byte(sounds))
	})
}
