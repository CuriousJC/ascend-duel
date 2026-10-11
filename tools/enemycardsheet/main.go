// Command enemycardsheet writes the creature cards' picture catalog: every flavor in
// data/enemy_card_art.json drawn as a card in each of the five elements, beside every creature card
// across the roster that wears it, and a table of how each motif leans across the three forms.
//
//	go run ./tools/enemycardsheet
//
// A flavor's pictures are spread over hundreds of cards in eighteen motif files, and a creature is
// met one at a time — so "which cards will this picture be on" and "does this motif still lean the
// way it should" cannot be answered by playing. The page answers both.
//
// # It is validated
//
// It reads data.LoadMotifs and data.LoadEnemyCardArt, and both refuse what the game refuses: a card
// with no form, a flavor the catalog does not have, a flavor of another form. So a page that
// builds is a roster that launches.
//
// # Output
//
// One PNG per flavor per element plus an index.html, written into `docs/sheets/enemycardsheet/`
// and committed on the terms every other sheet is.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

func main() {
	dir := flag.String("dir", filepath.Join("docs", "sheets", "enemycardsheet"),
		"directory to write the PNGs and index.html into")
	flag.Parse()

	if err := run(*dir); err != nil {
		log.Fatal(err)
	}
}

// use is one creature card wearing a flavor.
type use struct {
	Motif, Creature, Tier, Label string
	Cost, Amount, Copies         int
}

type face struct {
	Element string
	File    string
}

type plate struct {
	Flavor, Form, Name, Draw, Art string
	Faces                         []face
	Uses                          []use
	Cards, Copies, MotifCount     int
	Motifs                        string
}

type share struct {
	Form string
	Pct  float64
}

type motifRow struct {
	Name    string
	Lead    string
	Second  string
	Shares  []share
	Flavors []string
}

type page struct {
	Cards, Flavors, Pictures, Drawn int
	Forms                           []share
	Motifs                          []motifRow
	Plates                          []plate
	Filter                          template.HTML
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	// **Every PNG is written fresh**, so one left over from a retired flavor would sit in the
	// directory looking current.
	stale, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	for _, f := range stale {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	faces, err := cards.NewFaces(assets.LoadFontData()["kubasta"])
	if err != nil {
		return err
	}
	images := assets.LoadImageData()
	art := data.LoadEnemyCardArt()
	motifs := data.LoadMotifs()

	uses := map[string][]use{}
	formCopies := map[string]int{}
	total, totalCopies := 0, 0
	var rows []motifRow
	for _, key := range data.MotifOrder(motifs) {
		m := motifs[key]
		byForm := map[string]int{}
		byFlavor := map[string]int{}
		copies := 0
		for _, r := range m.Records {
			for _, c := range r.Cards {
				uses[c.Flavor] = append(uses[c.Flavor], use{
					Motif: m.Name, Creature: r.FullName(), Tier: r.Tier, Label: c.Label,
					Cost: c.Cost, Amount: c.Amount, Copies: c.Copies,
				})
				byForm[c.Form] += c.Copies
				byFlavor[c.Flavor] += c.Copies
				formCopies[c.Form] += c.Copies
				copies += c.Copies
				total++
			}
		}
		totalCopies += copies
		rows = append(rows, motifRow{Name: m.Name, Shares: shares(byForm, copies), Flavors: ranked(byFlavor)})
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].Name < rows[b].Name })
	for i := range rows {
		ordered := append([]share(nil), rows[i].Shares...)
		sort.SliceStable(ordered, func(a, b int) bool { return ordered[a].Pct > ordered[b].Pct })
		rows[i].Lead = ordered[0].Form
		if ordered[1].Pct > 0 {
			rows[i].Second = ordered[1].Form
		}
	}

	p := page{Cards: total, Flavors: len(art), Pictures: len(art) * len(data.AffinityElements),
		Forms: shares(formCopies, totalCopies), Motifs: rows}

	formCount := map[string]int{}
	for _, key := range data.EnemyCardArtFileOrder() {
		f := art[key]
		pl := plate{Flavor: f.Flavor, Form: f.Form, Name: f.Name, Draw: f.Draw, Art: f.Art, Uses: uses[key]}
		sort.SliceStable(pl.Uses, func(a, b int) bool {
			if pl.Uses[a].Motif != pl.Uses[b].Motif {
				return pl.Uses[a].Motif < pl.Uses[b].Motif
			}
			return pl.Uses[a].Cost < pl.Uses[b].Cost
		})
		seen := map[string]bool{}
		var tokens []string
		for _, u := range pl.Uses {
			pl.Cards++
			pl.Copies += u.Copies
			if !seen[u.Motif] {
				seen[u.Motif] = true
				tokens = append(tokens, token(u.Motif))
			}
		}
		pl.MotifCount = len(seen)
		pl.Motifs = strings.Join(tokens, " ")
		formCount[f.Form]++

		for _, el := range data.AffinityElements {
			picture, err := picture(images, f.ArtKey(el))
			if err != nil {
				return fmt.Errorf("%s in %s: %w", key, el, err)
			}
			if picture != nil {
				p.Drawn++
			}
			img, err := cards.Render(cards.Spec{
				Name: f.Name, Form: cardForm(f.Form), Cost: 2, Element: cardElement(el),
				Enabled: true, Art: picture,
			}, cards.Hand, faces)
			if err != nil {
				return fmt.Errorf("rendering %s in %s: %w", key, el, err)
			}
			name := key + "-" + el + ".png"
			if err := writePNG(filepath.Join(dir, name), img); err != nil {
				return err
			}
			pl.Faces = append(pl.Faces, face{Element: el, File: name})
		}
		p.Plates = append(p.Plates, pl)
	}

	var formValues []sheetfilter.Value
	for _, f := range data.EnemyForms {
		formValues = append(formValues, sheetfilter.Value{Value: f, Count: formCount[f]})
	}
	var motifValues []sheetfilter.Value
	for _, r := range rows {
		motifValues = append(motifValues, sheetfilter.Value{Value: token(r.Name), Label: r.Name})
	}
	p.Filter = sheetfilter.Bar([]sheetfilter.Facet{
		{Key: "form", Label: "Form", Values: formValues},
		{Key: "motif", Label: "Motif", Values: motifValues},
	})

	out := filepath.Join(dir, "index.html")
	file, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	defer file.Close()
	if err := tmpl.Execute(file, p); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Printf("wrote %s — %d flavors over %d creature cards, %d of %d pictures drawn\n",
		out, p.Flavors, p.Cards, p.Drawn, p.Pictures)
	return nil
}

