package breakrules

import (
	"testing"
)

func TestCompileSimple(t *testing.T) {
	src := `
$A = [\p{Lu}];
$B = [\p{Ll}];
$A $B;
.;
`
	result, err := Compile([]byte(src), CompileOptions{NumCategories: 3})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if result.DFA == nil {
		t.Fatal("nil DFA")
	}
	if len(result.DFA.States) == 0 {
		t.Fatal("no DFA states")
	}
}

func TestCompileWithChaining(t *testing.T) {
	src := `
!!chain;
$A = [\p{Lu}];
$B = [\p{Ll}];
$C = [\p{Nd}];
$A $B;
$B $C;
.;
`
	result, err := Compile([]byte(src), CompileOptions{NumCategories: 4})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !result.RuleSet.Controls["chain"] {
		t.Fatal("missing !!chain control")
	}
	if result.DFA == nil {
		t.Fatal("nil DFA")
	}
}

func TestCompileParseError(t *testing.T) {
	src := `$A = ;`
	_, err := Compile([]byte(src), CompileOptions{NumCategories: 1})
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
}

func TestCompileResolveError(t *testing.T) {
	src := `$Undefined;`
	_, err := Compile([]byte(src), CompileOptions{NumCategories: 1})
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
}

func TestCompileEndToEnd(t *testing.T) {
	src := `
!!chain;
!!lookAheadHardBreak;

$CR = [\p{Grapheme_Cluster_Break = CR}];
$LF = [\p{Grapheme_Cluster_Break = LF}];
$Control = [\p{Grapheme_Cluster_Break = Control}];
$Extend = [\p{Grapheme_Cluster_Break = Extend}];

$CR $LF;
.;
`
	result, err := Compile([]byte(src), CompileOptions{NumCategories: 5})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if result.DFA == nil {
		t.Fatal("nil DFA")
	}
	if len(result.RuleSet.Assignments) != 4 {
		t.Fatalf("expected 4 assignments, got %d", len(result.RuleSet.Assignments))
	}
	if len(result.RuleSet.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(result.RuleSet.Rules))
	}
}
