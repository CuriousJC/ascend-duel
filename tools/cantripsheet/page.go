package main

import "html/template"

// The page. One static file, no JavaScript, no build step — the rune sheet's shape, and the same
// natural-size, pixelated images for the same reason: a browser scaling a one-pixel rim lies about
// the art.
var tmpl = template.Must(template.New("cantripsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Duello — cantrip sheet</title>
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
  h3.family {
    font-size: 15px; font-weight: 600;
    margin: 34px 0 0; padding-bottom: 7px; border-bottom: 2px solid var(--rule);
  }
  h3.family span { font-weight: 400; font-size: 12px; color: var(--dim); margin-left: 10px; }
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
  .text { margin: 10px 0 0; font-size: 13px; white-space: pre-line; }
  .rule, .example {
    margin: 10px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim);
  }
  .example { margin: 3px 0 0; color: var(--ink); }
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
  figure { margin: 0; }
  figcaption { color: var(--dim); font-size: 11.5px; margin-top: 7px; max-width: 180px; }
</style>

<h1>Cantrip sheet</h1>
<p class="facts">
  {{.Count}} cantrips, {{.Undrawn}} of them drawing the default face, {{.Unwritten}} with no
  subject paragraph written. A bundle of scrolls holds <code>{{.Bundles}}</code>, keep one,
  <strong>drawn with replacement</strong> — so a bundle can repeat. A run carries at most
  <code>{{.Cap}}</code> consumables of every kind together. Examples are cast onto the shipped
  duelist: <code>{{.Body}}</code>. Shown at 1:1.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/cantripsheet</code> and refresh. Every card is drawn by
  <code>internal/cards</code>, the code the game blits, and every word beside it is read out of
  <code>data/cantrips.json</code> through <code>internal/session</code>'s own validation.
</p>
<p class="note">
  <strong>Read the sentence against the rule, and the rule against the example.</strong> The line
  under each name is <code>Text</code>, the tooltip's line. The monospace rule under it is what the
  record resolves to; the lines under that are <code>Cantrip.Cast</code> run on the duelist at full
  life, at half, and twice in a row — the game's own arithmetic, since a cast reads the duelist the
  last one left.
</p>
<p class="note">
  <strong>A cantrip lasts the fight it is cast in.</strong> It is cast between turns, onto the
  fighter; nothing is written to the run. At a win the life it added comes off the ceiling and a
  life above the run's own ceiling comes back as the ceiling — see MECHANICS.md §Cantrips.
</p>
<p class="note">
  <strong>The subject paragraph is the art brief.</strong> The quoted block is <code>Draw</code>;
  nothing in the game reads it. A cantrip with no subject and no art is the backlog — both lines go
  pink. <code>default-cantrip.png</code> is the seat art goes into.
</p>

<h2>What the catalog does</h2>
<ul class="effects">
{{range .Effects}}
  <li class="effect{{if not .Count}} empty{{end}}"><strong>{{.Name}}</strong>
    {{.Count}} cantrips{{if not .Count}} — nobody has authored one{{end}}</li>
{{end}}
</ul>

<h2>The catalog, by family</h2>
{{range .Families}}
<h3 class="family">{{.Name}} <span>{{.Count}} {{.Noun}}</span></h3>
<div class="plates">
  {{range .Cantrips}}
    <div class="plate">
      <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}" alt="{{.Name}}">
      <div class="about">
        <p class="name">{{.Name}}</p>
        <div class="record">{{.Record}}</div>
        <p class="text">{{.Text}}</p>
        {{if .Draw}}
          <p class="draw">{{.Draw}}</p>
        {{else}}
          <p class="draw missing">no subject written yet</p>
        {{end}}
        <p class="rule">{{.Rule}}</p>
        {{range .Examples}}<p class="example">{{.}}</p>{{end}}
        {{if .Default}}
          <p class="art missing">no art of its own — drawing default-cantrip.png</p>
        {{else}}
          <p class="art">art: <code>{{.Art}}</code></p>
        {{end}}
      </div>
    </div>
  {{end}}
</div>
{{end}}

<h2>Card states</h2>
<p class="note">
  The two states a carried cantrip is drawn in. There is no selected state: a cantrip is aimed at
  nothing, so a click casts it.
</p>
<div class="plates">
  {{range .States}}
    <figure>
      <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
      <figcaption>{{.Label}}</figcaption>
    </figure>
  {{end}}
</div>
`))
