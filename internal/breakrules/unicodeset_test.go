package breakrules

import (
	"fmt"
	"strings"
	"testing"
)

func testResolver(expr string, negated bool) ([]uint16, error) {
	normalizedExpr := strings.ReplaceAll(expr, " ", "")
	normalizedExpr = strings.ToLower(normalizedExpr)
	m := map[string][]uint16{
		"grapheme_cluster_break=cr":          {1},
		"grapheme_cluster_break=lf":          {2},
		"grapheme_cluster_break=control":     {3},
		"grapheme_cluster_break=extend":      {4},
		"grapheme_cluster_break=zwj":         {5},
		"grapheme_cluster_break=ri":          {6},
		"grapheme_cluster_break=prepend":     {7},
		"grapheme_cluster_break=spacingmark": {8},
		"extended_pictographic":              {14},
		"incb=linker":                        {15},
		"incb=consonant":                     {16},
		"incb=extend":                        {17},
		"lu":                                 {100},
		"ll":                                 {101},
		"letter":                             {100, 101},
	}
	ids, ok := m[normalizedExpr]
	if !ok {
		return nil, fmt.Errorf("unknown property %q", expr)
	}
	if negated {
		return nil, fmt.Errorf("negation not tested for %q", expr)
	}
	return ids, nil
}

func TestUnicodeSetPropertyExpr(t *testing.T) {
	ids, err := parseUnicodeSetExpr(`\p{Grapheme_Cluster_Break=CR}`, 18, testResolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected [1], got %v", ids)
	}
}

func TestUnicodeSetBracketedProperty(t *testing.T) {
	ids, err := parseUnicodeSetExpr(`[\p{Grapheme_Cluster_Break=CR}]`, 18, testResolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected [1], got %v", ids)
	}
}

func TestUnicodeSetUnion(t *testing.T) {
	ids, err := parseUnicodeSetExpr(`[\p{Grapheme_Cluster_Break=CR} \p{Grapheme_Cluster_Break=LF}]`, 18, testResolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 categories, got %v", ids)
	}
	s := toSet(ids)
	if !s[1] || !s[2] {
		t.Fatalf("expected {1,2}, got %v", ids)
	}
}

func TestUnicodeSetDifference(t *testing.T) {
	vars := map[string][]uint16{
		"Control": {3},
		"CR":      {1},
		"LF":      {2},
	}
	// [^$Control $CR $LF] — complement of union of Control, CR, LF
	ids, err := parseUnicodeSetExpr(`[^$Control $CR $LF]`, 18, testResolver, vars)
	if err != nil {
		t.Fatal(err)
	}
	s := toSet(ids)
	if s[1] || s[2] || s[3] {
		t.Fatalf("should not contain 1, 2, or 3: got %v", ids)
	}
	if len(ids) != 15 {
		t.Fatalf("expected 15 categories (18 - 3), got %d: %v", len(ids), ids)
	}
}

func TestUnicodeSetIntersection(t *testing.T) {
	vars := map[string][]uint16{
		"A": {1, 2, 3, 4},
		"B": {3, 4, 5, 6},
	}
	ids, err := parseUnicodeSetExpr(`[$A & [$B]]`, 18, testResolver, vars)
	if err != nil {
		t.Fatal(err)
	}
	s := toSet(ids)
	if len(ids) != 2 || !s[3] || !s[4] {
		t.Fatalf("expected {3,4}, got %v", ids)
	}
}

func TestUnicodeSetSetDifference(t *testing.T) {
	vars := map[string][]uint16{
		"A": {1, 2, 3, 4},
		"B": {3, 4},
	}
	ids, err := parseUnicodeSetExpr(`[$A - [$B]]`, 18, testResolver, vars)
	if err != nil {
		t.Fatal(err)
	}
	s := toSet(ids)
	if len(ids) != 2 || !s[1] || !s[2] {
		t.Fatalf("expected {1,2}, got %v", ids)
	}
}

func TestUnicodeSetVarRef(t *testing.T) {
	vars := map[string][]uint16{
		"Extend": {4},
		"ZWJ":    {5},
	}
	ids, err := parseUnicodeSetExpr(`[$Extend $ZWJ]`, 18, testResolver, vars)
	if err != nil {
		t.Fatal(err)
	}
	s := toSet(ids)
	if len(ids) != 2 || !s[4] || !s[5] {
		t.Fatalf("expected {4,5}, got %v", ids)
	}
}

func TestUnicodeSetNested(t *testing.T) {
	vars := map[string][]uint16{
		"A": {1, 2},
		"B": {2, 3},
		"C": {3, 4},
	}
	ids, err := parseUnicodeSetExpr(`[[$A & $B] - [$C]]`, 18, testResolver, vars)
	if err != nil {
		t.Fatal(err)
	}
	// A & B = {2}, minus C = {2} - {3,4} = {2}
	if len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("expected {2}, got %v", ids)
	}
}

func TestUnicodeSetPOSIXProperty(t *testing.T) {
	ids, err := parseUnicodeSetExpr(`[:Lu:]`, 200, testResolver, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 100 {
		t.Fatalf("expected [100], got %v", ids)
	}
}

func TestResolveUnicodeSetsEndToEnd(t *testing.T) {
	src := `
$CR = [\p{Grapheme_Cluster_Break = CR}];
$LF = [\p{Grapheme_Cluster_Break = LF}];
$CR $LF;
`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if resolveErrs := Resolve(rs); resolveErrs != nil {
		t.Fatalf("resolve: %v", resolveErrs)
	}
	setErrs := ResolveUnicodeSets(rs, 18, testResolver)
	if setErrs != nil {
		t.Fatalf("unicode set: %v", setErrs)
	}
	// Check assignments got resolved
	for _, a := range rs.Assignments {
		if len(a.Expr.Classes) == 0 {
			t.Fatalf("assignment $%s has no classes", a.Name)
		}
	}
	// Check rule expression — after resolve, $CR and $LF are replaced with
	// their CharClass nodes, which should now have Classes populated.
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	for i, c := range r.Expr.Children {
		if c.Kind != NodeCharClass {
			t.Fatalf("child %d: expected NodeCharClass, got %d", i, c.Kind)
		}
		if len(c.Classes) == 0 {
			t.Fatalf("child %d: no classes after resolve", i)
		}
	}
}

func TestResolveUnicodeSetsWithVarInSet(t *testing.T) {
	src := `
$Control = [\p{Grapheme_Cluster_Break = Control}];
$CR = [\p{Grapheme_Cluster_Break = CR}];
$LF = [\p{Grapheme_Cluster_Break = LF}];
$NotControl = [^$Control $CR $LF];
$NotControl;
`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if resolveErrs := Resolve(rs); resolveErrs != nil {
		t.Fatalf("resolve: %v", resolveErrs)
	}
	setErrs := ResolveUnicodeSets(rs, 18, testResolver)
	if setErrs != nil {
		t.Fatalf("unicode set: %v", setErrs)
	}
	// $NotControl should be complement of {1,2,3} = everything else
	r := rs.Rules[0]
	s := toSet(r.Expr.Classes)
	if s[1] || s[2] || s[3] {
		t.Fatalf("$NotControl should not contain CR/LF/Control: %v", r.Expr.Classes)
	}
	if len(r.Expr.Classes) != 15 {
		t.Fatalf("expected 15 categories, got %d", len(r.Expr.Classes))
	}
}
