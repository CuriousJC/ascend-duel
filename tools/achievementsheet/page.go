package main

import "html/template"

// The page. One static file, no JavaScript, no build step — the tonic sheet's shape.
var tmpl = template.Must(template.New("achievementsheet").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Duello — achievement sheet</title>
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
    margin: 0;
    padding: 32px 28px 64px;
    background: var(--ground);
    color: var(--ink);
    font: 14px/1.5 -apple-system, "Segoe UI", system-ui, sans-serif;
  }
  h1 { font-size: 20px; margin: 0 0 4px; font-weight: 600; }
  .facts { color: var(--dim); font-size: 12px; margin: 0 0 8px; }
  .note { color: var(--dim); font-size: 12.5px; max-width: 72ch; margin: 12px 0 0; }
  .plates { display: flex; flex-direction: column; gap: 20px; margin-top: 26px; }
  .plate {
    display: flex; gap: 22px; align-items: flex-start;
    background: var(--panel); border: 1px solid var(--rule); border-radius: 8px; padding: 16px;
  }
  .icons { display: grid; grid-template-columns: auto auto; gap: 8px 12px; align-items: end; }
  .icons figure { margin: 0; }
  .icons figcaption { font-size: 11px; color: var(--dim); }
  img { display: block; }
  .about { min-width: 0; flex: 1; }
  .name { font-size: 15px; font-weight: 600; margin: 0 0 2px; }
  table.steam { border-collapse: collapse; margin-top: 8px; font-size: 12.5px; }
  table.steam th { text-align: left; color: var(--dim); font-weight: 600; padding: 2px 14px 2px 0; }
  table.steam td { padding: 2px 0; }
  code, .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11.5px; }
  .said { margin: 10px 0 0; font-size: 12.5px; font-style: italic; }
  .said p { margin: 0; }
  .rule { margin: 10px 0 0; color: var(--dim); word-break: break-word; }
  .art { margin: 10px 0 0; font-size: 11.5px; color: var(--dim); }
  .art.missing { color: var(--pink); }
  .draw {
    margin: 10px 0 0; font-size: 12.5px; color: var(--dim);
    border-left: 2px solid var(--rule); padding-left: 9px;
  }
  .draw.missing { color: var(--pink); border-left-color: var(--pink); }
</style>

<h1>Achievement sheet</h1>
<p class="facts">
  {{.Count}} achievements, {{.Undrawn}} of them drawing the default icon, {{.Unwritten}} with no
  subject paragraph written, {{.Hidden}} hidden.
</p>
<p class="note">
  Regenerate with <code>go run ./tools/achievementsheet</code> and refresh. Every field is read out
  of <code>data/achievements.json</code> through <code>internal/achieve</code>'s own validation.
</p>
<p class="note">
  <strong>The fields under each name are Steamworks' own</strong>, under the names its admin page
  uses, so this page is what that page is filled in from. <strong>The two {{.Size}}px icons are the
  upload files</strong>: <code>achieved-*.png</code> is the Achieved Icon and
  <code>unachieved-*.png</code> the Unachieved Icon. The gray one is derived, by
  <code>systems.ArtMarkGray</code> — the same call that draws a locked row in the game.
</p>
<p class="note">
  <strong>Judge an icon at {{.Shown}}px, not at {{.Size}}.</strong> That is the size Steam's pop-up
  and the game's achievements page both draw it, and the small pair is reduced by the game's own
  averaging rather than by the browser.
</p>

<div class="plates">
  {{range .Plates}}
    <div class="plate">
      <div class="icons">
        <figure><img src="{{.Achieved}}" width="{{$.Size}}" height="{{$.Size}}" alt="{{.DisplayName}}"><figcaption>achieved, {{$.Size}}</figcaption></figure>
        <figure><img src="{{.Unachieved}}" width="{{$.Size}}" height="{{$.Size}}" alt="{{.DisplayName}}, unachieved"><figcaption>unachieved, {{$.Size}}</figcaption></figure>
        <figure><img src="{{.Achieved64}}" width="{{$.Shown}}" height="{{$.Shown}}" alt=""><figcaption>{{$.Shown}}</figcaption></figure>
        <figure><img src="{{.Unachieved64}}" width="{{$.Shown}}" height="{{$.Shown}}" alt=""><figcaption>{{$.Shown}}</figcaption></figure>
      </div>
      <div class="about">
        <p class="name">{{.DisplayName}}</p>
        <table class="steam">
          <tr><th>API Name</th><td class="mono">{{.APIName}}</td></tr>
          <tr><th>Display Name</th><td>{{.DisplayName}}</td></tr>
          <tr><th>Description</th><td>{{.Description}}</td></tr>
          <tr><th>Set By</th><td>{{.SetBy}}</td></tr>
          <tr><th>Hidden?</th><td>{{if .Hidden}}yes{{else}}no{{end}}</td></tr>
        </table>
        <div class="said">{{range .Said}}<p>{{.}}</p>{{end}}</div>
        {{if .Unlocks}}<p class="rule mono">unlocks: {{.Unlocks}}</p>{{end}}
        <p class="rule mono">{{.Trigger}}</p>
        {{if .Draw}}
          <p class="draw">{{.Draw}}</p>
        {{else}}
          <p class="draw missing">no subject written yet</p>
        {{end}}
        {{if .Icon}}
          <p class="art">icon: <code>{{.Icon}}</code></p>
        {{else}}
          <p class="art missing">no icon of its own — drawing the default</p>
        {{end}}
      </div>
    </div>
  {{end}}
</div>
`))
