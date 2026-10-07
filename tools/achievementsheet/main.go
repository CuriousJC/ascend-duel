// Command achievementsheet writes every achievement's Steam icon pair and a page that shows each
// one beside the Steamworks fields its record carries.
//
//	go run ./tools/achievementsheet
//
// An achievement is earned once and the page that lists them shows each icon at one size, in one
// state — so "does this icon still read at 64, and does it still read in gray" cannot be answered
// in a launched game without earning half the catalog. The sheet draws all four at once: achieved
// and unachieved, at Steam's 256 upload size and at the 64 Steam and the game both show.
//
// # It is the upload set as well as a report
//
// **The two 256 files per record are what goes into the Steamworks admin page** — `achieved-<api
// name>.png` for the Achieved Icon and `unachieved-<api name>.png` for the Unachieved Icon. The gray
// one comes out of systems.ArtMarkGray, the same call the game's page makes, so the locked row in
// the game and the file on Steam are one set of pixels.
//
// # It is validated
//
// The catalog is read through internal/achieve, so a record the game would refuse — a SetBy that
// is not `client`, a trigger naming nothing — fails the sheet exactly as it fails a launch.
//
// # Output
//
// Loose PNGs plus an index.html, written into `docs/sheets/achievementsheet/` and committed on the
// terms every other sheet is.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// ground is screens.screenGround, the light slate blue the achievements page is drawn on.
const ground = "#a8bcd4"

// shownSize is the side the game's page and Steam's pop-up both draw an icon at.
const shownSize = 64

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "achievementsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	// **Every PNG is written fresh**, so one left over from a renamed or retired record would sit in
	// the directory looking current. Cleared first, the same way the relic sheet clears its own.
	stale, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	for _, f := range stale {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	catalog := achieve.Loaded()
	records := data.LoadAchievements()
	page := page{Ground: ground, Count: len(records), Size: data.AchievementIconSize, Shown: shownSize}

	for _, r := range records {
		a, ok := catalog.Find(r.APIName)
		if !ok {
			return fmt.Errorf("%s is in the file and not in the catalog", r.APIName)
		}
		key := a.AchievedIconKey
		full, small := data.AchievementIconSize, shownSize
		files := []struct {
			name string
			img  *image.RGBA
		}{
			{"achieved-" + r.APIName + ".png", systems.ArtMark(key, full, full)},
			{"unachieved-" + r.APIName + ".png", systems.ArtMarkGray(key, full, full)},
			{"achieved-" + r.APIName + "-64.png", systems.ArtMark(key, small, small)},
			{"unachieved-" + r.APIName + "-64.png", systems.ArtMarkGray(key, small, small)},
		}
		for _, f := range files {
			if f.img == nil {
				return fmt.Errorf("%s: no embedded icon under %q", r.APIName, key)
			}
			if err := writePNG(filepath.Join(dir, f.name), f.img); err != nil {
				return err
			}
		}

		trigger, err := json.Marshal(r.Trigger)
		if err != nil {
			return err
		}
		p := plate{
			APIName:      r.APIName,
			DisplayName:  r.DisplayName,
			Description:  r.Description,
			SetBy:        r.SetBy,
			Hidden:       r.Hidden,
			Achieved:     files[0].name,
			Unachieved:   files[1].name,
			Achieved64:   files[2].name,
			Unachieved64: files[3].name,
			Icon:         r.AchievedIcon,
			Draw:         r.Draw,
			Said:         r.Said,
			Unlocks:      strings.Join(r.Unlocks, ", "),
			Trigger:      string(trigger),
		}
		if p.Icon == "" {
			page.Undrawn++
		}
		if p.Draw == "" {
			page.Unwritten++
		}
		if p.Hidden {
			page.Hidden++
		}
		page.Plates = append(page.Plates, p)
	}

	out := filepath.Join(dir, "index.html")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Printf("wrote %s and %d PNGs — %d achievements, %d with an icon of their own and %d with a subject\n",
		out, 4*len(page.Plates), page.Count, page.Count-page.Undrawn, page.Count-page.Unwritten)
	return nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return nil
}

type plate struct {
	APIName      string
	DisplayName  string
	Description  string
	SetBy        string
	Hidden       bool
	Achieved     string
	Unachieved   string
	Achieved64   string
	Unachieved64 string
	Icon         string
	Draw         string
	Said         []string
	Unlocks      string
	Trigger      string
}

type page struct {
	Ground    string
	Count     int
	Undrawn   int
	Unwritten int
	Hidden    int
	Size      int
	Shown     int
	Plates    []plate
}
