package breakrules

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes ICU RBBI rule source text.
type Lexer struct {
	src            []byte
	pos            int
	quotedLiterals bool // true after !!quoted_literals_only
}

// NewLexer returns a lexer for the given source text.
func NewLexer(src []byte) *Lexer {
	return &Lexer{src: src}
}

// Next returns the next token. Returns tokEOF at end of input.
func (l *Lexer) Next() Token {
	l.skipWhitespaceAndComments()
	if l.pos >= len(l.src) {
		return Token{Kind: tokEOF, Pos: l.pos}
	}
	start := l.pos
	ch, sz := utf8.DecodeRune(l.src[l.pos:])

	switch {
	case ch == '!' && l.peek(1) == '!':
		return l.lexControl()
	case ch == '$':
		return l.lexVariable()
	case ch == '[':
		return l.lexUnicodeSet()
	case ch == '\\':
		return l.lexEscape()
	case ch == '\'':
		return l.lexQuoted()
	case ch == '.':
		l.pos += sz
		return Token{Kind: tokDot, Pos: start}
	case ch == '*':
		l.pos += sz
		return Token{Kind: tokStar, Pos: start}
	case ch == '+':
		l.pos += sz
		return Token{Kind: tokPlus, Pos: start}
	case ch == '?':
		l.pos += sz
		return Token{Kind: tokQuestion, Pos: start}
	case ch == '|':
		l.pos += sz
		return Token{Kind: tokPipe, Pos: start}
	case ch == '/':
		l.pos += sz
		return Token{Kind: tokSlash, Pos: start}
	case ch == '^':
		l.pos += sz
		return Token{Kind: tokCaret, Pos: start}
	case ch == '(':
		l.pos += sz
		return Token{Kind: tokLParen, Pos: start}
	case ch == ')':
		l.pos += sz
		return Token{Kind: tokRParen, Pos: start}
	case ch == '{':
		return l.lexBrace()
	case ch == ';':
		l.pos += sz
		return Token{Kind: tokSemicolon, Pos: start}
	case ch == '=':
		l.pos += sz
		return Token{Kind: tokEquals, Pos: start}
	case l.quotedLiterals:
		return l.errToken(start, "unexpected character %q (!!quoted_literals_only is active)", ch)
	default:
		l.pos += sz
		return Token{Kind: tokChar, Value: string(ch), Pos: start}
	}
}

func (l *Lexer) peek(ahead int) rune {
	p := l.pos
	for i := 0; i < ahead; i++ {
		if p >= len(l.src) {
			return 0
		}
		_, sz := utf8.DecodeRune(l.src[p:])
		p += sz
	}
	if p >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.src[p:])
	return r
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == '#' {
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' {
			l.pos++
			continue
		}
		break
	}
}

// lexControl handles !!directive tokens.
func (l *Lexer) lexControl() Token {
	start := l.pos
	l.pos += 2 // skip '!!'
	nameStart := l.pos
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			l.pos++
		} else {
			break
		}
	}
	name := string(l.src[nameStart:l.pos])
	if name == "" {
		return l.errToken(start, "empty !! directive")
	}
	if name == "quoted_literals_only" {
		l.quotedLiterals = true
	}
	// The semicolon is consumed by the parser, not the lexer.
	return Token{Kind: tokControl, Value: name, Pos: start}
}

// lexVariable handles $name tokens.
func (l *Lexer) lexVariable() Token {
	start := l.pos
	l.pos++ // skip '$'
	nameStart := l.pos
	for l.pos < len(l.src) {
		r, sz := utf8.DecodeRune(l.src[l.pos:])
		if r == '_' || unicode.IsLetter(r) || (l.pos > nameStart && unicode.IsDigit(r)) {
			l.pos += sz
		} else {
			break
		}
	}
	name := string(l.src[nameStart:l.pos])
	if name == "" {
		return l.errToken(start, "empty variable name")
	}
	return Token{Kind: tokVariable, Value: name, Pos: start}
}

// lexUnicodeSet handles [...] expressions, tracking bracket nesting.
// Returns the raw text including the outer brackets.
func (l *Lexer) lexUnicodeSet() Token {
	start := l.pos
	depth := 0
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		switch ch {
		case '[':
			depth++
			l.pos++
		case ']':
			depth--
			l.pos++
			if depth == 0 {
				return Token{Kind: tokUnicodeSet, Value: string(l.src[start:l.pos]), Pos: start}
			}
		case '\\':
			l.pos++
			if l.pos < len(l.src) {
				_, sz := utf8.DecodeRune(l.src[l.pos:])
				l.pos += sz
			}
		case '\'':
			l.pos++
			for l.pos < len(l.src) {
				if l.src[l.pos] == '\'' {
					l.pos++
					if l.pos < len(l.src) && l.src[l.pos] == '\'' {
						l.pos++ // escaped single quote ''
						continue
					}
					break
				}
				_, sz := utf8.DecodeRune(l.src[l.pos:])
				l.pos += sz
			}
		default:
			_, sz := utf8.DecodeRune(l.src[l.pos:])
			l.pos += sz
		}
	}
	return l.errToken(start, "unterminated UnicodeSet expression")
}

