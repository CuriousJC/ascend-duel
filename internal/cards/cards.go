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
	Basic: {R: 150, G: 154, B: 163, A: 255},
	Fire:  {R: 235, G: 120, B: 45, A: 255},
	Ice:   {R: 80, G: 155, B: 230, A: 255},
	// **Darkened on 2026-08-19** from {240,205,55}, which is a fine yellow on a dark ground and
	// nearly invisible on two of the three light ones this game draws on — the off-white card
	// surface and the combat screen's cream. It first showed up as an unreadable damage figure in
	// the hand sum, where the number is drawn straight onto the cream; the border had the same
	// problem more quietly. This is `screens.attentionYellow`'s value, which took the same
	// correction for the same reason when the ground went cream.
	Lightning: {R: 214, G: 152, B: 12, A: 255},
	Earth:     {R: 76, G: 140, B: 52, A: 255},

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

// Form is which group of cards this one belongs to, and it is drawn in the corner where
// the phase glyph used to sit.
//
// **It replaced the category** *(2026-08-15)*. A card used to say which phase it resolved in —
// prepare, attack, defend — and with the deck rebuilt around three attack forms that fact
// became both less useful and derivable: everything in Stab, Slash and Crush is an attack and
// everything in Plan is not. What a card cannot say any other way is *which of the three ways of
// hitting* it is, because that is what a pair is counted on.
//
// It is its own type rather than the string it used to be so the mapping to art is a
// switch the compiler can see, not a lookup that fails quietly on a typo. internal/screens
// converts combat.Form into this.
type Form int

const (
	// FormNone draws nothing. Rings and the two fighter cards use it: they belong to no form.
	FormNone Form = iota
	FormStab
	FormSlash
	FormCrush
	FormPlan
)

var formNames = [...]string{
	FormNone:  "",
	FormStab:  "stab",
	FormSlash: "slash",
	FormCrush: "crush",
	FormPlan:  "plan",
}

func (f Form) String() string {
	if int(f) >= len(formNames) {
		return "?"
	}
	return formNames[f]
}

// Forms is the four real ones, for the contact sheet.
func Forms() []Form {
	return []Form{FormStab, FormSlash, FormCrush, FormPlan}
}

// formGlyphs is the picture each form carries in the card's corner: a spear, a sword, an axe and
// a bulb, which say the form without a legend.
//
// **FormNone is absent on purpose**, so the lookup below reports it as having no glyph — a ring
// and both fighter cards belong to no form, and the slot has to stay empty for them.
var formGlyphs = map[Form]systems.GlyphKind{
	FormStab:  systems.GlyphFormStab,
	FormSlash: systems.GlyphFormSlash,
	FormCrush: systems.GlyphFormCrush,
	FormPlan:  systems.GlyphFormPlan,
}

// glyph is the art for this form, and whether it has any.
func (f Form) glyph() (systems.GlyphKind, bool) {
	k, ok := formGlyphs[f]
	return k, ok
}

// Surface is every card's face, whatever its element. One constant, deliberately: the left
// column — the tinted form mark and the cost ticks under it — is what says which element a card
// is, and a surface that also shifted would be saying it twice and leaving nothing for the next
// distinction.
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

	// NumberInk is a figure: a stat row's value, the health fraction. It used to be the glyph
	// palette's Specular (pure white), which was legible on a coloured surface and is not on
	// this one.
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

	// ValueInk overrides the colour the figure is set in. **Zero alpha means the default**, the
	// same convention TextInk follows, so a caller that never thinks about it gets NumberInk.
	//
	// **It exists because vitae is crimson everywhere it is written** *(2026-08-22)*: the purse on
	// the duelist card is the figure the reward screen makes climb, and it is the same red the
	// word is set in while that screen reads the payout out. The state colouring still applies on
	// top, so a disabled card fades with everything else.
	ValueInk color.RGBA
}

// MaxStatLines is how many stat rows a card can carry.
//
// **It is what the layout fits, not headroom over it.** DuelistStyle's three rows run from
// y=56 to y=137 against a health bar at y=161, and a fourth at that pitch lands on the bar —
// TestStatRowsClearTheHealthBar fails rather than drawing it. So a fourth figure is a layout
// change and the cost of that is charged here on purpose, exactly as a fifth cost tier is by
// TestLeftColumnDoesNotCollide.
const MaxStatLines = 3

