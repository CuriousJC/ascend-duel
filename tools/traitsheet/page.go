package main

import "html/template"

// The page. One static file, no JavaScript, no build step — the tonic sheet's shape.
var tmpl = template.Must(template.New("traitsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Duello — trait sheet</title>
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
  .note { color: var(--dim); font-size: 12.5px; max-width: 72ch; margin: 12px 0 0; }
  .plates { display: flex; flex-wrap: wrap; gap: 24px; margin-top: 22px; }
  .plate {
    display: flex; gap: 18px; align-items: flex-start;
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 16px; width: 560px;
  }
  img { display: block; image-rendering: pixelated; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .text { margin: 10px 0 0; font-size: 13px; }
  ul { padding: 0 0 0 16px; margin: 8px 0 0; font-size: 12.5px; }
  .does { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11.5px; color: var(--dim); }
  .art { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
  .art.missing { color: var(--pink); }
  .draw {
    margin: 10px 0 0; font-size: 12.5px; color: var(--dim);
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
  .draw.missing { color: var(--pink); border-left-color: var(--pink); }
  .pair { display: flex; gap: 24px; margin-top: 22px; }
  .pair figure { margin: 0; }
  .pair figcaption { font-size: 12px; color: var(--dim); margin-top: 6px; }
</style>

<h1>Trait sheet</h1>
<p class="facts">{{len .Relic}} relic traits from <code>data/relic_traits.json</code> ·
{{len .Card}} card traits from <code>data/card_traits.json</code> ·
{{.ShelfAnyText}} of shelf relics carry at least one trait ·
{{.Undrawn}} with no art · {{.Unwritten}} with no brief</p>
<p class="note">A trait is what a wearing or a card is besides itself, and it composes with
everything else — see MECHANICS.md §Traits. The odds are per shelf seat and per bundled card;
each plate also says how often a player actually meets one.</p>

{{define "plate"}}
<div class="plate">
  <img src="{{.Image}}" width="120" alt="{{.Name}}">
  <div>
    <p class="name">{{.Name}}</p>
    <p class="record">{{.Kind}} trait · {{.Key}}</p>
    <p class="text">{{.Text}}</p>
    <ul>{{range .Odds}}<li>{{.}}</li>{{end}}</ul>
    <ul class="does">{{range .Does}}<li>{{.}}</li>{{end}}</ul>
    {{if .NoArt}}<p class="art missing">no art — draws the placeholder</p>{{else}}<p class="art">art: {{.Art}}</p>{{end}}
    {{if .NoDraw}}<p class="draw missing">no brief written</p>{{else}}<p class="draw">{{.Draw}}</p>{{end}}
  </div>
</div>
{{end}}

<h2>Relic traits</h2>
<div class="plates">{{range .Relic}}{{template "plate" .}}{{end}}</div>

<h2>Card traits</h2>
<div class="plates">{{range .Card}}{{template "plate" .}}{{end}}</div>

<h2>A mirrored card</h2>
<p class="note">The echo is drawn into the face, so this is the card as the hand deals it. An
effortless card is drawn aloft — lifted, drifting, shimmering — which is motion and is looked at in
the game, not here.</p>
<div class="pair">{{range .Mirror}}<figure><img src="{{.Image}}" alt="{{.Label}}"><figcaption>{{.Label}}</figcaption></figure>{{end}}</div>
`))
