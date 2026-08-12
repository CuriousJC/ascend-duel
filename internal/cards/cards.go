// Package cards draws an action card to a plain Go image.
//
// **It creates no Ebitengine images**, which is the whole reason it exists as its own
// package. `systems.RenderGlyph` already established the pattern — art that returns an
// `image.Image` can be rendered by the game *and* by a command-line tool with no window,
// so a card can be reviewed by opening a file instead of by launching the game and
// dealing yourself the hand that shows it. `tools/cardsheet` is that tool.
//
// The important consequence is that the sheet cannot lie. Both callers run this same
// code and get the same pixels; there is no second drawing path to drift out of sync,
// which is what went wrong with the first version of the glyph sheet when it previewed
// glyphs at a scale the cards did not use.
//
// # What a card looks like, and what changed
//
// The surface is a constant off-white for every card and **a thick coloured border
// carries the element**. That reverses the decision recorded in CLAUDE.md's colour
// section on 2026-08-03, where the surface was the element and the border was
// incidental. Two things follow from the reversal:
//
//   - **A near-white border on an off-white card is invisible.** The elementless card
//     used to be near-white surface and that read fine; as a border it would vanish.
//     BorderOf therefore gives Basic a mid grey rather than the {235,235,235} the screen
//     uses for its surface today.
//   - **The glyphs are hueless near-white by design.** Their Specular is pure white and
//     their Highlight is {232,236,242}, so on an off-white surface the lit side of every
//     bevel largely disappears and a glyph reads as outline-plus-shading. The near-black
//     Outline carries legibility, so nothing is broken, but the bevel that glyphs.go
//     exists to produce is mostly spent. Surface is a named constant so this is one edit
//     to test.
//
// # Rounded corners are rasterised here, not masked
//
// CLAUDE.md points at `CreateRoundedRecMask` + `ebiten.BlendSourceIn` as the repo's one
// rounding approach. **That approach cannot be used here**: it takes an `*ebiten.Image`,
// its body is `vector.DrawFilledCircle`, and `BlendSourceIn` is a GPU blend mode — none
// of which exist without a graphics context. Being window-free is the requirement that
// makes the review tool possible at all, so it wins, and corners are rasterised in plain
// Go below.
//
// This does leave two rounding approaches in the tree: cards here, health bars still on
// the mask-and-blend path in `combat_hud.go`. That is a real inconsistency rather than an
// oversight. Migrating health bars onto this one is possible later and is not this
// change.
//
// Corners are hard-edged — no antialiasing — because the cards were already drawn that
// way (`vector.DrawFilledRect(..., false)`) and because the glyphs sitting on them are
// 1:1 pixel art with a one-pixel rim. An antialiased card edge around pixel-art contents
// reads as two different pictures.
package cards

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// Element is what a card's border says it is. It mirrors the `element` type in
// internal/screens rather than importing it: that type is expected to move into
// internal/combat when elements stop being decoration (see combat_deck.go), and a
// drawing package should not be in the way of that move. `internal/screens` maps its
// element onto this one in card_art.go, and a test there pins the mapping.
type Element int

const (
	Basic Element = iota

	// The four primaries, which is all of them. `data/cards.json` deals
	// 12 concepts x 5 borders (these four plus Basic) = the 60-card deck.
	//
	// **Poison was here and is gone.** It was a secondary element that had never had a
	// card — `cards.json` contains no poison at all — so it only ever appeared as a
	// sixth row on the contact sheet showing a colour nothing could deal. The screen's
	// own `element` enum still has it, because that is game design recorded in
	// MECHANICS.md rather than art; it maps to Basic here.
	Fire
	Ice
	Lightning
	Earth

	// Ring is not an element and never appears on an action card. Rings reuse the whole
	// card format — same size, same corners, same border treatment — with a pink border
	// and artwork instead of glyphs, so they read as belonging to the same game while
	// never being mistaken for something playable from the hand.
	//
	// It is deliberately outside Elements(): anything iterating the elements is asking
	// about cards, and a ring is not one.
	Ring
)

