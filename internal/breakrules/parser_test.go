package breakrules

import (
	"testing"
)

func TestParseControl(t *testing.T) {
	rs, errs := Parse([]byte(`!!chain; !!lookAheadHardBreak;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if !rs.Controls["chain"] {
		t.Fatal("missing !!chain")
	}
	if !rs.Controls["lookAheadHardBreak"] {
		t.Fatal("missing !!lookAheadHardBreak")
	}
	if len(rs.Assignments) != 0 {
		t.Fatalf("expected 0 assignments, got %d", len(rs.Assignments))
	}
	if len(rs.Rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(rs.Rules))
	}
}

func TestParseAssignment(t *testing.T) {
	rs, errs := Parse([]byte(`$CR = [\p{Grapheme_Cluster_Break = CR}];`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if len(rs.Assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(rs.Assignments))
	}
	a := rs.Assignments[0]
	if a.Name != "CR" {
		t.Fatalf("expected name CR, got %q", a.Name)
	}
	if a.Expr == nil {
		t.Fatal("nil expression")
	}
	if a.Expr.Kind != NodeCharClass {
		t.Fatalf("expected NodeCharClass, got %d", a.Expr.Kind)
	}
	if a.Expr.Name != `[\p{Grapheme_Cluster_Break = CR}]` {
		t.Fatalf("wrong set name: %q", a.Expr.Name)
	}
}

func TestParseSimpleRule(t *testing.T) {
	rs, errs := Parse([]byte(`$CR $LF;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if len(rs.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rs.Rules))
	}
	r := rs.Rules[0]
	if r.NoChanIn {
		t.Fatal("unexpected NoChanIn")
	}
	if r.Tag != -1 {
		t.Fatalf("expected tag -1, got %d", r.Tag)
	}
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	if len(r.Expr.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(r.Expr.Children))
	}
	if r.Expr.Children[0].Kind != NodeVariable || r.Expr.Children[0].Name != "CR" {
		t.Fatalf("child 0: %+v", r.Expr.Children[0])
	}
	if r.Expr.Children[1].Kind != NodeVariable || r.Expr.Children[1].Name != "LF" {
		t.Fatalf("child 1: %+v", r.Expr.Children[1])
	}
}

