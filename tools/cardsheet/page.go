package main

import "html/template"

// The page. Deliberately one static file with no JavaScript and no build step: the
// iteration loop is "edit code, re-run the tool, refresh the tab", the same loop as every
// other tool in this repo.
//
// **Images are shown at their natural size with image-rendering: pixelated.** A card is
// pixel art whose rim is one pixel thick, and a browser that scales it — even by the
// fractional amount a max-width rule can introduce — resamples that rim into a blur and
// makes the sheet lie about the art. This is the same constraint that keeps
// systems.CardGlyphScale a whole number.
//
// The ground is the combat screen's own background, so a card is judged against what it
// actually sits on. A light page would flatter an off-white card enormously.
var tmpl = template.Must(template.New("cardsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — card sheet</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #e8e8ea;
    --dim: #9a9aa2;
    --rule: #4a4a4a;
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
  .note {
    color: var(--dim); font-size: 12.5px; max-width: 62ch;
    margin: 12px 0 0;
  }
  .row { margin-top: 22px; }
  .row-label {
    color: var(--dim); font-size: 12px; letter-spacing: .04em;
    margin-bottom: 10px;
  }
  .cells { display: flex; flex-wrap: wrap; gap: 20px; align-items: flex-start; }
  /* Overlapped: each card slides left over the one before it, so only its left edge
     shows. That edge is where the border colour and the cost dashes are, which is the
     whole argument for the deck overlay doing this. */
  .stack { display: flex; align-items: flex-start; padding-left: 4px; }
  .stack img + img { margin-left: calc(var(--overlap) - 90px); }
  .stack-label {
    color: var(--dim); font-size: 12px; width: 90px; flex: none;
    padding-top: 30px; text-align: right; padding-right: 14px;
  }
  .stack-row { display: flex; align-items: flex-start; margin-bottom: 10px; }
  figure { margin: 0; }
  /* Natural size, never scaled. See the comment above. */
  img { display: block; image-rendering: pixelated; }
  figcaption {
    color: var(--dim); font-size: 11.5px; margin-top: 7px;
    max-width: 180px;
  }
</style>

<h1>Card sheet</h1>
<p class="facts">
  Hand card <code>{{index .Style "width"}}&times;{{index .Style "height"}}</code>,
  corner radius <code>{{index .Style "cornerRadius"}}</code>,
  border <code>{{index .Style "borderWidth"}}</code>,
  dashes <code>{{index .Style "dashWidth"}}&times;{{index .Style "dashHeight"}}</code>
  on a <code>{{index .Style "dashGap"}}</code> gap.
  Shown at 1:1 on the combat screen's own background.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/cardsheet</code> and refresh. Every image here is
  drawn by <code>internal/cards</code>, the same code the game blits, so what is on this
  page is what the game draws.
</p>

<h2>Border colour &times; action-point cost</h2>
{{range .Borders}}
  <div class="row">
    <div class="row-label">{{.Label}}</div>
    <div class="cells">
      {{range .Cells}}
        <figure>
          <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          <figcaption>{{.Label}}</figcaption>
        </figure>
      {{end}}
    </div>
  </div>
{{end}}

<h2>Card state</h2>
<p class="note">
  State used to be carried by dimming the element-coloured surface. With the surface now
  constant, it is the border moving toward that surface instead — away from it for
  selected, toward it for disabled. Note that scaling a colour down (the usual rule) would
  do the wrong thing here: on a light card a darker border reads as <em>louder</em>, not
  quieter.
</p>
{{range .States}}
  <div class="row">
    <div class="row-label">{{.Label}}</div>
    <div class="cells">
      {{range .Cells}}
        <figure>
          <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          <figcaption>{{.Label}}</figcaption>
        </figure>
      {{end}}
    </div>
  </div>
{{end}}

<h2>Category glyph</h2>
<p class="note">
  The glyph sits in the top-left corner, cropped by the card's own curve, and
  <em>replaced the category word</em> that used to run under the name. Attack reuses the same
  sword the damage badge was drawn from — the badge is gone, so on a card this is now the only
  sword there is.
</p>
{{range .Categories}}
  <div class="row">
    <div class="row-label">{{.Label}}</div>
    <div class="cells">
      {{range .Cells}}
        <figure>
          <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          <figcaption>{{.Label}}</figcaption>
        </figure>
      {{end}}
    </div>
  </div>
{{end}}

<h2>Real cards</h2>
<p class="note">
  The two grids above hold the card constant, which hides the things that only go wrong on
  a specific card: a long name against the cost dashes, and a card with no damage leaving
  the glyph column empty.
</p>
{{range .Shapes}}
  <div class="row">
    <div class="row-label">{{.Label}}</div>
    <div class="cells">
      {{range .Cells}}
        <figure>
          <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          <figcaption>{{.Label}}</figcaption>
        </figure>
      {{end}}
    </div>
  </div>
{{end}}

<h2>The back</h2>
<p class="note">
  What the draw pile is a stack of, and what a card shows through the first half of a flip.
  <strong>The same silhouette as a face</strong> &mdash; same footprint, same corner radius
  &mdash; because these are one object seen from two sides; the face on the right is there to
  check that against. No border, because the border is where the element is said and a back
  that carried one would be naming the card underneath it. The mark is sized as a proportion
  of the card rather than from <code>Style</code>, so both sizes are the same drawing.
</p>
{{range .Backs}}
  <div class="row">
    <div class="row-label">{{.Label}}</div>
    <div class="cells">
      {{range .Cells}}
        <figure>
          <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          <figcaption>{{.Label}}</figcaption>
        </figure>
      {{end}}
    </div>
  </div>
{{end}}

<h2>Deck overlay — half size, stacked</h2>
<p class="note">
  One row per element, twelve concepts each: the whole 60-card deck, which the old 8&times;3
  grid of half-size cards could not do — it held 24 and printed &ldquo;+N more not
  shown&rdquo; on every look. <strong>A third-size card shows no glyph and no text</strong>,
  and that is forced rather than chosen: 64-pixel art cannot be scaled to fit 59 pixels
  without destroying a one-pixel rim, and text at a third size is unreadable. What survives
  is the border colour and the count of dashes, so a row says how many of an element you
  hold and at what costs &mdash; but no longer <em>which concepts</em>. That is the real
  loss and the thing to judge here.
</p>
{{range .Deck}}
  <div class="stack-row">
    <div class="stack-label">{{.Label}}</div>
    <div class="stack" style="--overlap: {{.Overlap}}px">
      {{range .Cells}}
        <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="">
      {{end}}
    </div>
  </div>
{{end}}

<h2>Rings — the same format, pink border</h2>
<p class="note">
  Same footprint, corners and border treatment as a card, so the two read as one game. No
  cost dashes, no category glyph, no effect text: a ring is not played from a hand and has
  no phase. <strong>Not wired into the game</strong> &mdash; nothing builds one of these yet.
</p>
<p class="note">
  The artwork is <code>assets/fire-ring.png</code> &mdash; the one thing on this page that is
  not generated, and first-party work, so it ships without a licence question. Only the
  sheets prefixed <code>tyrian_</code> come from the Tyrian set that CLAUDE.md names as a
  release blocker; this is not one of them.
</p>
{{range .Rings}}
  <div class="row">
    <div class="row-label">{{.Label}}</div>
    <div class="cells">
      {{range .Cells}}
        <figure>
          <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          <figcaption>{{.Label}}</figcaption>
        </figure>
      {{end}}
    </div>
  </div>
{{end}}
`))