// shares is each form's share of a set of copies, in EnemyForms order.
func shares(by map[string]int, total int) []share {
	out := make([]share, 0, len(data.EnemyForms))
	for _, f := range data.EnemyForms {
		pct := 0.0
		if total > 0 {
			pct = 100 * float64(by[f]) / float64(total)
		}
		out = append(out, share{Form: f, Pct: pct})
	}
	return out
}

// ranked is a motif's flavors, most copies first, ties by name so the page does not reorder itself
// between runs.
func ranked(by map[string]int) []string {
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if by[keys[a]] != by[keys[b]] {
			return by[keys[a]] > by[keys[b]]
		}
		return keys[a] < keys[b]
	})
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = fmt.Sprintf("%s %d", k, by[k])
	}
	return out
}

func token(name string) string { return strings.ReplaceAll(strings.ToLower(name), " ", "-") }

// picture decodes one flavor's picture, or nil for one not drawn yet. **A key with an Art stem and
// no file is an error**, where the game draws the card without one: a review tool that quietly drew
// nothing would hide exactly what it is for.
func picture(images map[string][]byte, key string) (image.Image, error) {
	if key == "" {
		return nil, nil
	}
	raw := images[key]
	if len(raw) == 0 {
		return nil, fmt.Errorf("no embedded picture %q", key)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

func cardForm(name string) cards.Form {
	for _, f := range cards.Forms() {
		if f != cards.FormNone && f.String() == name {
			return f
		}
	}
	return cards.FormNone
}

func cardElement(name string) cards.Element {
	for _, e := range cards.Elements() {
		if e.String() == name {
			return e
		}
	}
	return cards.Basic
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

var tmpl = template.Must(template.New("page").Funcs(template.FuncMap{
	"pct": func(f float64) string { return fmt.Sprintf("%.0f%%", f) },
	"w":   func(f float64) template.CSS { return template.CSS(fmt.Sprintf("width:%.1f%%", f)) },
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Enemy card sheet</title>
<style>
:root { --ground: #a8bcd4; --panel: #eef2f7; --ink: #1d2030; --muted: #4c556a; --line: #c4cedb;
  --stab: #8a6d2f; --slash: #3f5f80; --crush: #6e5a52; }
body { margin: 0; background: var(--ground); color: var(--ink); font: 15px/1.45 "Segoe UI", sans-serif; }
main { max-width: 1180px; margin: 0 auto; padding: 24px 16px 64px; display: grid; gap: 18px; }
h1 { margin: 0; font-size: 1.9rem; } h2 { margin: 12px 0 0; font-size: 1.4rem; }
.lede { margin: 0; max-width: 80ch; color: var(--muted); }
.f-stab { --tone: var(--stab); } .f-slash { --tone: var(--slash); } .f-crush { --tone: var(--crush); }
table { border-collapse: collapse; width: 100%; background: var(--panel); border-radius: 6px; overflow: hidden; }
th, td { text-align: left; padding: 5px 9px; border-bottom: 1px solid var(--line); vertical-align: top; }
th { font-size: .78rem; letter-spacing: .05em; text-transform: uppercase; color: var(--muted); }
.num { text-align: right; font-variant-numeric: tabular-nums; }
.stack { display: flex; height: 12px; min-width: 160px; border-radius: 3px; overflow: hidden; background: var(--line); }
.seg { background: var(--tone); } .seg + .seg { border-left: 2px solid var(--panel); }
.lead { font-weight: 700; color: var(--tone); }
.chips { font-size: .78rem; color: var(--muted); }
.plate { background: var(--panel); border-top: 4px solid var(--tone); border-radius: 6px; padding: 14px 16px; display: grid; gap: 10px; }
.plate h3 { margin: 0; display: flex; gap: 12px; align-items: baseline; flex-wrap: wrap; }
.plate h3 code { color: var(--tone); font-size: 1.1rem; }
.meta { font-size: .85rem; color: var(--muted); }
.faces { display: flex; gap: 8px; flex-wrap: wrap; }
.faces figure { margin: 0; text-align: center; font-size: .75rem; color: var(--muted); }
.faces img { display: block; width: 120px; height: auto; }
.draw { margin: 0; max-width: 90ch; }
.unwritten { color: #b03a5b; font-weight: 700; }
.tw { overflow-x: auto; }
details summary { cursor: pointer; color: var(--muted); }
</style></head><body><main>
<h1>Enemy card sheet</h1>
<p class="lede">Every creature card wears one of {{.Flavors}} pictures from <code>data/enemy_card_art.json</code>, each drawn once per element: {{.Pictures}} pictures, {{.Drawn}} drawn so far. {{.Cards}} creature cards share them. Across the roster, weighted by copies dealt:
{{range .Forms}}<span class="f-{{.Form}} lead">{{.Form}} {{pct .Pct}}</span> {{end}}.
Briefs are pasted after <code>docs/art/enemy_card_art_prompt.MD</code>; the procedure is the <code>enemy-cards</code> skill.</p>

<h2>By motif</h2>
<p class="lede">A realm is chosen by motif, so each motif should lean on one form. The lead is the form with the most copies dealt.</p>
<div class="tw"><table>
<thead><tr><th>Motif</th><th>Lead</th><th>Then</th><th>Forms by copies</th><th class="num">stab</th><th class="num">slash</th><th class="num">crush</th><th>Flavors by copies</th></tr></thead>
<tbody>{{range .Motifs}}<tr>
<td><b>{{.Name}}</b></td><td class="f-{{.Lead}} lead">{{.Lead}}</td><td class="f-{{.Second}} lead">{{.Second}}</td>
<td><div class="stack">{{range .Shares}}<span class="seg f-{{.Form}}" style="{{w .Pct}}"></span>{{end}}</div></td>
{{range .Shares}}<td class="num">{{pct .Pct}}</td>{{end}}
<td class="chips">{{range $i, $f := .Flavors}}{{if $i}} · {{end}}{{$f}}{{end}}</td>
</tr>{{end}}</tbody></table></div>

<h2>By flavor</h2>
{{.Filter}}
{{range .Plates}}<section class="plate sheet-item f-{{.Form}}" data-form="{{.Form}}" data-motif="{{.Motifs}}">
<h3><code>{{.Flavor}}</code> {{.Name}} <span class="meta">{{.Cards}} cards · {{.Copies}} copies · {{.MotifCount}} motifs · {{if .Art}}art <code>{{.Art}}-&lt;element&gt;</code>{{else}}no art yet{{end}}</span></h3>
<div class="faces">{{range .Faces}}<figure><img src="{{.File}}" alt="{{.Element}}"><figcaption>{{.Element}}</figcaption></figure>{{end}}</div>
<p class="draw">{{if .Draw}}{{.Draw}}{{else}}<span class="unwritten">No brief written.</span>{{end}}</p>
<details><summary>Every card wearing it</summary><div class="tw"><table>
<thead><tr><th>Motif</th><th>Creature</th><th>Card</th><th class="num">AP</th><th class="num">Amount</th><th class="num">Copies</th></tr></thead>
<tbody>{{range .Uses}}<tr><td>{{.Motif}}</td><td>{{.Creature}} <span class="meta">{{.Tier}}</span></td><td><b>{{.Label}}</b></td><td class="num">{{.Cost}}</td><td class="num">{{.Amount}}</td><td class="num">×{{.Copies}}</td></tr>{{end}}</tbody>
</table></div></details>
</section>
{{end}}
</main></body></html>
`))
