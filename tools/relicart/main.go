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
//	go run ./tools/relicart -kind cantrip-relic  # <record>-relic.png, transparent: makes the relic and the scroll
//	go run ./tools/relicart -kind card     # the playing cards instead
//	go run ./tools/relicart -kind upgrade  # the upgrade art an altered card wears over its face
//	go run ./tools/relicart -kind deck     # the card backs, one per deck
//	go run ./tools/relicart -kind achievement  # the achievement icons, square at Steam's 256
//	go run ./tools/relicart -kind enemy-card   # the creature cards, <flavor>-<weight>-<element>.png, fitted into the field
//	go run ./tools/relicart -n                 # say what would happen and touch nothing
//	go run ./tools/relicart -blocky            # quantize to the block grid on the way down
//
// # Every catalog, one command
//
// **Every catalog here carries Art and Draw and draws its picture at the card's own size**, so
// filing a picture is the identical four steps for each and a second command would be this file
// copied with three strings changed. `-kind` is the parameter and `catalogs` is the whole
// difference: an inbox, an asset directory, a JSON file, and the name of that file's record key.
// The name stays `relicart` because the relics are what it is reached for; renaming it would break
// the muscle memory and the CLAUDE.md line for nothing.
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
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/curiousjc/ascend-duel/data"
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

	// square, when set, is the side of a square picture this catalog is committed at instead of
	// the card's own 200x280. **Only the achievement icons set it**: they are not a card at all but
	// Steam's 256x256 icon, and the page draws them at Steam's 64.
	square int

	// elements, when set, makes this a catalog of **one record, several pictures**: a file is
	// `<record>-<element>.png`, it is committed under that name, and the record's Art is the stem
	// `<record>` with the element appended at draw time. **Only the creature cards set it** — a
	// flavor is drawn once per element. Art is written only once every element is on disk, because
	// a stem with one missing element is a card that silently draws nothing in that realm.
	elements []string

	// weights, when set alongside elements, puts a weight between the record and the element: a
	// file is `<record>-<weight>-<element>.png`. **Only the creature cards set it** — a flavor is
	// drawn once per weight per element — and a weight is listed on the record's `Weights` only once
	// its every element is on disk, for the reason above. See setWeights for how it is written.
	weights []string

	// field, when set, is the open part of the card a picture is fitted into rather than stretched
	// across the whole of it: the figure is cropped to what it actually draws, scaled to fill its
	// weight's box inside the field (see data.EnemyCardFieldFor), and centered. **Only the creature
	// cards set it**, because their pictures are
	// a figure laid over a face that prints a mark, ticks, a badge and a name around it, and a figure
	// that strays under those makes them unreadable. docs/art/enemy_card_art_prompt.MD asks for the
	// same box; this is what holds it when a generator does not.
	field image.Rectangle

	// suffix, when set, is on the end of every file's stem and not on the record's key: a file is
	// `<record><suffix>.png`, it is committed under that name, and the whole stem is what is written
	// into the art field. **Only the cantrip-relics set it**, because a cantrip record carries two
	// pictures — the scroll and the relic it casts — and both are filed into one flat directory.
	suffix string

	// scroll, when set, makes one transparent picture into **both of a cantrip's faces**: the relic,
	// cut and laid on the near-black ground, and the scroll, the blank parchment with the same cut
	// spirit laid on its sheet. **Only the cantrip-relics set it**, so a cantrip has one picture
	// and its scroll is generated from it — see cantripFaces.
	scroll bool
}

// size is what a picture in this catalog is reduced to.
func (c catalog) size() (int, int) {
	if c.square > 0 {
		return c.square, c.square
	}
	return cards.RelicStyle.Width, cards.RelicStyle.Height
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

	// art is the field the picture's stem is written into, and empty means `Art`. **Only the
	// achievements differ**, whose field is `AchievedIcon` because the file wears Steamworks' names.
	art string

	// block, when set, is the object inside the record the art field lives in. **Only the
	// cantrip-relics set it**: their picture is the `Art` inside a cantrip's `Relic` block, not the
	// scroll's own `Art` above it.
	block string
}