// Elements is every element a card can have, in a fixed order, for the contact sheet and
// for anything else that iterates them. A slice rather than a map, because Go randomises
// map order and a sheet whose rows moved between runs would be useless as a diff.
//
// Ring is not in it. See the constant.
func Elements() []Element {
	return []Element{Basic, Fire, Ice, Lightning, Earth}
}

var elementNames = [...]string{
	Basic:     "basic",
	Fire:      "fire",
	Ice:       "ice",
	Lightning: "lightning",
	Earth:     "earth",
	Ring:      "ring",
}

func (e Element) String() string {
	if int(e) >= len(elementNames) {
		return "?"
	}
	return elementNames[e]
}

// borderColors is the element signal, now that the surface no longer carries it.
//
// These are the screen's existing element colours with one deliberate exception: Basic.
// As a surface, near-white meant "this card makes no claim" and worked. As a border on
// an off-white card it would be invisible, so it becomes a mid grey — still the quietest
// of the set, still obviously the absence of an element, but actually a border.
var borderColors = [...]color.RGBA{
	Basic:     {R: 150, G: 154, B: 163, A: 255},
	Fire:      {R: 235, G: 120, B: 45, A: 255},
	Ice:       {R: 80, G: 155, B: 230, A: 255},
	Lightning: {R: 240, G: 205, B: 55, A: 255},
	Earth:     {R: 150, G: 105, B: 60, A: 255},

	// Pink, and deliberately unlike any of the four above — a ring has to be
	// unmistakable at a glance, because the one thing that must never happen is reaching
	// for a ring thinking it is a card you can play.
	Ring: {R: 232, G: 106, B: 168, A: 255},
}

// BorderOf is the colour this element's border is drawn in at full strength. States
// scale it down; see Render.
func BorderOf(e Element) color.RGBA {
	if int(e) >= len(borderColors) {
		return borderColors[Basic]
	}
	return borderColors[e]
}

// Category is which phase a card resolves in, and it is drawn as a glyph rather than
// written as a word.
//
// It is its own type rather than the string it used to be so the mapping to art is a
// switch the compiler can see, not a lookup that fails quietly on a typo. internal/screens
// converts combat.Category into this.
type Category int

const (
	// CategoryNone draws nothing. Rings use it: they have no phase.
	CategoryNone Category = iota
	CategoryPrepare
	CategoryAttack
	CategoryDefend
)

var categoryNames = [...]string{
	CategoryNone:    "",
	CategoryPrepare: "prepare",
	CategoryAttack:  "attack",
	CategoryDefend:  "defend",
}

func (c Category) String() string {
	if int(c) >= len(categoryNames) {
		return "?"
	}
	return categoryNames[c]
}

// Categories is the three real ones, for the contact sheet.
func Categories() []Category {
	return []Category{CategoryPrepare, CategoryAttack, CategoryDefend}
}

// glyph is the art for this category.
//
// Attack shares the sword with the damage badge rather than getting a second weapon:
// they say related things, a card shows only one of them in the category slot, and two
// different swords on one face would imply a distinction that does not exist.
func (c Category) glyph() (systems.GlyphKind, bool) {
	switch c {
	case CategoryPrepare:
		return systems.GlyphPrepare, true
	case CategoryAttack:
		return systems.GlyphAttack, true
	case CategoryDefend:
		return systems.GlyphDefend, true
	default:
		return 0, false
	}
}

// Surface is every card's face, whatever its element. One constant, deliberately: the
// border is now the only thing saying which element a card is, and a surface that also
// shifted would be saying it twice and leaving nothing for the next distinction.
//
// It is off-white rather than white so the card reads as a card rather than as a hole in
// the screen, and so a pure-white glyph Specular still has somewhere to go.
var Surface = color.RGBA{R: 240, G: 239, B: 234, A: 255}

