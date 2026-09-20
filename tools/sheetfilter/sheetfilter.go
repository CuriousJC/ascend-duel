// Package sheetfilter is the chip bar a review sheet narrows itself with: a row of toggles above
// the page, and the rule that decides what stays on screen when some of them are down.
//
// # Why a sheet needs one
//
// Every page under docs/sheets/ shows a whole catalog, grouped on the one axis its review is
// conducted along — relics by family, the roster by motif, stones by axis. That is the right
// default and it is not the only question asked of a catalog: "what does floor 1 actually field"
// and "show me the rares together" are both questions about a slice, and the answer today is
// scrolling past everything else.
//
// # What it is, and what it may not become
//
// **A facet's values come off the data, never out of the tool.** A sheet counts what its own
// records carry and hands the list over; a list typed in here would be a second opinion about a
// catalog, which is the stale-sheet failure one layer up.
//
// **The page is complete before any of this runs.** Filtering is additive: nothing is fetched,
// nothing is templated client-side, and a browser with scripting off shows the whole catalog
// exactly as it did before. That is what keeps a sheet a static file — the cost here is script in
// the page, never a build step.
//
// # The contract with a page
//
// A page marks up two kinds of element and tags them with `data-` attributes named after the
// facet keys:
//
//   - `.sheet-item` is one record — a relic's plate, a creature's plate.
//   - `.sheet-group` is a heading and the records under it, which needs a wrapping element.
//
// An attribute holds a space-separated token list, so a record answering several values of one
// facet — a creature dealt as four elements — carries all of them. **An element with no attribute
// for a facet is not filtered by it**, which is what lets a group be cut on the floor band while
// the plates inside it are cut on tier.
//
// Selected values within one facet are an OR and facets are an AND, which is what makes "floor 1,
// boss, fire or ice" one reading. A group survives if it matches on its own attributes and still
// holds a visible record.
package sheetfilter

import (
	"fmt"
	"html/template"
	"strings"
)

// Facet is one axis a page can be narrowed on.
type Facet struct {
	// Key is the attribute the page tags its elements with, without the `data-` — "floor" reads
	// `data-floor`. Lowercase, no spaces.
	Key string

	// Label is what the row of chips is called.
	Label string

	// Values are the chips, in the order they should be offered. That order is the page's to
	// decide: a floor is ascending, a rarity is cheapest-first, and neither is alphabetical.
	Values []Value
}

// Value is one chip.
type Value struct {
	// Value is the token matched against the page's `data-` attributes.
	Value string

	// Label is what the chip reads. Empty takes Value.
	Label string

	// Count is how many records answer this value, drawn small beside the label. Zero is drawn as
	// no count rather than as "0", since a facet may be offered without one.
	Count int
}

// Bar renders the whole widget — its styles, the chips, and the script that applies them — as one
// block a page drops in above its contents.
//
// **One insertion point rather than three** so a page cannot take the chips and forget the script,
// which is a filter bar that silently does nothing.
func Bar(facets []Facet) template.HTML {
	if len(facets) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(style)
	b.WriteString(`<div class="sheet-filters" hidden>` + "\n")
	for _, f := range facets {
		if len(f.Values) == 0 {
			continue
		}
		fmt.Fprintf(&b, `  <div class="sheet-facet"><span class="sheet-facet-label">%s</span>`+"\n",
			template.HTMLEscapeString(f.Label))
		for _, v := range f.Values {
			label := v.Label
			if label == "" {
				label = v.Value
			}
			fmt.Fprintf(&b, `    <button type="button" data-facet="%s" data-value="%s" aria-pressed="false">%s`,
				template.HTMLEscapeString(f.Key),
				template.HTMLEscapeString(v.Value),
				template.HTMLEscapeString(label))
			if v.Count > 0 {
				fmt.Fprintf(&b, `<i>%d</i>`, v.Count)
			}
			b.WriteString("</button>\n")
		}
		b.WriteString("  </div>\n")
	}
	b.WriteString(`  <div class="sheet-filters-end">` + "\n")
	b.WriteString(`    <button type="button" class="sheet-clear" hidden>clear</button>` + "\n")
	b.WriteString(`    <span class="sheet-tally"></span>` + "\n")
	b.WriteString("  </div>\n</div>\n")
	b.WriteString(script)
	return template.HTML(b.String())
}

