package main

import "html/template"

type page struct {
	Rooms        []room
	Bare         []string // motifs with no room authored: every fight on them draws the default
	FallbackLink string
	Filters      template.HTML
}

// room is one backdrop record, and one group on the page.
type room struct {
	Motif, MotifName string
	Backdrop, Name   string
	Tier, TierLabel  string
	Draw             string
	Cells            []cell
}

// cell is one element of one room: its picture, or the default standing in for it.
type cell struct {
	Element   string
	Key       string // the picture's stem, which is what the file under assets/ is called
	Direction string // the room's ElementDraw for this element
	Thumb     string
	Link      string
	Painted   bool
}

// State is the cell's `data-state` token.
func (c cell) State() string {
	if c.Painted {
		return "painted"
	}
	return "unpainted"
}

var tmpl = template.Must(template.New("backdropsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Duello — backdrop sheet</title>
<style>
  :root { --ground: #a8bcd4; --ink: #2c2822; --dim: #5a6472; --rule: #8fa3bd; --panel: #c6d5e6;
          --pink: #c8508c; }
  * { box-sizing: border-box; }
  body { margin: 0; padding: 32px 24px 64px; background: var(--ground); color: var(--ink);
         font: 14px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif; }
  .wrap { max-width: 1400px; margin: 0 auto; }
  h1 { font-size: 20px; margin: 0 0 4px; font-weight: 600; }
  h2 { font-size: 17px; margin: 0; font-weight: 600; }
  .note { color: var(--dim); font-size: 12.5px; max-width: 90ch; margin: 8px 0 0; }
  code, .key { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
  .room { background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
          padding: 18px 20px; margin-top: 22px; }
  .head { display: flex; align-items: baseline; gap: 14px; flex-wrap: wrap; }
  .facts { color: var(--dim); font-size: 12.5px; }
  .draw { font-size: 13px; max-width: 110ch; margin: 8px 0 14px; }
  .cells { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; }
  figure { margin: 0; }
  figure img { display: block; width: 100%; aspect-ratio: 16 / 9; border-radius: 4px;
               border: 1px solid var(--rule); }
  figure.unpainted img { opacity: .45; border: 2px dashed var(--pink); }
  figcaption { font-size: 12.5px; margin-top: 6px; }
  .el { font-weight: 600; text-transform: uppercase; letter-spacing: .06em; font-size: 11.5px; }
  .miss { color: var(--pink); font-weight: 600; margin-left: 6px; }
  .dir { color: var(--dim); font-size: 12px; margin-top: 3px; }
  .tbd { color: var(--pink); font-weight: 600; }
</style>

<div class="wrap">
  <h1>Backdrop sheet</h1>
  <p class="note">
    Every room a motif's fights are drawn in front of, once per element it is painted in, with the
    brief it was painted from: the room's <code>Draw</code> on top and each element's
    <code>ElementDraw</code> under its picture. A picture not painted yet is shown as the default
    backdrop the fight really draws, dimmed and outlined in pink. Each picture links to the full
    file under <code>assets/motifs/</code>.
  </p>
  {{if .Bare}}<p class="note"><strong>No rooms authored:</strong>
    {{range $i, $m := .Bare}}{{if $i}}, {{end}}{{$m}}{{end}} — every fight on these draws the default.</p>{{end}}

  {{.Filters}}

  {{range .Rooms}}
  <section class="room sheet-group" data-motif="{{.Motif}}" data-tier="{{.Tier}}">
    <div class="head">
      <h2>{{.Name}}</h2>
      <span class="key">{{.Backdrop}}</span>
      <span class="facts">{{.MotifName}} · {{.TierLabel}} · {{len .Cells}} elements</span>
    </div>
    <p class="draw">{{if .Draw}}{{.Draw}}{{else}}<span class="tbd">No Draw written.</span>{{end}}</p>
    <div class="cells">
      {{range .Cells}}
      <figure class="sheet-item {{.State}}" data-element="{{.Element}}" data-state="{{.State}}">
        <a href="{{.Link}}"><img src="{{.Thumb}}" alt="{{.Key}}" loading="lazy"></a>
        <figcaption>
          <span class="el">{{.Element}}</span> <span class="key">{{.Key}}.jpg</span>
          {{if not .Painted}}<span class="miss">not painted</span>{{end}}
          <div class="dir">{{if .Direction}}{{.Direction}}{{else}}<span class="tbd">No ElementDraw written.</span>{{end}}</div>
        </figcaption>
      </figure>
      {{end}}
    </div>
  </section>
  {{end}}
</div>
`))