// The back of a card: a dark face with a pale mark centred on it, and nothing else. It
// says "a card, and you may not see which" — the draw pile is shuffled, so a back that
// carried an element or a category would leak the very thing the shuffle protects.
//
// BackSurface is near-black rather than pure black for the same reason Surface is
// off-white rather than white: a card has to read as an object on the screen and not as a
// hole cut in it. BackInk is pure white, the one place on a card that colour is spent on
// nothing but contrast.
// BackRim is a thin neutral edge around the back.
//
// **It is what makes a pile read as a pile.** Three backs offset by a few pixels are three
// near-black shapes on a dark screen, and without an edge they merge into one card with a
// lopsided corner — which is exactly how the first version drew, and the reason this exists.
//
// Neutral rather than the element colour, which keeps the rule the back is built on: the
// border is where a card says which card it is, so a back may have an edge but never a
// *coloured* one.
var (
	BackSurface = color.RGBA{R: 14, G: 14, B: 18, A: 255}
	BackInk     = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	BackRim     = color.RGBA{R: 96, G: 98, B: 108, A: 255}
)

// BackMark is which shape a card back carries: **whose deck this is**, not what the card is.
//
// **Named designs drawn in code rather than a recolour or a picture** *(2026-08-11)*. A
// recolour would have been one data field and nearly invisible on a near-black card at the
// draw pile's 44 pixels; a picture per duelist would put the provenance question back on the
// one part of the game that had escaped it. A silhouette is what reads at that size and what
// can be generated, which is the same argument the glyphs are built on.
//
// The cost is deliberate and worth stating: **adding a back is a code change**, not a data
// one. `data/duelists.json` can only choose among what is drawn here.
//
// **Append, never insert.** The screen's card cache keys on the Spec, so these ordinals are
// part of a cache key — the same hazard GlyphKind carries.
type BackMark int

const (
	// MarkTriangle is the zero value and what every card had before backs were named, so a
	// Spec that says nothing about its back draws what the game shipped with.
	MarkTriangle BackMark = iota
	MarkDiamond
	MarkChevron
)

var backMarkNames = [...]string{
	MarkTriangle: "triangle",
	MarkDiamond:  "diamond",
	MarkChevron:  "chevron",
}

func (m BackMark) String() string {
	if int(m) >= len(backMarkNames) {
		return "?"
	}
	return backMarkNames[m]
}

// BackMarks is every mark, in a fixed order, for the contact sheet. A slice rather than a
// map: Go randomises map order and a sheet whose rows moved between runs is useless as a
// diff.
func BackMarks() []BackMark { return []BackMark{MarkTriangle, MarkDiamond, MarkChevron} }

// ParseBackMark resolves the name in `data/duelists.json`.
//
// **It falls back to the triangle rather than failing**, and reports that it did. A back is
// cosmetic — the wrong shape on a pile is a bug worth a log line, and refusing to start a
// duel over one would be a worse outcome than the bug. Same reasoning as
// combat.ParsePlanStyle falling back to brute.
func ParseBackMark(name string) (BackMark, bool) {
	for i, n := range backMarkNames {
		if n == name {
			return BackMark(i), true
		}
	}
	return MarkTriangle, false
}

// Ink colours. Hueless on purpose — the border is carrying the only colour on the face.
var (
	// NameInk is the concept's name across the top, the thing read first.
	NameInk = color.RGBA{R: 28, G: 30, B: 36, A: 255}

	// NumberInk is the damage figure beside its glyph. It used to be the glyph palette's
	// Specular (pure white), which was legible on a coloured surface and is not on this
	// one.
	NumberInk = color.RGBA{R: 40, G: 43, B: 52, A: 255}

	// LabelInk is the word half of a stat row — "DMG", "AP", "Vitae". Quieter than the
	// figure beside it, because the figure is what is read and the label is what says
	// which figure it is. Colour is what carries that hierarchy here; the two halves share
	// a baseline, so they cannot differ in size without reading as a mistake.
	LabelInk = color.RGBA{R: 108, G: 112, B: 124, A: 255}
)

// StatLine is one labelled figure on a card face: a word on the left, a number on the
// right, both on the same baseline.
//
// Both halves are strings, so this package never has to know that "12 / 40" is a fraction
// and "6" is a budget. Formatting a figure is the caller's business, exactly as the card's
// name is.
type StatLine struct {
	Label string
	Value string
}

