package roster

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

// The page and the types it walks.
//
// One static file and no build step: the loop is "edit a motif file, re-run the tool, refresh the
// tab", the same loop every other sheet here has. The only script on it is the chip bar from
// tools/sheetfilter, which narrows what is already on the page — the whole roster is in the file
// and readable with scripting off.
//
// **Grouped by floor rather than listed alphabetically**, for the reason the relic sheet groups by
// rarity: the floor is the whole placement decision, so the review question is "does anything in
// this band belong a floor deeper", which a flat list of ninety-six records cannot answer. Each
// heading carries the band's stat spread, so an outlier shows up as a number before it shows up as
// a card.

// cell is one picture on the page, at the size it was drawn.
type cell struct {
	File   string
	Width  int
	Height int
}

// row is one concept in an opponent's deck.
type row struct {
	Label  string
	Verb   string
	Effect string
	Cost   int
	Copies int
}

// plate is one opponent: the strip, and everything the file says about it.
type plate struct {
	Entry   Entry
	Cell    cell
	Affixes string

	// Deck is the expanded pile size — every copy of every concept — read off internal/decks
	// rather than added up here, so the figure is the pile a duel actually shuffles.
	Deck int

	Rows []row

	// Elems is the record's colours as the element chips match them — space-separated, where
	// Affixes is the same list written for a reader.
	Elems string

	// Others is the record's remaining colours as one row of cards, and OtherLabels names them in
	// the order it draws them. **Both are empty for a record whose other colours are undrawn** —
	// see anyDrawn — so a plate either has an expander with pictures in it or no expander at all.
	Others      cell
	OtherLabels []string
}

// group is one floor's worth of the catalog, with the band's own spread beside it.
//
// **The spread is what makes the grouping worth having.** A band whose HP runs 120 to 900 is not a
// band, it is two; and that is invisible while reading cards one at a time.
type group struct {
	Label  string
	Order  int
	Plates []plate

	// Motif is the record key of the motif this section is, and Floors is its band as the floor
	// chips match it. A floor is a fact about the motif rather than about one creature, so it is
	// the section that carries it and the section the chips cut.
	Motif  string
	Floors string

	MinHP, MaxHP   int
	MinDMG, MaxDMG int
	MinAP, MaxAP   int

	// Mix is the tier spread within the motif, written out on the heading — "3 outer, 3 inner,
	// 3 boss". The stat spread says whether the motif is pitched right and this says whether it
	// can fill a floor at all, which is the question the numbers cannot answer.
	Mix string

	// Coverage is the motif's grid: how many records can field each room at each element.
	//
	// **It is the one thing about a motif that cannot be seen by reading its records one at a
	// time.** A floor picks a motif and an element, so what has to hold is that every element can
	// field all three rooms — and the loader refuses a file that cannot, out of this same
	// function, so the page and the launch can never disagree about it.
	Coverage []coverRow
}

// coverRow is one element's row of the coverage grid: the element's name, and one cell per tier.
type coverRow struct {
	Element string
	Cells   []coverCell
}

// coverCell is one (tier, element) fight: how many records can be dealt into it, and whether that
// is fewer than a floor needs.
type coverCell struct {
	Count int
	Short bool
}

type page struct {
	Ground     string
	Title      string
	Blurb      string
	GroupLabel string
	Count      int
	Style      map[string]int
	Filters    template.HTML
	Groups     []group

	// SpanLo and SpanHi are the shallowest and deepest floor any motif reaches, which is what a
	// motif written with no band is expanded against.
	SpanLo, SpanHi int

	// CardWidth and Gap are the pitch the colour row is composited at, handed to the stylesheet so
	// the names under it sit on the cards. **Read off the style and the constant rather than typed
	// into the template**, which is the rule styleFacts is already under: a page quoting a pitch it
	// is not drawing at is a page with its labels one card to the left.
	CardWidth, Gap int
}

