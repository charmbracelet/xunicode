// Package sentence implements Unicode sentence segmentation as defined by UAX #29.
package sentence

import (
	"charm.land/xunicode/internal/segmenter"
	"golang.org/x/text/language"
)

var trie = newSentenceTrie(0)

var ruleData = segmenter.RuleBreakData{
	PropertyLookup:        trie.lookup,
	BreakStateTable:       breakTable[:],
	PropertyCount:         stride,
	LastCodepointProperty: lastCP,
	SOTProperty:           sot,
	EOTProperty:           eot,
}

// Segmenter iterates over the sentences in a byte slice.
// The usage pattern is:
//
//	seg := sentence.NewSegmenter(input)
//	for seg.Next() {
//	    fmt.Println(seg.Bytes())
//	}
type Segmenter struct {
	s *segmenter.Segmenter
}

// Options configures sentence segmentation.
// The zero value uses default UAX #29 rules with no locale tailoring.
type Options struct {
	// Locale provides locale-tailored sentence breaking.
	// When set, the segmenter applies locale-specific rules that
	// override the default UAX #29 properties for certain characters.
	//
	// Supported locales:
	//   - Greek (el): treats semicolon (U+003B) and Greek question
	//     mark (U+037E) as sentence terminators (STerm).
	//
	// The zero value applies no locale tailoring.
	Locale language.Tag
}

// greekOverride remaps U+003B (semicolon) and U+037E (Greek question mark)
// to STerm for Greek sentence segmentation. In standard UAX #29 these are
// Other; Greek uses them as sentence terminators.
func greekOverride(prop uint8, r rune) uint8 {
	switch r {
	case ';', '\u037E':
		return uint8(STerm)
	}
	return prop
}

// NewSegmenter returns a Segmenter that iterates over the sentences
// in the given input using default options.
func NewSegmenter(input []byte) *Segmenter {
	return &Segmenter{s: segmenter.New(&ruleData, input)}
}

// NewSegmenter returns a Segmenter that iterates over the sentences
// in the given input, configured by o.
func (o *Options) NewSegmenter(input []byte) *Segmenter {
	seg := segmenter.New(&ruleData, input)
	if o.Locale != language.Und {
		base, _ := o.Locale.Base()
		switch base {
		case language.MustParseBase("el"):
			seg.SetOverrideLookup(greekOverride)
		}
	}
	return &Segmenter{s: seg}
}

// isSafeASCII reports whether b is an ASCII byte that never participates in
// sentence break rules: letters, digits, and space.
func isSafeASCII(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == ' '
}

// asciiProp returns the sentence break property for a safe ASCII byte.
func asciiProp(b byte) uint8 {
	if b >= 'a' && b <= 'z' {
		return uint8(Lower)
	}
	if b >= 'A' && b <= 'Z' {
		return uint8(Upper)
	}
	if b >= '0' && b <= '9' {
		return uint8(Numeric)
	}
	return uint8(Sp)
}

// Next advances to the next sentence boundary segment. It returns false when
// the end of input has been reached.
func (se *Segmenter) Next() bool {
	input := se.s.Input()
	pos := se.s.End()
	if pos >= len(input) {
		return false
	}

	// ASCII fast path: skip over contiguous safe ASCII bytes (letters, digits,
	// space) that can never trigger sentence breaks, avoiding per-byte trie
	// lookups. If the run reaches EOF, emit it all via FastForward. Otherwise,
	// rewind to one byte before the end of the safe run (SetEnd) so the state
	// machine sees correct left-context when it hits the terminator, then fix
	// up the start (SetStart) to include the skipped prefix.
	end := pos
	for end < len(input) && isSafeASCII(input[end]) {
		end++
	}

	if end >= len(input) {
		se.s.FastForward(end, asciiProp(input[end-1]))
		return true
	}

	if end > pos {
		se.s.SetEnd(end - 1)
	}

	ok := se.s.Next()
	if ok && end > pos {
		se.s.SetStart(pos)
	}
	return ok
}

// Bytes returns the current sentence as a byte slice.
func (se *Segmenter) Bytes() []byte { return se.s.Bytes() }

// Text returns the current sentence as a string.
func (se *Segmenter) Text() string { return se.s.Text() }

// Position returns the byte offsets [start, end) of the current sentence.
func (se *Segmenter) Position() (start, end int) { return se.s.Position() }
