package ui

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/models"
)

// **A scrollbar's arrows are buttons, so they sit on a height tier**: the smallest one, written in
// models because models sits under this package and cannot read the tier itself.
func TestTheScrollArrowsAreTiny(t *testing.T) {
	if models.ScrollArrowSize != ButtonTiny {
		t.Errorf("the scroll arrows are %d tall, want ButtonTiny (%d)", models.ScrollArrowSize, ButtonTiny)
	}
}
