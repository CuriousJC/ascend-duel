package main

import (
	"html/template"
	"slices"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
)

type page struct {
	Plates  []plate
	Total   score
	Filters template.HTML
}

// plate is one motif.
type plate struct {
	Motif, Name, Realms string
	Score               score

	CreatureGrid []gridRow
	RoomGrid     []gridRow
	Creatures    []creature
	Rooms        []room
	Todo         []todo

	missing []string
}

// Missing reports whether this motif still has work of a kind.
func (p plate) Missing(kind string) bool { return slices.Contains(p.missing, kind) }

// Tokens is the plate's `data-missing` attribute: every kind of work still open on it.
func (p plate) Tokens() string {
	var out []string
	for _, kind := range todoKinds {
		if p.Missing(kind) {
			out = append(out, token(kind))
		}
	}
	return strings.Join(out, " ")
}

type gridRow struct {
	Tier  string
	Cells []gridCell
}

type gridCell struct{ Text, State string }

type creature struct {
	Record, Name, Tier string
	Brief              bool
	Elements           []cell
}

type room struct {
	Backdrop, Name, Tier string
	Brief                bool
	Elements             []cell
}

// cell is one element of one record: whether it is painted, and whether it carries direction of
// its own — a creature's specific override, or a room's ElementDraw.
type cell struct {
	Element  string
	Pictured bool
	Specific bool
}

type todo struct{ Kind, What string }

// Elements is the column order of both grids.
func (page) Elements() []string { return data.AffinityElements }

