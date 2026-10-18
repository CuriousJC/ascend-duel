package screens

// combat_sound.go — the fight's sound effects, which today are one: a relic toasting.
//
// **The cue goes where the toast does.** Every relic that rattles in the row is a relic activating,
// so shakeRelicAt raises it — a figure out of the sum, a DMG raise, a ring's shield leaving it, the
// pearl's copy — and so does each beat of the deal's cascade. One toast is one sound, and toasts
// that overlap layer, because each play is its own player (internal/music/sound.go).
//
// **Toasts on one frame are spread a little**, in the sound only. The sum's lines run in parallel,
// so two relics paying into one figure start on the same frame, and two copies of one file started
// together are one sound twice as loud rather than two relics. The picture is not moved: the
// rattle is still the frame the figure leaves, which is the rule the shake is under.
//
// **It may never change an outcome.** Every cue here is raised by playback, after the round is
// resolved.

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/music"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// toastStaggerTicks is how far apart two toasts raised on one frame play: a sixth of a beat, about a
// fifteenth of a second at the tuned speed — enough to hear as two and too little to read as late.
// **On the beat like every other duration**, so a faster game spreads its toasts tighter.
func toastStaggerTicks() int { return ui.Beat(1, 6) }

// toastSound plays a relic's toast, or queues it a few ticks behind the toasts already raised this
// frame.
func (s *CombatScene) toastSound() {
	if s.toastLastFrame != s.toastFrame {
		s.toastLastFrame, s.toastsThisFrame = s.toastFrame, 0
	}
	delay := s.toastsThisFrame * toastStaggerTicks()
	s.toastsThisFrame++
	if delay == 0 {
		music.PlayCue(data.CueRelicToast)
		return
	}
	s.toasts = append(s.toasts, delay)
}

// tickToasts advances the frame count and plays every queued toast whose wait is over.
func (s *CombatScene) tickToasts() {
	s.toastFrame++
	kept := s.toasts[:0]
	for _, wait := range s.toasts {
		if wait--; wait <= 0 {
			music.PlayCue(data.CueRelicToast)
			continue
		}
		kept = append(kept, wait)
	}
	s.toasts = kept
}
