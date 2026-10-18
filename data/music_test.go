package data

import "testing"

func TestTheMusicCatalogLoads(t *testing.T) {
	cat := LoadMusic()
	if len(cat.Tracks) == 0 {
		t.Fatal("music.json names no track")
	}
}

// TestTheMusicCatalogRefusesWhatWouldShipWrong walks the refusals one record at a time, each a
// file that would otherwise build a game-ready folder nobody chose.
func TestTheMusicCatalogRefusesWhatWouldShipWrong(t *testing.T) {
	const packs = `"Packs": [{"Pack": "p", "Name": "A pack"}]`
	cases := map[string]string{
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
	for name, track := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("music.json with %s loaded", track)
				}
			}()
			ParseMusic([]byte(`{` + packs + `, "Tracks": [` + track + `]}`))
		})
	}

	t.Run("a credited pack nothing comes from", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("a pack with no track loaded")
			}
		}()
		ParseMusic([]byte(`{"Packs": [{"Pack": "p", "Name": "A"}, {"Pack": "q", "Name": "B"}], "Tracks": [` +
			`{"File": "a.ogg", "Pack": "p", "Source": "p/a.wav", "GainDB": 0, "Quality": 6}]}`))
	})
}