func TestParseAlternation(t *testing.T) {
	rs, errs := Parse([]byte(`$A | $B;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeAlt {
		t.Fatalf("expected NodeAlt, got %d", r.Expr.Kind)
	}
	if len(r.Expr.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(r.Expr.Children))
	}
}

func TestParsePostfixOps(t *testing.T) {
	rs, errs := Parse([]byte(`$A* $B+ $C?;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	children := r.Expr.Children
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	if children[0].Kind != NodeStar || children[0].Child.Name != "A" {
		t.Fatalf("child 0: %+v", children[0])
	}
	if children[1].Kind != NodePlus || children[1].Child.Name != "B" {
		t.Fatalf("child 1: %+v", children[1])
	}
	if children[2].Kind != NodeQuest || children[2].Child.Name != "C" {
		t.Fatalf("child 2: %+v", children[2])
	}
}

func TestParseGroupedExpr(t *testing.T) {
	rs, errs := Parse([]byte(`($A | $B)* $C;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	star := r.Expr.Children[0]
	if star.Kind != NodeStar {
		t.Fatalf("expected NodeStar, got %d", star.Kind)
	}
	alt := star.Child
	if alt.Kind != NodeAlt {
		t.Fatalf("expected NodeAlt inside star, got %d", alt.Kind)
	}
}

func TestParseCaret(t *testing.T) {
	rs, errs := Parse([]byte(`^$A $B;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if !r.NoChanIn {
		t.Fatal("expected NoChanIn=true")
	}
}

func TestParseLookahead(t *testing.T) {
	rs, errs := Parse([]byte(`$A $B / $C;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	found := false
	for _, c := range r.Expr.Children {
		if c.Kind == NodeSlash {
			found = true
		}
	}
	if !found {
		t.Fatal("expected NodeSlash in expression")
	}
}

func TestParseStatusTag(t *testing.T) {
	rs, errs := Parse([]byte(`$A {200};`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Tag != 200 {
		t.Fatalf("expected tag 200, got %d", r.Tag)
	}
}

func TestParseDot(t *testing.T) {
	rs, errs := Parse([]byte(`.;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeDot {
		t.Fatalf("expected NodeDot, got %d", r.Expr.Kind)
	}
}

func TestParseBOFEOF(t *testing.T) {
	rs, errs := Parse([]byte(`{bof} $A; $B {eof};`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if len(rs.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rs.Rules))
	}
	r0 := rs.Rules[0]
	if r0.Expr.Kind != NodeConcat {
		t.Fatalf("rule 0: expected NodeConcat, got %d", r0.Expr.Kind)
	}
	if r0.Expr.Children[0].Kind != NodeBOF {
		t.Fatalf("rule 0 child 0: expected NodeBOF, got %d", r0.Expr.Children[0].Kind)
	}
	r1 := rs.Rules[1]
	if r1.Expr.Kind != NodeConcat {
		t.Fatalf("rule 1: expected NodeConcat, got %d", r1.Expr.Kind)
	}
	if r1.Expr.Children[1].Kind != NodeEOF {
		t.Fatalf("rule 1 child 1: expected NodeEOF, got %d", r1.Expr.Children[1].Kind)
	}
}

func TestParseMultipleAssignmentsAndRules(t *testing.T) {
	src := `
!!chain;
$CR = [\p{Grapheme_Cluster_Break = CR}];
$LF = [\p{Grapheme_Cluster_Break = LF}];
$Control = [\p{Grapheme_Cluster_Break = Control}];
$CR $LF;
[^$Control $CR $LF] ($Extend | $ZWJ);
.;
`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if !rs.Controls["chain"] {
		t.Fatal("missing !!chain")
	}
	if len(rs.Assignments) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(rs.Assignments))
	}
	if len(rs.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rs.Rules))
	}
}

func TestParsePrecedence(t *testing.T) {
	// $A $B | $C should parse as ($A $B) | $C
	rs, errs := Parse([]byte(`$A $B | $C;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeAlt {
		t.Fatalf("expected NodeAlt at top, got %d", r.Expr.Kind)
	}
	left := r.Expr.Children[0]
	if left.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat on left of alt, got %d", left.Kind)
	}
	if len(left.Children) != 2 {
		t.Fatalf("expected 2 children in concat, got %d", len(left.Children))
	}
	right := r.Expr.Children[1]
	if right.Kind != NodeVariable || right.Name != "C" {
		t.Fatalf("expected variable C on right, got %+v", right)
	}
}

func TestParseNestedGroups(t *testing.T) {
	// (($A | $B) $C)*
	rs, errs := Parse([]byte(`(($A | $B) $C)*;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeStar {
		t.Fatalf("expected NodeStar, got %d", r.Expr.Kind)
	}
	concat := r.Expr.Child
	if concat.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat inside star, got %d", concat.Kind)
	}
	if concat.Children[0].Kind != NodeAlt {
		t.Fatalf("expected NodeAlt as first child of concat, got %d", concat.Children[0].Kind)
	}
}

func TestParseComplexRule(t *testing.T) {
	// Mimics a simplified LB25-like rule: $NU ($SY | $IS)* $CL
	rs, errs := Parse([]byte(`$NU ($SY | $IS)* $CL;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	if len(r.Expr.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(r.Expr.Children))
	}
	if r.Expr.Children[0].Kind != NodeVariable || r.Expr.Children[0].Name != "NU" {
		t.Fatalf("child 0: %+v", r.Expr.Children[0])
	}
	star := r.Expr.Children[1]
	if star.Kind != NodeStar {
		t.Fatalf("child 1: expected NodeStar, got %d", star.Kind)
	}
	alt := star.Child
	if alt.Kind != NodeAlt {
		t.Fatalf("inside star: expected NodeAlt, got %d", alt.Kind)
	}
	if r.Expr.Children[2].Kind != NodeVariable || r.Expr.Children[2].Name != "CL" {
		t.Fatalf("child 2: %+v", r.Expr.Children[2])
	}
}

func TestParseCaretLookaheadTag(t *testing.T) {
	// ^$A $B / $C {100};
	rs, errs := Parse([]byte(`^$A $B / $C {100};`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if !r.NoChanIn {
		t.Fatal("expected NoChanIn")
	}
	if r.Tag != 100 {
		t.Fatalf("expected tag 100, got %d", r.Tag)
	}
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	hasSlash := false
	for _, c := range r.Expr.Children {
		if c.Kind == NodeSlash {
			hasSlash = true
		}
	}
	if !hasSlash {
		t.Fatal("expected NodeSlash")
	}
}

func TestParseUnicodeSetInRule(t *testing.T) {
	rs, errs := Parse([]byte(`[^$Control $CR $LF] $Extend;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeConcat {
		t.Fatalf("expected NodeConcat, got %d", r.Expr.Kind)
	}
	if r.Expr.Children[0].Kind != NodeCharClass {
		t.Fatalf("child 0: expected NodeCharClass, got %d", r.Expr.Children[0].Kind)
	}
}

func TestParseGraphemeRules(t *testing.T) {
	src := `
!!quoted_literals_only;
!!chain;
!!lookAheadHardBreak;

$CR          = [\p{Grapheme_Cluster_Break = CR}];
$LF          = [\p{Grapheme_Cluster_Break = LF}];
$Control     = [\p{Grapheme_Cluster_Break = Control}];
$Extend      = [\p{Grapheme_Cluster_Break = Extend}];
$ZWJ         = [\p{Grapheme_Cluster_Break = ZWJ}];
$Prepend     = [\p{Grapheme_Cluster_Break = Prepend}];
$SpacingMark = [\p{Grapheme_Cluster_Break = SpacingMark}];

# GB3
$CR $LF;

# GB6-GB8
$LF;
$Control;
$CR;

# GB9
[^$Control $CR $LF] ($Extend | $ZWJ);

# GB9a
[^$Control $CR $LF] $SpacingMark;

# GB9b
$Prepend [^$Control $CR $LF];

# GB999
.;
`
	rs, errs := Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if !rs.Controls["quoted_literals_only"] {
		t.Fatal("missing !!quoted_literals_only")
	}
	if !rs.Controls["chain"] {
		t.Fatal("missing !!chain")
	}
	if !rs.Controls["lookAheadHardBreak"] {
		t.Fatal("missing !!lookAheadHardBreak")
	}
	if len(rs.Assignments) != 7 {
		t.Fatalf("expected 7 assignments, got %d", len(rs.Assignments))
	}
	if len(rs.Rules) != 8 {
		t.Fatalf("expected 8 rules, got %d", len(rs.Rules))
	}
}

func TestParseMultiAlt(t *testing.T) {
	// $A | $B | $C should parse as ($A | $B) | $C (left-associative)
	rs, errs := Parse([]byte(`$A | $B | $C;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeAlt {
		t.Fatalf("expected NodeAlt at top, got %d", r.Expr.Kind)
	}
	left := r.Expr.Children[0]
	if left.Kind != NodeAlt {
		t.Fatalf("expected NodeAlt on left (left-assoc), got %d", left.Kind)
	}
}

func TestParseEmptyInput(t *testing.T) {
	rs, errs := Parse([]byte(``))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	if len(rs.Controls) != 0 || len(rs.Assignments) != 0 || len(rs.Rules) != 0 {
		t.Fatal("expected empty RuleSet")
	}
}

func TestParseCharLiteral(t *testing.T) {
	rs, errs := Parse([]byte(`\u002D;`))
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	r := rs.Rules[0]
	if r.Expr.Kind != NodeCharClass {
		t.Fatalf("expected NodeCharClass, got %d", r.Expr.Kind)
	}
	if r.Expr.Name != "-" {
		t.Fatalf("expected '-', got %q", r.Expr.Name)
	}
}
