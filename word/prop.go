package word

import "unicode/utf8"

// Properties provides access to word break properties of a rune.
type Properties struct {
	entry uint8
}

// Class returns the word break class.
func (p Properties) Class() Class {
	return Class(p.entry)
}

// IsNewline reports whether the rune has class Newline, CR, or LF (WB3/WB3a/WB3b).
func (p Properties) IsNewline() bool {
	c := p.entry
	return c == uint8(Newline) || c == uint8(CR) || c == uint8(LF)
}

// IsIgnored reports whether the rune has class Extend, Format, or ZWJ — the
// set of classes ignored between other classes under rule WB4.
func (p Properties) IsIgnored() bool {
	c := p.entry
	return c == uint8(Extend) || c == uint8(Format) || c == uint8(ZWJ)
}

// IsAHLetter reports whether the rune has class ALetter, Hebrew_Letter, or
// ALetter_Extended_Pictographic — the combined AHLetter group used in
// rules WB5–WB7.
func (p Properties) IsAHLetter() bool {
	c := p.entry
	return c == uint8(ALetter) || c == uint8(Hebrew_Letter) || c == uint8(ALetter_Extended_Pictographic)
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
