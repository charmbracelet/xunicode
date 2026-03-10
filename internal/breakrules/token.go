package breakrules

// TokenKind classifies lexer tokens.
type TokenKind int

const (
	// Literals and identifiers.
	tokEOF       TokenKind = iota
	tokError               // lexer error; Value contains the message
	tokChar                // single literal character (Value[0])
	tokDot                 // '.' — match any
	tokVariable            // '$name'
	tokNumber              // integer (e.g. inside {100})
	tokUnicodeSet          // '[...]' or '[:...:]' or '\p{...}' — raw text for set parser

	// Operators.
	tokStar      // '*'
	tokPlus      // '+'
	tokQuestion  // '?'
	tokPipe      // '|'
	tokSlash     // '/' — lookahead break point
	tokCaret     // '^' — no-chain-in
	tokLParen    // '('
	tokRParen    // ')'
	tokEquals    // '='
	tokLBrace    // '{'
	tokRBrace    // '}'
	tokSemicolon // ';'

	// Controls — the !! directives.
	tokControl // !!chain, !!forward, etc. — Value holds the directive name
)

// Token is a single lexical unit from a .rules file.
type Token struct {
	Kind  TokenKind
	Value string // textual content (variable name, set expression, control name, error message, etc.)
	Pos   int    // byte offset in the source
}
