// Command ringart takes the art generator's output and files it: every PNG in the inbox is
// reduced to the ring card's own size, committed under assets/ring, recorded on its record in
// data/rings.json, and struck from the worklist in docs/art/rings_to_draw.md.
//
// It exists because that is four steps done by hand, once per ring, a hundred and twenty times —
// and the two that fail silently are the ones a person gets wrong. A picture committed at the
// generator's 1060x1484 is caught by TestEveryBleedingCardArtIsTheCardsOwnSize, but an "Art"
// field left empty just draws default-ring.png, and a worklist entry left standing is a ring
// that gets drawn twice.
//
//	go run ./tools/ringart              # file everything in the inbox
//	go run ./tools/ringart -n           # say what would happen and touch nothing
//	go run ./tools/ringart -blocky      # quantize to the block grid on the way down
//
// The stem of each file is the record id, exactly as assets/ring keys are: brass-knuckles-ring.png
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
	ringsJSON = "data/rings.json"
	worklist  = "docs/art/rings_to_draw.md"
)

func main() {
	in := flag.String("in", filepath.Join(".scratch", "to-process-ring-art"), "inbox of generated PNGs")
	out := flag.String("out", filepath.Join("assets", "ring"), "where the reduced art is committed")
	done := flag.String("done", filepath.Join(".scratch", "processed-rings"), "where the originals are kept")
	blocky := flag.Bool("blocky", false, "quantize to the 40x56 block grid, then scale up by a whole number")
	dry := flag.Bool("n", false, "report what would happen and write nothing")
	flag.Parse()

	w, h := cards.RingStyle.Width, cards.RingStyle.Height

	records, err := recordIDs(ringsJSON)
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
			log.Fatalf("%s names no record in %s — a key that matches nothing draws the fallback and nothing fails", e.Name(), ringsJSON)
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
		fmt.Printf("\n%d ring(s) would be filed; %s and %s untouched\n", len(keys), ringsJSON, worklist)
		return
	}
	if err := setArt(ringsJSON, keys); err != nil {
		log.Fatal(err)
	}
	left, err := strike(worklist, keys)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%d ring(s) filed. %d still to draw.\n", len(keys), left)
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
		RingRecord string `json:"RingRecord"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	ids := make(map[string]bool, len(file))
	for _, r := range file {
		ids[r.RingRecord] = true
	}
	return ids, nil
}

// setArt rewrites one line per record rather than re-encoding the file. data/rings.json is
// hand-formatted — a rule's If clause sits on one line — and a round-trip through encoding/json
// would reflow all of it, burying a six-line change in a twelve-hundred-line diff.
func setArt(path string, keys []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(raw)
	for _, key := range keys {
		re := regexp.MustCompile(`("RingRecord": "` + regexp.QuoteMeta(key) + `",\n(?:[^\n]*\n)??[ \t]*"Art": )"[^"]*"`)
		if !re.MatchString(s) {
			return fmt.Errorf("%s: found no Art field on record %q", path, key)
		}
		s = re.ReplaceAllString(s, "${1}\""+key+"\"")
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

var (
	entryRe   = regexp.MustCompile("(?m)^### [^\n]*\n\n- \\*\\*Key:\\*\\* `([a-z0-9-]+)`\n(?:[^\n]*\n)*?\n")
	sectionRe = regexp.MustCompile(`(?m)^(## )(Common|Uncommon|Rare)( — )\d+( rings)$`)
	totalRe   = regexp.MustCompile(`for the \d+ rings with no artwork yet`)
)

// strike removes the worklist entry for each ring that now has art, then recomputes the counts in
// the heading and in every section title. A worklist whose length is wrong is a worklist nobody
// trusts the length of, which is the whole reason the file says to delete finished entries.
func strike(path string, keys []string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := string(raw)
	drawn := make(map[string]bool, len(keys))
	var missing []string
	for _, k := range keys {
		drawn[k] = true
		if !strings.Contains(s, "`"+k+"`") {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		fmt.Printf("  note: no worklist entry for %s — already struck?\n", strings.Join(missing, ", "))
	}
	s = entryRe.ReplaceAllStringFunc(s, func(m string) string {
		if drawn[entryRe.FindStringSubmatch(m)[1]] {
			return ""
		}
		return m
	})

	// Count what is left off the file itself rather than by arithmetic on what was removed — the
	// file is the truth, and a subtraction can only ever drift from it.
	left := 0
	counts := map[string]int{}
	for _, name := range []string{"Common", "Uncommon", "Rare"} {
		counts[name] = countIn(s, name)
		left += counts[name]
	}
	s = sectionRe.ReplaceAllStringFunc(s, func(m string) string {
		p := sectionRe.FindStringSubmatch(m)
		return fmt.Sprintf("%s%s%s%d%s", p[1], p[2], p[3], counts[p[2]], p[4])
	})
	s = totalRe.ReplaceAllString(s, fmt.Sprintf("for the %d rings with no artwork yet", left))
	return left, os.WriteFile(path, []byte(s), 0o644)
}

func countIn(s, section string) int {
	start := strings.Index(s, "## "+section+" — ")
	if start < 0 {
		return 0
	}
	rest := s[start:]
	if next := strings.Index(rest[3:], "\n## "); next >= 0 {
		rest = rest[:next+3]
	}
	return strings.Count(rest, "- **Key:** ")
}
