package ui

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
)

// lineText is a block of lines as one string, for a test that cares whether anything was written
// rather than what.
func lineText(lines []session.LedgerLine) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Text())
		b.WriteString("\n")
	}
	return b.String()
}

// anAct is the line every outcome attaches to, so a kind that attaches has something to attach to.
func anAct() session.LedgerRecord {
	return session.LedgerRecord{
		Kind: session.KindAct, Side: session.SideYou, Name: "Duelist",
		Card: "Jab", Element: "fire", Verb: session.InkAttack,
	}
}

// **This is the tripwire the whole split rests on.** The ledger stores records and the panel words
// them here, so a kind with no case in the translator draws as nothing at all — and a record nobody
// worded and a record nobody wrote look identical on screen. Adding a kind without wording it is
// the one mistake this arrangement makes easy, so it fails here instead.
//
// **It checks that the record changed what was written, not that it reads well.** Whether a
// sentence is any good is a thing to look at; whether it exists at all is a thing to test.
func TestEveryRecordKindReadsAsSomething(t *testing.T) {
	base := lineText(LedgerLines([]session.LedgerRecord{anAct()}))

	for _, kind := range session.RecordKinds() {
		rec := session.LedgerRecord{
			Kind: kind, Side: session.SideYou, Target: session.SideFoe, Name: "Giant Bat",
			Card: "Jab", Element: "fire", Verb: session.InkAttack, Status: "burning",
			Relic: "Keen", Amount: 3, Hand: "Pair", Multiplier: 100,
			Subject: "a fire jab", Into: "an ice jab",
		}

		got := lineText(LedgerLines([]session.LedgerRecord{anAct(), rec}))
		if got == base {
			t.Errorf("a %q record wrote nothing, so the panel draws a blank where it should read", kind)
		}
	}
}

// A term is several kinds of line in one column and each is picked by its role, so a role with no
// case falls through to the card term and prints a card that is not there.
func TestEveryTermRoleReadsAsSomething(t *testing.T) {
	roles := []string{session.RoleHit, session.RoleCard, session.RoleFlat, session.RoleDMG,
		session.RoleSum, session.RoleTotal}

	for _, role := range roles {
		rec := session.LedgerRecord{
			Kind: session.KindTerm, Role: role, Card: "Bash", Element: "fire",
			Relic: "Keen", Note: "the purse", Hand: "Pair",
			Base: 20, Amount: 8, Multiplier: 150, Total: 45,
		}
		if got := strings.TrimSpace(termLine(rec).Text()); got == "" {
			t.Errorf("a %q term wrote nothing", role)
		}
	}
}

// **A record read back has to say who it belongs to**, because the swatch is how the panel is
// scanned before it is read. An act's side becomes its voice, and a thing done *to* a duelist
// carries the victim's rather than the actor's.
func TestALineIsSpokenInItsOwnVoice(t *testing.T) {
	lines := LedgerLines([]session.LedgerRecord{
		anAct(),
		{Kind: session.KindDefeated, Target: session.SideFoe, Name: "Giant Bat"},
	})
	if len(lines) != 2 {
		t.Fatalf("want an act and a fall, got %q", lineText(lines))
	}
	if lines[0].Voice != session.VoiceYou {
		t.Errorf("the act is in the %q voice, want %q", lines[0].Voice, session.VoiceYou)
	}
	if lines[1].Voice != session.VoiceFoe {
		t.Errorf("the fall is in the %q voice, want %q", lines[1].Voice, session.VoiceFoe)
	}
}

// **An outcome joins the sentence above it rather than opening one**, which is what keeps a busy
// round readable: a card played and what became of it is one line, not four.
func TestAnOutcomeJoinsTheLineAboveIt(t *testing.T) {
	lines := LedgerLines([]session.LedgerRecord{
		anAct(),
		{Kind: session.KindDamage, Side: session.SideYou, Amount: 12},
		{Kind: session.KindStatus, Side: session.SideYou, Status: "burning"},
	})
	if len(lines) != 1 {
		t.Fatalf("a card and its outcomes wrote %d lines, want one: %q", len(lines), lineText(lines))
	}
	if got := lines[0].Text(); !strings.Contains(got, "12 damage") || !strings.Contains(got, "burning") {
		t.Errorf("the line reads %q, want the blow and what it did", got)
	}
}

// **An announcement has nothing above it to join**, so an outcome arriving after one may not
// silently attach itself to a sentence about somebody else.
func TestAnOutcomeDoesNotAttachToAnAnnouncement(t *testing.T) {
	lines := LedgerLines([]session.LedgerRecord{
		{Kind: session.KindDefeated, Target: session.SideFoe, Name: "Giant Bat"},
		{Kind: session.KindDamage, Side: session.SideYou, Amount: 12},
	})
	if len(lines) != 1 {
		t.Fatalf("want the fall alone, got %q", lineText(lines))
	}
	if strings.Contains(lines[0].Text(), "12") {
		t.Errorf("the fall reads %q, and the damage after it is not part of it", lines[0].Text())
	}
}

// The article has to be corrected rather than followed: two of the five elements begin with a
// vowel, so "a earth strike" would be a third of the act lines in the game.
func TestTheArticleAgreesWithTheElement(t *testing.T) {
	rec := anAct()
	rec.Element = "earth"

	got := LedgerLines([]session.LedgerRecord{rec})[0].Text()
	if !strings.Contains(got, "an earth") {
		t.Errorf("the line reads %q, want the article to agree with the element", got)
	}
}

// **An outcome finds its own hit's line**, however many lines were written after it: a blow writes
// every hit's working before any hit lands, so the line written last is the total and not the hit
// the damage belongs to.
func TestAHitsOutcomeAttachesToThatHitsLine(t *testing.T) {
	lines := LedgerLines([]session.LedgerRecord{
		{Kind: session.KindBlow, Side: session.SideYou, Hand: "Pair"},
		{Kind: session.KindTerm, Role: session.RoleHit, Hit: 1, Card: "Bash", Total: 10, Multiplier: 100},
		{Kind: session.KindTerm, Role: session.RoleHit, Hit: 2, Card: "Jab", Total: 5, Multiplier: 100},
		{Kind: session.KindTerm, Role: session.RoleTotal, Total: 15},
		{Kind: session.KindMissed, Side: session.SideYou, Hit: 1},
		{Kind: session.KindDamage, Side: session.SideYou, Amount: 5, Hit: 2},
	})
	if len(lines) != 4 {
		t.Fatalf("want a heading, two hits and a total, got %q", lineText(lines))
	}
	if got := lines[1].Text(); !strings.Contains(got, "misses") || strings.Contains(got, "damage") {
		t.Errorf("the first hit reads %q, want its miss and nothing else", got)
	}
	if got := lines[2].Text(); !strings.Contains(got, "5 damage") || strings.Contains(got, "misses") {
		t.Errorf("the second hit reads %q, want its damage and nothing else", got)
	}
	if got := lines[3].Text(); strings.Contains(got, "damage") || strings.Contains(got, "misses") {
		t.Errorf("the total reads %q, and no outcome belongs to it", got)
	}
}
