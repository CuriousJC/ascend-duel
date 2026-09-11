package main

import (
	"html/template"
	"image/color"
)

// groundRGBA is `ground` as pixels, for the strips the cards are composited onto.
//
// **Derived from the hex rather than typed twice**, which is the mistake tools/roster is currently
// carrying: its two copies of the ground have drifted apart, so its strips are composited onto a
// cream that its page no longer uses. One of these has to be the source and it may as well be the
// one the browser reads.
var groundRGBA = mustHex(ground)

func mustHex(s string) color.RGBA {
	v := uint32(0)
	for _, c := range s[1:] {
		switch {
		case c >= '0' && c <= '9':
			v = v<<4 | uint32(c-'0')
		case c >= 'a' && c <= 'f':
			v = v<<4 | uint32(c-'a'+10)
		default:
			panic("scenariosheet: ground is not a six-digit hex colour: " + s)
		}
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}

// cell is one picture on the page, at its natural size.
type cell struct {
	File   string
	Label  string
	Width  int
	Height int
}

// named is a relic, a parasite or a stone: what the fixture wrote, what it is called, and the
// sentence it prints.
type named struct {
	Key  string
	Name string
	Text string
	Kind string // "bucket" or "pouch" for a held good; empty for a relic
}

// line is one row of a hand or a deck, as the fixture writes it.
type line struct {
	Card    string
	Element string
	Copies  int
	Cost    int
	Riders  string
}

// plate is one scenario.
type plate struct {
	Record  string
	Note    string
	Facts   []string
	Strips  []cell
	Relics  []named
	Held    []named
	Hand    []line
	Deck    []line
	DeckSum int
}

// Command is the line that launches this fixture, ready to copy. **Every plate carries one**,
// including the first — the first entry in the file is what a bare `go run -tags scenario .`
// launches, and a page that said so in prose would be a rule the reader has to remember rather
// than a command they can paste.
func (p plate) Command() string {
	return "ASCEND_DUEL_SCENARIO=" + p.Record + " go run -tags scenario ."
}

type page struct {
	Ground string
	Source string
	Count  int
	Plates []plate
}

// The page. One static file, no JavaScript, no build step: the loop is "edit scenarios.json,
// re-run the tool, refresh the tab", the same loop every other tool here has.
//
// **Images are at their natural size with image-rendering: pixelated**, for the reason every other
// sheet's template gives — a card's rim is one pixel thick and any scaling resamples it into a
// blur, which makes the sheet lie about the art. A strip can be wider than the window, so each one
// scrolls inside its own box rather than stretching the page.
var tmpl = template.Must(template.New("scenariosheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — scenario sheet</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #2c2822;
    --dim: #5a6472;
    --rule: #8fa3bd;
    --panel: #c6d5e6;
    --pink: #c8508c;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 32px 28px 64px;
    background: var(--ground); color: var(--ink);
    font: 14px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif;
  }
  .wrap { max-width: 1180px; margin: 0 auto; }
  h1 { font-size: 20px; margin: 0 0 4px; font-weight: 600; }
  .note { color: var(--dim); font-size: 12.5px; max-width: 76ch; margin: 8px 0 0; }
  .note code, code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }

  .plate {
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px;
    padding: 18px 20px; margin-top: 22px;
  }
  .plate.first { border-color: var(--pink); border-width: 2px; }
  .record { font-size: 17px; font-weight: 600; font-family: ui-monospace, monospace; }
  .default {
    font-family: inherit; font-size: 11px; font-weight: 600; letter-spacing: .07em;
    text-transform: uppercase; color: var(--pink); margin-left: 10px;
  }
  .facts { color: var(--dim); font-size: 12px; margin: 4px 0 0; }
  .cmd {
    display: block; margin: 10px 0 0; padding: 8px 10px;
    background: #edf3fa; border: 1px solid var(--rule); border-radius: 5px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px;
    overflow-x: auto; white-space: pre;
  }
  .blurb { margin: 12px 0 0; font-size: 13px; max-width: 86ch; }

  .strip { margin-top: 16px; }
  .strip .cap {
    font-size: 11px; text-transform: uppercase; letter-spacing: .08em;
    color: var(--dim); font-weight: 600; margin-bottom: 6px;
  }
  .scroll { overflow-x: auto; padding-bottom: 4px; }
  img { display: block; image-rendering: pixelated; }

  table { border-collapse: collapse; margin-top: 14px; font-size: 12.5px; }
  th {
    text-align: left; font-weight: 600; font-size: 11px; text-transform: uppercase;
    letter-spacing: .07em; color: var(--dim); border-bottom: 1px solid var(--rule);
    padding: 3px 14px 4px 0;
  }
  td { padding: 3px 14px 3px 0; vertical-align: top; }
  td.key, td.rider { font-family: ui-monospace, monospace; font-size: 11.5px; color: var(--dim); }
  td.said { color: var(--dim); max-width: 52ch; }
  .cols { display: flex; flex-wrap: wrap; gap: 34px; }
</style>

<div class="wrap">
  <h1>Scenario sheet</h1>
  <p class="note">
    The {{.Count}} fixtures in <code>{{.Source}}</code>: what each one plugs in, and the command
    that launches it. Compiled out of every build that is not <code>-tags scenario</code>, so none
    of this is reachable from a shipped binary.
  </p>
  <p class="note">
    The cards are drawn by <code>internal/cards</code>, the same code the game blits — but
    <strong>plain</strong>. A rider is printed as the word the file writes rather than painted onto
    the face; what an upgrade looks like is the upgrade sheet's subject.
  </p>

  {{range $i, $p := .Plates}}
    <div class="plate{{if eq $i 0}} first{{end}}">
      <div class="record">{{$p.Record}}{{if eq $i 0}}<span class="default">first in the file — the bare launch</span>{{end}}</div>
      <div class="facts">{{range $j, $f := $p.Facts}}{{if $j}} · {{end}}{{$f}}{{end}}</div>
      <code class="cmd">{{$p.Command}}</code>
      {{with $p.Note}}<p class="blurb">{{.}}</p>{{end}}

      {{range $p.Strips}}
        <div class="strip">
          <div class="cap">{{.Label}}</div>
          <div class="scroll"><img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}"></div>
        </div>
      {{end}}

      <div class="cols">
        {{with $p.Relics}}
          <table>
            <tr><th>Worn, in order</th><th>Relic</th><th>Says</th></tr>
            {{range .}}<tr><td class="key">{{.Key}}</td><td>{{.Name}}</td><td class="said">{{.Text}}</td></tr>{{end}}
          </table>
        {{end}}
        {{with $p.Held}}
          <table>
            <tr><th>Carried</th><th>Where</th><th>Says</th></tr>
            {{range .}}<tr><td class="key">{{.Key}}</td><td>{{.Kind}}</td><td class="said">{{.Text}}</td></tr>{{end}}
          </table>
        {{end}}
        {{with $p.Hand}}
          <table>
            <tr><th>Opening hand</th><th>Element</th><th>AP</th><th>Riders</th></tr>
            {{range .}}<tr><td>{{.Card}}</td><td>{{.Element}}</td><td>{{.Cost}}</td><td class="rider">{{.Riders}}</td></tr>{{end}}
          </table>
        {{end}}
        {{with $p.Deck}}
          <table>
            <tr><th>Deck — {{$p.DeckSum}} cards</th><th>Element</th><th>Copies</th><th>AP</th><th>Riders</th></tr>
            {{range .}}<tr><td>{{.Card}}</td><td>{{.Element}}</td><td>{{.Copies}}</td><td>{{.Cost}}</td><td class="rider">{{.Riders}}</td></tr>{{end}}
          </table>
        {{end}}
      </div>
    </div>
  {{end}}
</div>
`))
