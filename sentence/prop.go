package sentence

import "unicode/utf8"

// Properties provides access to sentence break properties of a rune.
type Properties struct {
	entry uint8
}

// Class returns the sentence break class.
func (p Properties) Class() Class {
	return Class(p.entry)
}

// IsSep reports whether the rune has class Sep, CR, or LF — the paragraph
// separator set (SB4).
func (p Properties) IsSep() bool {
	c := p.entry
	return c == uint8(Sep) || c == uint8(CR) || c == uint8(LF)
}

// IsIgnored reports whether the rune has class Extend or Format — the set of
// classes ignored between other classes under rule SB5.
func (p Properties) IsIgnored() bool {
	c := p.entry
	return c == uint8(Extend) || c == uint8(Format)
}

// IsSATerm reports whether the rune has class STerm, ATerm, or SContinue.
func (p Properties) IsSATerm() bool {
	c := p.entry
	return c == uint8(STerm) || c == uint8(ATerm) || c == uint8(SContinue)
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
