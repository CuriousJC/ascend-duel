// Command relicart takes the art generator's output and files it: every PNG in the inbox is
// reduced to the card's own size, committed under the catalog's asset directory, and recorded
// on its record in the catalog's JSON file.
//
// It exists because that is four steps done by hand, once per record, a hundred and twenty times —
// and the two that fail silently are the ones a person gets wrong. A picture committed at the
// generator's 1060x1484 is caught by TestEveryBleedingCardArtIsTheCardsOwnSize, but an "Art"
// field left empty just draws the default face and nothing fails.
//
// **There is no worklist to strike.** A record with no Art is one still to draw, and its brief is
// the Draw field on the same record — the catalog's review sheet counts both and marks both.
//
//	go run ./tools/relicart                    # file everything in the relic inbox
//	go run ./tools/relicart -kind essence         # the essences instead
//	go run ./tools/relicart -kind rune     # the runes instead
//	go run ./tools/relicart -kind stone    # the stones instead
//	go run ./tools/relicart -kind card     # the playing cards instead
//	go run ./tools/relicart -n                 # say what would happen and touch nothing
//	go run ./tools/relicart -blocky            # quantize to the block grid on the way down
//
// # Four catalogs, one command
//
// **Relics, essences, runes and stones all carry Art and Draw and all draw a full-bleed card**, so
// filing a picture is the identical four steps for each and a second command would be this file copied
// with three strings changed. `-kind` is the parameter and `catalogs` is the whole difference:
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

// catalog is one family of records this tool can file art for: where the generator's output
// lands, where the reduced pictures are committed, which file records them, and what that file
// calls its key.
//
// **The key field is a string rather than a reflected struct tag**, because the tool reads the
// JSON twice for two different questions — which ids exist, and which lines to rewrite — and both
// are done with the minimum that answers them. A full round-trip through encoding/json would
// reflow a hand-formatted file; see setArt.
type catalog struct {
	inbox   string
	out     string
	sources []source
}

// source is one JSON file a catalog files art into, and what that file calls its key.
//
// **It is a list because one inbox can span two catalogs**: the potions and the sealed goods share
// docs/art/other_card_art_prompt.MD, so they are generated in one batch and filing them is one
// command over two files. Every other kind has exactly one source, and a single-source catalog
// behaves exactly as it did.
type source struct {
	json string
	key  string
}

var catalogs = map[string]catalog{
	"relic": {
		inbox:   filepath.Join(".scratch", "to-process-relic-art"),
		out:     filepath.Join("assets", "relic"),
		sources: []source{{json: "data/relics.json", key: "RelicRecord"}},
	},
	"essence": {
		inbox:   filepath.Join(".scratch", "to-process-essence-art"),
		out:     filepath.Join("assets", "essence"),
		sources: []source{{json: "data/essences.json", key: "EssenceRecord"}},
	},
	"rune": {
		inbox:   filepath.Join(".scratch", "to-process-rune-art"),
		out:     filepath.Join("assets", "rune"),
		sources: []source{{json: "data/runes.json", key: "RuneRecord"}},
	},
	"stone": {
		inbox:   filepath.Join(".scratch", "to-process-stone-art"),
		out:     filepath.Join("assets", "stone"),
		sources: []source{{json: "data/stones.json", key: "StoneRecord"}},
	},
	"card": {
		inbox:   filepath.Join(".scratch", "to-process-playing-card-art"),
		out:     filepath.Join("assets", "card"),
		sources: []source{{json: "data/card_art.json", key: "CardArtRecord"}},
	},
	"other": {
		inbox: filepath.Join(".scratch", "to-process-other-art"),
		out:   filepath.Join("assets", "other"),
		sources: []source{
			{json: "data/potions.json", key: "PotionRecord"},
			{json: "data/goods.json", key: "GoodRecord"},
		},
	},
}

// kindList is the -kind flag's vocabulary, sorted, for the error a misspelling gets.
func kindList() string {
	names := make([]string, 0, len(catalogs))
	for k := range catalogs {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, " / ")
}

