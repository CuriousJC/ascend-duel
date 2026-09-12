package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "change a style or a tint,
// re-run the tool, refresh the tab", the same loop every other tool here has.
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
// **The ground is the one the cards actually sit on.** A gold card on white is a different card
// from a gold card on the game's own blue.
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
  /* The tooltip as the game draws it: a titled box of short lines, dark on the game's own panel
     colours rather than on the page, so a line is judged against the ground it lands on. */
  .tip {
    display: inline-block; margin: 12px 0 0; padding: 9px 14px 11px;
    background: #f0eee8; border: 2px solid #8d8b84; border-radius: 7px;
    font: 12.5px/1.65 ui-monospace, SFMono-Regular, Menlo, monospace;
    letter-spacing: .04em; min-width: 190px;
  }
  .tip b { display: block; font-weight: 700; margin-bottom: 3px; }
  .grants code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
</style>

<h1>Upgrade sheet</h1>
<p class="facts">
  All {{len .Plates}} <em>visible upgrades</em> a card can carry, each drawn on all four form
  marks. The card is <code>cards.Hand</code>, {{index .Style "width"}}×{{index .Style "height"}};
  the form mark's box is <code>{{index .Style "formSize"}}px</code> at
  ({{index .Style "dashLeft"}},{{index .Style "formTop"}}); the ticks are
  {{index .Style "dashWidth"}}×{{index .Style "dashHeight"}}. Every ink is
  {{.InkSize}}×{{.InkSize}}; a washed card goes <code>{{.WashPct}}%</code> of the way toward it and a
  washed border <code>{{.BorderPct}}%</code>. The game draws <code>{{.Default}}</code>.
</p>
<p class="note">
  A card has a form, an element and an action — and then <strong>one upgrade</strong>. A second
  upgrade replaces the first outright. <code>Spec.Element</code> is still what the card <em>is</em>,
  and the left column still states it; the wildcard is the one upgrade that takes the hue off the
  form mark, because it is the one whose subject is that the card counts as every element at once.
</p>
<p class="note">
  <strong>Three styles, and the question is which one to keep.</strong> <code>border</code> paints
  the 3px ring and leaves the face alone — the border was freed up in August when the element moved
  off it, and what is left in that slot is the card's <em>state</em>, which is a wash away from
  neutral rather than a hue. <code>wash</code> takes the whole card, border included: nobody misses
  a gold card, and nothing on the face is unaffected. <code>wash-face</code> is the middle answer.
  Read them against the plain row above, and against a whole hand rather than one card — five loud
  cards in a row is the failure mode the border style exists to avoid.
</p>
<p class="note">
  <strong>Eight of these colours are placeholders.</strong> Hue is spent — five elements, the relic
  pink, the two verbs, the two duelists, the ground — so what is here is picked to be told apart
  rather than to mean anything. Gold and silver are the exception: they are metals, and they are
  what the mechanic is called. Retune the rest in <code>systems.upgradeTint</code> and re-run.
</p>

<p class="note">
  Each upgrade below carries the <strong>tooltip a card wearing it actually shows</strong>, built by
  <code>internal/carddesc</code> — the same call <code>screens.cardTip</code> makes, not a copy of
  it. The card is a fire Skewer and the duelist behind it hits for 10, so a figure that looks wrong
  can be checked by eye. There are no relics on, which is why the block is the whole panel: the game
  appends its damage chain only when a relic has moved something. <strong>The colouring is the game's
  too</strong> — the same <code>cards.ElementRuns</code> table the screen reads — which is why FIRE
  is red and CHROMATIC is not: the wheel has no hue left for "all of them".
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
<div class="tip">
  <b>{{range .Tip.Title}}<span{{if .Ink}} style="color:{{.Ink}}"{{end}}>{{.Text}}</span>{{end}}</b>
  {{range .Tip.Lines}}<div>{{range .}}<span{{if .Ink}} style="color:{{.Ink}}"{{end}}>{{.Text}}</span>{{end}}</div>{{end}}
</div>

{{range .Styles}}
<h4><code>{{.Style}}</code>{{if .Default}}<span class="tag">the game draws this</span>{{end}}</h4>
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
  The same files, blown up nearest-neighbour, one row per style in the same order. Read it
  <em>after</em> the rows above, never instead of them — a border that only works at 3× is a border
  that does not work.
</p>
{{range $s := .Styles}}
<div class="row">
  {{range $s.Cells}}
  <div class="cardbox">
    <img class="big" src="{{.File}}" alt="{{.Label}}">
    <div class="cap">{{$.Zoom}}× · {{$s.Style}} · {{.Note}}</div>
  </div>
  {{end}}
</div>
{{end}}

<h4>states</h4>
<p class="note">
  In the style the game draws. The upgrade goes on after the state colouring, so a card that cannot
  be afforded has to still read as unavailable through it and a queued one has to still read as
  queued — which is the sharpest test of the border style, since state is what the border was
  already saying. A row where the three look the same is a bug in the order, not a matter of taste.
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
