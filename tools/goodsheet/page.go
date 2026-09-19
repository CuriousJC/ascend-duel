package main

import "html/template"

// The page. One static file, no JavaScript, no build step: the loop is "edit goods.json, re-run
// the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are shown at their natural size with image-rendering: pixelated**, for the reason the
// relic sheet's template gives: a card's rim is one pixel thick and a browser that scales it
// resamples that rim into a blur, which makes the sheet lie about the art.
//
// **The ground is the one the goods actually sit on.** A gray border on white is a different card
// from a gray border on the game's own blue.
//
// **The tooltip is drawn as a panel rather than printed as a list.** The face says nothing now, so
// the tip is the whole of what a player reads before paying, and a bare list of strings under the
// card reads as metadata about the card instead of as the card's own words.
//
// **A contents with no vessel is still a heading**, drawn as an empty seat rather than skipped.
// That is the one layout decision this template makes that the essence sheet's does not, and it is
// the stone sheet's argument on the other axis: the gap is the finding.
var tmpl = template.Must(template.New("goodsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — sealed goods sheet</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #5a6472;
    --rule: #8fa3bd;
    --panel: #c6d5e6;
    --pink: #c8508c;
    --tip: #2c3440;
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
    padding: 16px; width: 520px;
  }
  /* Natural size, never scaled. See the comment above. */
  img { display: block; image-rendering: pixelated; }
  /* A contents with no vessel keeps the card's footprint, so the ladders stay comparable. */
  .empty {
    width: {{index .Style "width"}}px; height: {{index .Style "height"}}px;
    border: 2px dashed var(--rule); border-radius: {{index .Style "cornerRadius"}}px;
    display: flex; align-items: center; justify-content: center;
    color: var(--dim); font-size: 12px; text-align: center; padding: 12px;
    flex: none;
  }
  .about { min-width: 0; flex: 1; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  .record { color: var(--dim); font-size: 11.5px; font-family: ui-monospace, monospace; }
  .offer { margin: 10px 0 0; font-size: 13px; }
  .offer b { font-variant-numeric: tabular-nums; }
  .step { margin: 6px 0 0; font-size: 12.5px; color: var(--dim); }
  .step b { color: var(--ink); font-variant-numeric: tabular-nums; }
  /* The tip drawn as the panel a player rests on, not as a list of fields. */
  .tip {
    margin: 12px 0 0; background: var(--tip); color: #e8eef6;
    border-radius: 6px; padding: 10px 12px; font-size: 12.5px; line-height: 1.45;
  }
  .tip .t { font-weight: 600; letter-spacing: .04em; }
  .tip div { color: #c3cfdd; }
  .dialog {
    margin: 10px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 11.5px; color: var(--dim);
  }
  .art { margin: 8px 0 0; font-size: 11.5px; color: var(--pink); }
  /* The subject paragraph an art generator was given, set apart from the authored line a player
     reads: indented, quieted, and marked when nobody has written one yet. Same pink the
     missing-art line takes, because they are one backlog. */
  .draw {
    margin: 10px 0 0; font-size: 12.5px; color: var(--dim);
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
  .draw.missing { color: var(--pink); border-left-color: var(--pink); }
</style>

<h1>Sealed goods sheet</h1>
<p class="facts">
  {{.Count}} goods over {{.Catalog}} kinds of contents{{if .Unheld}} — {{.Unheld}} kind(s) have no
  vessel at all{{end}}. {{.Seats}} stand on a shelf at a time, one kind of contents each.
  Good card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  text band from <code>y={{index .Style "textBandTop"}}</code> — which nothing here fills.
  Shown at 1:1 on the ground the goods are actually offered on.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/goodsheet</code> and refresh. Every card here is drawn by
  <code>internal/cards</code>, the same code the game blits, and every word beside it is read out
  of <code>data/goods.json</code> through <code>internal/session</code>'s own validation — so a
  good this page refuses to draw is a good the game refuses to start with.
</p>
<p class="note">
  <strong>The face is a picture and a name, so the tooltip is the card.</strong> A good carried its
  offer across the lower half of its art until 2026-09-15. What a player can know before paying is
  now said entirely by resting on it, and the dark panel beside each card is that paragraph, built
  the way the game builds it: the count and the noun computed from the record, the prose lines read
  off it, the price last.
</p>
<p class="note">
  <strong>What a bigger vessel sells is a wider choice, not more cards.</strong> Every good gives
  exactly one of what it holds whatever its size, so the figure to review is what the extra options
  cost over the size below — printed under each card. A ladder charging more for one option than
  for the next is a rung nobody has a reason to buy, and it is invisible reading prices down a
  column.
</p>
<p class="note">
  <strong>Read the size against the catalog under it.</strong> The share on each card is how much
  of its contents the player is shown at once, computed from the live catalog rather than authored.
  A vessel showing most of what there is to see is a sealed good that is barely sealed — the choice
  it sells is nearly the whole shelf.
</p>
<p class="note">
  <strong>The quoted paragraph under each card is its <code>Draw</code></strong> — the subject an
  art generator was handed, against the picture that came back. The style, the ground and the
  lighting are not in it: those are <code>docs/art/other_card_art_prompt.MD</code>, which is about
  no record at all, and this says only what the object is. The engine ignores the field, so nothing
  but this page reads it and nothing fails when a picture stops matching its brief.
</p>
<p class="note">
  <strong>An unpainted good borrows the face of what is inside it</strong> —
  <code>screens.goodArt</code> falls back to the default stone, essence or rune picture. That is
  the "nobody has painted this yet" state, and it is marked below: two vessels borrowing one
  contents' default face are two cards a player cannot tell apart on a shelf.
</p>

<h2>The vessels, by what is inside</h2>
<p class="note">
  <strong>Grouped by contents and walked smallest first</strong>, which is how the shelf groups
  them: the roll picks a kind of contents and then takes whichever size it landed on. Reading a
  ladder down is reading the one decision a player makes between shops.
</p>

{{range .Groups}}
<h3 class="group">
  {{.Contents}}
  <span>
    {{if .Goods}}{{len .Goods}} vessel(s), drawn from {{.Held}} in the catalog{{else}}nothing holds them{{end}}
  </span>
</h3>
<div class="plates">
  {{if .Goods}}
    {{range .Goods}}
      <div class="plate">
        <img src="{{.Cell.File}}" width="{{.Cell.Width}}" height="{{.Cell.Height}}" alt="{{.Name}}">
        <div class="about">
          <p class="name">{{.Name}}</p>
          <div class="record">{{.Record}} · {{.Family}}</div>
          <p class="offer">
            <b>{{.Size}}</b> {{.Contents}}, keep <b>1</b>, for <b>{{.Price}}</b> vitae
            — <b>{{.Share}}%</b> of the catalog seen at once
          </p>
          {{if .HasStep}}
            <p class="step">
              <b>+{{.StepSize}}</b> option(s) over the vessel below, for <b>+{{.StepPrice}}</b> vitae
            </p>
          {{else}}
            <p class="step">the smallest vessel of this catalog — nothing to compare it against</p>
          {{end}}
          <div class="tip">
            <div class="t">{{.TipTitle}}</div>
            {{range .Tip}}<div>{{.}}</div>{{end}}
          </div>
          <p class="dialog">opens headed &ldquo;{{.Title}}&rdquo; — {{.Hint}}</p>
          {{if .Draw}}
            <p class="draw">{{.Draw}}</p>
          {{else}}
            <p class="draw missing">no subject written yet</p>
          {{end}}
          {{if .Borrowed}}
            <p class="art">no art of its own — borrowing the default {{.Contents}} face</p>
          {{end}}
        </div>
      </div>
    {{end}}
  {{else}}
    <div class="plate">
      <div class="empty">no vessel holds them</div>
      <div class="about">
        <p class="name">{{.Contents}}</p>
        <div class="record">{{.Held}} in the catalog, and no way to be offered any</div>
        <p class="step">
          a contents the rules can open and the shop cannot sell — a record in
          <code>goods.json</code> is all it wants
        </p>
      </div>
    </div>
  {{end}}
</div>
{{end}}

<h2>Card states</h2>
<p class="note">
  The two states a sealed good is drawn in. A good is bought rather than chosen out of a set, so
  the shelf dims what the purse cannot reach and lights nothing — there is no third.
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
