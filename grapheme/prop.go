package grapheme

import "unicode/utf8"

// Properties provides access to grapheme cluster break properties of a rune.
type Properties struct {
	entry uint8
}

// Class returns the grapheme cluster break class.
func (p Properties) Class() Class {
	return Class(p.entry)
}

// IsControl reports whether the rune has class CR, LF, or Control (GB4/GB5).
func (p Properties) IsControl() bool {
	c := p.entry
	return c == uint8(CR) || c == uint8(LF) || c == uint8(Control)
}

// IsExtend reports whether the rune has class Extend, ZWJ, InCBExtend, or
// InCBLinker — the set of classes that are ignored between other classes
// under rules GB9/GB9a/GB9c.
func (p Properties) IsExtend() bool {
	c := p.entry
	return c == uint8(Extend) || c == uint8(ZWJ) || c == uint8(InCBExtend) || c == uint8(InCBLinker)
}

// IsHangul reports whether the rune has a Hangul jamo or syllable class
// (L, V, T, LV, or LVT).
func (p Properties) IsHangul() bool {
	return p.entry >= uint8(L) && p.entry <= uint8(LVT)
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
