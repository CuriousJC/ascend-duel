package main

// page.go — the HTML the sheet writes.
//
// **Kept beside the tool rather than in a shared template**, on the terms the mark sheet's own page
// records: the sheets have almost nothing in common but `png.Encode`, and a shared layout would be
// a seam that exists to be a seam. What this one needs that no other does is a matrix walked by
// *value* rather than by form — eleven rows of six — because the defect it exists to catch is a
// numeral drifting between the colors of one value.

// pageHTML is the whole page. It enforces the mark sheet's one visual rule: **actual size first,
// enlarged underneath**, so nothing is ever judged only at a size the game does not draw.
const pageHTML = `<!doctype html>
<meta charset="utf-8">
<title>Badge sheet</title>
<style>
  body { background: {{.Ground}}; color: #12141a; font: 14px system-ui, sans-serif;
         margin: 0; padding: 32px; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 17px; margin: 32px 0 2px; }
  p.note { margin: 0 0 16px; max-width: 74ch; opacity: .78; }
  table { border-collapse: collapse; background: {{.Surface}}; border-radius: 10px;
          overflow: hidden; margin: 0 0 8px; }
  th, td { padding: 8px 12px; text-align: center; vertical-align: bottom; }
  th { font-weight: 600; font-size: 12px; text-transform: uppercase; letter-spacing: .06em;
       opacity: .6; }
  th.value { text-align: right; font-size: 15px; text-transform: none; opacity: .85; }
  td.cell { border-top: 1px solid rgba(0,0,0,.07); }
  img { image-rendering: pixelated; display: block; margin: 0 auto 4px; }
  .figures { font: 11px ui-monospace, monospace; opacity: .55; white-space: nowrap; }
  .missing { color: #c02c4a; font-weight: 600; font-size: 12px; }
  .zoomed img { transform-origin: bottom center; transform: scale({{.Zoom}}); }
  .zoomed td.cell { padding: 24px 40px 8px; }
</style>

<h1>Badge sheet</h1>
<p class="note">
  Every damage badge the game can draw: each multiplier on the ladder, in the five elements plus
  the neutral one, in the <strong>{{.Shape}}</strong> outline. <strong>Read the actual-size table
  first</strong> — there is no zoom anywhere in the game.
</p>
<p class="note">
  <strong>Across a row, these six are meant to be one drawing in six inks.</strong> A numeral that
  sits lower, or reads heavier, in one color than in the other five is the defect this page exists
  for, and it is a redraw of one file rather than of the batch. <strong>Down a column, can you read
  every value at 16 pixels?</strong> The fractions are the hard ones. Under each cell: the ink's
  drawn size, how many pixels of it are opaque, and what share of those are near-black — the
  contour and the numeral together, and a 0% is a badge with nothing holding it against a pale
  card. A gap is a file nobody drew, never a fallback.
</p>

{{range .Sizes}}
<h2>{{.Size}}px — {{.Where}}</h2>
<table>
  <tr>
    <th></th>
    <th>fire</th><th>ice</th><th>lightning</th><th>earth</th><th>arcane</th><th>neutral</th>
  </tr>
  {{range .Values}}
  <tr>
    <th class="value">{{.Label}}</th>
    {{range .Cells}}
    <td class="cell">
      {{if .Missing}}<span class="missing">no file</span>
      {{else}}<img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Key}}">
      <div class="figures">{{.Ink}} · {{.Pixels}}px · {{.Dark}}% dark</div>{{end}}
    </td>
    {{end}}
  </tr>
  {{end}}
</table>
{{end}}

<h2>The three outlines, unnumbered</h2>
<p class="note">
  The shape is still an open choice, so all three ship and one is drawn — see
  <code>DefaultBadgeShape</code>. These are the blank badges, which are also what a multiplier
  nobody drew falls back to, with its figure printed on top in the card's own type.
</p>
<table class="zoomed">
  <tr>
    <th></th>
    <th>fire</th><th>ice</th><th>lightning</th><th>earth</th><th>arcane</th><th>neutral</th>
  </tr>
  {{range .Shapes}}
  <tr>
    <th class="value">{{.Shape}}</th>
    {{range .Cells}}
    <td class="cell">
      {{if .Missing}}<span class="missing">no file</span>
      {{else}}<img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Key}}">{{end}}
    </td>
    {{end}}
  </tr>
  {{end}}
</table>
`