// lexEscape handles \ escapes. Produces either a tokChar or the start of a
// \p{...} / \P{...} UnicodeSet.
func (l *Lexer) lexEscape() Token {
	start := l.pos
	l.pos++ // skip '\'
	if l.pos >= len(l.src) {
		return l.errToken(start, "trailing backslash")
	}
	r, sz := utf8.DecodeRune(l.src[l.pos:])
	switch r {
	case 'p', 'P':
		l.pos += sz
		if l.pos < len(l.src) && l.src[l.pos] == '{' {
			braceStart := l.pos
			l.pos++ // skip '{'
			for l.pos < len(l.src) && l.src[l.pos] != '}' {
				l.pos++
			}
			if l.pos >= len(l.src) {
				return l.errToken(start, "unterminated \\%c{...}", r)
			}
			l.pos++ // skip '}'
			_ = braceStart
			return Token{Kind: tokUnicodeSet, Value: string(l.src[start:l.pos]), Pos: start}
		}
		// \pL — single-letter property name
		if l.pos < len(l.src) {
			_, sz2 := utf8.DecodeRune(l.src[l.pos:])
			l.pos += sz2
		}
		return Token{Kind: tokUnicodeSet, Value: string(l.src[start:l.pos]), Pos: start}
	case 'u':
		return l.lexUnicodeEscape(start, 4)
	case 'U':
		return l.lexUnicodeEscape(start, 8)
	case 'x':
		return l.lexHexEscape(start)
	case 'a':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\a", Pos: start}
	case 'b':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\b", Pos: start}
	case 't':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\t", Pos: start}
	case 'n':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\n", Pos: start}
	case 'v':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\v", Pos: start}
	case 'f':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\f", Pos: start}
	case 'r':
		l.pos += sz
		return Token{Kind: tokChar, Value: "\r", Pos: start}
	default:
		l.pos += sz
		return Token{Kind: tokChar, Value: string(r), Pos: start}
	}
}

func (l *Lexer) lexUnicodeEscape(start, digits int) Token {
	l.pos++ // skip 'u' or 'U'
	hex := make([]byte, 0, digits)
	for i := 0; i < digits && l.pos < len(l.src); i++ {
		hex = append(hex, l.src[l.pos])
		l.pos++
	}
	if len(hex) != digits {
		return l.errToken(start, "incomplete \\u escape: need %d hex digits", digits)
	}
	var cp rune
	for _, b := range hex {
		cp <<= 4
		switch {
		case b >= '0' && b <= '9':
			cp |= rune(b - '0')
		case b >= 'a' && b <= 'f':
			cp |= rune(b - 'a' + 10)
		case b >= 'A' && b <= 'F':
			cp |= rune(b - 'A' + 10)
		default:
			return l.errToken(start, "invalid hex digit %q in \\u escape", b)
		}
	}
	return Token{Kind: tokChar, Value: string(cp), Pos: start}
}

func (l *Lexer) lexHexEscape(start int) Token {
	l.pos++ // skip 'x'
	var cp rune
	count := 0
	for count < 2 && l.pos < len(l.src) {
		b := l.src[l.pos]
		d, ok := hexVal(b)
		if !ok {
			break
		}
		cp = cp<<4 | d
		l.pos++
		count++
	}
	if count == 0 {
		return l.errToken(start, "incomplete \\x escape")
	}
	return Token{Kind: tokChar, Value: string(cp), Pos: start}
}

func hexVal(b byte) (rune, bool) {
	switch {
	case b >= '0' && b <= '9':
		return rune(b - '0'), true
	case b >= 'a' && b <= 'f':
		return rune(b - 'a' + 10), true
	case b >= 'A' && b <= 'F':
		return rune(b - 'A' + 10), true
	default:
		return 0, false
	}
}

// lexQuoted handles '...' quoted sequences. Two adjacent single quotes ''
// represent a literal single quote.
func (l *Lexer) lexQuoted() Token {
	start := l.pos
	l.pos++ // skip opening '
	var result []byte
	for l.pos < len(l.src) {
		if l.src[l.pos] == '\'' {
			l.pos++
			if l.pos < len(l.src) && l.src[l.pos] == '\'' {
				result = append(result, '\'')
				l.pos++
				continue
			}
			// End of quoted sequence. If single char, return as tokChar.
			s := string(result)
			r, rsz := utf8.DecodeRuneInString(s)
			if rsz == len(s) {
				return Token{Kind: tokChar, Value: string(r), Pos: start}
			}
			// Multi-char quoted literal — the parser will handle it.
			return Token{Kind: tokChar, Value: s, Pos: start}
		}
		r, sz := utf8.DecodeRune(l.src[l.pos:])
		result = append(result, []byte(string(r))...)
		l.pos += sz
	}
	return l.errToken(start, "unterminated quoted literal")
}

// lexBrace handles {N} status tags and {bof}/{eof} pseudo-anchors.
func (l *Lexer) lexBrace() Token {
	start := l.pos
	l.pos++ // skip '{'
	contentStart := l.pos
	for l.pos < len(l.src) && l.src[l.pos] != '}' {
		l.pos++
	}
	if l.pos >= len(l.src) {
		return l.errToken(start, "unterminated {}")
	}
	content := string(l.src[contentStart:l.pos])
	l.pos++ // skip '}'
	// Check for special anchors.
	if content == "bof" {
		return Token{Kind: tokChar, Value: "{bof}", Pos: start}
	}
	if content == "eof" {
		return Token{Kind: tokChar, Value: "{eof}", Pos: start}
	}
	// Otherwise it's a numeric tag.
	return Token{Kind: tokNumber, Value: content, Pos: start}
}

func (l *Lexer) errToken(pos int, format string, args ...any) Token {
	return Token{Kind: tokError, Value: fmt.Sprintf(format, args...), Pos: pos}
}