var tmpl = template.Must(template.New("motifreport").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Duello — motif report</title>
<style>
  :root { --ground: #a8bcd4; --ink: #2c2822; --dim: #5a6472; --rule: #8fa3bd; --panel: #c6d5e6;
          --pink: #c8508c; --ok: #3f7a4a; --warn: #9a6b12; }
  * { box-sizing: border-box; }
  body { margin: 0; padding: 32px 24px 64px; background: var(--ground); color: var(--ink);
         font: 14px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif; }
  .wrap { max-width: 1180px; margin: 0 auto; }
  h1 { font-size: 20px; margin: 0 0 4px; font-weight: 600; }
  h2 { font-size: 17px; margin: 0; font-weight: 600; }
  h3 { font-size: 11px; text-transform: uppercase; letter-spacing: .08em; color: var(--dim);
       margin: 16px 0 6px; font-weight: 600; }
  .note { color: var(--dim); font-size: 12.5px; max-width: 80ch; margin: 8px 0 0; }
  code, .key { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
  .plate { background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
           padding: 18px 20px; margin-top: 22px; }
  .head { display: flex; align-items: baseline; gap: 14px; flex-wrap: wrap; }
  .pct { font-size: 22px; font-weight: 700; }
  .pct.full { color: var(--ok); }
  .facts { color: var(--dim); font-size: 12.5px; }
  .bars { display: grid; grid-template-columns: repeat(4, minmax(150px, 1fr)); gap: 10px 22px; margin-top: 10px; }
  .bar { font-size: 12px; }
  .bar .track { height: 6px; background: #edf3fa; border-radius: 3px; overflow: hidden; margin-top: 3px; }
  .bar .fill { height: 100%; background: var(--ok); }
  .cols { display: flex; flex-wrap: wrap; gap: 34px; }
  table { border-collapse: collapse; font-size: 12.5px; }
  th { text-align: left; font-weight: 600; font-size: 11px; text-transform: uppercase; letter-spacing: .07em;
       color: var(--dim); border-bottom: 1px solid var(--rule); padding: 3px 12px 4px 0; }
  td { padding: 3px 12px 3px 0; vertical-align: top; }
  td.g { text-align: center; min-width: 58px; font-family: ui-monospace, monospace; font-size: 12px; border-radius: 4px; }
  td.ok { background: #d6e8d4; } td.hole { background: #f1d0dc; color: var(--pink); font-weight: 600; }
  td.brief { background: #f3e4c4; color: var(--warn); }
  .pill { display: inline-block; font-size: 11px; padding: 0 6px; margin: 0 3px 2px 0; border-radius: 9px;
          border: 1px solid var(--rule); background: #edf3fa; }
  .pill.miss { border-color: var(--pink); color: var(--pink); background: #f7e3ea; }
  .pill.spec::after { content: " ★"; }
  .tbd { color: var(--pink); font-weight: 600; }
  ul.todo { margin: 0; padding-left: 18px; font-size: 12.5px; columns: 2; column-gap: 34px; }
  ul.todo li { break-inside: avoid; }
  .kind { font-size: 10.5px; text-transform: uppercase; letter-spacing: .06em; color: var(--dim); margin-right: 6px; }
  .done { color: var(--ok); font-weight: 600; font-size: 12.5px; }
</style>

<div class="wrap">
  <h1>Motif report</h1>
  <p class="note">
    How full each motif is, and everything still to write or to paint. Counted off
    <code>data/motifs/</code> and the pictures under <code>assets/motifs/</code> — no pictures are
    drawn here; the motif sheet is where a creature is looked at. A pill is one element of one
    record: pink is no picture yet, ★ is direction written for that element on the record itself
    (a creature's own override, a room's ElementDraw).
  </p>
  <p class="note">
    <strong>Across the roster:</strong> {{.Total.Percent}}% ·
    briefs {{.Total.Briefs}} · creature art {{.Total.CreatureArt}} ·
    fights with a room {{.Total.Rooms}} · fights with a painted room {{.Total.RoomArt}}.
    A fight is one tier in one element, fifteen to a motif; one with no room draws the plain default backdrop.
  </p>

  {{.Filters}}

  {{$els := .Elements}}
  {{range .Plates}}
  <section class="plate sheet-item" data-missing="{{.Tokens}}">
    <div class="head">
      <h2>{{.Name}} <span class="key">{{.Motif}}</span></h2>
      <span class="pct{{if eq .Score.Percent 100}} full{{end}}">{{.Score.Percent}}%</span>
      <span class="facts">{{.Realms}} · {{len .Creatures}} creatures · {{len .Rooms}} rooms</span>
    </div>
    <div class="bars">
      <div class="bar">briefs {{.Score.Briefs}}<div class="track"><div class="fill" style="width: {{.Score.Briefs.Percent}}%"></div></div></div>
      <div class="bar">creature art {{.Score.CreatureArt}}<div class="track"><div class="fill" style="width: {{.Score.CreatureArt.Percent}}%"></div></div></div>
      <div class="bar">fights with a room {{.Score.Rooms}}<div class="track"><div class="fill" style="width: {{.Score.Rooms.Percent}}%"></div></div></div>
      <div class="bar">rooms painted {{.Score.RoomArt}}<div class="track"><div class="fill" style="width: {{.Score.RoomArt.Percent}}%"></div></div></div>
    </div>

    <div class="cols">
      <div>
        <h3>Creatures per fight</h3>
        <table>
          <tr><th></th>{{range $els}}<th>{{.}}</th>{{end}}</tr>
          {{range .CreatureGrid}}<tr><td>{{.Tier}}</td>{{range .Cells}}<td class="g {{.State}}">{{.Text}}</td>{{end}}</tr>{{end}}
        </table>
      </div>
      <div>
        <h3>Rooms per fight — painted / authored</h3>
        <table>
          <tr><th></th>{{range $els}}<th>{{.}}</th>{{end}}</tr>
          {{range .RoomGrid}}<tr><td>{{.Tier}}</td>{{range .Cells}}<td class="g {{.State}}">{{.Text}}</td>{{end}}</tr>{{end}}
        </table>
      </div>
    </div>

    <h3>Rooms</h3>
    {{if .Rooms}}
    <table>
      <tr><th>Backdrop</th><th>Name</th><th>Tier</th><th>Draw</th><th>Elements</th></tr>
      {{range .Rooms}}<tr>
        <td class="key">{{.Backdrop}}</td><td>{{.Name}}</td><td>{{.Tier}}</td>
        <td>{{if .Brief}}written{{else}}<span class="tbd">TBD</span>{{end}}</td>
        <td>{{range .Elements}}<span class="pill{{if not .Pictured}} miss{{end}}{{if .Specific}} spec{{end}}">{{.Element}}</span>{{end}}</td>
      </tr>{{end}}
    </table>
    {{else}}<p class="note">None authored — every fight on this motif draws the default backdrop.</p>{{end}}

    <h3>Creatures</h3>
    <table>
      <tr><th>Record</th><th>Name</th><th>Tier</th><th>Draw</th><th>Elements</th></tr>
      {{range .Creatures}}<tr>
        <td class="key">{{.Record}}</td><td>{{.Name}}</td><td>{{.Tier}}</td>
        <td>{{if .Brief}}written{{else}}<span class="tbd">TBD</span>{{end}}</td>
        <td>{{range .Elements}}<span class="pill{{if not .Pictured}} miss{{end}}{{if .Specific}} spec{{end}}">{{.Element}}</span>{{end}}</td>
      </tr>{{end}}
    </table>

    <h3>Still to do — {{len .Todo}}</h3>
    {{if .Todo}}<ul class="todo">{{range .Todo}}<li><span class="kind">{{.Kind}}</span>{{.What}}</li>{{end}}</ul>
    {{else}}<p class="done">Nothing — this motif is full.</p>{{end}}
  </section>
  {{end}}
</div>
`))
