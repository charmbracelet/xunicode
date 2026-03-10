package breakrules

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// PropertyResolver maps a property expression to a set of category IDs.
// The expression is the content of \p{...} (e.g. "Grapheme_Cluster_Break=CR")
// or [:...:] (e.g. "Letter"). The negated flag is true for \P{...} or [:^...:].
type PropertyResolver func(expr string, negated bool) ([]uint16, error)

// ResolveUnicodeSets walks the AST and resolves all NodeCharClass nodes
// that have a Name (raw text) but empty Classes. It parses the Name as a
// UnicodeSet expression and populates Classes with category IDs.
//
// The resolver function maps property expressions to category ID sets.
// The vars map provides resolved variable name → Classes for $var references
// inside [...] expressions.
func ResolveUnicodeSets(rs *RuleSet, numCats int, resolver PropertyResolver) []error {
	vars := make(map[string][]uint16)
	for _, a := range rs.Assignments {
		if a.Expr != nil && a.Expr.Kind == NodeCharClass && len(a.Expr.Classes) > 0 {
			vars[a.Name] = a.Expr.Classes
		}
	}

	var errs []error
	for _, a := range rs.Assignments {
		if err := resolveNodeSets(a.Expr, numCats, resolver, vars); err != nil {
			errs = append(errs, err...)
		}
		if a.Expr != nil && a.Expr.Kind == NodeCharClass && len(a.Expr.Classes) > 0 {
			vars[a.Name] = a.Expr.Classes
		}
	}
	for _, r := range rs.Rules {
		if err := resolveNodeSets(r.Expr, numCats, resolver, vars); err != nil {
			errs = append(errs, err...)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func resolveNodeSets(n *Node, numCats int, resolver PropertyResolver, vars map[string][]uint16) []error {
	if n == nil {
		return nil
	}
	var errs []error
	switch n.Kind {
	case NodeCharClass:
		if n.Name != "" && len(n.Classes) == 0 {
			classes, err := parseUnicodeSetExpr(n.Name, numCats, resolver, vars)
			if err != nil {
				errs = append(errs, err)
			} else {
				n.Classes = classes
				n.Name = ""
			}
		}
	case NodeConcat, NodeAlt:
		for _, c := range n.Children {
			if e := resolveNodeSets(c, numCats, resolver, vars); e != nil {
				errs = append(errs, e...)
			}
		}
	case NodeStar, NodePlus, NodeQuest:
		if e := resolveNodeSets(n.Child, numCats, resolver, vars); e != nil {
			errs = append(errs, e...)
		}
	}
	return errs
}

// parseUnicodeSetExpr parses a raw UnicodeSet expression string and returns
// the set of matching category IDs.
//
// Supported forms:
//   - \p{prop=value} or \p{prop}    → property lookup
//   - \P{prop=value} or \P{prop}    → negated property lookup
//   - \pL                           → single-letter property shorthand
//   - [...]                         → bracketed set expression
//   - single character literal      → exact character lookup (via resolver)
func parseUnicodeSetExpr(raw string, numCats int, resolver PropertyResolver, vars map[string][]uint16) ([]uint16, error) {
	if strings.HasPrefix(raw, `\p{`) || strings.HasPrefix(raw, `\P{`) {
		negated := raw[1] == 'P'
		inner := raw[3 : len(raw)-1]
		return resolver(inner, negated)
	}
	if strings.HasPrefix(raw, `\p`) || strings.HasPrefix(raw, `\P`) {
		negated := raw[1] == 'P'
		inner := raw[2:]
		return resolver(inner, negated)
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		p := &setParser{
			src:      raw,
			pos:      0,
			numCats:  numCats,
			resolver: resolver,
			vars:     vars,
		}
		return p.parseTopLevel()
	}
	return resolver(raw, false)
}

type setParser struct {
	src      string
	pos      int
	numCats  int
	resolver PropertyResolver
	vars     map[string][]uint16
}

func (p *setParser) parseTopLevel() ([]uint16, error) {
	if p.pos >= len(p.src) || p.src[p.pos] != '[' {
		return nil, fmt.Errorf("expected '[' at pos %d", p.pos)
	}
	return p.parseSet()
}

// parseSet parses a [...] expression, returning category IDs.
func (p *setParser) parseSet() ([]uint16, error) {
	if p.pos >= len(p.src) || p.src[p.pos] != '[' {
		return nil, fmt.Errorf("expected '[' at pos %d", p.pos)
	}
	p.pos++ // skip '['

	complement := false
	if p.pos < len(p.src) && p.src[p.pos] == '^' {
		complement = true
		p.pos++
	}

	result := make(map[uint16]bool)

	for p.pos < len(p.src) && p.src[p.pos] != ']' {
		p.skipSpaces()
		if p.pos >= len(p.src) || p.src[p.pos] == ']' {
			break
		}

		// Set operations: & (intersection) and - (difference)
		if p.src[p.pos] == '&' {
			p.pos++
			p.skipSpaces()
			right, err := p.parseSetOperand()
			if err != nil {
				return nil, err
			}
			rightSet := toSet(right)
			for k := range result {
				if !rightSet[k] {
					delete(result, k)
				}
			}
			continue
		}
		if p.src[p.pos] == '-' {
			p.pos++
			p.skipSpaces()
			right, err := p.parseSetOperand()
			if err != nil {
				return nil, err
			}
			for _, v := range right {
				delete(result, v)
			}
			continue
		}

		operand, err := p.parseSetOperand()
		if err != nil {
			return nil, err
		}
		for _, v := range operand {
			result[v] = true
		}
	}

	if p.pos < len(p.src) && p.src[p.pos] == ']' {
		p.pos++
	}

	cats := fromSet(result)

	if complement {
		compSet := make(map[uint16]bool)
		for i := uint16(0); i < uint16(p.numCats); i++ {
			compSet[i] = true
		}
		for _, c := range cats {
			delete(compSet, c)
		}
		cats = fromSet(compSet)
	}

	return cats, nil
}

// parseSetOperand parses a single operand within a set expression:
// a nested [...], a property expression, a $variable, or literal chars.
func (p *setParser) parseSetOperand() ([]uint16, error) {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of set expression")
	}

	ch := p.src[p.pos]

	// Nested set: [...]
	if ch == '[' {
		return p.parseSet()
	}

	// POSIX-style property: [:prop=value:]
	if ch == ':' && p.pos+1 < len(p.src) {
		return p.parsePOSIXProperty()
	}

	// Backslash escape: \p{...}, \P{...}, \uHHHH, etc.
	if ch == '\\' {
		return p.parseBackslashInSet()
	}

	// $variable reference
	if ch == '$' {
		return p.parseVarRef()
	}

	// Literal character(s) — treat as name to resolve
	return p.parseLiteralInSet()
}

func (p *setParser) parsePOSIXProperty() ([]uint16, error) {
	if p.src[p.pos] != ':' {
		return nil, fmt.Errorf("expected ':' at pos %d", p.pos)
	}
	p.pos++ // skip ':'

	negated := false
	if p.pos < len(p.src) && p.src[p.pos] == '^' {
		negated = true
		p.pos++
	}

	start := p.pos
	for p.pos < len(p.src) {
		if p.src[p.pos] == ':' && p.pos+1 < len(p.src) && p.src[p.pos+1] == ']' {
			break
		}
		p.pos++
	}
	expr := p.src[start:p.pos]
	if p.pos < len(p.src) && p.src[p.pos] == ':' {
		p.pos++ // skip closing ':'
	}
	// The closing ']' is consumed by the outer parseSet
	return p.resolver(expr, negated)
}

func (p *setParser) parseBackslashInSet() ([]uint16, error) {
	p.pos++ // skip '\'
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("trailing backslash in set expression")
	}
	ch := p.src[p.pos]
	if ch == 'p' || ch == 'P' {
		negated := ch == 'P'
		p.pos++
		if p.pos < len(p.src) && p.src[p.pos] == '{' {
			p.pos++ // skip '{'
			start := p.pos
			for p.pos < len(p.src) && p.src[p.pos] != '}' {
				p.pos++
			}
			expr := p.src[start:p.pos]
			if p.pos < len(p.src) {
				p.pos++ // skip '}'
			}
			return p.resolver(expr, negated)
		}
		// Single-letter property
		if p.pos < len(p.src) {
			_, sz := utf8.DecodeRuneInString(p.src[p.pos:])
			expr := p.src[p.pos : p.pos+sz]
			p.pos += sz
			return p.resolver(expr, negated)
		}
		return nil, fmt.Errorf("incomplete \\%c in set expression", ch)
	}
	// Other escape: \uHHHH, \n, etc. — treat as literal
	_, sz := utf8.DecodeRuneInString(p.src[p.pos:])
	literal := p.src[p.pos : p.pos+sz]
	p.pos += sz
	return p.resolver(literal, false)
}

func (p *setParser) parseVarRef() ([]uint16, error) {
	p.pos++ // skip '$'
	start := p.pos
	for p.pos < len(p.src) {
		r, sz := utf8.DecodeRuneInString(p.src[p.pos:])
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(p.pos > start && r >= '0' && r <= '9') {
			p.pos += sz
		} else {
			break
		}
	}
	name := p.src[start:p.pos]
	if name == "" {
		return nil, fmt.Errorf("empty variable name in set expression")
	}
	cats, ok := p.vars[name]
	if !ok {
		return nil, fmt.Errorf("undefined variable $%s in set expression", name)
	}
	return cats, nil
}

func (p *setParser) parseLiteralInSet() ([]uint16, error) {
	_, sz := utf8.DecodeRuneInString(p.src[p.pos:])
	literal := p.src[p.pos : p.pos+sz]
	p.pos += sz
	return p.resolver(literal, false)
}

func (p *setParser) skipSpaces() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t') {
		p.pos++
	}
}

func toSet(ids []uint16) map[uint16]bool {
	m := make(map[uint16]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func fromSet(m map[uint16]bool) []uint16 {
	result := make([]uint16, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	// Sort for determinism
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j-1] > result[j]; j-- {
			result[j-1], result[j] = result[j], result[j-1]
		}
	}
	return result
}
