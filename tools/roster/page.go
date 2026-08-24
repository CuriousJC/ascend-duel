package roster

import "html/template"

// The page and the types it walks.
//
// One static file, no JavaScript, no build step: the loop is "edit enemies.json, re-run the tool,
// refresh the tab", the same loop every other sheet here has.
//
// **Grouped by floor rather than listed alphabetically**, for the reason the ring sheet groups by
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
}

// group is one floor's worth of the catalogue, with the band's own spread beside it.
//
// **The spread is what makes the grouping worth having.** A band whose HP runs 120 to 900 is not a
// band, it is two; and that is invisible while reading cards one at a time.
type group struct {
	Label  string
	Order  int
	Plates []plate

	MinHP, MaxHP   int
	MinDMG, MaxDMG int
	MinAP, MaxAP   int
}

type page struct {
	Ground     string
	Title      string
	Blurb      string
	GroupLabel string
	Count      int
	Style      map[string]int
	Groups     []group
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
			Label: pl.Entry.Floors,
			Order: pl.Entry.Group,
			MinHP: pl.Entry.HP, MaxHP: pl.Entry.HP,
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
// **The ground is the cream the cards actually sit on**, not a page colour chosen to flatter them.
var tmpl = template.Must(template.New("roster").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — {{.Title}}</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #6b6355;
    --rule: #c4b294;
    --panel: #ede0c8;
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

{{range .Groups}}
  <h2 class="floor">
    {{.Label}}
    <span>{{len .Plates}} records · HP {{.MinHP}}–{{.MaxHP}} · DMG {{.MinDMG}}–{{.MaxDMG}} ·
      AP {{.MinAP}}–{{.MaxAP}}</span>
  </h2>

  {{range .Plates}}
    <div class="plate">
      <div class="head">
        <span class="named">
          <span class="name">{{.Entry.Name}}</span>
          {{if .Entry.Title}}<span class="title">{{.Entry.Title}}</span>{{end}}
        </span>
        <span class="record">{{.Entry.Record}}</span>
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

      {{if .Affixes}}<p class="affix">Affixes it may be themed with: {{.Affixes}}</p>{{end}}
    </div>
  {{end}}
{{end}}
`))