// artField is the name of the field this file records a picture in.
func (s source) artField() string {
	if s.art == "" {
		return "Art"
	}
	return s.art
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
	"cantrip-relic": {
		inbox:   filepath.Join(".scratch", "to-process-cantrip-relic-art"),
		out:     filepath.Join("assets", "cantrip"),
		sources: []source{{json: "data/cantrips.json", key: "CantripRecord", block: "Relic"}},
		suffix:  "-relic",
		scroll:  true,
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
		inbox:   filepath.Join(".scratch", "to-process-card-art"),
		out:     filepath.Join("assets", "card"),
		sources: []source{{json: "data/card_art.json", key: "CardArtRecord"}},
	},
	"upgrade": {
		inbox:   filepath.Join(".scratch", "to-process-upgrade-art"),
		out:     filepath.Join("assets", "upgrade-art"),
		sources: []source{{json: "data/upgrade_art.json", key: "UpgradeArtRecord"}},
	},
	"deck": {
		inbox:   filepath.Join(".scratch", "to-process-deck-art"),
		out:     filepath.Join("assets", "deck"),
		sources: []source{{json: "data/decks.json", key: "DeckRecord"}},
	},
	"other": {
		inbox: filepath.Join(".scratch", "to-process-other-art"),
		out:   filepath.Join("assets", "other"),
		sources: []source{
			{json: "data/potions.json", key: "PotionRecord"},
			{json: "data/tonics.json", key: "TonicRecord"},
			{json: "data/goods.json", key: "GoodRecord"},
		},
	},
	"enemy-card": {
		inbox:    filepath.Join(".scratch", "to-process-enemy-card-art"),
		out:      filepath.Join("assets", "enemy-card"),
		sources:  []source{{json: "data/enemy_card_art.json", key: "Flavor"}},
		elements: data.AffinityElements,
		weights:  data.EnemyCardWeights,
		field:    data.EnemyCardArtField,
	},
	"achievement": {
		inbox:   filepath.Join(".scratch", "to-process-achievement-art"),
		out:     filepath.Join("assets", data.AchievementIconDir),
		sources: []source{{json: "data/achievements.json", key: "APIName", art: "AchievedIcon"}},
		square:  data.AchievementIconSize,
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

	// **Every card catalog draws a full-bleed card at RelicStyle's size.** EssenceStyle is the
	// same width and height — the two differ in the text band, not in the picture — so one target
	// size is a fact about the card rather than a shortcut. TestEveryBleedingCardArtIsTheCardsOwnSize
	// is what fails if that stops being true. The achievement icons are the square exception.
	w, h := cat.size()

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
		if _, ok := records[cat.recordOf(key)]; !ok {
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
		field := cat.field
		if _, weight := cat.split(key); weight != "" {
			field = data.EnemyCardFieldFor(weight)
		}
		var (
			img, scrollFace image.Image
			size            string
			err             error
		)
		if cat.scroll {
			img, scrollFace, size, err = cantripFaces(src, w, h)
		} else {
			img, size, err = reduce(src, w, h, *blocky, field)
		}
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
		if scrollFace != nil {
			if err := writePNG(filepath.Join(*out, cat.recordOf(key)+".png"), scrollFace); err != nil {
				log.Fatal(err)
			}
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
	stems, finished, waiting := cat.complete(keys, *out)
	for _, line := range waiting {
		fmt.Println("  waiting: " + line)
	}
	if cat.weights != nil {
		if err := setWeights(cat.sources[0].json, finished); err != nil {
			log.Fatal(err)
		}
	} else if err := setArt(records, stems, cat.suffix); err != nil {
		log.Fatal(err)
	}
	// **The scroll each picture made is recorded on the cantrip's own Art**, by the same rewrite, as
	// a record with no suffix and no block.
	if cat.scroll {
		scrolls := map[string]source{}
		var recs []string
		for _, stem := range stems {
			rec := cat.recordOf(stem)
			src := records[rec]
			scrolls[rec] = source{json: src.json, key: src.key}
			recs = append(recs, rec)
		}
		if err := setArt(scrolls, recs, ""); err != nil {
			log.Fatal(err)
		}
	}
	left, err := undrawn(cat)
	if *kind == "achievement" {
		fmt.Println("the unachieved icons are derived: go run ./tools/achievementsheet writes both of each pair")
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%d %s(s) filed. %d still to draw.\n", len(keys), *kind, left)
}

// recordOf is the record a file's stem belongs to: the stem itself, or for a per-element catalog
// the stem with its `-<element>` — and its `-<weight>`, where the catalog has weights — taken off.
// A stem naming no element or no weight answers "", which names no record and is refused like any
// other misspelling.
func (c catalog) recordOf(stem string) string {
	rec, _ := c.split(stem)
	return rec
}

// split is recordOf plus the weight it took off, "" for a catalog without weights.
func (c catalog) split(stem string) (record, weight string) {
	if c.elements == nil {
		if c.suffix != "" {
			rec, ok := strings.CutSuffix(stem, c.suffix)
			if !ok {
				return "", ""
			}
			return rec, ""
		}
		return stem, ""
	}
	for _, el := range c.elements {
		rest, ok := strings.CutSuffix(stem, "-"+el)
		if !ok {
			continue
		}
		if c.weights == nil {
			return rest, ""
		}
		for _, w := range c.weights {
			if rec, ok := strings.CutSuffix(rest, "-"+w); ok {
				return rec, w
			}
		}
		return "", ""
	}
	return "", ""
}

// complete is what can be written now, and a line for each set that has to wait. For an ordinary
// catalog that is every key filed. For a per-element one it is each record — or each record's
// weight — touched by this batch that now has **every** element on disk in out, counting pictures
// filed by earlier runs; one still short is left out and says which elements it is missing.
//
// stems is the records for setArt; done is each record's newly complete weights, for setWeights.
func (c catalog) complete(keys []string, out string) (stems []string, done map[string][]string, waiting []string) {
	if c.elements == nil {
		return keys, nil, nil
	}
	done = map[string][]string{}
	seen := map[string]bool{}
	for _, key := range keys {
		rec, weight := c.split(key)
		set := rec
		if weight != "" {
			set += "-" + weight
		}
		if seen[set] {
			continue
		}
		seen[set] = true
		var missing []string
		for _, el := range c.elements {
			if _, err := os.Stat(filepath.Join(out, set+"-"+el+".png")); err != nil {
				missing = append(missing, el)
			}
		}
		if len(missing) > 0 {
			waiting = append(waiting, fmt.Sprintf("%s has no %s yet, so it draws no picture", set, strings.Join(missing, ", ")))
			continue
		}
		if len(done[rec]) == 0 {
			stems = append(stems, rec)
		}
		done[rec] = append(done[rec], weight)
	}
	return stems, done, waiting
}

// setWeights adds each record's newly complete weights to its `Weights`, in the catalog's weight
// order, and writes its stem into `Art`.
//
// **A round trip through encoding/json rather than setArt's line rewrite**, because a list is the
// one field a single-line pattern cannot rewrite safely, and data/enemy_card_art.json is
// machine-formatted — two-space indent, keys in struct order, a Ladder's keys sorted — which is
// exactly what MarshalIndent writes back, so an untouched record comes back byte for byte.
func setWeights(path string, done map[string][]string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var file []data.EnemyCardArtData
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for i, f := range file {
		add := done[f.Flavor]
		if len(add) == 0 {
			continue
		}
		has := map[string]bool{}
		for _, w := range append(append([]string(nil), f.Weights...), add...) {
			has[w] = true
		}
		file[i].Weights = []string{}
		for _, w := range data.EnemyCardWeights {
			if has[w] {
				file[i].Weights = append(file[i].Weights, w)
			}
		}
		file[i].Art = f.Flavor
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(file); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// reduce reads one generated PNG and scales it to the card's own size. The smooth path is
// CatmullRom, which is what the committed catalog was made with. The blocky path quantizes to
// the block grid the prompt asks for and scales back up by a whole number, so every block lands
// on an exact square instead of being resampled across one.
//
// **With a field, the picture is fitted rather than stretched**: see fitField.
func reduce(path string, w, h int, blocky bool, field image.Rectangle) (image.Image, string, error) {
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

	// A generator that cannot write transparency is asked for a flat magenta ground instead, and
	// that ground is taken out here, before the reduction — see keyMagenta.
	if keyed, ok := keyMagenta(src); ok {
		src = keyed
		size += ", magenta ground keyed out"
	}

	// The art is fitted into the card by drawArt, never cropped, so an aspect ratio that is not
	// the card's letterboxes. Worth saying out loud rather than finding on the sheet.
	if got, want := float64(b.Dx())/float64(b.Dy()), float64(w)/float64(h); got < want*0.99 || got > want*1.01 {
		fmt.Printf("  warning: %s is %s, which is not the card's %d:%d — it will letterbox\n",
			filepath.Base(path), size, w, h)
	}

	if !field.Empty() {
		if blocky {
			return nil, "", fmt.Errorf("-blocky does not apply to a catalog fitted into a field")
		}
		dst, scale := fitField(src, w, h, field)
		return dst, fmt.Sprintf("%s, fitted at %.2f", size, scale), nil
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

// fitField draws a transparent figure into a w x h card, filling field, and reports the scale it
// was drawn at against a plain reduction to the card.
//
// The figure is cropped to the pixels it actually draws — alpha above a tenth, so a haze of nearly
// invisible smoke does not stretch the box — and then **scaled until it just fills the box, up or
// down**: one batch's light figure drawn at a quarter of the field and another's at a half both
// come back the size a light figure is, so a weight ladder read across batches steps evenly. The
// source is several times the card's size, so even a figure scaled up is still a reduction. It is
// centered in the box, and drawn in one CatmullRom step from the full-size source, the same single
// reduction every other catalog gets.
func fitField(src image.Image, w, h int, field image.Rectangle) (*image.RGBA, float64) {
	b := src.Bounds()
	crop := image.Rectangle{Min: b.Max, Max: b.Min}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := src.At(x, y).RGBA(); a > 0x1800 {
				crop = crop.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if crop.Empty() {
		return dst, 0
	}

	// The figure's size on the card if the whole canvas were simply reduced to it.
	natW := float64(crop.Dx()) * float64(w) / float64(b.Dx())
	natH := float64(crop.Dy()) * float64(h) / float64(b.Dy())
	scale := min(float64(field.Dx())/natW, float64(field.Dy())/natH)
	tw, th := int(natW*scale+0.5), int(natH*scale+0.5)
	x := field.Min.X + (field.Dx()-tw)/2
	y := field.Min.Y + (field.Dy()-th)/2
	xdraw.CatmullRom.Scale(dst, image.Rect(x, y, x+tw, y+th), src, crop, xdraw.Src, nil)
	return dst, scale
}

// keyMagenta takes a flat magenta ground out of a picture, and reports whether it found one.
//
// **It only acts on a picture whose four corners are all magenta**, the fallback the card prompt
// asks for when a generator cannot write a real alpha channel — so a picture with a real ground,
// or real transparency, comes back untouched. Each pixel's alpha is how far it sits from magenta,
// measured the way a soft key is: the smallest alpha that lays the pixel's color over magenta to
// produce what was drawn. An anti-aliased edge therefore keeps its softness instead of carrying a
// pink fringe.
func keyMagenta(src image.Image) (image.Image, bool) {
	b := src.Bounds()
	magentaAt := func(x, y int) bool {
		r, g, bl, a := src.At(x, y).RGBA()
		return a > 0xf000 && r > 0xd000 && bl > 0xd000 && g < 0x3000
	}
	for _, p := range []image.Point{{b.Min.X, b.Min.Y}, {b.Max.X - 1, b.Min.Y}, {b.Min.X, b.Max.Y - 1}, {b.Max.X - 1, b.Max.Y - 1}} {
		if !magentaAt(p.X, p.Y) {
			return src, false
		}
	}
	key := [3]int{255, 0, 255}
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := src.At(x, y).RGBA()
			p := [3]int{int(r >> 8), int(g >> 8), int(bl >> 8)}
			// Alpha out of 255: each channel's distance from the key as a share of how far it could
			// have moved in that direction; the largest share wins.
			alpha := 0
			for i := range p {
				d, room := p[i]-key[i], 255-key[i]
				if d < 0 {
					d, room = -d, key[i]
				}
				if room > 0 {
					if a := d * 255 / room; a > alpha {
						alpha = a
					}
				}
			}
			if alpha > 255 {
				alpha = 255
			}
			if alpha < 8 {
				continue
			}
			// The color that, laid over the key at that alpha, reproduces the pixel.
			var c [3]uint8
			for i := range p {
				v := key[i] + (p[i]-key[i])*255/alpha
				c[i] = uint8(min(max(v, 0), 255))
			}
			out.SetNRGBA(x-b.Min.X, y-b.Min.Y, color.NRGBA{R: c[0], G: c[1], B: c[2], A: uint8(alpha)})
		}
	}
	return out, true
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
//
// Each stem is written whole; the record it lands on is the stem with suffix taken off, which for
// every catalog but the cantrip-relics is the stem itself.
func setArt(records map[string]source, stems []string, suffix string) error {
	// Grouped by file so each one is read once and written once, however a batch was mixed.
	byFile := map[source][]string{}
	for _, stem := range stems {
		src := records[strings.TrimSuffix(stem, suffix)]
		byFile[src] = append(byFile[src], stem)
	}
	for src, keys := range byFile {
		raw, err := os.ReadFile(src.json)
		if err != nil {
			return err
		}
		s := string(raw)
		for _, stem := range keys {
			key := strings.TrimSuffix(stem, suffix)
			// **Bounded and lazy, so it cannot walk into the next record.** The header fields between
			// RelicRecord and Art are authored and have grown once already — Family landed between Name
			// and Art on 2026-09-12, and a pattern allowing exactly one intervening line then matched no
			// record in the file. Four is headroom for the next one; an unbounded `*` would silently
			// retarget a record whose own Art was missing.
			// **A nested field skips into its block first**, by the same bounded, lazy walk: the
			// cantrip's own header — Name, Family, Art, Draw, Text — then the block's opening line,
			// then the block's own header up to its Art. Without the block step the scroll's Art,
			// which comes first, is the one rewritten.
			lines := func(n int) string { return `(?:[^\n]*\n){0,` + fmt.Sprint(n) + `}?` }
			pat := `("` + src.key + `": "` + regexp.QuoteMeta(key) + `",\n`
			if src.block != "" {
				pat += lines(8) + `[ \t]*"` + src.block + `": \{\n`
			}
			pat += lines(4) + `[ \t]*"` + src.artField() + `": )"[^"]*"`
			re := regexp.MustCompile(pat)
			if !re.MatchString(s) {
				return fmt.Errorf("%s: found no %s field on record %q", src.json, src.artField(), key)
			}
			s = re.ReplaceAllString(s, "${1}\""+stem+"\"")
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
		var file []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &file); err != nil {
			return 0, fmt.Errorf("%s: %w", src.json, err)
		}
		for _, r := range file {
			if src.block != "" {
				var inner map[string]json.RawMessage
				if err := json.Unmarshal(r[src.block], &inner); err != nil {
					return 0, fmt.Errorf("%s: a %s is not an object: %w", src.json, src.block, err)
				}
				r = inner
			}
			var art string
			if v, ok := r[src.artField()]; ok {
				if err := json.Unmarshal(v, &art); err != nil {
					return 0, fmt.Errorf("%s: an %s is not a string: %w", src.json, src.artField(), err)
				}
			}
			if art == "" {
				left++
			}
		}
	}
	return left, nil
}

// scrollBlank is the blank parchment every cantrip's scroll is generated on, committed beside this
// tool because nothing else reads it. scrollSheet is its open sheet, as a share of the blank's size:
// where the spirit is laid.
const scrollBlank = "tools/relicart/cantrip-scroll-blank.png"

var scrollSheet = [4]float64{130.0 / 1060, 330.0 / 1484, 930.0 / 1060, 1080.0 / 1484}

// scrollFill is how much of the sheet the spirit fills, at most, along whichever side binds first.
const scrollFill = 0.95

// cutLow and cutHigh are the hard cut every cantrip spirit takes before it is laid on either
// ground: alpha below cutLow is gone, above cutHigh is solid, a ramp between. **It drops the faint
// haze a generator leaves round a glowing figure**, which reads as a smudge on the parchment, and
// keeps the figure itself solid at card size.
const cutLow, cutHigh = 18, 70

// cantripGround is the near-black a cantrip-relic's spirit is laid on: the ground every relic
// picture is drawn on.
var cantripGround = color.NRGBA{R: 20, G: 22, B: 28, A: 255}

// cantripFaces makes both of a cantrip's faces out of one transparent picture of its spirit: the
// relic, the cut spirit on the near-black ground, and the scroll, the blank parchment with the same
// cut spirit laid on its sheet. Both come back at the card's own size.
//
// **A picture with no transparency is refused**: there is nothing to cut, and laying an opaque
// rectangle on the parchment would put a dark box on the scroll.
func cantripFaces(path string, w, h int) (relic, scroll image.Image, size string, err error) {
	src, err := decodePNG(path)
	if err != nil {
		return nil, nil, "", err
	}
	b := src.Bounds()
	size = fmt.Sprintf("%dx%d", b.Dx(), b.Dy())
	if _, _, _, a := src.At(b.Min.X, b.Min.Y).RGBA(); a != 0 {
		return nil, nil, "", fmt.Errorf("%s has no transparent ground; the cantrip prompt asks for one", filepath.Base(path))
	}

	spirit := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			switch {
			case c.A < cutLow:
				c.A = 0
			case c.A > cutHigh:
				c.A = 255
			default:
				c.A = uint8((int(c.A) - cutLow) * 255 / (cutHigh - cutLow))
			}
			spirit.SetNRGBA(x-b.Min.X, y-b.Min.Y, c)
		}
	}

	// The relic: the spirit on the near-black, reduced to the card.
	full := image.NewRGBA(spirit.Bounds())
	xdraw.Draw(full, full.Bounds(), &image.Uniform{C: cantripGround}, image.Point{}, xdraw.Src)
	xdraw.Draw(full, full.Bounds(), spirit, image.Point{}, xdraw.Over)
	relicFace := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(relicFace, relicFace.Bounds(), full, full.Bounds(), xdraw.Src, nil)

	// The scroll: the spirit fitted into the blank's sheet, then the whole reduced to the card.
	blank, err := decodePNG(scrollBlank)
	if err != nil {
		return nil, nil, "", err
	}
	bb := blank.Bounds()
	page := image.NewRGBA(image.Rect(0, 0, bb.Dx(), bb.Dy()))
	xdraw.Draw(page, page.Bounds(), blank, bb.Min, xdraw.Src)
	sheet := image.Rect(int(scrollSheet[0]*float64(bb.Dx())), int(scrollSheet[1]*float64(bb.Dy())),
		int(scrollSheet[2]*float64(bb.Dx())), int(scrollSheet[3]*float64(bb.Dy())))
	sb := spirit.Bounds()
	k := scrollFill * min(float64(sheet.Dx())/float64(sb.Dx()), float64(sheet.Dy())/float64(sb.Dy()))
	sw, sh := int(float64(sb.Dx())*k), int(float64(sb.Dy())*k)
	min0 := image.Pt(sheet.Min.X+(sheet.Dx()-sw)/2, sheet.Min.Y+(sheet.Dy()-sh)/2)
	xdraw.CatmullRom.Scale(page, image.Rectangle{Min: min0, Max: min0.Add(image.Pt(sw, sh))},
		spirit, sb, xdraw.Over, nil)
	scrollFace := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(scrollFace, scrollFace.Bounds(), page, page.Bounds(), xdraw.Src, nil)

	return relicFace, scrollFace, size + ", cut, laid on near-black and on the scroll", nil
}

// decodePNG reads one PNG off disk.
func decodePNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