// add files one opponent under its floor, opening the section if it is the first.
//
// **Append rather than sort**, because the entries arrive in EnemyOrder / BossOrder — both of
// which are floor-first — so the sections come out in floor order by construction. Sorting here
// would be a second opinion about an order the data package already owns.
func (p *page) add(pl plate) {
	i := len(p.Groups) - 1
	// **Cut on the written band, not on the number it sorts by.** Two creatures can share a
	// lowest floor and differ in their highest — 1–2 and 1–3 — and a section keyed on the sort
	// number would file both under whichever label arrived first, which is a page saying a
	// creature reaches a floor it does not.
	if i < 0 || p.Groups[i].Label != pl.Entry.Floors {
		p.Groups = append(p.Groups, group{
			Label:  pl.Entry.Floors,
			Order:  pl.Entry.Group,
			Motif:  pl.Entry.Motif,
			Floors: floorTokens(pl.Entry.Band, p.SpanLo, p.SpanHi),
			MinHP:  pl.Entry.HP, MaxHP: pl.Entry.HP,
			MinDMG: pl.Entry.DMG, MaxDMG: pl.Entry.DMG,
			MinAP: pl.Entry.Actions, MaxAP: pl.Entry.Actions,
		})
		i = len(p.Groups) - 1
	}

	g := &p.Groups[i]
	g.Plates = append(g.Plates, pl)
	stretch(&g.MinHP, &g.MaxHP, pl.Entry.HP)
	stretch(&g.MinDMG, &g.MaxDMG, pl.Entry.DMG)
	stretch(&g.MinAP, &g.MaxAP, pl.Entry.Actions)
	g.Mix = familyMix(g.Plates)
	g.Coverage = coverageOf(pl.Entry.Motif)
}

// coverageOf reads the motif's grid straight out of data, so the page reports exactly what the
// loader checked.
//
// **A motif key that names nothing gives no grid**, which is what a pool that is not the motif
// pool would produce — the page then simply has no grid rather than an empty one.
func coverageOf(motif string) []coverRow {
	if motif == "" {
		return nil
	}
	motifs := data.LoadMotifs()
	m, ok := motifs[motif]
	if !ok {
		return nil
	}

	counts := data.CoverageOf(m).Counts
	rows := make([]coverRow, 0, len(data.AffinityElements))
	for ai, element := range data.AffinityElements {
		row := coverRow{Element: element}
		for ti, tier := range data.TierOrder {
			n := counts[ti][ai]
			row.Cells = append(row.Cells, coverCell{Count: n, Short: n < data.MinCoverageFor(tier)})
		}
		rows = append(rows, row)
	}
	return rows
}

// familyMix is the band's family spread, in the order the families first appear in it — which is
// the file's order, since the entries arrive in it.
//
// **Recomputed on every add rather than once at the end**, because the page is assembled as the
// strips are written and there is no second pass to hang it off. It is a walk over at most a few
// dozen plates a band, so the cost is nothing and the alternative is a finalise step somebody has
// to remember to call.
//
// **A record with no family is counted as "unfamilied"** rather than skipped: a band whose spread
// does not add up to its own count would be a heading that quietly lies.
func familyMix(plates []plate) string {
	order := make([]string, 0, 4)
	counts := map[string]int{}
	for _, p := range plates {
		name := p.Entry.Family
		if name == "" {
			name = "unfamilied"
		}
		if counts[name] == 0 {
			order = append(order, name)
		}
		counts[name]++
	}
	parts := make([]string, 0, len(order))
	for _, name := range order {
		parts = append(parts, fmt.Sprintf("%d %s", counts[name], name))
	}
	return strings.Join(parts, ", ")
}

func stretch(lo, hi *int, v int) {
	if v < *lo {
		*lo = v
	}
	if v > *hi {
		*hi = v
	}
}

