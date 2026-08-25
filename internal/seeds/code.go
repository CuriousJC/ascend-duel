package seeds

import (
	"errors"
	"fmt"
	"strings"
)

// A run seed is a **six-character code**, and this file is the only place that spelling exists.
// `RunSeed` is still an `int64` because every stream derives from it by arithmetic; the code is
// how that number is written down, said out loud and typed back in.
//
// **The alphabet is Crockford base32** — the ten digits and the twenty-six letters, less `I`,
// `L`, `O` and `U`. That is 32 characters, so the space is 32^6: 1,073,741,824 runs. Reducing
// the seed to that range does not weaken the derivation, because each stream XORs a salt spread
// across the full int64 — two adjacent codes do not produce neighbouring shuffles.
//
// **The four are dropped because a code is transcribed by eye** *(owner's call, 2026-08-25)*.
// `0`/`O` and `1`/`I`/`L` are the pairs a person reads wrong off a screen, and the failure they
// cause is the quiet one: not an error message, but a different, perfectly valid run. `U` goes
// for a different reason — it is the letter that turns a random six-character code into a word
// somebody has to read out.
//
// **Dropping them from the alphabet is only half of it; `Parse` still accepts them.** Someone
// who types `O` meant `0` and someone who types `I` or `L` meant `1`, so those fold on the way
// in rather than being refused — refusing is a correct-but-useless answer to a person holding
// the right code. `U` does not fold, because it is not a mistake for anything; a code with a
// `U` in it was never one this game issued.
//
// **Case is not information.** A code is read off a screen and typed back by someone who has
// no reason to know whether the letter was capital, so `Parse` accepts either and `Code` only
// ever emits upper case. Anything comparing two codes compares the parsed numbers, never the
// strings.

const (
	// alphabet is the ordered digit set — Crockford base32, so no `I`, `L`, `O` or `U`. Its
	// order is the numeral system, so **it may never be reordered and nothing may be added to
	// it or taken out** — the same append-only hazard the enum ordinals carry, one layer down:
	// either change re-points every code ever written down at a different run.
	alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

	// folded are the characters Parse accepts even though Code never emits them, paired with
	// what each one was meant to be. See the note above: a person holding the right code and
	// typing the letter that looks like the digit should get their run, not an error.
	folded = "OoIiLl"

	// foldedTo is what each character in `folded` becomes, index for index.
	foldedTo = "001111"

	// CodeLen is how many characters a code is. Fixed rather than variable, so a code is
	// recognisable as one and a field can size itself.
	CodeLen = 6

	// Base is the alphabet's size.
	Base = int64(len(alphabet))
)

// Space is how many distinct runs a code can name: Base^CodeLen.
var Space = func() int64 {
	n := int64(1)
	for i := 0; i < CodeLen; i++ {
		n *= Base
	}
	return n
}()

// ErrBadCode is what Parse returns for anything that is not a code. One error rather than a set,
// because the only caller that can act on it is a text field, and every reason it can fail comes
// out as the same sentence to the player.
var ErrBadCode = errors.New("seeds: a run code is six letters or digits, and has no I, L, O or U in it")

// Normalize folds any int64 into the code space, so the clock — or any other wide number — names
// a run that can actually be written down. Euclidean, so a negative input lands in range rather
// than staying negative.
func Normalize(n int64) int64 {
	n %= Space
	if n < 0 {
		n += Space
	}
	return n
}

// Code renders a run seed as its six-character code.
//
// **It panics on a seed outside the space** rather than folding it, matching the rest of this
// package: quietly rendering a number that will not round-trip is how a shared code comes to
// name a different run than the one it was copied from. Call Normalize at the one place a seed
// is chosen; everything downstream is already in range.
func Code(runSeed int64) string {
	if runSeed < 0 || runSeed >= Space {
		panic(fmt.Sprintf("seeds: run seed %d is outside the code space, call Normalize", runSeed))
	}
	out := make([]byte, CodeLen)
	for i := CodeLen - 1; i >= 0; i-- {
		out[i] = alphabet[runSeed%Base]
		runSeed /= Base
	}
	return string(out)
}

// Parse reads a code back to a run seed. Case-insensitive; `O` folds to `0` and `I`/`L` fold to
// `1`, because that is what the person typing them meant; and surrounding whitespace is forgiven
// because a pasted code brings some with it.
func Parse(code string) (int64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != CodeLen {
		return 0, ErrBadCode
	}
	var n int64
	for i := 0; i < len(code); i++ {
		c := code[i]
		if f := strings.IndexByte(folded, c); f >= 0 {
			c = foldedTo[f]
		}
		d := strings.IndexByte(alphabet, c)
		if d < 0 {
			return 0, ErrBadCode
		}
		n = n*Base + int64(d)
	}
	return n, nil
}
