package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit hands.json or the
// deck, re-run the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// card sheet's template gives: a card's rim is one pixel thick and a browser that scales it
// resamples that rim into a blur, which makes the sheet lie about the art.
//
// The layout is a row per rung — what it pays on the left, the cards on the right — because the
// review question is a comparison down the column, not a look at any one hand.
var tmpl = template.Must(template.New("handsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — hand sheet</title>
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
  .note { color: var(--dim); font-size: 12.5px; max-width: 72ch; margin: 12px 0 0; }
  .rungs { margin-top: 22px; }
  .rung {
    display: flex; gap: 22px; align-items: flex-start;
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 14px 16px; margin-top: 12px;
  }
  .pays { width: 210px; flex: none; }
  .pays .mult { font-size: 22px; font-weight: 600; line-height: 1.1; }
  .pays .name { font-size: 14px; font-weight: 600; margin-top: 2px; }
  .pays .axis { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .pays .cost { font-size: 12.5px; margin-top: 8px; }
  .pays .cost.over { color: var(--pink); font-weight: 600; }
  .hand { min-width: 0; }
  .cards { display: flex; gap: 6px; flex-wrap: wrap; }
  /* Natural size, never scaled. See the comment above. */
  img { display: block; image-rendering: pixelated; }
  .shares { color: var(--dim); font-size: 11.5px; margin-top: 8px; }
  .missing { color: var(--pink); font-size: 12.5px; }
</style>

<h1>Hand sheet</h1>
<p class="facts">
  {{len .Rows}} hands, cheapest-paying first. Deck of <code>{{.DeckSize}}</code>,
  <code>{{.Attacks}}</code> of them attacks;
  <code>{{index .PerValue "concept"}}</code> cards per concept,
  <code>{{index .PerValue "form"}}</code> per form,
  <code>{{index .PerValue "element"}}</code> per element.
  A round has <code>{{.Budget}}</code> action points and plays at most
  <code>{{.MaxCards}}</code> cards. Mini card
  <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>, shown at 1:1 on the
  table the hand is actually laid out on.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/handsheet</code> and refresh. Every rung comes from
  <code>internal/combat</code>'s live catalogue and every example is built out of cards that are
  really in <code>data/duelist_cards.json</code> — so a rung with no cards beside it is a rung the
  shipping deck cannot form at all.
</p>
<p class="note">
  <strong>Ordered by multiplier, across all three axes at once.</strong> That is the comparison
  the numbers are making — an Elemental Three of a Kind at 195 is claiming to be worth about what
  a Form Full House is not — and it is the comparison <code>hands.json</code>'s axis-by-axis
  layout hides.
</p>
<p class="note">
  <strong>The AP figure is not the difficulty.</strong> It is what the example above costs
  <em>once you hold the cards</em>. How often you hold them is
  <code>go run ./tools/handodds</code>, which deals two million hands and prints the
  reachability the multipliers are actually priced against. Two tools reporting the same
  probability by different methods would be two numbers that can disagree, so this one does not
  sample.
</p>
{{if .TooManyAP}}
<p class="note">
  <strong>{{.TooManyAP}} of these examples cost more than a round has</strong>, on action points or
  on the card cap. That is the <em>example</em>, not the rung: each one varies everything the rung
  does not count, so it is picked to illustrate rather than to be cheap. Every rung on this ladder
  has some build a 6 AP round can pay for — an Elemental Five of a Kind is Jab, Cut, Bash, Ward and
  Thrust in one colour, 6 AP exactly.
</p>
{{end}}

<h2>The ladder</h2>
<div class="rungs">
  {{range .Rows}}
    <div class="rung">
      <div class="pays">
        <div class="mult">{{.Pays}}</div>
        <div class="name">{{.Name}}</div>
        <div class="axis">{{.Match}} {{.Groups}} &middot; {{.Cards}} cards &middot; {{.Key}}</div>
        <div class="cost{{if not .Affordable}} over{{end}}">
          {{.Cost}} AP{{if not .Affordable}} — more than a round can play{{end}}
        </div>
      </div>
      <div class="hand">
        {{if .Cells}}
          <div class="cards">
            {{range .Cells}}
              <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
            {{end}}
          </div>
          <div class="shares">
            {{range $i, $s := .Shares}}{{if $i}} &middot; {{end}}{{$s}}{{end}}
          </div>
        {{else}}
          <div class="missing">no example — no set of cards in the deck forms this rung</div>
        {{end}}
      </div>
    </div>
  {{end}}
</div>
`))
