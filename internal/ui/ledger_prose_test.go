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
			Card: "Jab", Element: "fire", Verb: session.InkAttack,
			Relic: "Keen", Amount: 3, Hand: "Pair", Multiplier: 100,
			Subject: "a fire jab", Into: "an ice jab",
		}

		got := lineText(LedgerLines([]session.LedgerRecord{anAct(), rec}))
		if got == base {
			t.Errorf("a %q record wrote nothing, so the panel draws a blank where it should read", kind)
		}
	}
}

// TestARoundMayOpenWithAnyRecord holds the ledger to a block whose first record has no line before
// it to attach to — a held card's payout comes at the very top of a round, before anything is
// played. **The panel builds every fight of the run whenever it opens**, so one such round makes the
// ledger unopenable for the rest of the run.
func TestARoundMayOpenWithAnyRecord(t *testing.T) {
	for _, kind := range session.RecordKinds() {
		rec := session.LedgerRecord{
			Kind: kind, Side: session.SideYou, Target: session.SideFoe, Name: "Giant Bat",
			Card: "Jab", Element: "fire", Verb: session.InkAttack,
			Relic: "Keen", Amount: 3, Hand: "Pair", Multiplier: 100,
			Subject: "a fire jab", Into: "an ice jab",
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("a round opening with a %q record panics: %v", kind, r)
				}
			}()
			LedgerLines([]session.LedgerRecord{rec})
		}()
	}
}

// TestAHeldCardHasALineOfItsOwn holds a held payout to its own line naming the card it paid for:
// never hung off the card played before it, which was not the one kept back, and never lost at the
// top of a round, where nothing comes before it. A record written before the card was named reads
// as a card all the same.
func TestAHeldCardHasALineOfItsOwn(t *testing.T) {
	held := session.LedgerRecord{Kind: session.KindHeld, Side: session.SideYou, Card: "Jab", Element: "ice", Amount: 8}

	alone := LedgerLines([]session.LedgerRecord{held})
	if len(alone) != 1 || !strings.Contains(alone[0].Text(), "An ice jab") || !strings.Contains(alone[0].Text(), "8") {
		t.Errorf("a round opening with a held payout reads %q", lineText(alone))
	}

	after := LedgerLines([]session.LedgerRecord{anAct(), held})
	act := lineText(LedgerLines([]session.LedgerRecord{anAct()}))
	if len(after) != 2 || after[0].Text()+"\n" != act {
		t.Errorf("a held payout after a played card changed that card's line: %q", lineText(after))
	}

	old := LedgerLines([]session.LedgerRecord{{Kind: session.KindHeld, Side: session.SideYou, Amount: 8}})
	if len(old) != 1 || strings.TrimSpace(old[0].Text()) == "" || strings.Contains(old[0].Text(), "{") {
		t.Errorf("a held payout with no card named reads %q", lineText(old))
	}
}

// A term is several kinds of line in one column and each is picked by its role; a role this build
// does not write draws as a hit, so a line read back from an older account still says something.
func TestEveryTermRoleReadsAsSomething(t *testing.T) {
	roles := []string{session.RoleHit, session.RoleDMG, session.RoleTotal, "a-role-nothing-writes"}

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
		{Kind: session.KindDrained, Side: session.SideYou, Relic: "Vampiric Fang", Amount: 3},
	})
	if len(lines) != 1 {
		t.Fatalf("a card and its outcomes wrote %d lines, want one: %q", len(lines), lineText(lines))
	}
	if got := lines[0].Text(); !strings.Contains(got, "12 damage") || !strings.Contains(got, "drains 3") {
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
		{Kind: session.KindFizzled, Side: session.SideYou, Hit: 1},
		{Kind: session.KindDamage, Side: session.SideYou, Amount: 5, Hit: 2},
	})
	// **A hit's outcome sits on its `=` step**, beside what the hit came to.
	var sums []string
	for _, l := range lines {
		if strings.HasPrefix(l.Text(), "= ") {
			sums = append(sums, l.Text())
		}
	}
	if len(sums) != 2 {
		t.Fatalf("want two hits each ending on its sum, got %q", lineText(lines))
	}
	if got := sums[0]; !strings.Contains(got, "= 10") || !strings.Contains(got, "fizzles, no damage applied") ||
		strings.Contains(got, "Damage Applied") {
		t.Errorf("the first hit's sum reads %q, want its fizzle and nothing else", got)
	}
	if got := sums[1]; !strings.Contains(got, "= 5 Damage Applied") || strings.Contains(got, "fizzles") {
		t.Errorf("the second hit's sum reads %q, want it applied and nothing else", got)
	}
	if got := lines[0].Text(); !strings.Contains(got, "Grand Total 15") {
		t.Errorf("the hand's heading reads %q, want every hit's 15 as its grand total", got)
	}
}

