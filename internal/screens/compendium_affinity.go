package screens

// **The relic book's affinities: what a relic is about, read off its own rules.**
//
// An affinity is an element or a form a relic names — in what it matches or in what it turns a card
// into, so a flip to arcane and a flip from arcane are both arcane — or one of four things it works
// on: an attack, a defense, the action budget, a hand. **Derived, never authored**: a relic retuned to
// a different element moves chips with nothing edited. **A relic naming none of those is SUPPORT** —
// a stat, the clock, the purse, a prize, a consumable — so every relic stands under at least one chip.

import (
	"slices"

	"github.com/curiousjc/ascend-duel/data"
)

// The affinities that are not an element or a form, as their chips read.
const (
	affinityAttack  = "ATTACK"
	affinityDefend  = "DEFEND"
	affinityAP      = "AP"
	affinityHand    = "HAND"
	affinitySupport = "SUPPORT"
)

// The affinity vocabulary's sorting of the rule grammar: which moments and which effects say a relic
// works on an attack, a defense, the action budget or a hand. **Every moment and every verb the
// grammar has is in at least one list here**, the neutral ones included, and
// TestEveryMomentAndVerbIsSorted fails on one that is not — so a verb added to internal/combat is a
// decision about its affinity rather than a relic quietly filed under SUPPORT.
var (
	attackMoments = []string{"card-damage", "attack-lands"}
	attackVerbs   = []string{"scale-damage", "add-dmg", "add-dmg-per-held", "add-dmg-per-vitae",
		"scale-damage-per-vitae", "repeat-card", "echo-attack", "pierce-element", "drain-damage",
		"grow-on-hit", "awaken"}
	defendMoments = []string{"hit-blocked"}
	defendVerbs   = []string{"raise-shield", "shield-per-card", "surge-on-block", "keep-shields",
		"heal-on-block", "vitae-on-block", "match-foe-shields", "reflect-damage"}
	apMoments = []string{"card-cost"}
	apVerbs   = []string{"adjust-cost", "demote-card", "surge-on-block", "surge-per-card"}
	handVerbs = []string{"add-hand-dmg", "scale-hand-damage"}

	// The neutral ones: a moment that says nothing about what is being worked on, and a verb that
	// claims no affinity of its own. **A setter's affinity is its element or form**, read off the
	// effect; **a growth verb feeds the relic's own accumulator**, which its other effects read and
	// carry the affinity of; the rest are a stat, the clock, the purse, a prize or a consumable — the
	// SUPPORT relics.
	neutralMoments = []string{"equipped", "fight-won", "prizes-dealt", "blow-formed", "turn-taken",
		"card-drawn", "turn-start", "essence-spent", "fight-begun"}
	neutralVerbs = []string{"set-element", "set-form", "grow-on-win", "grow-on-turn", "grow-per-card",
		"grow-on-streak", "reset-growth", "add-hp", "scale-hp", "heal-share", "scale-propagation",
		"adjust-picks", "adjust-prize-vitae", "scale-rolls", "adjust-round-limit",
		"adjust-essence-targets", "copy-consumable"}
)

// affinityOrder is every affinity chip in the order it stands: the elements, the three attacking forms,
// the four works-on affinities, and SUPPORT.
func affinityOrder() []string {
	var out []string
	for _, e := range data.AffinityElements {
		out = append(out, e)
	}
	return append(out, "stab", "slash", "crush", affinityAttack, affinityDefend, affinityAP, affinityHand,
		affinitySupport)
}

// relicAffinities is every affinity one relic carries, in affinityOrder.
func relicAffinities(r data.RelicData) []string {
	has := map[string]bool{}
	for _, rule := range r.Rules {
		if slices.Contains(attackMoments, rule.When) {
			has[affinityAttack] = true
		}
		if slices.Contains(defendMoments, rule.When) {
			has[affinityDefend] = true
		}
		if slices.Contains(apMoments, rule.When) {
			has[affinityAP] = true
		}
		if p := rule.If; p != nil {
			has[p.Element] = true
			if p.Form == "defend" {
				has[affinityDefend] = true
			} else {
				has[p.Form] = true
			}
			if p.Hand != "" || len(p.Hands) > 0 || p.MinForms > 0 {
				has[affinityHand] = true
			}
		}
		for _, t := range rule.Then {
			has[t.Element] = true
			has[t.Form] = true
			switch {
			case slices.Contains(attackVerbs, t.Do):
				has[affinityAttack] = true
			case slices.Contains(defendVerbs, t.Do):
				has[affinityDefend] = true
			}
			if slices.Contains(apVerbs, t.Do) {
				has[affinityAP] = true
			}
			if slices.Contains(handVerbs, t.Do) {
				has[affinityHand] = true
			}
		}
	}
	var out []string
	for _, a := range affinityOrder() {
		if has[a] {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		out = []string{affinitySupport}
	}
	return out
}
