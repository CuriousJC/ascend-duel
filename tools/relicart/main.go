// Command relicart takes the art generator's output and files it: every PNG in the inbox is
// reduced to the relic card's own size, committed under assets/relic, and recorded on its
// record in data/relics.json.
//
// It exists because that is four steps done by hand, once per relic, a hundred and twenty times —
// and the two that fail silently are the ones a person gets wrong. A picture committed at the
// generator's 1060x1484 is caught by TestEveryBleedingCardArtIsTheCardsOwnSize, but an "Art"
// field left empty just draws default-relic.png and nothing fails.
//
// **There is no worklist to strike.** A relic with no Art is one still to draw, and its brief is
// the Draw field on the same record — tools/relicsheet counts both and marks both.
//
//	go run ./tools/relicart              # file everything in the inbox
//	go run ./tools/relicart -n           # say what would happen and touch nothing
//	go run ./tools/relicart -blocky      # quantize to the block grid on the way down
//
// The stem of each file is the record id, exactly as assets/relic keys are: brass-knuckles-ring.png
// is the record "brass-knuckles-ring". A stem naming no record is refused rather than filed,
// because a misspelled key is invisible in play — the card simply draws the fallback.
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
	"regexp"
	"sort"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/curiousjc/ascend-duel/internal/cards"
)

const (
	relicsJSON = "data/relics.json"
)

func main() {
	in := flag.String("in", filepath.Join(".scratch", "to-process-relic-art"), "inbox of generated PNGs")
	out := flag.String("out", filepath.Join("assets", "relic"), "where the reduced art is committed")
	done := flag.String("done", filepath.Join(".scratch", "processed-relics"), "where the originals are kept")
	blocky := flag.Bool("blocky", false, "quantize to the 40x56 block grid, then scale up by a whole number")
	dry := flag.Bool("n", false, "report what would happen and write nothing")
	flag.Parse()

	w, h := cards.RelicStyle.Width, cards.RelicStyle.Height

	records, err := recordIDs(relicsJSON)
	if err != nil {
		log.Fatal(err)
	}

	entries, err := os.ReadDir(*in)
	if err != nil {
		log.Fatalf("inbox: %v", err)
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".png") {
			continue
		}
		key := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if !records[key] {
			log.Fatalf("%s names no record in %s — a key that matches nothing draws the fallback and nothing fails", e.Name(), relicsJSON)
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		fmt.Printf("nothing in %s\n", *in)
		return
	}
	sort.Strings(keys)

	for _, key := range keys {
		src := filepath.Join(*in, key+".png")
		img, size, err := reduce(src, w, h, *blocky)
		if err != nil {
			log.Fatalf("%s: %v", key, err)
		}
		fmt.Printf("%-40s %s -> %dx%d\n", key, size, w, h)
		if *dry {
			continue
		}
		if err := writePNG(filepath.Join(*out, key+".png"), img); err != nil {
			log.Fatal(err)
		}
		if err := os.MkdirAll(*done, 0o755); err != nil {
			log.Fatal(err)
		}
		if err := os.Rename(src, filepath.Join(*done, key+".png")); err != nil {
			log.Fatal(err)
		}
	}

	if *dry {
		fmt.Printf("\n%d relic(s) would be filed; %s untouched\n", len(keys), relicsJSON)
		return
	}
	if err := setArt(relicsJSON, keys); err != nil {
		log.Fatal(err)
	}
	left, err := undrawn(relicsJSON)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%d relic(s) filed. %d still to draw.\n", len(keys), left)
}

// reduce reads one generated PNG and scales it to the card's own size. The smooth path is
// CatmullRom, which is what the committed catalogue was made with. The blocky path quantizes to
// the block grid the prompt asks for and scales back up by a whole number, so every block lands
// on an exact square instead of being resampled across one.
func reduce(path string, w, h int, blocky bool) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil, "", err
	}
	b := src.Bounds()
	size := fmt.Sprintf("%dx%d", b.Dx(), b.Dy())

	// The art is fitted into the card by drawArt, never cropped, so an aspect ratio that is not
	// the card's letterboxes. Worth saying out loud rather than finding on the sheet.
	if got, want := float64(b.Dx())/float64(b.Dy()), float64(w)/float64(h); got < want*0.99 || got > want*1.01 {
		fmt.Printf("  warning: %s is %s, which is not the card's %d:%d — it will letterbox\n",
			filepath.Base(path), size, w, h)
	}

	if !blocky {
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
		return dst, size, nil
	}

	// The prompt asks for about 40 blocks across, and 200/40 is exactly 5 — so a block is a whole
	// number of pixels on the committed card and the hard edges survive the reduction.
	const blocksAcross = 40
	if w%blocksAcross != 0 {
		return nil, "", fmt.Errorf("a %dpx card does not divide into %d blocks", w, blocksAcross)
	}
	scale := w / blocksAcross
	small := image.NewRGBA(image.Rect(0, 0, w/scale, h/scale))
	xdraw.CatmullRom.Scale(small, small.Bounds(), src, b, xdraw.Src, nil)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.NearestNeighbor.Scale(dst, dst.Bounds(), small, small.Bounds(), xdraw.Src, nil)
	return dst, size, nil
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func recordIDs(path string) (map[string]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file []struct {
		RelicRecord string `json:"RelicRecord"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	ids := make(map[string]bool, len(file))
	for _, r := range file {
		ids[r.RelicRecord] = true
	}
	return ids, nil
}

// setArt rewrites one line per record rather than re-encoding the file. data/relics.json is
// hand-formatted — a rule's If clause sits on one line — and a round-trip through encoding/json
// would reflow all of it, burying a six-line change in a twelve-hundred-line diff.
func setArt(path string, keys []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(raw)
	for _, key := range keys {
		re := regexp.MustCompile(`("RelicRecord": "` + regexp.QuoteMeta(key) + `",\n(?:[^\n]*\n)??[ \t]*"Art": )"[^"]*"`)
		if !re.MatchString(s) {
			return fmt.Errorf("%s: found no Art field on record %q", path, key)
		}
		s = re.ReplaceAllString(s, "${1}\""+key+"\"")
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

// undrawn counts the records still carrying no art, read back off the file rather than
// subtracted from what was just filed — the file is the truth, and an arithmetic count can
// only ever drift from it. It is the whole of what the worklist used to be.
func undrawn(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var file []struct {
		Art string `json:"Art"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return 0, fmt.Errorf("%s: %w", path, err)
	}
	left := 0
	for _, r := range file {
		if r.Art == "" {
			left++
		}
	}
	return left, nil
}
