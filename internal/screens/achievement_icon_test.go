package screens

import (
	"bytes"
	"image"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/achieve"
)

// TestEveryAchievementIconIsSteamsSquare holds the icon family to the one size Steamworks takes: a
// key naming no file draws nothing on the page and fails the upload, and a picture filed at the
// generator's own size reduces to the page's 64 by a factor that is not a whole number.
func TestEveryAchievementIconIsSteamsSquare(t *testing.T) {
	images := assets.LoadImageData()
	for _, a := range achieve.Loaded().All() {
		raw := images[a.AchievedIconKey]
		if len(raw) == 0 {
			t.Errorf("%s: no embedded icon under %q", a.APIName, a.AchievedIconKey)
			continue
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			t.Errorf("%s: %v", a.APIName, err)
			continue
		}
		if cfg.Width != data.AchievementIconSize || cfg.Height != data.AchievementIconSize {
			t.Errorf("%s: icon is %dx%d, want %dx%d", a.APIName, cfg.Width, cfg.Height,
				data.AchievementIconSize, data.AchievementIconSize)
		}
	}
}
