package main

// page.go — the HTML the sheet writes. Kept beside the tool, on the terms every sheet's page is.

const pageHTML = `<!doctype html>
<meta charset="utf-8">
<title>Interface sheet</title>
<style>
  body { background: #7eace4; color: #12141a; font: 14px system-ui, sans-serif; margin: 0; padding: 32px; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 17px; margin: 36px 0 4px; }
  p.note { margin: 0 0 14px; max-width: 78ch; opacity: .8; }
  figure { margin: 0 0 14px; }
  figcaption { font: 12px ui-monospace, monospace; opacity: .7; margin-top: 2px; }
  img { image-rendering: auto; display: block; }
  .missing { color: #a3122c; font-weight: 600; font-size: 12px; }
  .grid { display: flex; flex-wrap: wrap; gap: 8px 20px; align-items: flex-end; }
</style>

<h1>Interface sheet</h1>
<p class="note">
  The interface's own art: the two lettering sheets, every button face, every square button and the
  parts square buttons are assembled from, and the bar cells. Each is drawn at the sizes the game
  uses it, on the table's blue and, for the lettering, on a dark panel too.
</p>

<h2>Prose — the reading set</h2>
<p class="note">
  The whole repertoire and a sentence at each reading tier. Prose is white under a thin outline and
  the game multiplies it by an ink, so it is shown as drawn, on both grounds.
</p>
{{range .Prose}}<figure><img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Caption}}">
<figcaption>{{.Caption}}</figcaption></figure>{{end}}

<h2>Figures — the interface's lettering</h2>
<p class="note">
  Every ink of the figure set, the same alphabet in each, at two sizes. <strong>A character an ink
  is missing is listed under it</strong>: a line falls back to the font whole when any glyph is
  missing, so one gap changes the typeface of a whole word.
</p>
{{range .Figures}}<figure>
{{if .File}}<img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Ink}}">{{end}}
<figcaption>{{.Ink}}{{if .Missing}} — <span class="missing">missing: {{.Missing}}</span>{{end}}</figcaption>
</figure>{{end}}

<h2>Word-button faces</h2>
<p class="note">
  Each face at the four button heights — 80, 68, 44, 32 — at its own proportions and stretched to a
  wide button. The wide one is composed here by the screen's rule, ends scaled to the height and the
  middle filling the rest.
</p>
{{range .Faces}}<figure><img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Caption}}">
<figcaption>{{.Caption}}</figcaption></figure>{{end}}

<h2>Square buttons, drawn whole</h2>
<p class="note">Each at 68, 44 and 32 pixels. A <code>-pressed</code> file is the latched drawing of the button above it.</p>
<div class="grid">
{{range .Squares}}<figure><img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Name}}">
<figcaption>{{.Name}}</figcaption></figure>{{end}}
</div>

<h2>Square buttons, assembled</h2>
<p class="note">
  Every glyph laid over every blank tile, at 68, 44 and 32 pixels — the way a square button is built
  from parts. The first column is the tile alone.
</p>
{{range .Assembled}}<figure><img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Caption}}">
<figcaption>{{.Caption}}</figcaption></figure>{{end}}

<h2>Bars</h2>
<p class="note">The bar cells and the health bar's pieces, as delivered.</p>
<div class="grid">
{{range .Bars}}<figure><img src="{{.File}}" width="{{.W}}" height="{{.H}}" alt="{{.Caption}}">
<figcaption>{{.Caption}}</figcaption></figure>{{end}}
</div>
`
