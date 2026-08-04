//go:build debugtrace

// The live half of package trace. See trace_off.go for what this is and why it is behind a
// build tag; that file carries the package doc so it is readable in the default build.
package trace

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// frameEvery is how many ticks pass between screen captures, at 60 TPS. Capturing is a
	// GPU-to-CPU readback and it stalls the frame it happens on, so this is deliberately
	// slow — the point is to see the screen, not to film it.
	frameEvery = 120

	dirEnv     = "ASCEND_TRACE_DIR"
	defaultDir = "trace"
	frameName  = "frame.png"
)

var (
	// tick is the simulation tick every line is stamped with. Written once per Update from
	// the game loop and read from the same goroutine, so it needs no synchronisation.
	tick int

	// encoding guards the single background encoder. Capture is throttled anyway, but a
	// slow disk must never be able to pile up goroutines each holding a screen's worth of
	// pixels.
	encoding atomic.Bool

	// announced keeps the capture path to one line for the whole session.
	announced atomic.Bool
)

func Enabled() bool { return true }

func Tick(n int) { tick = n }

func Logf(category, format string, args ...any) {
	fmt.Printf("[%07d] %-9s %s\n", tick, category, fmt.Sprintf(format, args...))
}

func Section(name string) {
	fmt.Printf("[%07d] %-9s ---- %s ----\n", tick, "trace", name)
}

// Rect writes one named rectangle. Both the extent and the size, because a layout bug is
// usually one or the other: a box in the wrong place, or a box the wrong size for what has
// to go in it.
func Rect(name string, r image.Rectangle) {
	Logf("layout", "%-18s x %4d..%-4d y %4d..%-4d  %dx%d",
		name, r.Min.X, r.Max.X, r.Min.Y, r.Max.Y, r.Dx(), r.Dy())
}

// Frame writes the current screen to <dir>/frame.png, overwriting the previous one, so the
// path is stable and always holds the most recent capture.
//
// ReadPixels has to happen here on the game goroutine — it is a GPU operation — but the PNG
// encode and the file write do not, so they go to a background goroutine and the frame only
// pays for the readback.
func Frame(screen *ebiten.Image) {
	if tick%frameEvery != 0 {
		return
	}
	if !encoding.CompareAndSwap(false, true) {
		return // the previous capture is still being written
	}

	b := screen.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	screen.ReadPixels(img.Pix)

	at := tick
	go func() {
		defer encoding.Store(false)
		if err := writeFrame(img); err != nil {
			fmt.Printf("[%07d] %-9s frame capture failed: %v\n", at, "trace", err)
			return
		}
		// Announced once, not every two seconds. The path never changes, so repeating it
		// says nothing new and drowns the log — a twelve-minute session put 380 of these
		// among 218 lines that actually carried information.
		if announced.CompareAndSwap(false, true) {
			fmt.Printf("[%07d] %-9s capturing every %d ticks -> %s\n",
				at, "trace", frameEvery, framePath())
		}
	}()
}

func traceDir() string {
	if d := os.Getenv(dirEnv); d != "" {
		return d
	}
	return defaultDir
}

func framePath() string { return filepath.Join(traceDir(), frameName) }

// writeFrame encodes to memory, writes a temporary file and renames it over the target.
// A reader that opens frame.png while it is being written would otherwise get a truncated
// PNG, and the whole point of the file is that something else reads it.
func writeFrame(img image.Image) error {
	dir := traceDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	tmp := filepath.Join(dir, frameName+".tmp")
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, framePath())
}
