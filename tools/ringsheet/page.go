package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit rings.json,
// re-run the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason
// the card sheet's template gives: a card's rim is one pixel thick and a browser that scales
// it — even by the fraction a max-width rule can introduce — resamples that rim into a blur
// and makes the sheet lie about the art.
//
// **The ground is the cream the rings actually sit on**, not a page colour chosen to flatter
// them. A pink border on white is a different card from a pink border on cream.
//
// The layout is a card beside a block of text rather than a grid of cards, because what is
// being reviewed here is not only the picture: it is the picture against the price, the
// authored line, and the rules — four things nothing else in the project shows together.
var tmpl = template.Must(template.New("ringsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — ring sheet</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #6b6355;
    --rule: #c4b294;
    --panel: #ede0c8;
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
  /* Natural size, never scaled. See the comment above. */
  img { display: block; image-rendering: pixelated; }
  .about { min-width: 0; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .price { margin: 10px 0 0; font-size: 12.5px; }
  .text { margin: 10px 0 0; font-size: 13px; }
  .rules { margin: 10px 0 0; padding: 0; list-style: none; }
  .rules li {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim); margin-top: 3px;
  }
  .art { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
  .art.missing { color: var(--pink); }
  .cells { display: flex; flex-wrap: wrap; gap: 20px; margin-top: 22px; }
  figure { margin: 0; }
  figcaption { color: var(--dim); font-size: 11.5px; margin-top: 7px; max-width: 180px; }
</style>

<h1>Ring sheet</h1>
<p class="facts">
  {{.Count}} rings, {{.Undrawn}} of them drawing the default face.
  Ring card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  art box inset <code>{{index .Style "artInset"}}</code> from
  <code>y={{index .Style "artTop"}}</code>, at most
  <code>{{index .Style "artMaxH"}}</code> tall.
  Shown at 1:1 on the cream the rings are actually drawn on.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/ringsheet</code> and refresh. Every card here is
  drawn by <code>internal/cards</code>, the same code the game blits, and every word beside
  it is read out of <code>data/rings.json</code> through the same registration the game
  runs at start-up — so a ring this page refuses to draw is a ring the game refuses to start
  with.
</p>
<p class="note">
  <strong>Read the sentence against the rules.</strong> The line under each name is the
  <code>Text</code> field, which is what the hover tooltip prints verbatim; the monospace
  lines under it are the rules that actually fire. Nothing in the codebase checks one
  against the other, so a rule edited without its sentence is a ring that lies to the
  player, and this is the only place the two are visible together.
</p>

<h2>The catalogue</h2>
<div class="plates">
  {{range .Rings}}
    <div class="plate">
      <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}"
           alt="{{.Name}}">
      <div class="about">
        <p class="name">{{.Name}}</p>
        <div class="record">{{.Record}}</div>
        <p class="price">{{.Price}} vitae, sells back for {{.Sell}}</p>
        <p class="text">{{.Text}}</p>
        <ul class="rules">
          {{range .Rules}}<li>{{.}}</li>{{end}}
        </ul>
        {{if .Default}}
          <p class="art missing">no art of its own — drawing default-ring.png</p>
        {{else}}
          <p class="art">art: <code>{{.Art}}</code></p>
        {{end}}
      </div>
    </div>
  {{end}}
</div>

<h2>Card states</h2>
<p class="note">
  The three states a ring card is drawn in. A ring the run neither owns nor has been offered
  is not on screen at all, so there is no fourth.
</p>
<div class="cells">
  {{range .States}}
    <figure>
      <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
      <figcaption>{{.Label}}</figcaption>
    </figure>
  {{end}}
</div>
`))
