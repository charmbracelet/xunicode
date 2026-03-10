package breakrules

import (
	"testing"
)

func TestLexerBasicTokens(t *testing.T) {
	src := `!!chain; !!quoted_literals_only;`
	lex := NewLexer([]byte(src))
	expect := []struct {
		kind TokenKind
		val  string
	}{
		{tokControl, "chain"},
		{tokSemicolon, ""},
		{tokControl, "quoted_literals_only"},
		{tokSemicolon, ""},
		{tokEOF, ""},
	}
	for i, e := range expect {
		tok := lex.Next()
		if tok.Kind != e.kind {
			t.Fatalf("token %d: got kind %d, want %d (val=%q)", i, tok.Kind, e.kind, tok.Value)
		}
		if e.val != "" && tok.Value != e.val {
			t.Fatalf("token %d: got value %q, want %q", i, tok.Value, e.val)
		}
	}
}

func TestLexerVariablesAndSets(t *testing.T) {
	src := `$CR = [\p{Grapheme_Cluster_Break = CR}];`
	lex := NewLexer([]byte(src))
	expect := []struct {
		kind TokenKind
		val  string
	}{
		{tokVariable, "CR"},
		{tokEquals, ""},
		{tokUnicodeSet, `[\p{Grapheme_Cluster_Break = CR}]`},
		{tokSemicolon, ""},
		{tokEOF, ""},
	}
	for i, e := range expect {
		tok := lex.Next()
		if tok.Kind != e.kind {
			t.Fatalf("token %d: got kind %d, want %d (val=%q)", i, tok.Kind, e.kind, tok.Value)
		}
		if e.val != "" && tok.Value != e.val {
			t.Fatalf("token %d: got value %q, want %q", i, tok.Value, e.val)
		}
	}
}

func TestLexerRuleExpr(t *testing.T) {
	src := `$CR $LF;`
	lex := NewLexer([]byte(src))
	expect := []struct {
		kind TokenKind
		val  string
	}{
		{tokVariable, "CR"},
		{tokVariable, "LF"},
		{tokSemicolon, ""},
		{tokEOF, ""},
	}
	for i, e := range expect {
		tok := lex.Next()
		if tok.Kind != e.kind {
			t.Fatalf("token %d: got kind %d, want %d (val=%q)", i, tok.Kind, e.kind, tok.Value)
		}
		if e.val != "" && tok.Value != e.val {
			t.Fatalf("token %d: got value %q, want %q", i, tok.Value, e.val)
		}
	}
}

func TestLexerOperators(t *testing.T) {
	src := `($A | $B)* $C+ $D? . ;`
	lex := NewLexer([]byte(src))
	expect := []TokenKind{
		tokLParen, tokVariable, tokPipe, tokVariable, tokRParen, tokStar,
		tokVariable, tokPlus, tokVariable, tokQuestion, tokDot, tokSemicolon, tokEOF,
	}
	for i, ek := range expect {
		tok := lex.Next()
		if tok.Kind != ek {
			t.Fatalf("token %d: got kind %d, want %d (val=%q)", i, tok.Kind, ek, tok.Value)
		}
	}
}

func TestLexerComment(t *testing.T) {
	src := "# this is a comment\n$A;"
	lex := NewLexer([]byte(src))
	tok := lex.Next()
	if tok.Kind != tokVariable || tok.Value != "A" {
		t.Fatalf("got %+v, want variable A", tok)
	}
}

func TestLexerBackslashEscapes(t *testing.T) {
	src := `\u0041 \U00000042 \n \p{Lu}`
	lex := NewLexer([]byte(src))

	tok := lex.Next()
	if tok.Kind != tokChar || tok.Value != "A" {
		t.Fatalf("\\u0041: got %+v", tok)
	}
	tok = lex.Next()
	if tok.Kind != tokChar || tok.Value != "B" {
		t.Fatalf("\\U00000042: got %+v", tok)
	}
	tok = lex.Next()
	if tok.Kind != tokChar || tok.Value != "\n" {
		t.Fatalf("\\n: got %+v", tok)
	}
	tok = lex.Next()
	if tok.Kind != tokUnicodeSet || tok.Value != `\p{Lu}` {
		t.Fatalf("\\p{Lu}: got %+v", tok)
	}
}

