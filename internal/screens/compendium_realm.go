package screens

// **The Compendium's realm book: every motif in every element it can theme, room by room.**
//
// A realm is a motif and an element — see MECHANICS.md §A realm is a motif and an element — and it
// has three rooms, outer, inner and portal, so the book holds one entry per realm per room, motif by
// motif in the roster's order, each in the affinity order, each room in the order they are walked.
// **Its picture is the room**, the motif's first backdrop of that tier in that element, reduced to a
// landscape thumbnail rather than squeezed onto a card: a player who has stood in it knows it on
// sight. A found room with no picture painted yet draws the default backdrop darkened, on the
// catalogs' found-but-undrawn rule; an unfound one draws the portal swirl and nothing of its room.
//
// **A room is found once any of its tier's creatures has been fought in its colour**, won or lost,
// off the bestiary's own per-element families — so the two books cannot disagree about whether a
// player has been somewhere. **Nothing is set under the picture**: resting on a found one opens the
// room large, with the realm and the room, what the portal said about the realm, how often the room
// has been fought in and how often won — see compendium_room.go.

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The thumbnail: the size that stands four across and five down between the filter columns, at
// the room's own 16:9.
const (
	realmThumbW = 290
	realmThumbH = 163

	// realmThumbsPerFrame is how many rooms are decoded in one frame. **A room is a full-size JPEG**,
	// so a page of them decoded at once is a visible hitch on opening the book; one a frame fills the
	// page in a fraction of a second and the rest of the screen keeps moving.
	realmThumbsPerFrame = 1
)

// realmThumbs holds each room reduced, by asset key. **Only the thumbnail is kept**, never the decoded
// room, which is eight megabytes a picture.
var realmThumbs = map[string]*ebiten.Image{}

// realmRooms is a realm's three rooms in the order they are walked, each with the word its chip
// and its tooltip use. The portal room is the boss tier's.
var realmRooms = []struct{ tier, label string }{
	{data.TierOuter, "OUTER"},
	{data.TierInner, "INNER"},
	{data.TierBoss, "PORTAL"},
}

// roomLabel is the chip word for a tier: which room of a realm it is, or which room a creature
// stands in.
func roomLabel(tier string) string {
	for _, r := range realmRooms {
		if r.tier == tier {
			return r.label
		}
	}
	return ""
}

// realmsDialog is the realm book.
func realmsDialog(gs *state.GlobalState) *compendiumDialog {
	all := everything(gs)
	var out []compendiumEntry
	for _, key := range data.MotifOrder(gs.Motifs) {
		motif := gs.Motifs[key]
		for _, element := range data.AffinityElements {
			if !dealsIn(motif, element) {
				continue
			}
			name := elementTitle(element) + " " + motifName(motif)
			about, inElement := motif.PortalLines(element)
			for _, room := range realmRooms {
				beaten, fell := 0, 0
				for _, r := range motif.Records {
					if r.Tier != room.tier || !r.HasAffinity(element) {
						continue
					}
					beaten += all.Total(session.CreatureAs(session.StatCreatureDefeatedAs, r.Record, element))
					fell += all.Total(session.CreatureAs(session.StatCreatureKillerAs, r.Record, element))
				}
				e := entry(cards.Spec{}, bleedStyle(false), beaten, "WON", name+" - "+titleCase(strings.ToLower(room.label)),
					[]string{about, inElement, "ENCOUNTERED " + times(beaten+fell)})
				e.found = beaten+fell > 0
				e.element, e.motif, e.room = element, key, room.label
				e.backdrop = realmRoom(motif, room.tier, element)
				out = append(out, e)
			}
		}
	}
	d := catalogDialog(gs, "REALMS", out)
	d.pic = image.Pt(realmThumbW, realmThumbH)
	d.roster(gs, func(bool, string, string) []models.TipLine { return nil })
	d.rooms(gs)
	return d
}

// dealsIn reports whether a motif fields any creature in an element: whether it is a realm at all.
func dealsIn(m data.MotifData, element string) bool {
	for _, r := range m.Records {
		if r.HasAffinity(element) {
			return true
		}
	}
	return false
}