func main() {
	kind := flag.String("kind", "relic", "which catalog to file art for: "+kindList())
	in := flag.String("in", "", "inbox of generated PNGs (default: the catalog's own)")
	out := flag.String("out", "", "where the reduced art is committed (default: the catalog's own)")
	done := flag.String("done", "", "where the originals are kept (default: .scratch/processed-<kind>s)")
	blocky := flag.Bool("blocky", false, "quantize to the 40x56 block grid, then scale up by a whole number")
	dry := flag.Bool("n", false, "report what would happen and write nothing")
	flag.Parse()

	cat, ok := catalogs[*kind]
	if !ok {
		log.Fatalf("-kind %q is not one of %s", *kind, kindList())
	}
	// **The three directory flags default to the catalog's own and still override**, so the
	// everyday call is `-kind essence` and a one-off batch sitting somewhere else is still one flag
	// away. An empty string is the sentinel rather than the catalog being copied into the flag
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

	// **Every one of the three draws a full-bleed card at RelicStyle's size.** EssenceStyle is the
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
		if _, ok := records[key]; !ok {
			log.Fatalf("%s names no record in %s — a key that matches nothing draws the fallback and nothing fails", e.Name(), cat.files())
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
		fmt.Printf("\n%d %s(s) would be filed; %s untouched\n", len(keys), *kind, cat.files())
		return
	}
	if err := setArt(records, keys); err != nil {
		log.Fatal(err)
	}
	left, err := undrawn(cat)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%d %s(s) filed. %d still to draw.\n", len(keys), *kind, left)
}

// reduce reads one generated PNG and scales it to the card's own size. The smooth path is
// CatmullRom, which is what the committed catalog was made with. The blocky path quantizes to
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

// recordIDs is every key the catalog's files write, against the file that wrote it — so a picture
// is filed on the right record without the caller knowing which of a catalog's files that is.
//
// **Decoded into a map rather than a struct**, because the key's field name differs per catalog
// — RelicRecord, EssenceRecord, RuneRecord — and a struct per catalog would be three types
// that exist to hold one string each. Everything else in the record is ignored here.
//
// **One id may not appear in two of a catalog's files.** Both keys index one flat asset map, so a
// collision is two records drawing one picture — which is exactly the silent failure this tool
// exists to refuse.
func recordIDs(cat catalog) (map[string]source, error) {
	ids := make(map[string]source)
	for _, src := range cat.sources {
		raw, err := os.ReadFile(src.json)
		if err != nil {
			return nil, err
		}
		var file []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, fmt.Errorf("%s: %w", src.json, err)
		}
		for _, r := range file {
			raw, ok := r[src.key]
			if !ok {
				continue
			}
			var id string
			if err := json.Unmarshal(raw, &id); err != nil {
				return nil, fmt.Errorf("%s: a %s is not a string: %w", src.json, src.key, err)
			}
			if was, dup := ids[id]; dup {
				return nil, fmt.Errorf("%q is a record in both %s and %s — one id, one picture", id, was.json, src.json)
			}
			ids[id] = src
		}
	}
	return ids, nil
}

// files is the catalog's JSON files, for an error message.
func (c catalog) files() string {
	names := make([]string, 0, len(c.sources))
	for _, src := range c.sources {
		names = append(names, src.json)
	}
	return strings.Join(names, " or ")
}

// setArt rewrites one line per record rather than re-encoding the file. data/relics.json is
// hand-formatted — a rule's If clause sits on one line — and a round-trip through encoding/json
// would reflow all of it, burying a six-line change in a twelve-hundred-line diff. The essence and
// rune files are machine-formatted today and would survive a round-trip, but one path through
// this function is worth more than the difference.
func setArt(records map[string]source, keys []string) error {
	// Grouped by file so each one is read once and written once, however a batch was mixed.
	byFile := map[source][]string{}
	for _, key := range keys {
		src := records[key]
		byFile[src] = append(byFile[src], key)
	}
	for src, keys := range byFile {
		raw, err := os.ReadFile(src.json)
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
			re := regexp.MustCompile(`("` + src.key + `": "` + regexp.QuoteMeta(key) + `",
(?:[^
]*
){0,4}?[ 	]*"Art": )"[^"]*"`)
			if !re.MatchString(s) {
				return fmt.Errorf("%s: found no Art field on record %q", src.json, key)
			}
			s = re.ReplaceAllString(s, "${1}\""+key+"\"")
		}
		if err := os.WriteFile(src.json, []byte(s), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// undrawn counts the records still carrying no art, read back off the file rather than
// subtracted from what was just filed — the file is the truth, and an arithmetic count can
// only ever drift from it. It is the whole of what the worklist used to be.
func undrawn(cat catalog) (int, error) {
	left := 0
	for _, src := range cat.sources {
		raw, err := os.ReadFile(src.json)
		if err != nil {
			return 0, err
		}
		var file []struct {
			Art string `json:"Art"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			return 0, fmt.Errorf("%s: %w", src.json, err)
		}
		for _, r := range file {
			if r.Art == "" {
				left++
			}
		}
	}
	return left, nil
}
