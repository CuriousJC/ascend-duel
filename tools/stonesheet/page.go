package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit stones.json, re-run
// the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// ring sheet's template gives: a card's rim is one pixel thick and a browser that scales it
// resamples that rim into a blur, which makes the sheet lie about the art.
//
// **The ground is the cream the stones actually sit on.** A grey border on white is a different
// card from a grey border on cream.
//
// **A rung with no stone is still a row**, drawn as an empty seat rather than skipped. That is the
// one layout decision this template makes that the worm sheet's does not, and it is why the page
// is worth having: the gap is the finding.
var tmpl = template.Must(template.New("stonesheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — stone sheet</title>
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
  /* An unstoned rung keeps the card's footprint, so the ladder stays a ladder. */
  .empty {
    width: {{index .Style "width"}}px; height: {{index .Style "height"}}px;
    border: 2px dashed var(--rule); border-radius: {{index .Style "cornerRadius"}}px;
    display: flex; align-items: center; justify-content: center;
    color: var(--dim); font-size: 12px; text-align: center; padding: 12px;
    flex: none;
  }
  .about { min-width: 0; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .text { margin: 10px 0 0; font-size: 13px; white-space: pre-line; }
  .rule {
    margin: 10px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim);
  }
  .worth { margin: 10px 0 0; font-size: 13px; }
  .worth b { font-variant-numeric: tabular-nums; }
  .art { margin: 6px 0 0; font-size: 11.5px; color: var(--pink); }
</style>

<h1>Stone sheet</h1>
<p class="facts">
  {{.Count}} stones over {{.Rungs}} rungs{{if .Unstoned}} — {{.Unstoned}} rung(s) have none{{end}}.
  A sealed bag costs <code>{{.BagPrice}}</code> vitae and draws <code>{{.BagSize}}</code>, keep
  one — {{.Share}}% of the catalogue gets a seat.
  Stone card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  art box inset <code>{{index .Style "artInset"}}</code> from
  <code>y={{index .Style "artTop"}}</code> at most <code>{{index .Style "artMaxH"}}</code> tall,
  text band from <code>y={{index .Style "textBandTop"}}</code>.
  Shown at 1:1 on the cream the stones are actually offered on.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/stonesheet</code> and refresh. Every card here is drawn by
  <code>internal/cards</code>, the same code the game blits, and every word beside it is read out
  of <code>data/stones.json</code> through <code>internal/session</code>'s own validation — so a
  stone this page refuses to draw is a stone the game refuses to start with.
</p>
<p class="note">
  <strong>The +N is computed, not authored.</strong> A stone adds a tenth of its rung's catalogue
  multiplier — <code>combat.StoneValue</code> — so the figure on the face follows
  <code>hands.json</code> without anything being edited here. That is the split worth
  sanity-checking: the record carries the sentence, the rules carry the arithmetic.
</p>
<p class="note">
  <strong>Read the sentence against the key.</strong> The line under each name is the
  <code>Text</code> field, which is what the card prints verbatim; the monospace line under it is
  the rung it is actually keyed to. Nothing in the codebase checks one against the other, and a
  stone naming the wrong rung would be invisible everywhere but here.
</p>
<p class="note">
  <strong>Every stone draws the same boulder.</strong> It is a generated glyph rather than a file
  — <code>systems.GlyphStone</code> — so there is nothing to review about the picture and no
  provenance question to answer. What is being reviewed is the wording and the numbers.
</p>

<h2>The ladder, by axis</h2>
<p class="note">
  <strong>Grouped by what a rung counts on</strong>, and walked in the catalogue's own order
  inside each. This is deliberately not the hand sheet's layout: that one interleaves all three
  axes by ascending multiplier, because a player forming a hand chooses among all of them at once.
  A stone is bought against one rung, so the question here is whether an axis' ladder is priced
  sensibly against itself.
</p>

{{range .Groups}}
<h3 class="group">
  {{.Axis}}
  <span>{{len .Rungs}} rungs, {{.Stoned}} with a stone</span>
</h3>
<div class="plates">
  {{range .Rungs}}
    <div class="plate">
      {{if .Has}}
        <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}"
             alt="{{.Name}}">
      {{else}}
        <div class="empty">no stone raises this rung</div>
      {{end}}
      <div class="about">
        {{if .Has}}
          <p class="name">{{.Name}}</p>
          <div class="record">{{.Record}}</div>
          <p class="text">{{.Text}}</p>
          <p class="rule">raises {{.HandKey}}</p>
          <p class="worth">
            <b>{{.Multiplier}}</b> &rarr; <b>{{.Raised}}</b> with one stone
            (<b>+{{.Worth}}</b> each)
          </p>
          <p class="art">no art of its own — drawing the generated boulder</p>
        {{else}}
          <p class="name">{{.Hand}}</p>
          <div class="record">{{.HandKey}}</div>
          <p class="worth"><b>{{.Multiplier}}</b> — nothing can raise it</p>
          <p class="art">a rung the catalogue has not authored a stone for</p>
        {{end}}
        <p class="rule">{{.CardsWanted}} cards, counted on {{.Axis}}</p>
      </div>
    </div>
  {{end}}
</div>
{{end}}

<h2>Card states</h2>
<p class="note">
  The two states a stone card is drawn in. The bag's dialog dims the three rocks that were not
  kept rather than lighting the one that was, so there is no third.
</p>
<div class="plates">
  {{range .States}}
    <figure style="margin:0">
      <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
      <figcaption style="color:var(--dim);font-size:11.5px;margin-top:7px;max-width:180px">
        {{.Label}}
      </figcaption>
    </figure>
  {{end}}
</div>
`))
