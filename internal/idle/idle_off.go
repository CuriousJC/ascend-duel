//go:build !idleexit

package idle

// Tick reports whether the game has been idle long enough to close itself. Always false in
// this build — the whole feature is compiled out.
//
// Arguments are all cheap to produce at the call site, so unlike trace's log lines there is
// nothing here worth guarding behind an Enabled() check.
func Tick(mouseX, mouseY int, focused, clicking bool) bool { return false }
