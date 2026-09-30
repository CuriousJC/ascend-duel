package main

import "html/template"

// The page. One static file, no JavaScript, no build step — the cantrip sheet's shape.
var tmpl = template.Must(template.New("tonicsheet").Funcs(template.FuncMap{
	"inc": func(i int) int { return i + 1 },
}).Parse(`<!doctype html>
<meta charset="utf-8">
<title>Duello — tonic sheet</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #5a6472;
    --rule: #8fa3bd;
    --panel: #c6d5e6;
    --pink: #c8508c;
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
  h2 {
    font-size: 13px; text-transform: uppercase; letter-spacing: .09em;
    color: var(--dim); font-weight: 600;
    margin: 44px 0 0; padding-bottom: 8px; border-bottom: 1px solid var(--rule);
  }
  .facts { color: var(--dim); font-size: 12px; margin: 0 0 8px; }
  .facts code { color: var(--ink); }
  .note { color: var(--dim); font-size: 12.5px; max-width: 68ch; margin: 12px 0 0; }
  .plates { display: flex; flex-wrap: wrap; gap: 24px; margin-top: 22px; }
  .plate {
    display: flex; gap: 18px; align-items: flex-start;
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 16px; width: 520px;
  }
  img { display: block; image-rendering: pixelated; }
  .about { min-width: 0; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .tip { margin: 2px 0 0; font-size: 13px; }
  .tip:first-of-type { margin-top: 10px; }
  .rule {
    margin: 10px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim);
  }
  .art { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
  .art.missing { color: var(--pink); }
  .draw {
    margin: 10px 0 0; font-size: 12.5px; color: var(--dim);
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
  .draw.missing { color: var(--pink); border-left-color: var(--pink); }
  ul.effects { list-style: none; padding: 0; margin: 14px 0 0; }
  li.effect {
    font-size: 12.5px; color: var(--dim); padding: 5px 0 5px 10px;
    border-left: 3px solid var(--rule); margin-bottom: 3px;
  }
  li.effect strong { color: var(--ink); margin-right: 6px; font-family: ui-monospace, monospace; }
  li.effect.empty { opacity: .62; }
  table { border-collapse: collapse; margin-top: 16px; font-size: 12.5px; }
  th, td { text-align: left; padding: 5px 14px 5px 0; border-bottom: 1px solid var(--rule); }
  th { color: var(--dim); font-weight: 600; }
  td.code { font-family: ui-monospace, monospace; }
</style>

<h1>Tonic sheet</h1>
<p class="facts">
  {{.Count}} tonics, {{.Undrawn}} of them drawing the default face, {{.Unwritten}} with no subject
  paragraph written. One is offered per realm across {{.Realms}} realms. Shown at 1:1.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/tonicsheet</code> and refresh. Every card is drawn by
  <code>internal/cards</code>, the code the game blits, and every word beside it is read out of
  <code>data/tonics.json</code> through <code>internal/session</code>'s own validation.
</p>
<p class="note">
  <strong>A tonic changes the run's rules for the rest of the run, and is offered once.</strong>
  The run code shuffles the catalog; each realm offers the first tonic in that order the run may
  still have. Bought or passed over, it is never offered again; one whose requirement has not been
  drunk is skipped and comes back later. See MECHANICS.md §Tonics.
</p>
<p class="note">
  <strong>The subject paragraph is the art brief.</strong> The quoted block is <code>Draw</code>;
  nothing in the game reads it. A tonic with no subject and no art is the backlog — both lines go
  pink.
</p>

<h2>What the catalog does</h2>
<ul class="effects">
{{range .Effects}}
  <li class="effect{{if not .Count}} empty{{end}}"><strong>{{.Name}}</strong>
    {{.Count}} tonics{{if not .Count}} — nobody has authored one{{end}}</li>
{{end}}
</ul>

<h2>The catalog</h2>
<div class="plates">
  {{range .Plates}}
    <div class="plate">
      <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}" alt="{{.Name}}">
      <div class="about">
        <p class="name">{{.Name}}</p>
        <div class="record">{{.Record}}</div>
        {{range .Tip}}<p class="tip">{{.}}</p>{{end}}
        <p class="rule">{{.Rule}}</p>
        {{if .Draw}}
          <p class="draw">{{.Draw}}</p>
        {{else}}
          <p class="draw missing">no subject written yet</p>
        {{end}}
        {{if .Default}}
          <p class="art missing">no art of its own — drawing the default face</p>
        {{else}}
          <p class="art">art: <code>{{.Art}}</code></p>
        {{end}}
      </div>
    </div>
  {{end}}
</div>

<h2>What a discount charges</h2>
<p class="note">
  The share of the list price, rounded up so it is always at least one vitae, and never below a
  price of one — <code>session.DiscountCut</code>, the arithmetic the shop charges with.
</p>
<table>
  <tr><th>tonic</th><th>item</th><th>list</th><th>paid</th></tr>
  {{range .Prices}}
  <tr><td>{{.Tonic}}</td><td>{{.What}}</td><td>{{.List}}</td><td>{{.Paid}}</td></tr>
  {{end}}
</table>

<h2>What a run is offered</h2>
<p class="note">
  Each sample run code walked through every fight, through the shop's own <code>OfferTonic</code>
  and <code>DrinkTonic</code>: once buying every tonic offered, once buying none. A dash is a realm
  with nothing left to offer. Buying nothing is what holds a tonic with a requirement back for good.
</p>
<table>
  <tr><th>run code</th><th>buys</th>{{range $i, $_ := (index .Walks 0).BuyAll}}<th>realm {{inc $i}}</th>{{end}}</tr>
  {{range .Walks}}
  <tr><td class="code">{{.Code}}</td><td>every one</td>{{range .BuyAll}}<td>{{.}}</td>{{end}}</tr>
  <tr><td class="code">{{.Code}}</td><td>none</td>{{range .BuyNone}}<td>{{.}}</td>{{end}}</tr>
  {{end}}
</table>
`))
