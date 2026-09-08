package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "change a tint mode, re-run
// the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size first with image-rendering: pixelated.** A card's rim is
// one pixel thick and a browser that scales it resamples that rim into a blur, which makes the
// sheet lie about the art. The enlarged row is an honest nearest-neighbour blow-up of the same file
// rather than a second rendering — a kinder render at 3x would be a picture of a card the game
// never draws.
//
// **The actual-size row comes first on purpose**, which is the glyph sheet's rule: reviewing only
// the enlarged row is how a mark comes to look acceptable in review and clunky in play.
//
// **The ground is the one the cards actually sit on.** A rainbow column on white is a different
// column from a rainbow column on the game's own blue.
var tmpl = template.Must(template.New("upgradesheet").Funcs(funcs).Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — upgrade sheet</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #5a6472;
    --rule: #8fa3bd;
    --panel: #c6d5e6;
    --pink: #c8508c;
    --alarm: #b03030;
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
    font-size: 16px; font-weight: 600; text-transform: capitalize;
    margin: 38px 0 0; padding-bottom: 7px; border-bottom: 2px solid var(--rule);
  }
  h4 {
    font-size: 13px; font-weight: 600; margin: 26px 0 0;
    display: flex; align-items: baseline; gap: 10px;
  }
  h4 code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
  .facts { color: var(--dim); font-size: 12px; margin: 0 0 8px; }
  .facts code { color: var(--ink); }
  .note { color: var(--dim); font-size: 12.5px; max-width: 74ch; margin: 12px 0 0; }
  .tag {
    font-size: 10.5px; text-transform: uppercase; letter-spacing: .08em;
    background: var(--pink); color: #fff; padding: 2px 7px; border-radius: 10px;
    font-weight: 600;
  }
  .alarm { color: var(--alarm); font-weight: 600; }
  .row { display: flex; flex-wrap: wrap; gap: 18px; margin-top: 14px; align-items: flex-start; }
  .cardbox {
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px; padding: 12px;
  }
  .cardbox .cap {
    color: var(--dim); font-size: 11px; margin-top: 8px; text-align: center;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  }
  /* Natural size, never scaled. See the comment above. */
  img { display: block; image-rendering: pixelated; }
  img.big { width: {{mul (index .Style "width") .Zoom}}px; height: auto; }
  .grants { margin: 8px 0 0; font-size: 13px; }
  .grants code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
</style>

<h1>Upgrade sheet</h1>
<p class="facts">
  Every <em>visible upgrade</em> a card can carry, drawn on all four form marks in all
  {{len .Plates}} × 3 combinations of ink and mark. The card is
  <code>cards.Hand</code>, {{index .Style "width"}}×{{index .Style "height"}}; the form mark's box is
  <code>{{index .Style "formSize"}}px</code> at ({{index .Style "dashLeft"}},{{index .Style "formTop"}});
  the ticks are {{index .Style "dashWidth"}}×{{index .Style "dashHeight"}}. Every ink is authored at
  {{.InkSize}}×{{.InkSize}}. The game draws <code>{{.Default}}</code>.
</p>
<p class="note">
  An upgrade takes the card's <strong>left column</strong> — the form mark and the cost ticks under
  it — and paints it from a picture instead of from the element's one colour. That is the column
  that already states the element, so an upgrade is not a fourth thing on a card with three: it is
  the same statement about a card whose answer has changed. Everything else on the face is
  untouched, and <code>Spec.Element</code> is still what the card <em>is</em>.
</p>

<h2>The plain card</h2>
<p class="note">
  What the player has fifty-five of. Every judgement below is a comparison against this row, so it
  is here rather than at the bottom.
</p>
<div class="row">
  {{range .Plain}}
  <div class="cardbox">
    <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
    <div class="cap">{{.Note}}</div>
  </div>
  {{end}}
</div>

{{range .Plates}}
<h3 class="group">{{.Upgrade}}</h3>
<p class="grants">
  Rider: <code>{{.Riders}}</code>.
  {{if .Grants}}Granted by {{.Grants}}.{{else}}<span class="alarm">Nothing in data/parasites.json grants it — this upgrade cannot be acquired.</span>{{end}}
</p>

{{range .Modes}}
<h4><code>{{.Mode}}</code>{{if .Default}}<span class="tag">the game draws this</span>{{end}}</h4>
<div class="row">
  {{range .Cells}}
  <div class="cardbox">
    <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
    <div class="cap">{{.Note}}</div>
  </div>
  {{end}}
</div>
{{end}}

<h4>enlarged, {{$.Zoom}}×</h4>
<p class="note">
  The same files, blown up nearest-neighbour. Read it <em>after</em> the rows above, never instead
  of them.
</p>
{{range .Modes}}
<div class="row">
  {{range .Cells}}
  <div class="cardbox">
    <img class="big" src="{{.File}}" alt="{{.Label}}">
    <div class="cap">{{$.Zoom}}× · {{.Note}}</div>
  </div>
  {{end}}
</div>
{{end}}

<h4>states</h4>
<p class="note">
  The mark and the ticks are one statement and share one state switch, so a card that cannot be
  afforded fades both together and a queued one lights both together. A row where they disagree is
  a bug in <code>Spec.atState</code>, not a matter of taste.
</p>
<div class="row">
  {{range .States}}
  <div class="cardbox">
    <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
    <div class="cap">{{.Label}}</div>
  </div>
  {{end}}
</div>
{{end}}
`))

// funcs is the one arithmetic the template needs: the enlarged row's pixel width. It is a template
// function rather than a field on the page so the markup can state the zoom factor beside the
// number it produced, instead of quoting a width whose origin is invisible.
var funcs = template.FuncMap{
	"mul": func(a, b int) int { return a * b },
}
