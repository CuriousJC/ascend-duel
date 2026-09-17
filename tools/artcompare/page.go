package main

import "html/template"

// The page. One static file, no build step, no external anything — the loop is "re-run the tool,
// refresh the tab", the same loop every other tool in this repo has.
//
// **The ground is the combat screen's own background**, so a card is judged against what it
// actually sits on, and **images are shown at their natural size**: a card's rim is a two-pixel
// bevel and a browser asked to scale it resamples that into a blur, which would make the page lie
// about the art. Both constraints are tools/cardsheet's and are here for its reasons.
//
// **The script is the shortlist.** Clicking a card marks that set as the one to keep for that
// record, and the bar writes out the copy commands, grouped by the set they come from. That list
// is the actual output of a review like this — install-this-one, ninety-five times — and writing
// it out by hand is how one gets missed. Nothing is stored anywhere: a reload starts again, which
// is the honest behavior for a page whose PNGs are rewritten on every run of the tool.

// page is the whole page.
type page struct {
	Ground  string
	Catalog string
	File    string

	// Sets is the columns, in the order they are drawn. Labels is the same list flattened, for
	// the header line and the script.
	Sets   []set
	Labels []string

	// Dir is where a picked picture would be installed to — the catalog's own assets directory,
	// which is what the copy list writes to.
	Dir string

	Groups  []group
	Count   int
	Undrawn []string
}

// group is one heading and the records filed under it — the catalog's own Family, in the file's
// own order.
type group struct {
	Name string
	Rows []row
}

// row is one record, drawn once per set.
type row struct {
	Key     string
	Stem    string
	Caption string
	Cells   []cell
}

// cell is one rendered card. An empty File means that set did not hold the picture, which is
// drawn as a gap rather than dropped — an absent file is a thing to see.
type cell struct {
	File          string
	Label         string
	Width, Height int
}