// realmRoom is one of a realm's rooms: the motif's first backdrop of the tier drawn in the element,
// in file order, or the default backdrop when it has none.
func realmRoom(m data.MotifData, tier, element string) string {
	for _, b := range m.Backdrops {
		if b.Tier == tier && b.HasAffinity(element) {
			return b.ArtKey(element)
		}
	}
	return data.DefaultBackgroundArt
}

// drawRealmPicture draws a realm's room into its seat, decoding it first if the frame still has a
// decode to spend. An undecoded room is an empty frame until its turn comes.
func drawRealmPicture(gs *state.GlobalState, screen *ebiten.Image, e compendiumEntry, at image.Rectangle, decodes *int) {
	if !e.found {
		drawUnfoundRealm(gs, screen, at)
		return
	}
	key := e.backdrop
	painted := len(gs.ImageData[key]) > 0
	if !painted {
		key = data.DefaultBackgroundArt
	}
	thumb, ok := realmThumbs[key]
	if !ok && *decodes > 0 {
		*decodes--
		thumb = reduceRoom(gs.ImageData[key], at.Dx(), at.Dy())
		realmThumbs[key] = thumb
	}
	if thumb != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(at.Min.X), float64(at.Min.Y))
		if e.found && !painted {
			op.ColorScale.Scale(catalogUndrawnScale, catalogUndrawnScale, catalogUndrawnScale, 1)
		}
		screen.DrawImage(thumb, op)
	}
	vector.StrokeRect(screen, float32(at.Min.X), float32(at.Min.Y), float32(at.Dx()), float32(at.Dy()),
		2, systems.PanelEdgeInk, false)
}

// drawUnfoundRealm is a realm not yet found: the portal screen's own swirl, as tall as the seat, on a
// ground darker than the panel. **It gives nothing of the room away** — every unfound realm is the
// same picture — and it is the one thing the player has already seen standing for a realm unknown.
func drawUnfoundRealm(gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle) {
	vector.FillRect(screen, float32(at.Min.X), float32(at.Min.Y), float32(at.Dx()), float32(at.Dy()),
		realmUnfoundGround, false)
	if swirl := portalSwirl(gs); swirl != nil {
		sw, sh := swirl.Bounds().Dx(), swirl.Bounds().Dy()
		scale := min(float64(at.Dx())/float64(sw), float64(at.Dy())/float64(sh))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(sw)/2, -float64(sh)/2)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(at.Min.X+at.Dx()/2), float64(at.Min.Y+at.Dy()/2))
		op.Filter = ebiten.FilterLinear
		screen.DrawImage(swirl, op)
	}
	vector.StrokeRect(screen, float32(at.Min.X), float32(at.Min.Y), float32(at.Dx()), float32(at.Dy()),
		2, systems.PanelEdgeInk, false)
}

// realmUnfoundGround is what an unfound realm's swirl stands on: a step darker than the panel.
var realmUnfoundGround = color.RGBA{R: 14, G: 13, B: 18, A: 255}

// reduceRoom decodes a room and averages it down to w by h, cropped to cover. **A box average**, each
// thumbnail pixel the mean of the block of the room it stands for, which is what keeps a reduction
// this deep from aliasing. nil when there is nothing to decode.
func reduceRoom(raw []byte, w, h int) *ebiten.Image {
	if len(raw) == 0 {
		return nil
	}
	decoded, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b := decoded.Bounds()
	src := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(src, src.Bounds(), decoded, b.Min, draw.Src)

	// The window of the room that covers w:h, centered.
	cw, ch := b.Dx(), b.Dx()*h/w
	if ch > b.Dy() {
		cw, ch = b.Dy()*w/h, b.Dy()
	}
	left, top := (b.Dx()-cw)/2, (b.Dy()-ch)/2

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := top+y*ch/h, top+(y+1)*ch/h
		for x := 0; x < w; x++ {
			x0, x1 := left+x*cw/w, left+(x+1)*cw/w
			var r, g, bl, n int
			for sy := y0; sy < max(y1, y0+1); sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < max(x1, x0+1); sx++ {
					p := row[sx*4 : sx*4+3]
					r, g, bl, n = r+int(p[0]), g+int(p[1]), bl+int(p[2]), n+1
				}
			}
			o := out.Pix[y*out.Stride+x*4:]
			o[0], o[1], o[2], o[3] = uint8(r/n), uint8(g/n), uint8(bl/n), 255
		}
	}
	return ebiten.NewImageFromImage(out)
}
