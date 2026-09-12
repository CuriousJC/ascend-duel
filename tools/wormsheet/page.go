package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit worms.json, re-run
// the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// relic sheet's template gives: a card's rim is one pixel thick and a browser that scales it
// resamples that rim into a blur, which makes the sheet lie about the art.
//
// **The ground is the one the worms actually sit on.** A red border on white is a different
// card from a red border on the game's own blue.
var tmpl = template.Must(template.New("wormsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — worm sheet</title>
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
  .text { margin: 10px 0 0; font-size: 13px; }
  .rule {
    margin: 10px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim);
  }
  .border { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
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
  .cells { display: flex; flex-wrap: wrap; gap: 20px; margin-top: 22px; }
  figure { margin: 0; }
  figcaption { color: var(--dim); font-size: 11.5px; margin-top: 7px; max-width: 180px; }
</style>

<h1>Worm sheet</h1>
<p class="facts">
  {{.Count}} worms, {{.Undrawn}} of them drawing the default face, {{.Unwritten}} with no subject
  paragraph written. {{.Offered}} are offered after a won fight — {{.Share}}% of the catalogue gets
  a seat. Worm card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  art box inset <code>{{index .Style "artInset"}}</code> from
  <code>y={{index .Style "artTop"}}</code> at most <code>{{index .Style "artMaxH"}}</code> tall,
  text band from <code>y={{index .Style "textBandTop"}}</code>.
  Shown at 1:1 on the ground the worms are actually offered on.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/wormsheet</code> and refresh. Every card here is drawn by
  <code>internal/cards</code>, the same code the game blits, and every word beside it is read out
  of <code>data/worms.json</code> through <code>internal/session</code>'s own validation — so a
  worm this page refuses to draw is a worm the game refuses to start with.
</p>
<p class="note">
  <strong>Read the sentence against the rule.</strong> The line under each name is the
  <code>Text</code> field, which is what the card prints verbatim; the monospace line under it is
  the rule that actually fires. Nothing in the codebase checks one against the other.
</p>
<p class="note">
  <strong>The subject paragraph is the art brief, and it lives on the record.</strong> The quoted
  block under each worm is <code>Draw</code> in <code>data/worms.json</code>: what the thing
  <em>is</em> and what it is doing, in one sentence. Nothing in the game reads it. It is pasted
  under the shared prompt in <code>docs/art/card_art_prompt.MD</code>, which is the only part of a
  brief that is not about one record. <strong>A worm with no subject and no art is the
  backlog</strong> — both lines go pink, so the page can be scrolled for what still needs writing
  rather than a worklist being kept in step by hand. <code>default-worm.png</code> is the seat art
  goes into, not a fallback that has gone wrong.
</p>

<h2>What the catalogue changes</h2>
<p class="note">
  <strong>Every target, and how many worms sit at it.</strong> The target vocabulary is closed —
  a new one is a Go change plus one place applying it, never something a file can assert into
  existence — so a target with nothing under it is a mechanic that was built and never reached for.
  This was the page's grouping until families landed; it is a count now, because the target is a
  fact about one worm and a family is a block of them.
</p>
<ul class="targets">
{{range .Targets}}
  <li class="target{{if not .Count}} empty{{end}}"><strong>{{.Target}}</strong>
    {{.Count}} worms{{if not .Count}} — nobody has authored one{{end}}</li>
{{end}}
</ul>

<h2>The catalogue, by family</h2>
<p class="note">
  <strong>Grouped by the motif each worm was authored beside, in the file's own order.</strong>
  <code>Family</code> is authored and the engine ignores it, exactly as it ignores <code>Art</code>
  and <code>Draw</code> — so it can go quietly out of date when a worm is retargeted, and nothing
  fails. Treat a family that disagrees with the rule beside it as a label to fix.
</p>

{{range .Families}}
<h3 class="family">
  {{.Name}}
  <span>{{.Count}} {{.Noun}}</span>
</h3>
<div class="plates">
  {{range .Worms}}
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
        <p class="border">border: {{.Element}}</p>
        {{if .Default}}
          <p class="art missing">no art of its own — drawing default-worm.png</p>
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
  The two states a worm card is drawn in. The reward screen dims the offer that was not taken
  rather than lighting the one that was, so there is no third.
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
