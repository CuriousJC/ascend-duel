package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit parasites.json,
// re-run the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// relic sheet's template gives: a card's rim is one pixel thick and a browser that scales it
// resamples that rim into a blur, which makes the sheet lie about the art.
//
// **The ground is the one the parasites actually sit on**, which for these is the panel over a
// live fight rather than a shop shelf — the bucket opens between turns, which is the one thing
// about this catalogue the page has to say in words because no card can show it.
var tmpl = template.Must(template.New("parasitesheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — parasite sheet</title>
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
  h3.group {
    font-size: 15px; font-weight: 600; text-transform: capitalize;
    margin: 34px 0 0; padding-bottom: 7px; border-bottom: 2px solid var(--rule);
  }
  h3.group span {
    text-transform: none; font-weight: 400; font-size: 12px; color: var(--dim);
    margin-left: 10px;
  }
  .facts { color: var(--dim); font-size: 12px; margin: 0 0 8px; }
  .facts code { color: var(--ink); }
  .note { color: var(--dim); font-size: 12.5px; max-width: 68ch; margin: 12px 0 0; }
  .plates { display: flex; flex-wrap: wrap; gap: 24px; margin-top: 22px; }
  .plate {
    display: flex; gap: 18px; align-items: flex-start;
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 16px; width: 480px;
  }
  /* Natural size, never scaled. See the comment above. */
  img { display: block; image-rendering: pixelated; }
  .about { min-width: 0; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .text { margin: 10px 0 0; font-size: 13px; white-space: pre-line; }
  .rule {
    margin: 10px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim);
  }
  .cards { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
  .art { margin: 6px 0 0; font-size: 11.5px; color: var(--dim); }
  .art.missing { color: var(--pink); }
  /* The subject paragraph is an *input* to the art generator, so it is set apart from the
     authored line a player reads: indented, quieted, and marked when nobody has written one
     yet. Same pink the missing-art line takes, because they are one backlog. */
  .draw {
    margin: 10px 0 0; font-size: 12.5px; color: var(--dim);
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
  .draw.missing { color: var(--pink); border-left-color: var(--pink); }
  h3.family {
    font-size: 15px; font-weight: 600;
    margin: 34px 0 0; padding-bottom: 7px; border-bottom: 2px solid var(--rule);
  }
  h3.family span {
    font-weight: 400; font-size: 12px; color: var(--dim); margin-left: 10px;
  }
  /* The target counts, which used to be the page's headings and are now a list at the top. */
  ul.targets { list-style: none; padding: 0; margin: 14px 0 0; }
  li.target {
    font-size: 12.5px; color: var(--dim); padding: 5px 0 5px 10px;
    border-left: 3px solid var(--rule); margin-bottom: 3px;
  }
  li.target strong { color: var(--ink); text-transform: capitalize; margin-right: 6px; }
  li.target.empty { opacity: .62; }
  figure { margin: 0; }
  figcaption { color: var(--dim); font-size: 11.5px; margin-top: 7px; max-width: 180px; }
</style>

<h1>Parasite sheet</h1>
<p class="facts">
  {{.Count}} parasites, {{.Undrawn}} of them drawing the default face, {{.Unwritten}} with no
  subject paragraph written. A sealed bucket costs <code>{{.BucketPrice}}</code> vitae and draws
  <code>{{.BucketSize}}</code>, keep one — {{.Share}}% of the catalogue gets a seat. One parasite
  may name at most <code>{{.MaxTargets}}</code> cards.
  Card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  art box inset <code>{{index .Style "artInset"}}</code> from
  <code>y={{index .Style "artTop"}}</code> at most <code>{{index .Style "artMaxH"}}</code> tall,
  text band from <code>y={{index .Style "textBandTop"}}</code>.
  Shown at 1:1.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/parasitesheet</code> and refresh. Every card here is drawn
  by <code>internal/cards</code>, the same code the game blits, and every word beside it is read
  out of <code>data/parasites.json</code> through <code>internal/session</code>'s own validation —
  so a parasite this page refuses to draw is a parasite the game refuses to start with.
</p>
<p class="note">
  <strong>Read the sentence against the rule.</strong> The line under each name is the
  <code>Text</code> field, which is what the card prints verbatim; the monospace line under it is
  the rule that actually fires. Nothing in the codebase checks one against the other, and this is
  the least readable record in <code>data/</code> — which of <code>Rider</code>,
  <code>Value</code> and <code>Count</code> the rules read depends entirely on the target.
</p>
<p class="note">
  <strong>They are spent between the turns of a fight, never inside one.</strong> That is the
  thing about this catalogue no card can show: the bucket is gated on planning, because
  <code>ResolveRound</code> decides a whole round before playback starts and a card altered
  mid-playback would show a face disagreeing with a blow already computed.
</p>
<p class="note">
  <strong>The subject paragraph is the art brief, and it lives on the record.</strong> The quoted
  block under each parasite is <code>Draw</code> in <code>data/parasites.json</code>: what the
  thing <em>is</em> and what it is doing, in one sentence. Nothing in the game reads it. It is
  pasted under the shared prompt in <code>docs/art/card_art_prompt.MD</code>, which is the only
  part of a brief that is not about one record. <strong>A parasite with no subject and no art is
  the backlog</strong> — both lines go pink, so the page can be scrolled for what still needs
  writing. <code>default-parasite.png</code> is the seat art goes into; it was the worm's own
  placeholder until 2026-09-12, and the two split because one shared picture is a page where a
  drawn worm and an undrawn parasite look identical.
</p>

<h2>What the catalogue does</h2>
<p class="note">
  <strong>Every target, and how many parasites sit at it.</strong> The vocabulary is closed — a new
  target is a Go change plus one place applying it, never something a file can assert into
  existence — so a target with nothing under it is a mechanic built and never reached for. This was
  the page's grouping until families landed, and it made a poor heading once a third of the
  catalogue was a single <code>swap</code>: the motif is what tells one swap from another.
</p>
<ul class="targets">
{{range .Targets}}
  <li class="target{{if not .Count}} empty{{end}}"><strong>{{.Target}}</strong>
    {{.Count}} parasites{{if not .Count}} — nobody has authored one{{end}}</li>
{{end}}
</ul>

<h2>The catalogue, by family</h2>
<p class="note">
  <strong>Grouped by the motif each parasite was authored beside, in the file's own order.</strong>
  <code>Family</code> is authored and the engine ignores it, exactly as it ignores <code>Art</code>
  and <code>Draw</code> — so it can go quietly out of date when a record is retargeted, and nothing
  fails. Treat a family that disagrees with the rule beside it as a label to fix.
</p>

{{range .Families}}
<h3 class="family">
  {{.Name}}
  <span>{{.Count}} {{.Noun}}</span>
</h3>
<div class="plates">
  {{range .Parasites}}
    <div class="plate">
      <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}"
           alt="{{.Name}}">
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
        <p class="cards">cards asked for: {{.Cards}}</p>
        {{if .Default}}
          <p class="art missing">no art of its own — drawing default-parasite.png</p>
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
  The three states a parasite card is drawn in — one more than a worm has.
  <strong>Selected is a state here and is not one there</strong>: a parasite is armed first and
  aimed second, so the board piece has to say which one is in hand while the player picks what it
  eats.
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