func TestLexerQuotedLiterals(t *testing.T) {
	src := `'hello' '''' ;`
	lex := NewLexer([]byte(src))

	tok := lex.Next()
	if tok.Kind != tokChar || tok.Value != "hello" {
		t.Fatalf("'hello': got %+v", tok)
	}
	tok = lex.Next()
	if tok.Kind != tokChar || tok.Value != "'" {
		t.Fatalf("'''': got %+v", tok)
	}
	tok = lex.Next()
	if tok.Kind != tokSemicolon {
		t.Fatalf("semicolon: got %+v", tok)
	}
}

func TestLexerCaretSlashLookahead(t *testing.T) {
	src := `^$A $B / $C;`
	lex := NewLexer([]byte(src))
	expect := []TokenKind{
		tokCaret, tokVariable, tokVariable, tokSlash, tokVariable, tokSemicolon, tokEOF,
	}
	for i, ek := range expect {
		tok := lex.Next()
		if tok.Kind != ek {
			t.Fatalf("token %d: got kind %d, want %d (val=%q)", i, tok.Kind, ek, tok.Value)
		}
	}
}

func TestLexerStatusTag(t *testing.T) {
	src := `$A {200};`
	lex := NewLexer([]byte(src))
	tok := lex.Next() // $A
	if tok.Kind != tokVariable {
		t.Fatalf("got %+v, want variable", tok)
	}
	tok = lex.Next() // {200}
	if tok.Kind != tokNumber || tok.Value != "200" {
		t.Fatalf("got %+v, want number 200", tok)
	}
	tok = lex.Next() // ;
	if tok.Kind != tokSemicolon {
		t.Fatalf("got %+v, want semicolon", tok)
	}
}

func TestLexerNestedSets(t *testing.T) {
	src := `[[$A & $B] - [$C]]`
	lex := NewLexer([]byte(src))
	tok := lex.Next()
	if tok.Kind != tokUnicodeSet || tok.Value != src {
		t.Fatalf("got %+v, want unicode set %q", tok, src)
	}
}

func TestLexerBOFEOF(t *testing.T) {
	src := `{bof} {eof};`
	lex := NewLexer([]byte(src))
	tok := lex.Next()
	if tok.Kind != tokChar || tok.Value != "{bof}" {
		t.Fatalf("bof: got %+v", tok)
	}
	tok = lex.Next()
	if tok.Kind != tokChar || tok.Value != "{eof}" {
		t.Fatalf("eof: got %+v", tok)
	}
}

func TestLexerGraphemeRules(t *testing.T) {
	// Simplified version of ICU char.txt
	src := `!!quoted_literals_only;
!!chain;
!!lookAheadHardBreak;
$CR = [\p{Grapheme_Cluster_Break = CR}];
$LF = [\p{Grapheme_Cluster_Break = LF}];
$CR $LF;
[^$Control $CR $LF] ($Extend | $ZWJ);
.;
`
	lex := NewLexer([]byte(src))
	var tokens []Token
	for {
		tok := lex.Next()
		tokens = append(tokens, tok)
		if tok.Kind == tokEOF || tok.Kind == tokError {
			break
		}
	}
	last := tokens[len(tokens)-1]
	if last.Kind != tokEOF {
		t.Fatalf("expected EOF, got %+v", last)
	}
	// Should have no errors
	for _, tok := range tokens {
		if tok.Kind == tokError {
			t.Fatalf("lexer error: %s at pos %d", tok.Value, tok.Pos)
		}
	}
}