// MaxStatLines is how many stat rows a card can carry.
//
// **It is what the layout fits, not headroom over it.** DuelistStyle's three rows run from
// y=56 to y=137 against a health bar at y=161, and a fourth at that pitch lands on the bar —
// TestStatRowsClearTheHealthBar fails rather than drawing it. So a fourth figure is a layout
// change and the cost of that is charged here on purpose, exactly as a fifth cost tier is by
// TestLeftColumnDoesNotCollide.
const MaxStatLines = 3

// Spec is everything about one card that changes what it looks like.
//
// It is plain data rather than a combat.ActionKind on purpose. The contact sheet renders
// combinations that are not real cards — every border colour against every AP count —
// and a Spec built from the rules could not express those. It also keeps this package
// free of internal/combat, so the only thing it knows about the game is how to draw it.
type Spec struct {
	Name     string
	Category Category
	Damage   int // omitted entirely when zero: a sword reading 0 is worse than no sword
	Cost     int // action points, drawn as dash marks
	Element  Element

	// Art is optional artwork drawn on the face, scaled to fit and centred. Rings use
	// it; action cards do not, and their art is the generated glyphs instead.
	//
	// **This is the one thing on a card that is not generated**, so it is the one thing
	// with a provenance question. Anything put here needs a licence that survives a paid
	// release — see CLAUDE.md on the Tyrian set.
	Art image.Image

	// Stats are the labelled figures the duelist card carries — DMG, AP, Vitae — drawn one
	// per row by the styles that ask for them. An entry with neither a label nor a value
	// leaves its row blank rather than closing the gap, so a row is always at the height
	// the style says it is.
	//
	// **A fixed array rather than a slice, and that is load-bearing.** internal/screens
	// keys its card cache on the whole Spec, so a Spec has to stay comparable; a slice
	// here would not compile at the map lookup, and the fix at that point would be a
	// hand-written key that could disagree with what is actually drawn. See MaxStatLines
	// for why the size is what the card fits rather than a round number.
	Stats [MaxStatLines]StatLine

	// Life and MaxLife draw a health bar and a "42/60" line, on the styles that ask for
	// it. The enemy and duelist cards do — see EnemyStyle and DuelistStyle.
	//
	// **They are on the Spec rather than drawn over the card by the screen**, which means a
	// point of damage is a different Spec and therefore a different cache entry. That is
	// affordable because life only changes on a damage event — a handful of times a round,
	// not per frame — and it is what keeps the whole card in one place: the contact sheet
	// draws a wounded enemy without the tool having to reimplement a bar.
	Life, MaxLife int

	// Enabled is whether the fighter can currently afford it. Disabled reads as
	// unavailable first and as itself second.
	Enabled bool

	// Selected is queued for this round. The hand also lifts a selected card out of the
	// row, which is a layout concern and not this package's.
	Selected bool

	// Dragging is being carried by the cursor right now.
	//
	// It is its own state rather than a reuse of Enabled because it means the opposite
	// thing: a disabled card is one you cannot act on, and a dragged card is the one you
	// are acting on. Rendering them the same way would be the same mistake as dimming a
	// border on a light card. The hand's drag-to-reorder will want this; the ring on the
	// contact sheet is the first thing to use it.
	Dragging bool

	// Back is which mark this card's back carries, when FaceDown is set.
	//
	// **A duelist and a card back go together** *(2026-08-11)*: the plan is to offer
	// different duelists as different decks, and the back is how you tell at a glance whose
	// deck is on the table. It is named in `data/duelists.json` and parsed by ParseBackMark.
	//
	// The zero value is the triangle, which is what every card had before this existed — so
	// a Spec that says nothing about its back still draws the back the game shipped with.
	Back BackMark

	// FaceDown draws the back instead of the face, and every other field is ignored except
	// Back.
	//
	// **A field rather than a separate RenderBack, so the cache does not need to learn
	// about it.** internal/screens keys its cache on the Spec, so a face-down card is
	// cached by exactly the machinery that caches every other card — one entry per style,
	// since nothing else about the Spec reaches the drawing. A second entry point would
	// have needed a second key, and the two could then disagree about what a card is.
	//
	// It is what the draw pile is a stack of, and what a card shows during the first half
	// of a flip.
	FaceDown bool
}
