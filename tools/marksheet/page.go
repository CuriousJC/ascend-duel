package main

// page.go — the HTML the sheet writes.
//
// **Kept beside the tool rather than in a shared template.** Every sheet under `tools/` writes its
// own page for the reason `tools/sheets` gives for shelling out rather than importing: they have
// almost nothing in common but `png.Encode`, and a shared layout would be a seam that exists to be
// a seam. What this one needs that no other does is a matrix — forms down, elements across — and
// two of them at two sizes.

// pageHTML is the whole page. The one rule it enforces visually is the retired glyph sheet's: **actual size
// first, enlarged underneath**, so nothing is ever judged only at a size the game does not draw.
const pageHTML = `<!doctype html>
<meta charset="utf-8">
<title>Mark sheet</title>
<style>
  body { background: {{.Ground}}; color: #12141a; font: 14px system-ui, sans-serif;
         margin: 0; padding: 32px; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 17px; margin: 32px 0 2px; }
  p.note { margin: 0 0 16px; max-width: 70ch; opacity: .78; }
  table { border-collapse: collapse; background: {{.Surface}}; border-radius: 10px;
          overflow: hidden; margin: 0 0 8px; }
  th, td { padding: 8px 10px; text-align: center; vertical-align: bottom; }
  th { font-weight: 600; font-size: 12px; text-transform: uppercase; letter-spacing: .06em;
       opacity: .6; }
  th.form { text-align: right; }
  td.cell { border-top: 1px solid rgba(0,0,0,.07); }
  img { image-rendering: pixelated; display: block; margin: 0 auto 4px; }
  .figures { font: 11px ui-monospace, monospace; opacity: .55; white-space: nowrap; }
  .missing { color: #c02c4a; font-weight: 600; font-size: 12px; }
  .zoomed img { transform-origin: bottom center; }
</style>

<h1>Mark sheet</h1>
<p class="note">
  Every form mark and every cost tick, at the sizes the game draws them and blown up
  {{.Zoom}}x with nearest-neighbor. <strong>Read the actual-size table first</strong> — there is no
  zoom anywhere in the game, and reviewing only the enlarged one is how a mark comes to look
  acceptable in review and clunky in play. Under each cell: the ink's drawn size, how many pixels
  of it are opaque, and what share of those are near-black. That last figure is the contour, and a
  0% is a mark with nothing holding it against a pale card.
</p>

{{range .Marks}}
<h2>Form marks — {{.Size}}px, {{.Where}}</h2>
<table>
  <tr><th></th>{{range $.Elements}}<th>{{.}}</th>{{end}}</tr>
  {{range .Forms}}
  <tr>
    <th class="form">{{.Form}}</th>
    {{range .Cells}}
    <td class="cell">
      {{if .Missing}}<span class="missing">no file for {{.Key}}</span>
      {{else}}<img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Key}}">
      <span class="figures">{{.Ink}} · {{.Pixels}}px · {{.Dark}}% dark</span>{{end}}
    </td>
    {{end}}
  </tr>
  {{end}}
</table>

<table class="zoomed">
  <tr><th></th>{{range $.Elements}}<th>{{.}}</th>{{end}}</tr>
  {{range .Forms}}
  <tr>
    <th class="form">{{.Form}}</th>
    {{range .Cells}}
    <td class="cell">
      {{if .Missing}}<span class="missing">—</span>
      {{else}}<img src="{{.File}}" width="{{mul .W $.Zoom}}" height="{{mul .H $.Zoom}}" alt="{{.Key}}">{{end}}
    </td>
    {{end}}
  </tr>
  {{end}}
</table>
{{end}}

{{range .Ticks}}
<h2>Cost ticks — {{.Size}}, {{.Where}}</h2>
<table>
  <tr><th></th>{{range $.Elements}}<th>{{.}}</th>{{end}}</tr>
  <tr>
    <th class="form">one</th>
    {{range .Ones}}
    <td class="cell">
      {{if .Missing}}<span class="missing">no file for {{.Key}}</span>
      {{else}}<img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Key}}">
      <span class="figures">{{.Ink}} · {{.Pixels}}px · {{.Dark}}% dark</span>{{end}}
    </td>
    {{end}}
  </tr>
  <tr>
    <th class="form">a cost of three</th>
    {{range .Threes}}
    <td class="cell">
      {{if .Missing}}<span class="missing">—</span>
      {{else}}<img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Key}}">{{end}}
    </td>
    {{end}}
  </tr>
  <tr>
    <th class="form">enlarged</th>
    {{range .Threes}}
    <td class="cell">
      {{if .Missing}}<span class="missing">—</span>
      {{else}}<img src="{{.File}}" width="{{mul .W $.Zoom}}" height="{{mul .H $.Zoom}}" alt="{{.Key}}">{{end}}
    </td>
    {{end}}
  </tr>
</table>
{{end}}
`
