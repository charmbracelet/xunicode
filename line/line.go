// Package line implements Unicode line break segmentation as defined by UAX #14.
//
// A [Segmenter] iterates over the line break opportunities in a byte slice,
// returning segments between mandatory or allowed break positions.
//
// CSS line-break and word-break properties are supported via [Options].
package line

import (
	"charm.land/xunicode/grapheme"
	"charm.land/xunicode/internal/segmenter"
	"golang.org/x/text/language"
)

var trie = newLineTrie(0)

var ruleData = segmenter.RuleBreakData{
	PropertyLookup:        trie.lookup,
	BreakStateTable:       breakTable[:],
	PropertyCount:         stride,
	LastCodepointProperty: lastCP,
	SOTProperty:           sot,
	EOTProperty:           eot,
	ComplexProp:           uint8(SA),
}

// Strictness controls the strictness of line-breaking rules,
// corresponding to the CSS line-break property.
//
// See https://drafts.csswg.org/css-text-3/#line-break-property.
type Strictness uint8

const (
	// Strict uses the most stringent set of line-breaking rules.
	// This is the default behavior of the Unicode Line Breaking Algorithm,
	// resolving class CJ to NS (no extra break opportunities at
	// conditional Japanese starters).
	Strict Strictness = iota

	// Normal uses the most common set of line-breaking rules.
	// CJ class codepoints are treated as ID (ideographic), allowing
	// breaks before them.
	Normal

	// Loose uses the least restrictive set of line-breaking rules.
	// CJ class codepoints are treated as ID (ideographic), allowing
	// breaks before them.
	Loose

	// Anywhere allows breaks after every typographic character unit,
	// disregarding prohibitions against line breaks. Only mandatory
	// break rules (LB4, LB5) are preserved.
	Anywhere
)

// WordBreak controls line break opportunities between letters,
// corresponding to the CSS word-break property.
//
// See https://drafts.csswg.org/css-text-3/#word-break-property.
type WordBreak uint8

const (
	// WordNormal breaks words according to their customary rules.
	WordNormal WordBreak = iota

	// WordBreakAll treats all characters as having soft wrap
	// opportunities by remapping alphabetic (AL) and complex context
	// (SA) characters to ID (ideographic).
	WordBreakAll

	// WordKeepAll suppresses soft wrap opportunities between
	// typographic letter units by remapping ideographic (ID) and
	// conditional Japanese starter (CJ) characters to AL (alphabetic).
	WordKeepAll
)

// Segmenter iterates over the line break segments in a byte slice.
// The usage pattern is:
//
//	seg := line.NewSegmenter(input)
//	for seg.Next() {
//	    fmt.Println(seg.Bytes())
//	}
type Segmenter struct {
	s        *segmenter.Segmenter
	gs       *grapheme.Segmenter
	anywhere bool
}

// Options configures line break segmentation.
// The zero value uses [Strict] strictness with [WordNormal] word breaking
// and no locale tailoring.
type Options struct {
	// Strictness sets the CSS line-break strictness level, controlling
	// how aggressively the segmenter breaks lines at CJ (conditional
	// Japanese starter) characters and other context-dependent positions.
	// The default is [Strict].
	//
	// See https://drafts.csswg.org/css-text-3/#line-break-property.
	Strictness Strictness

	// WordBreak sets the CSS word-break behavior, controlling line
	// break opportunities between letters. [WordBreakAll] allows
	// breaks within words; [WordKeepAll] suppresses breaks between
	// CJK characters that would normally be allowed.
	// The default is [WordNormal].
	//
	// See https://drafts.csswg.org/css-text-3/#word-break-property.
	WordBreak WordBreak

	// Locale provides locale-tailored line breaking.
	// When set, the segmenter may allow additional break opportunities
	// under [Normal] or [Loose] strictness based on the content language.
	//
	// See https://drafts.csswg.org/css-text-3/#line-break-property for details.
	//
	// The zero value applies no locale tailoring.
	Locale language.Tag
}

// NewSegmenter returns a Segmenter that iterates over the line break
// segments in the given input using default options.
func NewSegmenter(input []byte) *Segmenter {
	return &Segmenter{s: segmenter.New(&ruleData, input)}
}

