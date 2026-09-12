// Command relicart takes the art generator's output and files it: every PNG in the inbox is
// reduced to the card's own size, committed under the catalogue's asset directory, and recorded
// on its record in the catalogue's JSON file.
//
// It exists because that is four steps done by hand, once per record, a hundred and twenty times —
// and the two that fail silently are the ones a person gets wrong. A picture committed at the
// generator's 1060x1484 is caught by TestEveryBleedingCardArtIsTheCardsOwnSize, but an "Art"
// field left empty just draws the default face and nothing fails.
//
// **There is no worklist to strike.** A record with no Art is one still to draw, and its brief is
// the Draw field on the same record — the catalogue's review sheet counts both and marks both.
//
//	go run ./tools/relicart                    # file everything in the relic inbox
//	go run ./tools/relicart -kind worm         # the worms instead
//	go run ./tools/relicart -kind parasite     # the parasites instead
//	go run ./tools/relicart -n                 # say what would happen and touch nothing
//	go run ./tools/relicart -blocky            # quantize to the block grid on the way down
//
// # Three catalogues, one command
//
// **Relics, worms and parasites all carry Art and Draw and all draw a full-bleed card**, so filing
// a picture is the identical four steps for each and a second command would be this file copied
// with three strings changed. `-kind` is the parameter and `catalogues` is the whole difference:
// an inbox, an asset directory, a JSON file, and the name of that file's record key. The name
// stays `relicart` because the relics are what it is reached for; renaming it would break the
// muscle memory and the CLAUDE.md line for nothing.
//
// The stem of each file is the record id, exactly as the asset keys are: brass-knuckles-ring.png
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

// catalogue is one family of records this tool can file art for: where the generator's output
// lands, where the reduced pictures are committed, which file records them, and what that file
// calls its key.
//
// **The key field is a string rather than a reflected struct tag**, because the tool reads the
// JSON twice for two different questions — which ids exist, and which lines to rewrite — and both
// are done with the minimum that answers them. A full round-trip through encoding/json would
// reflow a hand-formatted file; see setArt.
type catalogue struct {
	inbox string
	out   string
	json  string
	key   string
}

var catalogues = map[string]catalogue{
	"relic": {
		inbox: filepath.Join(".scratch", "to-process-relic-art"),
		out:   filepath.Join("assets", "relic"),
		json:  "data/relics.json",
		key:   "RelicRecord",
	},
	"worm": {
		inbox: filepath.Join(".scratch", "to-process-worm-art"),
		out:   filepath.Join("assets", "worm"),
		json:  "data/worms.json",
		key:   "WormRecord",
	},
	"parasite": {
		inbox: filepath.Join(".scratch", "to-process-parasite-art"),
		out:   filepath.Join("assets", "parasite"),
		json:  "data/parasites.json",
		key:   "ParasiteRecord",
	},
}

// kindList is the -kind flag's vocabulary, sorted, for the error a misspelling gets.
func kindList() string {
	names := make([]string, 0, len(catalogues))
	for k := range catalogues {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, " / ")
}

func main() {
	kind := flag.String("kind", "relic", "which catalogue to file art for: "+kindList())
	in := flag.String("in", "", "inbox of generated PNGs (default: the catalogue's own)")
	out := flag.String("out", "", "where the reduced art is committed (default: the catalogue's own)")
	done := flag.String("done", "", "where the originals are kept (default: .scratch/processed-<kind>s)")
	blocky := flag.Bool("blocky", false, "quantize to the 40x56 block grid, then scale up by a whole number")
	dry := flag.Bool("n", false, "report what would happen and write nothing")
	flag.Parse()

	cat, ok := catalogues[*kind]
	if !ok {
		log.Fatalf("-kind %q is not one of %s", *kind, kindList())
	}
	// **The three directory flags default to the catalogue's own and still override**, so the
	// everyday call is `-kind worm` and a one-off batch sitting somewhere else is still one flag
	// away. An empty string is the sentinel rather than the catalogue being copied into the flag
	// defaults, because flag defaults are read before -kind is.
	if *in == "" {
		*in = cat.inbox
	}
	if *out == "" {
		*out = cat.out
	}
	if *done == "" {
		*done = filepath.Join(".scratch", "processed-"+*kind+"s")
	}

	// **Every one of the three draws a full-bleed card at RelicStyle's size.** WormStyle is the
	// same width and height — the two differ in the text band, not in the picture — so one target
	// size is a fact about the card rather than a shortcut. TestEveryBleedingCardArtIsTheCardsOwnSize
	// is what fails if that stops being true.
	w, h := cards.RelicStyle.Width, cards.RelicStyle.Height

	records, err := recordIDs(cat)
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
			log.Fatalf("%s names no record in %s — a key that matches nothing draws the fallback and nothing fails", e.Name(), cat.json)
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
		fmt.Printf("\n%d %s(s) would be filed; %s untouched\n", len(keys), *kind, cat.json)
		return
	}
	if err := setArt(cat, keys); err != nil {
		log.Fatal(err)
	}
	left, err := undrawn(cat.json)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%d %s(s) filed. %d still to draw.\n", len(keys), *kind, left)
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

// recordIDs is every key the catalogue's file writes.
//
// **Decoded into a map rather than a struct**, because the key's field name differs per catalogue
// — RelicRecord, WormRecord, ParasiteRecord — and a struct per catalogue would be three types
// that exist to hold one string each. Everything else in the record is ignored here.
func recordIDs(cat catalogue) (map[string]bool, error) {
	raw, err := os.ReadFile(cat.json)
	if err != nil {
		return nil, err
	}
	var file []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", cat.json, err)
	}
	ids := make(map[string]bool, len(file))
	for _, r := range file {
		raw, ok := r[cat.key]
		if !ok {
			continue
		}
		var id string
		if err := json.Unmarshal(raw, &id); err != nil {
			return nil, fmt.Errorf("%s: a %s is not a string: %w", cat.json, cat.key, err)
		}
		ids[id] = true
	}
	return ids, nil
}

// setArt rewrites one line per record rather than re-encoding the file. data/relics.json is
// hand-formatted — a rule's If clause sits on one line — and a round-trip through encoding/json
// would reflow all of it, burying a six-line change in a twelve-hundred-line diff. The worm and
// parasite files are machine-formatted today and would survive a round-trip, but one path through
// this function is worth more than the difference.
func setArt(cat catalogue, keys []string) error {
	raw, err := os.ReadFile(cat.json)
	if err != nil {
		return err
	}
	s := string(raw)
	for _, key := range keys {
		// **Bounded and lazy, so it cannot walk into the next record.** The header fields between
		// RelicRecord and Art are authored and have grown once already — Family landed between Name
		// and Art on 2026-09-12, and a pattern allowing exactly one intervening line then matched no
		// record in the file. Four is headroom for the next one; an unbounded `*` would silently
		// retarget a record whose own Art was missing.
		re := regexp.MustCompile(`("` + cat.key + `": "` + regexp.QuoteMeta(key) + `",\n(?:[^\n]*\n){0,4}?[ \t]*"Art": )"[^"]*"`)
		if !re.MatchString(s) {
			return fmt.Errorf("%s: found no Art field on record %q", cat.json, key)
		}
		s = re.ReplaceAllString(s, "${1}\""+key+"\"")
	}
	return os.WriteFile(cat.json, []byte(s), 0o644)
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
