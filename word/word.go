// Package word implements Unicode word segmentation as defined by UAX #29.
package word

import (
	"github.com/charmbracelet/xunicode/internal/segmenter"
	"golang.org/x/text/language"
)

var trie = newWordTrie(0)

var ruleData = segmenter.RuleBreakData{
	PropertyLookup:        trie.lookup,
	BreakStateTable:       breakTable[:],
	PropertyCount:         stride,
	LastCodepointProperty: lastCP,
	SOTProperty:           sot,
	EOTProperty:           eot,
}

// WordType classifies a word segment.
type WordType uint8

const (
	WordNone   WordType = iota // not word-like (whitespace, punctuation, etc.)
	WordNumber                 // numeric segment
	WordLetter                 // word segment (letters, CJK ideographs, etc.)
)

// IsWordLike reports whether t represents a word-like segment (Letter or Number).
func IsWordLike(t WordType) bool { return t != WordNone }

// wordTypeTable maps property indices (including combined states from
// the state machine) to WordType. When a lookahead rolls back via
// NoMatch, BoundaryProperty may return a combined state index rather
// than a base codepoint property, so the table must cover those too.
var wordTypeTable = [...]WordType{
	Katakana:                      WordLetter,
	Hebrew_Letter:                 WordLetter,
	ALetter:                       WordLetter,
	Numeric:                       WordNumber,
	ExtendNumLet:                  WordLetter,
	ALetter_Extended_Pictographic: WordLetter,
	SA:                            WordLetter,
	ALetter_ZWJ:                   WordLetter,
	Hebrew_Letter_ZWJ:             WordLetter,
	Numeric_ZWJ:                   WordNumber,
	Katakana_ZWJ:                  WordLetter,
	ExtendNumLet_ZWJ:              WordLetter,
	ALetterEP_ZWJ:                 WordLetter,
	AHL_MidLetter:                 WordLetter,
	HL_MidLetter:                  WordLetter,
	Num_MidNum:                    WordNumber,
	HL_DQ:                         WordLetter,
}

// Segmenter iterates over the words in a byte slice.
type Segmenter struct {
	s *segmenter.Segmenter
}

// Options configures word segmentation.
// The zero value uses default UAX #29 rules with no locale tailoring.
type Options struct {
	// Locale provides locale-tailored word breaking.
	// When set, the segmenter applies locale-specific rules that
	// override the default UAX #29 properties for certain characters.
	//
	// Supported locales:
	//   - Finnish (fi), Swedish (sv): treat colon (U+003A) and its
	//     fullwidth/small variants as word breaks instead of MidLetter,
	//     so that e.g. "EU:ssa" segments into separate words.
	//
	// The zero value applies no locale tailoring.
	Locale language.Tag
}

// finnishOverride remaps colon characters from MidLetter to Other for
// Finnish and Swedish word segmentation. In standard UAX #29, colon is
// MidLetter, so "EU:ssa" is one word. Finnish/Swedish treat colon as a
// word break.
func finnishOverride(prop uint8, r rune) uint8 {
	switch r {
	case ':', '\uFE55', '\uFF1A':
		return uint8(Other)
	}
	return prop
}

// NewSegmenter returns a Segmenter that iterates over the words
// in the given input using default options.
func NewSegmenter(input []byte) *Segmenter {
	return &Segmenter{s: segmenter.New(&ruleData, input)}
}

// NewSegmenter returns a Segmenter that iterates over the words
// in the given input, configured by o.
func (o *Options) NewSegmenter(input []byte) *Segmenter {
	seg := segmenter.New(&ruleData, input)
	if o.Locale != language.Und {
		base, _ := o.Locale.Base()
		switch base {
		case language.MustParseBase("fi"), language.MustParseBase("sv"):
			seg.SetOverrideLookup(finnishOverride)
		}
	}
	return &Segmenter{s: seg}
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func isUnsafeAfterAlphaNum(b byte) bool {
	switch b {
	case '\'', '"', ',', '.', ':', ';', '_':
		return true
	}
	return isAlphaNum(b)
}

// Next advances to the next word boundary segment. It returns false when the
// end of input has been reached.
func (w *Segmenter) Next() bool {
	input := w.s.Input()
	pos := w.s.End()
	if pos >= len(input) {
		return false
	}

	// ASCII fast path: scan a contiguous run of ASCII alphanumerics
	// (ALetter/Numeric) and emit it as one segment, skipping the state machine.
	// Only safe if the byte after the run isn't mid-word punctuation
	// (MidLetter/MidNum/MidNumLet triggers lookahead), ExtendNumLet, another
	// alphanumeric, or non-ASCII (could be Extend/Format/ZWJ). The boundary
	// property is set from the last byte for correct WordType classification.
	b := input[pos]
	if b < 0x80 && isAlphaNum(b) {
		end := pos + 1
		for end < len(input) && isAlphaNum(input[end]) {
			end++
		}
		if end >= len(input) || (input[end] < 0x80 && !isUnsafeAfterAlphaNum(input[end])) {
			prop := ALetter
			if input[end-1] >= '0' && input[end-1] <= '9' {
				prop = Numeric
			}
			w.s.FastForward(end, uint8(prop))
			return true
		}
	}
	return w.s.Next()
}

// Bytes returns the current segment as a byte slice.
func (w *Segmenter) Bytes() []byte { return w.s.Bytes() }

// Text returns the current segment as a string.
func (w *Segmenter) Text() string { return w.s.Text() }

// Position returns the byte offsets [start, end) of the current segment.
func (w *Segmenter) Position() (start, end int) { return w.s.Position() }

// WordType returns the classification of the current segment.
func (w *Segmenter) WordType() WordType {
	p := w.s.BoundaryProperty()
	if int(p) < len(wordTypeTable) {
		return wordTypeTable[p]
	}
	return WordNone
}

// IsWordLike reports whether the current segment is word-like (letter or number).
func (w *Segmenter) IsWordLike() bool { return IsWordLike(w.WordType()) }