// NewSegmenter returns a Segmenter that iterates over the line break
// segments in the given input, configured by o.
func (o *Options) NewSegmenter(input []byte) *Segmenter {
	seg := segmenter.New(&ruleData, input)

	if override := buildOverride(o.Strictness, o.WordBreak); override != nil {
		seg.SetOverrideLookup(override)
	}

	l := &Segmenter{s: seg, anywhere: o.Strictness == Anywhere}
	if l.anywhere {
		l.gs = grapheme.NewSegmenter(input)
	}
	return l
}

// buildOverride returns a composed property override function for the given
// CSS settings, or nil if no overrides are needed (Strict + WordNormal).
func buildOverride(strictness Strictness, wb WordBreak) func(uint8, rune) uint8 {
	needStrictness := strictness == Normal || strictness == Loose
	needWordBreak := wb != WordNormal

	if !needStrictness && !needWordBreak {
		return nil
	}

	return func(prop uint8, r rune) uint8 {
		if needStrictness {
			if prop == uint8(CJ) {
				prop = uint8(ID)
			}
		}

		if strictness == Loose {
			switch prop {
			case uint8(NS):
				if isLooseNS(r) {
					prop = uint8(ID)
				}
			case uint8(IN):
				prop = uint8(ID)
			}
		}

		switch wb {
		case WordBreakAll:
			if prop == uint8(AL) || prop == uint8(AI) || prop == uint8(SA) {
				prop = uint8(ID)
			}
		case WordKeepAll:
			if prop == uint8(ID) || prop == uint8(ID_ExtPict) || prop == uint8(CJ) ||
				prop == uint8(H2) || prop == uint8(H3) || prop == uint8(JL) || prop == uint8(JV) || prop == uint8(JT) {
				prop = uint8(AL)
			}
		}

		return prop
	}
}

// isLooseNS reports whether r is an NS codepoint that should be treated as
// ID under CSS line-break: loose, allowing a break before it.
// Per CSS Text Level 3 §5.1, this includes CJK wave dashes, iteration marks,
// and certain centered punctuation marks.
func isLooseNS(r rune) bool {
	switch r {
	case '\u301C', '\u30A0':
		return true
	case '\u3005', '\u303B', '\u309D', '\u309E', '\u30FD', '\u30FE':
		return true
	case '\u30FB', '\uFF1A', '\uFF1B', '\uFF65':
		return true
	case '\u203C':
		return true
	}
	return r >= '\u2047' && r <= '\u2049'
}

// Next advances to the next line break segment. It returns false when the
// end of input has been reached.
func (l *Segmenter) Next() bool {
	if l.anywhere {
		return l.nextAnywhere()
	}
	return l.s.Next()
}

// nextAnywhere implements CSS line-break: anywhere by breaking after every
// extended grapheme cluster (typographic character unit per CSS Text 3).
func (l *Segmenter) nextAnywhere() bool {
	return l.gs.Next()
}

// Bytes returns the current segment as a byte slice.
func (l *Segmenter) Bytes() []byte {
	if l.anywhere {
		return l.gs.Bytes()
	}
	return l.s.Bytes()
}

// Text returns the current segment as a string.
func (l *Segmenter) Text() string {
	if l.anywhere {
		return l.gs.Text()
	}
	return l.s.Text()
}

// Position returns the byte offsets [start, end) of the current segment.
func (l *Segmenter) Position() (start, end int) {
	if l.anywhere {
		return l.gs.Position()
	}
	return l.s.Position()
}

// MustBreak returns whether there is a mandatory break at the current
// position. This is true for hard line breaks such as U+000A (LF) and U+000D
// (CR), but not for soft line breaks such as spaces.
func (l *Segmenter) MustBreak() bool {
	var p uint8
	if l.anywhere {
		b := l.gs.Bytes()
		if len(b) > 0 {
			p, _ = ruleData.PropertyLookup(b)
		}
	} else {
		p = l.s.BoundaryProperty()
	}
	return p == uint8(BK) || p == uint8(CR) || p == uint8(LF) || p == uint8(NL)
}