// tmpl is the page.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// card sheet's template gives: a card's rim is one pixel thick and a browser that scales it — even
// by the fraction a max-width rule can introduce — resamples that rim into a blur and makes the
// sheet lie about the art. A strip is wider than most windows, so it scrolls inside its own box
// rather than being shrunk to fit.
//
// **The ground is the one the cards actually sit on**, not a page color chosen to flatter them.
var tmpl = template.Must(template.New("roster").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — {{.Title}}</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #5a6472;
    --rule: #8fa3bd;
    --panel: #c6d5e6;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    padding: 32px 28px 64px;
    background: var(--ground);
    color: var(--ink);
    font: 14px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif;
  }
  h1 { font-size: 20px; margin: 0 0 4px; font-weight: 600; }
  h2.floor {
    font-size: 15px; font-weight: 600;
    margin: 40px 0 0; padding-bottom: 7px; border-bottom: 2px solid var(--rule);
  }
  h2.floor span {
    font-weight: 400; font-size: 12px; color: var(--dim); margin-left: 12px;
  }
  /* A motif folds away. The page is one section per motif and a roster this deep is mostly
     scrolling past the motifs you are not reviewing, so the heading is the handle: a <details>,
     open by default, which keeps the whole roster in the file and prints and reads with
     scripting off. */
  details.motif > summary { list-style: none; cursor: pointer; }
  details.motif > summary::-webkit-details-marker { display: none; }
  /* The caret is drawn in the heading rather than left to the browser's marker, which would sit
     on its own line above a block-level h2 and read as a bullet rather than as a handle. */
  h2.floor::before {
    content: "▾"; color: var(--dim); font-weight: 400;
    display: inline-block; width: 1em; margin-left: -1em;
  }
  details.motif:not([open]) > summary h2.floor::before { content: "▸"; }
  details.motif > summary:hover h2.floor { color: var(--dim); }
  .facts { color: var(--dim); font-size: 12px; margin: 0 0 8px; }
  .facts code { color: var(--ink); }
  .note { color: var(--dim); font-size: 12.5px; max-width: 74ch; margin: 12px 0 0; }
  .plate {
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 16px; margin-top: 18px;
  }
  .head { display: flex; align-items: baseline; gap: 12px; flex-wrap: wrap; }
  /* The name and the title stack, because they are two fields now and the card only carries the
     first — see BossData.Name. Showing them as one line here would hide the split the page exists
     to make reviewable. */
  .named { display: inline-block; }
  .name { font-size: 16px; font-weight: 600; display: block; }
  .title { font-size: 12.5px; color: var(--dim); display: block; margin-top: 1px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .stats { margin-left: auto; font-size: 13px; }
  .stats b { font-weight: 600; }
  .stats span { color: var(--dim); margin-left: 14px; }
  /* Natural size, never scaled — see the comment above. A strip is up to about 1100px wide, so
     it gets its own scrollbar rather than a max-width rule that would resample every rim. */
  .strip { overflow-x: auto; margin: 12px 0 0; }
  img { display: block; image-rendering: pixelated; }
  table { border-collapse: collapse; margin: 12px 0 0; font-size: 12.5px; }
  th, td { text-align: left; padding: 3px 16px 3px 0; }
  th { color: var(--dim); font-weight: 600; font-size: 11px;
       text-transform: uppercase; letter-spacing: .06em; }
  td.num { text-align: right; padding-right: 22px; font-variant-numeric: tabular-nums; }
  td.effect { color: var(--dim); }
  .affix { color: var(--dim); font-size: 12px; margin: 10px 0 0; }
  /* The other colours are folded away, because the question they answer is asked of one record at
     a time: a page that opened every one of them would be four cards wide everywhere and would
     bury the deck the plate is actually about. It is a <details>, so the whole row is in the file
     and prints and reads with scripting off. */
  .others { margin: 8px 0 0; }
  .others summary {
    color: var(--dim); font-size: 12px; cursor: pointer; width: fit-content;
  }
  .others summary:hover { color: var(--ink); }
  .elabels { display: flex; gap: {{.Gap}}px; margin: 4px 0 0; }
  .elabels span {
    width: {{.CardWidth}}px; text-align: center;
    font-size: 11.5px; color: var(--dim);
  }
  /* The family sits beside the record key, quiet and in small caps: it is what kind of thing this
     is rather than what it is called, so it reads as a label on the name rather than a second
     name. */
  .family {
    color: var(--dim); font-size: 11px; letter-spacing: .05em;
    text-transform: uppercase; margin-left: 10px;
  }
  /* The subject paragraph is an *input* to an art generator rather than anything the game reads,
     so it is set apart from the stat line and the deck: indented and quieted, under everything
     the record actually does. Every one of them reads TBD today. */
  .draw {
    color: var(--dim); font-size: 12px; margin: 8px 0 0;
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
table.cover { border-collapse: collapse; margin: 0 0 18px; font-size: 13px; }
table.cover caption { text-align: left; opacity: .6; padding-bottom: 6px; max-width: 60ch; }
table.cover th, table.cover td { border: 1px solid rgba(128,128,128,.35); padding: 3px 10px; }
table.cover th { font-weight: 600; text-align: left; opacity: .75; }
table.cover td.num { text-align: right; font-variant-numeric: tabular-nums; }
table.cover td.short { color: #b03a3a; font-weight: 700; }
</style>

<h1>{{.Title}}</h1>
<p class="facts">
  {{.Count}} records, grouped by {{.GroupLabel}} ·
  card <code>{{index .Style "width"}}x{{index .Style "height"}}</code>,
  portrait box <code>{{index .Style "artMaxH"}}</code> tall,
  name at <code>{{index .Style "nameSize"}}pt</code>
</p>
<p class="note">{{.Blurb}}</p>
<p class="note">
  Every card here is drawn by <code>internal/cards</code>, the same code the game blits, so this
  page cannot show a card the game would draw differently. The leftmost card on each strip is the
  opponent as it appears in the corner of the combat screen; the rest are its deck, one card per
  concept — the <em>copies</em> column says how many of each the pile holds.
</p>

{{.Filters}}

{{range .Groups}}
<section class="sheet-group" data-motif="{{.Motif}}" data-floor="{{.Floors}}">
  <details class="motif" open>
  <summary>
  <h2 class="floor">
    {{.Label}}
    <span>{{len .Plates}} records · HP {{.MinHP}}–{{.MaxHP}} · DMG {{.MinDMG}}–{{.MaxDMG}} ·
      AP {{.MinAP}}–{{.MaxAP}} · {{.Mix}}</span>
  </h2>
  </summary>

  {{if .Coverage}}
  <table class="cover">
    <caption>How many records can field each fight. A floor picks this motif and one element and
      holds three rooms, so every cell needs at least two.</caption>
    <tr><th></th><th>outer</th><th>inner</th><th>boss</th></tr>
    {{range .Coverage}}
      <tr>
        <th>{{.Element}}</th>
        {{range .Cells}}<td class="num{{if .Short}} short{{end}}">{{.Count}}</td>{{end}}
      </tr>
    {{end}}
  </table>
  {{end}}

  {{range .Plates}}
    <div class="plate sheet-item" data-tier="{{.Entry.Tier}}" data-element="{{.Elems}}">
      <div class="head">
        <span class="named">
          <span class="name">{{.Entry.Name}}</span>
          {{if .Entry.Title}}<span class="title">{{.Entry.Title}}</span>{{end}}
        </span>
        <span class="record">{{.Entry.Record}}</span>
        {{if .Entry.Family}}<span class="family">{{.Entry.Family}}</span>{{end}}
        <span class="stats">
          <b>HP {{.Entry.HP}}</b><span>DMG {{.Entry.DMG}}</span><span>AP {{.Entry.Actions}}</span>
          <span>{{.Deck}}-card deck</span>
        </span>
      </div>

      <div class="strip">
        <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}"
             alt="{{.Entry.Name}} and its deck">
      </div>

      <table>
        <tr><th>Card</th><th>Verb</th><th>Effect</th><th>AP</th><th>Copies</th></tr>
        {{range .Rows}}
          <tr>
            <td>{{.Label}}</td>
            <td>{{.Verb}}</td>
            <td class="effect">{{.Effect}}</td>
            <td class="num">{{.Cost}}</td>
            <td class="num">{{.Copies}}</td>
          </tr>
        {{end}}
      </table>

      {{if .Affixes}}<p class="affix">Dealt as: {{.Affixes}}</p>{{end}}

      {{if .Others.File}}
      <details class="others">
        <summary>Show its other {{len .OtherLabels}} colours</summary>
        <div class="strip">
          <div>
            <img src="{{.Others.File}}" width="{{.Others.Width}}" height="{{.Others.Height}}"
                 alt="{{.Entry.Name}} in its other colours">
            <div class="elabels">
              {{range .OtherLabels}}<span>{{.}}</span>{{end}}
            </div>
          </div>
        </div>
      </details>
      {{end}}
      {{if .Entry.Draw}}<p class="draw">{{.Entry.Draw}}</p>{{end}}
    </div>
  {{end}}
  </details>
</section>
{{end}}
`))