// **A hit that landed what it came to says so on its sum, underlined**: `= 288 Arcane Damage
// Applied`, with the fall it caused beside it. A hit that landed nothing because the target was
// already down is over damage — which is how the two read apart, and why the fall is not a line
// of its own.
func TestALandedHitsSumIsAppliedAndUnderlined(t *testing.T) {
	lines := LedgerLines([]session.LedgerRecord{
		{Kind: session.KindBlow, Side: session.SideYou, Hand: "Pair"},
		{Kind: session.KindTerm, Role: session.RoleHit, Hit: 1, Card: "Jab", Element: "arcane", Total: 288, Multiplier: 100},
		{Kind: session.KindTerm, Role: session.RoleHit, Hit: 2, Card: "Jab", Element: "arcane", Total: 288, Multiplier: 100},
		{Kind: session.KindTerm, Role: session.RoleTotal, Total: 576},
		{Kind: session.KindDamage, Side: session.SideYou, Amount: 288, Hit: 1},
		{Kind: session.KindDefeated, Target: session.SideFoe, Name: "Giant Rat"},
	})
	for _, l := range lines {
		if l.Text() == "Giant Rat falls" {
			t.Errorf("the fall is a line of its own, want it on the hit that landed it: %q", lineText(lines))
		}
	}
	var sums []session.LedgerLine
	for _, l := range lines {
		if strings.HasPrefix(l.Text(), "= ") {
			sums = append(sums, l)
		}
	}
	if len(sums) != 2 {
		t.Fatalf("want two sums, got %q", lineText(lines))
	}
	if got, want := sums[0].Text(), "= 288 Arcane Damage Applied - Giant Rat falls"; got != want {
		t.Errorf("the landed hit's sum reads %q, want %q", got, want)
	}
	for _, s := range sums[0].Spans[1:3] {
		if !s.Mark {
			t.Errorf("%q in the applied total is not underlined", s.Text)
		}
	}
	if got, want := sums[1].Text(), "= 288 Over Damage"; got != want {
		t.Errorf("the hit that landed nothing reads %q, want %q", got, want)
	}
	for _, s := range sums[1].Spans {
		if s.Mark {
			t.Errorf("%q in a sum that landed nothing is underlined", s.Text)
		}
	}
}

// A consumable reads as the duelist doing something with it: a cantrip is cast and says what it
// moved, and a rune is used on the cards it was aimed at.
func TestAConsumableUsedReadsAsASentence(t *testing.T) {
	got := lineText(LedgerLines([]session.LedgerRecord{
		{Kind: session.KindUsed, Side: session.SideYou, Name: "Duelist", Note: "cantrip",
			Subject: "Cantrip of Might", Into: "DMG 10 to 20"},
		{Kind: session.KindUsed, Side: session.SideYou, Name: "Duelist", Note: "rune",
			Subject: "Embermark", Into: "a jab and an ice bash"},
	}))
	want := "Duelist casts Cantrip of Might - DMG 10 to 20\nDuelist uses Embermark on a jab and an ice bash\n"
	if got != want {
		t.Errorf("the uses read as\n%s\nwant\n%s", got, want)
	}
}