// MaxEffects is how many status badges a card can show at once.
//
// **Four, because `statuses.json` holds four and a duelist carries at most one of each.** This
// package does not know that — it holds pictures — so the number is stated here as the width of the
// row the bottom band fits, and checked against the rules by
// `TestTheCardHoldsAsManyEffectsAsThereAreStatuses` in internal/screens, which is the layer that can
// see both.
//
// **It stopped being "one per element" on 2026-08-17**, when statuses became their own collection.
// The two were the same number only because a status *was* an element. Authoring a fifth status is
// therefore a layout change as well as a file edit, exactly as a fourth stat row is — though a cheap
// one: at 20px badges and a 6px gap the row fits six inside the borders, so the fix is this number
// rather than a redesign.
const MaxEffects = 4

// Spec is everything about one card that changes what it looks like.
//
// It is plain data rather than a combat.ActionKind on purpose. The contact sheet renders
// combinations that are not real cards — every border colour against every AP count —
// and a Spec built from the rules could not express those. It also keeps this package
// free of internal/combat, so the only thing it knows about the game is how to draw it.
type Spec struct {
	Name    string
	Form    Form
	Cost    int // action points, drawn as dash marks
	Element Element

	// Text is what the card does, in words, wrapped across the band under the left column.
	//
	// **Every action card carries one** *(2026-08-14)*. Half the deck could not be understood from
	// a card that showed only a name, a cost and a damage figure: what a plan card does is not
	// guessable from its price, and nine attacks on one ladder are told apart by a multiplier that
	// has to be printed. The band it goes in was being held for a long-press description; the
	// text is now printed and long press becomes the gesture that *pulls a card forward* so an
	// overlapped one can be read.
	//
	// **The wording lives with the screen, not here.** internal/cards knows how to set a line
	// of text on a card and nothing about what a card does; the table is in internal/screens
	// beside actionPhrases, and tools/cardsheet keeps its own snapshot exactly as it does for
	// names and costs.
	Text string

	// TextInk overrides the colour Text is set in. **Zero alpha means the default**, which is the
	// convention every other optional colour in this codebase follows — so a caller that never
	// thinks about it gets LabelInk and nothing changes.
	//
	// **It exists so a card can say that something else changed what it does** *(2026-08-21)*: a
	// slash doubled by a worn Keen Ring reads "Slashes for 4x DMG" in the ring pink, because the
	// figure is no longer the one the concept declares. The state colouring still applies on top,
	// so a disabled boosted card fades with everything else rather than staying loud.
	//
	// **This package still does not know what a ring is.** It is handed a colour and a string; the
	// decision that the two go together is `internal/screens`, which is where the wording lives.
	TextInk color.RGBA

	// TextHighlight is the one run of Text that TextInk applies to. **Empty means the whole
	// string**, so a caller wanting a differently coloured line still gets one.
	//
	// It exists because colouring the whole sentence was too loud *(owner's call, 2026-08-21)*: the
	// changed thing on "Slashes for 4x DMG" is the figure, and painting the verb and the unit with
	// it says the ring changed the card rather than the number.
	//
	// **The first occurrence wins, and it is drawn as a run rather than per glyph.** A card's figure
	// is one word — 4x — so the wrapper cannot split it across two lines, which is the case a
	// per-glyph scheme would be needed for. A highlight not found in Text simply does not appear.
	TextHighlight string

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

	// Effects are the status badges drawn in a row along the bottom edge, on the styles that
	// ask for them. Nil entries are skipped, so the row is as wide as the statuses actually
	// standing on the combatant and stays centred as they come and go.
	//
	// **What they mean is none of this package's business.** A caller hands over pictures in
	// the order it wants them read; `internal/screens` fills them from `Duelist.Statuses` in
	// element order, which is what makes the row's order deterministic. Putting the element
	// enum in here would give the drawing package an opinion about the rules.
	//
	// **A fixed array, and comparable for the same reason `Stats` is** — the screen's card
	// cache keys on the whole Spec. `image.Image` compares by dynamic type and pointer, which
	// is exactly what is wanted: the same decoded picture is the same cache entry.
	Effects [MaxEffects]image.Image

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

// borderBase is the colour a card's border is actually drawn from, and it is deliberately
// *not* BorderOf.
//
// **The element moved off the border and onto the form mark on 2026-08-23** *(owner's call)*.
// The border said the element from 2026-08-09, when the surface stopped carrying it; what that
// left was a card whose loudest feature named the one thing about it that a player already knows
// from the row it is sitting in, while the corner mark — the thing a hand is actually counted on
// — was painted in one hueless palette. Swapping them spends the colour on the mark and gives the
// border back to state, which is what borderRestToward and the rest were always about.
//
// **Ring keeps its pink**, because pink was never an element. It is the "this is not something
// you can play" signal, and it has to survive a change that is about elements.
func borderBase(e Element) color.RGBA {
	if e == Ring {
		return borderColors[Ring]
	}
	return borderColors[Basic]
}
