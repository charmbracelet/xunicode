package line

import "unicode/utf8"

// Properties provides access to line break properties of a rune.
type Properties struct {
	entry uint8
}

// Class returns the line break class.
func (p Properties) Class() Class {
	return Class(p.entry)
}

// IsMandatoryBreak reports whether the rune has class BK, CR, LF, or NL —
// the set of classes that force a mandatory line break.
func (p Properties) IsMandatoryBreak() bool {
	c := p.entry
	return c == uint8(BK) || c == uint8(CR) || c == uint8(LF) || c == uint8(NL)
}

// IsHangul reports whether the rune has a Hangul jamo or syllable class
// (JL, JV, JT, H2, or H3).
func (p Properties) IsHangul() bool {
	c := p.entry
	return c == uint8(JL) || c == uint8(JV) || c == uint8(JT) || c == uint8(H2) || c == uint8(H3)
}

// IsClosing reports whether the rune has class CL, CP, EX, IS, or SY — the
// set of closing or related punctuation that prohibits breaks before
// them (LB13).
func (p Properties) IsClosing() bool {
	c := p.entry
	return c == uint8(CL) || c == uint8(CP) || c == uint8(EX) || c == uint8(IS) || c == uint8(SY)
}

// Lookup returns properties for the first rune in s and the width in bytes of
// its encoding. The size will be 0 if s does not hold enough bytes to complete
// the encoding.
func Lookup(s []byte) (Properties, int) {
	v, sz := trie.lookup(s)
	return Properties{entry: v}, sz
}

// LookupString returns properties for the first rune in s and the width in
// bytes of its encoding. The size will be 0 if s does not hold enough bytes to
// complete the encoding.
func LookupString(s string) (Properties, int) {
	v, sz := trie.lookupString(s)
	return Properties{entry: v}, sz
}

// LookupRune returns properties for r.
func LookupRune(r rune) Properties {
	var buf [4]byte
	n := utf8.EncodeRune(buf[:], r)
	v, _ := trie.lookup(buf[:n])
	return Properties{entry: v}
}
