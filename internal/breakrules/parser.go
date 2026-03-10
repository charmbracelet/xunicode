package breakrules

import (
	"fmt"
	"strconv"
)

// Parser reads a token stream from a Lexer and produces a RuleSet AST.
type Parser struct {
	lex *Lexer
	tok Token
	err []string
}

// Parse parses the entire source and returns the RuleSet.
// If there are syntax errors, they are collected in the returned error slice.
func Parse(src []byte) (*RuleSet, []error) {
	p := &Parser{lex: NewLexer(src)}
	p.advance()
	rs := p.parseRuleSet()
	if len(p.err) > 0 {
		errs := make([]error, len(p.err))
		for i, e := range p.err {
			errs[i] = fmt.Errorf("%s", e)
		}
		return rs, errs
	}
	return rs, nil
}

func (p *Parser) advance() {
	p.tok = p.lex.Next()
}

func (p *Parser) expect(kind TokenKind) Token {
	if p.tok.Kind != kind {
		p.errorf("expected %d, got %d (%q) at pos %d", kind, p.tok.Kind, p.tok.Value, p.tok.Pos)
		return p.tok
	}
	t := p.tok
	p.advance()
	return t
}

func (p *Parser) errorf(format string, args ...any) {
	p.err = append(p.err, fmt.Sprintf(format, args...))
}

func (p *Parser) parseRuleSet() *RuleSet {
	rs := &RuleSet{Controls: make(map[string]bool)}
	for p.tok.Kind != tokEOF && p.tok.Kind != tokError {
		p.parseStatement(rs)
	}
	if p.tok.Kind == tokError {
		p.errorf("lexer error: %s at pos %d", p.tok.Value, p.tok.Pos)
	}
	return rs
}

func (p *Parser) parseStatement(rs *RuleSet) {
	switch p.tok.Kind {
	case tokControl:
		name := p.tok.Value
		p.advance()
		p.expect(tokSemicolon)
		rs.Controls[name] = true
	case tokVariable:
		if p.isAssignment() {
			p.parseAssignment(rs)
		} else {
			p.parseRule(rs)
		}
	default:
		p.parseRule(rs)
	}
}

// isAssignment peeks ahead to see if the next token after the current variable
// is '='. Uses a simple lookahead: save/restore position isn't needed because
// the lexer is forward-only, but we can peek by saving state.
func (p *Parser) isAssignment() bool {
	saved := *p.lex
	savedTok := p.tok
	p.advance()
	isEq := p.tok.Kind == tokEquals
	*p.lex = saved
	p.tok = savedTok
	return isEq
}

func (p *Parser) parseAssignment(rs *RuleSet) {
	pos := p.tok.Pos
	name := p.tok.Value
	p.advance()         // consume $variable
	p.expect(tokEquals) // consume '='
	expr := p.parseExpr()
	if expr == nil {
		p.errorf("empty expression in assignment $%s at pos %d", name, pos)
	}
	p.expect(tokSemicolon)
	rs.Assignments = append(rs.Assignments, &Assignment{
		Name:      name,
		Expr:      expr,
		SourcePos: pos,
	})
}

func (p *Parser) parseRule(rs *RuleSet) {
	pos := p.tok.Pos
	noChanIn := false
	if p.tok.Kind == tokCaret {
		noChanIn = true
		p.advance()
	}

	expr := p.parseExpr()
	tag := -1

	// Check for lookahead '/'
	if p.tok.Kind == tokSlash {
		slashPos := p.tok.Pos
		p.advance()
		postCtx := p.parseExpr()
		expr = &Node{
			Kind:     NodeConcat,
			Children: []*Node{expr, {Kind: NodeSlash, Tag: slashPos}, postCtx},
		}
	}

	// Check for status tag {N}
	if p.tok.Kind == tokNumber {
		n, err := strconv.Atoi(p.tok.Value)
		if err != nil {
			p.errorf("invalid tag number %q at pos %d", p.tok.Value, p.tok.Pos)
		} else {
			tag = n
		}
		p.advance()
	}

	p.expect(tokSemicolon)
	rs.Rules = append(rs.Rules, &Rule{
		Expr:      expr,
		NoChanIn:  noChanIn,
		Tag:       tag,
		SourcePos: pos,
	})
}

// parseExpr handles alternation (lowest precedence).
// expr ::= concat ('|' concat)*
func (p *Parser) parseExpr() *Node {
	left := p.parseConcat()
	if left == nil {
		return left
	}
	for p.tok.Kind == tokPipe {
		p.advance()
		right := p.parseConcat()
		if right == nil {
			p.errorf("expected expression after '|' at pos %d", p.tok.Pos)
			return left
		}
		left = &Node{Kind: NodeAlt, Children: []*Node{left, right}}
	}
	return left
}

// parseConcat handles concatenation (implicit, higher precedence than alternation).
// concat ::= postfix+
func (p *Parser) parseConcat() *Node {
	first := p.parsePostfix()
	if first == nil {
		return nil
	}
	parts := []*Node{first}
	for {
		next := p.parsePostfix()
		if next == nil {
			break
		}
		parts = append(parts, next)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return &Node{Kind: NodeConcat, Children: parts}
}

// parsePostfix handles postfix operators (*, +, ?).
// postfix ::= atom ('*' | '+' | '?')?
func (p *Parser) parsePostfix() *Node {
	atom := p.parseAtom()
	if atom == nil {
		return nil
	}
	switch p.tok.Kind {
	case tokStar:
		p.advance()
		return &Node{Kind: NodeStar, Child: atom}
	case tokPlus:
		p.advance()
		return &Node{Kind: NodePlus, Child: atom}
	case tokQuestion:
		p.advance()
		return &Node{Kind: NodeQuest, Child: atom}
	}
	return atom
}

// parseAtom handles atomic expressions.
// atom ::= char | set | variable | '(' expr ')' | '.'
func (p *Parser) parseAtom() *Node {
	switch p.tok.Kind {
	case tokChar:
		val := p.tok.Value
		p.advance()
		if val == "{bof}" {
			return &Node{Kind: NodeBOF}
		}
		if val == "{eof}" {
			return &Node{Kind: NodeEOF}
		}
		return &Node{Kind: NodeCharClass, Name: val}
	case tokDot:
		p.advance()
		return &Node{Kind: NodeDot}
	case tokVariable:
		name := p.tok.Value
		p.advance()
		return &Node{Kind: NodeVariable, Name: name}
	case tokUnicodeSet:
		val := p.tok.Value
		p.advance()
		return &Node{Kind: NodeCharClass, Name: val}
	case tokLParen:
		p.advance()
		expr := p.parseExpr()
		p.expect(tokRParen)
		return expr
	default:
		return nil
	}
}
