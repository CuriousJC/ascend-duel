package main

import "html/template"

// The page. One static file and no build step: the loop is "edit relics.json, re-run the tool,
// refresh the tab", the same loop every other tool here has. The only script on it is the chip bar
// from tools/sheetfilter, which narrows what is already there — the whole catalog is in the file
// and readable with scripting off.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason
// the card sheet's template gives: a card's rim is one pixel thick and a browser that scales
// it — even by the fraction a max-width rule can introduce — resamples that rim into a blur
// and makes the sheet lie about the art.
//
// **That reason is about the card, not about the picture on it, and the two now want opposite
// things** *(2026-09-14)*. The rim, the counter disc and the type are rasterized at 1:1 by
// internal/cards and want nearest-neighbour; the art inside is smooth-shaded and would go blocky
// under it. Nothing has to choose, because the image is never scaled — which was an implied
// property and is now an asserted one: `flex: none` stops the plate's flex row from shrinking a
// card to make room for a long rules block, which is the one way a natural-size image on this
// page could still be resampled.
//
// **The ground is the one the relics actually sit on**, not a page color chosen to flatter
// them. A pink border on white is a different card from a pink border on the game's own blue.
//
// The layout is a card beside a block of text rather than a grid of cards, because what is
// being reviewed here is not only the picture: it is the picture against the price, the
// authored line, and the rules — four things nothing else in the project shows together.
var tmpl = template.Must(template.New("relicsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — {{.Title}}</title>
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
  h3.family span {
    font-weight: 400; font-size: 12px; color: var(--dim); margin-left: 10px;
  }
  /* The shelf is a list rather than three headings now that the page groups by family, so the
     tier color moves onto the row. Common is the ground itself; rare is the relic pink the
     game already spends on "a relic did this". */
  ul.shelf { list-style: none; padding: 0; margin: 14px 0 0; }
  li.tier {
    font-size: 12.5px; color: var(--dim); padding: 5px 0 5px 10px;
    border-left: 3px solid var(--rule); margin-bottom: 3px;
  }
  li.tier strong { color: var(--ink); text-transform: capitalize; margin-right: 6px; }
  li.tier.common { border-left-color: var(--rule); }
  li.tier.uncommon { border-left-color: #6c8fb5; }
  li.tier.rare { border-left-color: var(--pink); }
  .facts { color: var(--dim); font-size: 12px; margin: 0 0 8px; }
  .facts code { color: var(--ink); }
  .note { color: var(--dim); font-size: 12.5px; max-width: 68ch; margin: 12px 0 0; }
  .plates { display: flex; flex-wrap: wrap; gap: 24px; margin-top: 22px; }
  .plate {
    display: flex; gap: 18px; align-items: flex-start;
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 16px; width: 520px;
  }
  /* Natural size, never scaled — flex: none is what makes that true rather than likely.
     See the comment above. */
  img { display: block; image-rendering: pixelated; flex: none; }
  .about { min-width: 0; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .price { margin: 10px 0 0; font-size: 12.5px; }
  /* The tier, on the relic rather than only on the shelf list and the family heading. A
     family is nearly always one rarity throughout, which is exactly what makes a mixed one
     worth reading card by card — and the price beside it is the tier's price, so the two
     belong on one line. Same three colors the shelf list uses. */
  .rarity {
    display: inline-block; margin-right: 7px; padding: 1px 7px 2px;
    border-radius: 9px; font-size: 11px; letter-spacing: .05em;
    text-transform: uppercase; font-weight: 600;
    color: var(--ground); background: var(--rule);
  }
  .rarity.uncommon { background: #6c8fb5; }
  .rarity.rare { background: var(--pink); }
  .text { margin: 10px 0 0; font-size: 13px; }
  .rules { margin: 10px 0 0; padding: 0; list-style: none; }
  .rules li {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim); margin-top: 3px;
  }
  .art { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
  /* The subject paragraph is an *input* to the art generator, so it is set apart from the
     authored sentence a player reads: indented, quieted, and marked when nobody has written
     one yet. Same pink the missing-art line takes, because they are one backlog. */
  .draw {
    margin: 10px 0 0; font-size: 12.5px; color: var(--dim);
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
  .draw.missing { color: var(--pink); border-left-color: var(--pink); }
  .art.missing { color: var(--pink); }
  .cells { display: flex; flex-wrap: wrap; gap: 20px; margin-top: 22px; }
  figure { margin: 0; }
  figcaption { color: var(--dim); font-size: 11.5px; margin-top: 7px; max-width: 180px; }
</style>

<h1>{{.Title}}</h1>
{{if .Archive}}
<p class="facts">
  {{.Count}} archived relics, {{.Undrawn}} of them drawing the default face,
  {{.Unwritten}} with no subject paragraph written, {{.Broken}} that would not load.
</p>
<p class="note">
  <strong>These relics are out of the game and kept in the repository.</strong> The records are
  <code>data/archive/relics.json</code> and the pictures are <code>assets/archive/relic/</code>;
  neither is embedded, so a release carries none of it and no shelf can offer one. Each is drawn
  exactly as the catalog's own sheet would draw it, family by family in the file's order.
</p>
<p class="note">
  <strong>Moving one back is moving the record and its picture</strong> — the record into
  <code>data/relics.json</code>, the picture into <code>assets/relic/</code> — and regenerating
  both pages. The archive is held to the relic grammar by <code>TestEveryArchivedRelicWouldLoad</code>,
  so a relic here loads the day it returns; one that has fallen out of step with the vocabulary
  carries the reason in pink on its plate.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/relicsheet -archive</code> and refresh.
</p>
{{else}}
<p class="facts">
  {{.Count}} relics, {{.Undrawn}} of them drawing the default face,
  {{.Unwritten}} with no subject paragraph written.
  Relic card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  art box inset <code>{{index .Style "artInset"}}</code> from
  <code>y={{index .Style "artTop"}}</code>, at most
  <code>{{index .Style "artMaxH"}}</code> tall.
  Accumulator badge <code>{{index .Style "counterSize"}}pt</code> on a
  <code>{{index .Style "counterDiameter"}}</code>-pixel disc,
  <code>{{index .Style "counterRight"}}</code> in from the right and
  <code>{{index .Style "counterBottom"}}</code> up from the bottom.
</p>
<p class="note">
  The badge is drawn at <strong>Grown&nbsp;0</strong> &mdash; what a fresh copy wears, which is the
  card the shelf shows. It is also the narrowest the figure gets: a run late in a climb reads
  <code>10.5</code> or <code>100</code>, so judge the size against those rather than against
  <code>1.0</code>. A relic with no badge is one that does not grow, which is most of the catalog.
  <strong>A decimal point means a multiplier and a bare number means a flat figure</strong>;
  there is no <code>x</code> after the multiplier and no <code>+</code> before the flat figure,
  because the point already says which is which, and a
  multiplier is <strong>always one decimal place</strong> — so four characters is the widest the
  figure ever gets. <strong>A wide figure is meant to outgrow the disc a little</strong>; what it
  may not do is leave the card.
  Shown at 1:1 on the ground the relics are actually drawn on.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/relicsheet</code> and refresh. Every card here is
  drawn by <code>internal/cards</code>, the same code the game blits, and every word beside
  it is read out of <code>data/relics.json</code> through the same registration the game
  runs at start-up — so a relic this page refuses to draw is a relic the game refuses to start
  with.
</p>
<p class="note">
  <strong>The subject paragraph is the art brief, and it lives on the record.</strong> The
  quoted block under each relic is <code>Draw</code> in <code>data/relics.json</code>: what the
  object <em>is</em> and what the effect is doing to it, in one sentence. Nothing in the game
  reads it. It is pasted under the catalog's own prompt,
  <code>docs/art/relic_art_prompt.MD</code>, which is the only part of a brief that is not
  about one relic. <strong>A relic with no
  subject and no art is the backlog</strong> — both lines go pink, so the page can be scrolled
  for what still needs writing rather than a worklist being kept in step by hand.
</p>
<p class="note">
  <strong>A relic out of the game is on its own page.</strong> The archive — records taken out of
  the catalog and kept in the repository — is <a href="../relicarchive/index.html">the relic
  archive</a>.
</p>
<p class="note">
  <strong>Read the sentence against the rules.</strong> The line under each name is the
  <code>Text</code> field, which is what the hover tooltip prints verbatim; the monospace
  lines under it are the rules that actually fire. Nothing in the codebase checks one
  against the other, so a rule edited without its sentence is a relic that lies to the
  player, and this is the only place the two are visible together.
</p>

<h2>The shelf</h2>
<p class="note">
  <strong>What a single shelf seat costs and how often it lands in each tier.</strong> Rarity is
  the only pricing dial there is — a relic is rebalanced by moving it between these three, never
  by writing a number — and the share is the tier's tickets over the whole catalog's.
</p>
<ul class="shelf">
{{range .Tiers}}
  <li class="tier {{.Rarity}}"><strong>{{.Rarity}}</strong>
    {{.Count}} relics &middot; {{.Price}} vitae, sells for {{.Sell}} &middot;
    weight {{.Weight}} each &middot; {{.Share}}% of a shelf draw</li>
{{end}}
</ul>
{{end}}

<h2>The catalog, by family</h2>
{{if not .Archive}}
<p class="note">
  <strong>Grouped by the motif each relic was authored beside, in the file's own order.</strong>
  The pricing review survives the move because nearly every family is one tier throughout: a
  heading reading <em>all common</em> asks "does one of these belong a tier up" of the whole block
  at once, and a family with a mix says so on its heading rather than hiding it.
</p>
{{end}}

{{.Filters}}

{{range .Families}}
<section class="sheet-group" data-family="{{.Key}}">
<h3 class="family">
  {{.Name}}
  <span>{{.Count}} {{.Noun}} &middot; {{.Mix}}</span>
</h3>
<div class="plates">
  {{range .Relics}}
    <div class="plate sheet-item" data-rarity="{{.Rarity}}">
      <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}"
           alt="{{.Name}}">
      <div class="about">
        <p class="name">{{.Name}}</p>
        <div class="record">{{.Record}}</div>
        {{if $.Archive}}
          <p class="price"><span class="rarity {{.Rarity}}">{{.Rarity}}</span></p>
        {{else}}
          <p class="price"><span class="rarity {{.Rarity}}">{{.Rarity}}</span>{{.Price}} vitae, sells back for {{.Sell}}</p>
        {{end}}
        {{if .Problem}}<p class="art missing">would not load: {{.Problem}}</p>{{end}}
        <p class="text">{{.Text}}</p>
        {{if .Draw}}
          <p class="draw">{{.Draw}}</p>
        {{else}}
          <p class="draw missing">no subject written yet</p>
        {{end}}
        {{if .Counter}}<p class="art">badge: <code>{{.Counter}}</code> at Grown 0</p>{{end}}
        <ul class="rules">
          {{range .Rules}}<li>{{.}}</li>{{end}}
        </ul>
        {{if .Default}}
          <p class="art missing">no art of its own — drawing default-relic.png</p>
        {{else}}
          <p class="art">art: <code>{{.Art}}</code></p>
        {{end}}
      </div>
    </div>
  {{end}}
</div>
</section>
{{end}}

{{if not .Archive}}
<h2>Card states</h2>
<p class="note">
  The three states a relic card is drawn in. A relic the run neither owns nor has been offered
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
{{end}}
`))