var tmpl = template.Must(template.New("artcompare").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — {{.Catalog}} art, side by side</title>
<style>
  :root {
    --ground: {{.Ground}};
    --ink: #e8e8ea;
    --dim: #9a9aa2;
    --rule: #4a4a4a;
    --pick: #e0609a;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 0 28px 64px;
    background: var(--ground); color: var(--ink);
    font: 14px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif;
  }
  h1 { font-size: 20px; margin: 0 0 4px; font-weight: 600; }
  h2 {
    font-size: 13px; text-transform: uppercase; letter-spacing: .09em;
    color: var(--dim); font-weight: 600;
    margin: 40px 0 0; padding-bottom: 8px; border-bottom: 1px solid var(--rule);
  }
  .facts { color: var(--dim); font-size: 12px; margin: 0; }
  .facts code { color: var(--ink); }
  .note { color: var(--dim); font-size: 12.5px; max-width: 76ch; margin: 12px 0 0; }
  header { padding: 28px 0 0; }

  .bar {
    position: sticky; top: 0; z-index: 5;
    display: flex; gap: 10px; align-items: center; flex-wrap: wrap;
    margin: 20px 0 0; padding: 12px 0;
    background: var(--ground); border-bottom: 1px solid var(--rule);
  }
  .bar .spacer { flex: 1 1 auto; }
  button {
    font: inherit; font-size: 12.5px; color: var(--ink);
    background: #3d3d3d; border: 1px solid var(--rule); border-radius: 4px;
    padding: 5px 11px; cursor: pointer;
  }
  button:hover { background: #4a4a4a; }
  button[aria-pressed="true"] { background: #565656; border-color: #7a7a7a; }
  .tally { color: var(--dim); font-size: 12.5px; }
  .tally b { color: var(--pick); font-weight: 600; }

  .rows { display: flex; flex-wrap: wrap; gap: 26px 30px; margin-top: 22px; }
  .cap { color: var(--dim); font-size: 12px; margin-bottom: 8px; }
  .cap code { color: var(--ink); }
  .cells { display: flex; gap: 10px; align-items: flex-start; }
  figure { margin: 0; }
  figcaption {
    color: var(--dim); font-size: 11px; letter-spacing: .05em; text-transform: uppercase;
    margin-top: 5px; text-align: center;
  }
  .card {
    display: block; padding: 0; border: 2px solid transparent; border-radius: 6px;
    background: none; cursor: pointer; line-height: 0;
  }
  .card img { display: block; }
  .card[aria-pressed="true"] { border-color: var(--pick); }
  .card[aria-pressed="true"] + figcaption { color: var(--pick); }
  .gone {
    border: 1px dashed var(--rule); border-radius: 6px;
    display: flex; align-items: center; justify-content: center;
    color: var(--dim); font-size: 11px; text-align: center; padding: 8px;
  }
  .cell.hidden { display: none; }

  .out {
    margin-top: 18px; width: 100%; min-height: 120px; display: none;
    font: 12px/1.5 ui-monospace, Consolas, monospace;
    color: var(--ink); background: #2a2a2a;
    border: 1px solid var(--rule); border-radius: 4px; padding: 10px;
  }
  .out.shown { display: block; }
  .undrawn { color: var(--dim); font-size: 12px; margin-top: 40px; max-width: 90ch; }
</style>

<header>
  <h1>{{.Catalog}} art, side by side</h1>
  <p class="facts">
    {{len .Sets}} sets ·
    {{range $i, $s := .Sets}}{{if $i}} · {{end}}<code>{{$s.Label}}</code> is <code>{{$s.Dir}}</code>{{end}}
    · {{.Count}} records from <code>{{.File}}</code>{{if .Undrawn}} · {{len .Undrawn}} undrawn{{end}}
  </p>
  <p class="note">
    Every cell is <code>cards.Render</code> at the catalog's own style, the same call the game
    makes, with only the picture differing — so what you are comparing is the card as it will be
    dealt, type and all. Click whichever one you want to keep; the bar counts the picks and writes
    the copy commands, grouped by the set they come from.
  </p>
</header>

<div class="bar">
  <button class="col" data-label="" aria-pressed="true">All sets</button>
  {{range .Labels}}<button class="col" data-label="{{.}}" aria-pressed="false">{{.}} only</button>{{end}}
  <span class="spacer"></span>
  <button id="clear">Clear picks</button>
  <button id="show">Write the copy list</button>
  <span class="tally" id="tally"></span>
</div>
<textarea class="out" id="out" spellcheck="false" readonly></textarea>

{{range .Groups}}
<h2>{{if .Name}}{{.Name}}{{else}}unfiled{{end}}</h2>
<div class="rows">
  {{range .Rows}}
  <div class="row" data-stem="{{.Stem}}">
    <div class="cap"><code>{{.Stem}}</code> · {{.Caption}}</div>
    <div class="cells">
      {{range .Cells}}
      <div class="cell" data-label="{{.Label}}">
        {{if .File}}
        <figure>
          <button class="card" data-label="{{.Label}}" aria-pressed="false">
            <img src="{{.File}}" width="{{.Width}}" height="{{.Height}}" alt="{{.Label}}">
          </button>
          <figcaption>{{.Label}}</figcaption>
        </figure>
        {{else}}<div class="gone">no<br>{{.Label}}<br>picture</div>{{end}}
      </div>
      {{end}}
    </div>
  </div>
  {{end}}
</div>
{{end}}

{{if .Undrawn}}
<p class="undrawn">
  Named no picture in <code>{{.File}}</code>, so there was nothing to compare:
  {{range $i, $k := .Undrawn}}{{if $i}}, {{end}}<code>{{$k}}</code>{{end}}
</p>
{{end}}

<script>
(function () {
  var picks = {};                                  // stem -> set label
  var labels = {{.Labels}};
  var dirs = {}; {{range .Sets}}dirs[{{.Label}}] = {{.Dir}}; {{end}}
  var into = {{.Dir}};
  var total = document.querySelectorAll(".row").length;

  function tally() {
    var counts = {}, picked = 0;
    labels.forEach(function (l) { counts[l] = 0; });
    for (var stem in picks) { counts[picks[stem]]++; picked++; }
    var parts = labels.map(function (l) { return "<b>" + counts[l] + "</b> " + l; });
    parts.push("<span>" + (total - picked) + " undecided</span>");
    document.getElementById("tally").innerHTML = parts.join(" · ");
  }

  document.querySelectorAll(".row").forEach(function (row) {
    row.querySelectorAll(".card").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var stem = row.dataset.stem, label = btn.dataset.label;
        if (picks[stem] === label) { delete picks[stem]; } else { picks[stem] = label; }
        row.querySelectorAll(".card").forEach(function (other) {
          other.setAttribute("aria-pressed", picks[stem] === other.dataset.label ? "true" : "false");
        });
        tally();
      });
    });
  });

  // Showing one set at a time is how a batch is read as a wall — the comparison answers "which
  // of these", and a single column answers "does this set hang together".
  document.querySelectorAll(".col").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var only = btn.dataset.label;
      document.querySelectorAll(".col").forEach(function (other) {
        other.setAttribute("aria-pressed", other.dataset.label === only ? "true" : "false");
      });
      document.querySelectorAll(".cell").forEach(function (cell) {
        cell.classList.toggle("hidden", only !== "" && cell.dataset.label !== only);
      });
    });
  });

  document.getElementById("clear").addEventListener("click", function () {
    picks = {};
    document.querySelectorAll(".card").forEach(function (b) { b.setAttribute("aria-pressed", "false"); });
    tally();
  });

  // The copy list is the point of the review: one line per record, from the set that won it into
  // the catalog's own directory. A pick from the installed set writes no line — it is already
  // there — and is counted in the comment so the arithmetic adds up.
  document.getElementById("show").addEventListener("click", function () {
    var out = document.getElementById("out"), byLabel = {}, kept = 0;
    Object.keys(picks).sort().forEach(function (stem) {
      var label = picks[stem];
      if (dirs[label] === into) { kept++; return; }
      (byLabel[label] = byLabel[label] || []).push(stem);
    });

    var lines = [];
    labels.forEach(function (label) {
      var stems = byLabel[label];
      if (!stems) { return; }
      lines.push("# " + stems.length + " from " + label + " (" + dirs[label] + ")");
      stems.forEach(function (stem) {
        lines.push("Copy-Item '" + dirs[label] + "\\" + stem + ".png' '" + into + "\\" + stem + ".png' -Force");
      });
      lines.push("");
    });
    if (kept) { lines.unshift("# " + kept + " already installed, nothing to copy"); }
    if (!lines.length) { lines.push("# nothing picked"); }

    out.value = lines.join("\n");
    out.classList.add("shown");
    out.select();
  });

  tally();
})();
</script>
`))