// style is the bar's own CSS.
//
// **It names the same custom properties every sheet already defines** — --ink, --dim, --rule,
// --panel — so the bar takes each page's colors rather than carrying a palette of its own, and a
// sheet that retunes its ground brings the bar with it. A page that has not defined them still
// gets a legible bar out of the fallbacks.
const style = `<style>
  .sheet-filters {
    position: sticky; top: 0; z-index: 5;
    margin: 14px 0 0; padding: 10px 12px;
    display: flex; flex-wrap: wrap; align-items: center; gap: 8px 18px;
    background: var(--panel, #c6d5e6);
    border: 1px solid var(--rule, #8fa3bd); border-radius: 8px;
  }
  .sheet-facet { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
  .sheet-facet-label {
    color: var(--dim, #5a6472); font-size: 11px; font-weight: 600;
    text-transform: uppercase; letter-spacing: .06em; margin-right: 2px;
  }
  .sheet-filters button {
    font: inherit; font-size: 12.5px; line-height: 1;
    padding: 5px 9px; border-radius: 999px; cursor: pointer;
    color: var(--ink, #2c2822);
    background: transparent;
    border: 1px solid var(--rule, #8fa3bd);
  }
  .sheet-filters button:hover { background: rgba(255,255,255,.45); }
  /* A chip that is down is filled rather than merely outlined: an outline shifting shade is the
     one state change that does not survive being looked at quickly. */
  .sheet-filters button[aria-pressed="true"] {
    background: var(--ink, #2c2822); color: var(--panel, #c6d5e6);
    border-color: var(--ink, #2c2822);
  }
  .sheet-filters button i {
    font-style: normal; opacity: .6; margin-left: 6px;
    font-variant-numeric: tabular-nums;
  }
  .sheet-filters-end { margin-left: auto; display: flex; align-items: center; gap: 12px; }
  .sheet-tally { color: var(--dim, #5a6472); font-size: 12px; font-variant-numeric: tabular-nums; }
</style>
`

// script is the whole behavior.
//
// **The bar is written hidden and this is what shows it**, so a page opened with scripting off
// does not offer controls that cannot work. Everything else here is the matching rule the package
// comment states, plus a tally, because a filter that hides most of a page should say how much.
//
// **It waits for the document before it counts anything.** The bar sits above the contents it
// filters, so a script that ran where it stands would collect its records out of a page the
// parser has not reached yet — chips that toggle over two empty lists, which is a filter that
// looks built and does nothing.
const script = `<script>
(function () {
  function start() {
    var bar = document.querySelector('.sheet-filters');
    if (!bar) return;
    bar.hidden = false;

    var chips = Array.prototype.slice.call(bar.querySelectorAll('button[data-facet]'));
    var clear = bar.querySelector('.sheet-clear');
    var tally = bar.querySelector('.sheet-tally');
    var items = Array.prototype.slice.call(document.querySelectorAll('.sheet-item'));
    var groups = Array.prototype.slice.call(document.querySelectorAll('.sheet-group'));
    var picked = {};

    function tokens(el, key) {
      var v = el.getAttribute('data-' + key);
      if (v === null) return null;
      return v.split(/\s+/).filter(Boolean);
    }

    function matches(el) {
      for (var key in picked) {
        var want = picked[key];
        if (!want.length) continue;
        var have = tokens(el, key);
        if (have === null) continue;
        var hit = false;
        for (var i = 0; i < have.length; i++) {
          if (want.indexOf(have[i]) !== -1) { hit = true; break; }
        }
        if (!hit) return false;
      }
      return true;
    }

    function apply() {
      var shown = 0, any = false;
      for (var key in picked) { if (picked[key].length) any = true; }

      var open = [];
      for (var g = 0; g < groups.length; g++) open.push(matches(groups[g]));

      for (var i = 0; i < items.length; i++) {
        var it = items[i];
        var owner = groups.indexOf(it.closest('.sheet-group'));
        var ok = matches(it) && (owner === -1 || open[owner]);
        it.hidden = !ok;
        if (ok) shown++;
      }

      for (var g2 = 0; g2 < groups.length; g2++) {
        var grp = groups[g2];
        var held = grp.querySelector('.sheet-item');
        var live = grp.querySelector('.sheet-item:not([hidden])');
        grp.hidden = !open[g2] || (held !== null && live === null);
      }

      if (clear) clear.hidden = !any;
      if (tally) tally.textContent = any ? 'showing ' + shown + ' of ' + items.length : '';
    }

    chips.forEach(function (chip) {
      chip.addEventListener('click', function () {
        var key = chip.getAttribute('data-facet');
        var val = chip.getAttribute('data-value');
        var list = picked[key] || (picked[key] = []);
        var at = list.indexOf(val);
        if (at === -1) { list.push(val); } else { list.splice(at, 1); }
        chip.setAttribute('aria-pressed', at === -1 ? 'true' : 'false');
        apply();
      });
    });

    if (clear) {
      clear.addEventListener('click', function () {
        picked = {};
        chips.forEach(function (c) { c.setAttribute('aria-pressed', 'false'); });
        apply();
      });
    }

    apply();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', start);
  } else {
    start();
  }
})();
</script>
`
