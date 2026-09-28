package combat

import "testing"

// The form orbs: set-form deals a card as its counterpart on another form's ladder.

func formOrb(t *testing.T, key string, from, to Form) RelicID {
	t.Helper()
	return relic(t, key, RelicRule{
		When: MomentCardDrawn,
		If:   RelicCondition{Form: from, HasForm: true},
		Then: []RelicEffect{{Do: DoSetForm, Form: to}},
	})
}

// TestAFormOrbDealsTheSameRungOfAnotherForm. A 2 AP slash under slash-to-crush is the 2 AP crush,
// and it keeps its own element.
func TestAFormOrbDealsTheSameRungOfAnotherForm(t *testing.T) {
	worn := []WornRelic{{Relic: formOrb(t, "orb.slash.crush", FormSlash, FormCrush)}}

	got := DealtAs(worn, Of(Slice, Fire))
	if got.Concept != Bash || got.Element != Fire {
		t.Errorf("a fire Slice was dealt as %s %v, want a fire Bash", ConceptOf(got.Concept).Label, got.Element)
	}
	if other := DealtAs(worn, Of(Jab, Fire)); other.Concept != Jab {
		t.Errorf("a stab was changed by a slash orb: %s", ConceptOf(other.Concept).Label)
	}
}

// TestFormOrbsChainInWornOrder. Stab-to-slash left of slash-to-crush deals a stab as a crush,
// through slash; swapped, the crush orb reads the stab before it has become a slash.
func TestFormOrbsChainInWornOrder(t *testing.T) {
	toSlash := formOrb(t, "orb.stab.slash", FormStab, FormSlash)
	toCrush := formOrb(t, "orb.slash.crush.chain", FormSlash, FormCrush)

	if got := DealtAs([]WornRelic{{Relic: toSlash}, {Relic: toCrush}}, Of(Thrust, Ice)); got.Concept != Bash {
		t.Errorf("stab-to-slash then slash-to-crush dealt a Thrust as %s, want Bash", ConceptOf(got.Concept).Label)
	}
	if got := DealtAs([]WornRelic{{Relic: toCrush}, {Relic: toSlash}}, Of(Thrust, Ice)); got.Concept != Slice {
		t.Errorf("slash-to-crush then stab-to-slash dealt a Thrust as %s, want Slice", ConceptOf(got.Concept).Label)
	}
}

// TestSetFormNamingNoFormIsRefused. FormNone is the zero value, so a record that forgot the form
// must not load as an orb that does nothing.
func TestSetFormNamingNoFormIsRefused(t *testing.T) {
	_, err := RegisterRelic("relictest.orb.noform", "no form", []RelicRule{{
		When: MomentCardDrawn,
		Then: []RelicEffect{{Do: DoSetForm}},
	}})
	if err == nil {
		t.Fatal("a set-form naming no form registered")
	}
}
