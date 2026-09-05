package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit hands.json or the
// deck, re-run the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// card sheet's template gives: a card's rim is one pixel thick and a browser that scales it
// resamples that rim into a blur, which makes the sheet lie about the art.
//
// The layout is a row per rung — what it pays on the left, how often it can be built in the middle,
// the cards on the right — because the review question is a comparison down the column, not a look
// at any one hand.
//
// **The reachability carries a meter as well as a figure** *(2026-09-05)*. The figure is the fact
// and the meter is the ranking: eighteen numbers running from 100% to 0.006% are not comparable by
// eye down a column, and the whole point of putting them on this page is to read them against the
// multiplier beside them. It is square-rooted, for the reason bar() in main.go gives.
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
  .odds { width: 190px; flex: none; }
  .odds .reach { font-size: 19px; font-weight: 600; line-height: 1.1; }
  .odds .one { color: var(--dim); font-size: 12px; }
  .odds .meter {
    height: 6px; margin: 8px 0 6px; border-radius: 3px;
    background: rgba(0, 0, 0, .09);
  }
  .odds .meter div { height: 100%; border-radius: 3px; background: var(--pink); }
  .odds .best { color: var(--dim); font-size: 11.5px; }
  .odds .unsampled { color: var(--dim); font-size: 12px; }
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
  <strong>The AP figure is not the difficulty, and the reachability is.</strong> The AP is what the
  example costs <em>once you hold the cards</em>. <strong>Reachable</strong> is how often you hold
  them: of <code>{{.Trials}}</code> opening hands of <code>{{.HandSize}}</code> dealt off this deck,
  the share that could afford some set forming the rung — which is the figure the multipliers are
  priced against. <strong>Best in hand</strong> is the share where it was the dearest-paying rung
  available, which is what a player taking the most they could would actually have landed.
</p>
<p class="note">
  Round one only: a later hand draws from a depleted pile and keeps what it did not spend, which
  the sample does not model. <code>{{.Nothing}}</code> of hands build no rung at all — the turns the
  High Card names. The same table, with the axes kept apart and an <code>-ap</code> flag for a turn
  holding cost discounts, is <code>go run ./tools/handodds</code>: one pinned sample in
  <code>tools/hands</code>, printed by both, so the two cannot disagree.
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
      <div class="odds">
        {{if .Sampled}}
          <div class="reach">{{.Reachable}}</div>
          <div class="one">{{.OneIn}} hands</div>
          <div class="meter"><div style="width: {{.Bar}}%"></div></div>
          <div class="best">best in hand {{.Best}}</div>
        {{else}}
          <div class="unsampled">not sampled — the fallback every attacking turn lands on</div>
        {{end}}
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
