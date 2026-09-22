package profile

import "testing"

// TestAnInstallIDIsMadeOnceAndKept is the whole of what an install id promises: several reports
// from one player group together, which they do not if the id is minted again at the next launch.
func TestAnInstallIDIsMadeOnceAndKept(t *testing.T) {
	s := At(t.TempDir())

	p, writable, err := LoadProfile(s)
	if err != nil || !writable {
		t.Fatalf("LoadProfile: %v, writable %v", err, writable)
	}
	if p.InstallID != "" {
		t.Fatalf("a fresh profile already carries an id: %q", p.InstallID)
	}
	if !p.EnsureInstallID() {
		t.Fatal("EnsureInstallID made nothing on a profile with no id")
	}
	first := p.InstallID
	if first == "" {
		t.Fatal("EnsureInstallID reported an id and set none")
	}
	if p.EnsureInstallID() {
		t.Fatal("EnsureInstallID replaced an id that was already there")
	}
	if err := SaveProfile(s, p); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	back, _, err := LoadProfile(s)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if back.InstallID != first {
		t.Fatalf("install id = %q after a round trip, want %q", back.InstallID, first)
	}
}

// TestAnInstallIDIsNotAName holds the reason it is random rather than derived: it is the only
// thing in a crash report that is about whom, and what leaves the machine must identify nobody.
func TestAnInstallIDIsNotAName(t *testing.T) {
	var a, b Profile
	a.EnsureInstallID()
	b.EnsureInstallID()
	if a.InstallID == b.InstallID {
		t.Fatalf("two profiles were given the same id: %q", a.InstallID)
	}
	if len(a.InstallID) != 16 {
		t.Fatalf("id = %q, want sixteen hex characters", a.InstallID)
	}
}
