package main

import "html/template"

// page is what the template is handed. Filters is the chip bar from tools/sheetfilter, which
// narrows what is already in the file — nothing is fetched and nothing is templated in the
// browser, so the page reads whole with scripting off.
type page struct {
	Categories []category
	Total      int
	Filters    template.HTML
}

// The page. Deliberately the plainest sheet here: a heading per category and a tile per file,
// each tile the picture over its filename. There is no card, no stat line and no rule text,
// because the question is "what is this file called and what does it look like" — anything more
// is the targeted sheet's job and is one click away from each heading.
//
// **Every picture is loaded lazily.** assets/ holds several hundred creature portraits, and a
// page that decodes all of them at once takes seconds to open on the one axis it is meant to be
// quick on.
var tmpl = template.Must(template.New("artsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Ascending Duel — all art</title>
<style>
  :root { --ground: #a8bcd4; --ink: #2c2822; --dim: #5a6472; --rule: #8fa3bd; --panel: #c6d5e6; }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 32px 24px 64px; background: var(--ground); color: var(--ink);
    font: 15px/1.6 -apple-system, "Segoe UI", system-ui, sans-serif;
  }
  .wrap { max-width: 1400px; margin: 0 auto; }
  h1 { font-size: 24px; margin: 0 0 6px; font-weight: 600; }
  p.note { color: var(--dim); font-size: 13px; margin: 0 0 6px; max-width: 760px; }
  a { color: inherit; }
  h2 { font-size: 18px; font-weight: 600; margin: 0; }
  .head {
    display: flex; flex-wrap: wrap; align-items: baseline; gap: 10px;
    margin: 34px 0 12px; padding-bottom: 6px; border-bottom: 1px solid var(--rule);
  }
  .head .count { color: var(--dim); font-size: 13px; }
  .head .note { color: var(--dim); font-size: 13px; }
  .head .sheet { margin-left: auto; font-size: 13px; }
  .grid { display: flex; flex-wrap: wrap; gap: 10px; }
  .tile {
    width: 168px; padding: 8px; border-radius: 8px;
    background: var(--panel); border: 1px solid var(--rule);
    display: flex; flex-direction: column; align-items: center; gap: 6px;
  }
  .tile .shot {
    width: 100%; height: 150px; display: flex; align-items: center; justify-content: center;
    background:
      linear-gradient(45deg, #9caec4 25%, transparent 25%, transparent 75%, #9caec4 75%),
      linear-gradient(45deg, #9caec4 25%, #b6c6da 25%, #b6c6da 75%, #9caec4 75%);
    background-size: 14px 14px; background-position: 0 0, 7px 7px;
    border-radius: 5px;
  }
  .tile img { max-width: 100%; max-height: 150px; image-rendering: auto; }
  .tile .name {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px;
    color: var(--ink); word-break: break-all; text-align: center; line-height: 1.35;
  }
  code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12.5px; }
</style>

<div class="wrap">
  <h1>All art</h1>
  <p class="note">
    Every picture in <code>assets/</code>, grouped by the directory it lives in, at its own size
    with its filename under it. This page writes no images of its own — each tile points straight
    at the committed file, so it cannot go stale and costs nothing to regenerate.
  </p>
  <p class="note">
    It is for finding a file, not for judging one. A picture is drawn under type on the card it
    ends up on, and the sheet linked beside each heading is where that can be seen.
    <strong>{{.Total}} pictures.</strong>
  </p>

  {{.Filters}}

  {{range $c := .Categories}}
    <section class="sheet-group" data-category="{{.Key}}">
      <div class="head">
        <h2>{{.Label}}</h2>
        <span class="count">{{len .Files}}</span>
        {{if .Note}}<span class="note">{{.Note}}</span>{{end}}
        {{if .Sheet}}<span class="sheet"><a href="{{.Sheet}}">review sheet &rarr;</a></span>{{end}}
      </div>
      <div class="grid">
        {{range .Files}}
          <figure class="tile sheet-item" data-category="{{$c.Key}}">
            <span class="shot"><img src="{{.Src}}" alt="{{.Name}}" loading="lazy"></span>
            <figcaption class="name">{{.Name}}</figcaption>
          </figure>
        {{end}}
      </div>
    </section>
  {{end}}
</div>
`))
