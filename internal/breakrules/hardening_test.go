package breakrules

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. Literal char handling in NodeCharClass.Name
// ---------------------------------------------------------------------------
// Bare literal characters (e.g. \u002D → "-") end up as NodeCharClass with
// Name set to the raw char string and Classes initially empty. The UnicodeSet
// resolver handles these via fallthrough: the caller-provided PropertyResolver
// receives the raw string and maps it to the correct category ID(s).
// These tests verify that pipeline works end-to-end.

func TestLiteralCharInRuleViaResolver(t *testing.T) {
	src := []byte(`
$A = [\p{Lu}];
$A \u002D;
`)
	resolver := func(expr string, negated bool) ([]uint16, error) {
		switch expr {
		case "Lu":
			return []uint16{0}, nil
		case "-":
			return []uint16{1}, nil
		}
		return nil, fmt.Errorf("unknown: %q", expr)
	}
	result, err := Compile(src, CompileOptions{
		NumCategories:    2,
		PropertyResolver: resolver,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dfa := result.DFA
	start := dfa.States[dfa.StartState]

	s1, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on cat 0 (Lu) from start")
	}
	s2, ok := dfa.States[s1].Trans[1]
	if !ok {
		t.Fatal("no transition on cat 1 (dash) after Lu")
	}
	if !dfa.States[s2].Accepting {
		t.Fatal("Lu followed by dash should be accepting")
	}
}

func TestLiteralCharFallsThroughToResolver(t *testing.T) {
	rs, errs := Parse([]byte(`x;`))
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	if len(rs.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rs.Rules))
	}
	expr := rs.Rules[0].Expr
	if expr.Kind != NodeCharClass {
		t.Fatalf("expected NodeCharClass, got %d", expr.Kind)
	}
	if expr.Name != "x" {
		t.Fatalf("expected Name=%q, got %q", "x", expr.Name)
	}
	if len(expr.Classes) != 0 {
		t.Fatalf("expected empty Classes before resolve, got %v", expr.Classes)
	}
}

// ---------------------------------------------------------------------------
// 2. !!lookAheadHardBreak
// ---------------------------------------------------------------------------
// This is a runtime flag. The compiler's job is to surface it in CompileResult.

func TestLookAheadHardBreakFlagSet(t *testing.T) {
	src := []byte(`
!!lookAheadHardBreak;
.;
`)
	result, err := Compile(src, CompileOptions{NumCategories: 1})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !result.LookAheadHardBreak {
		t.Fatal("expected LookAheadHardBreak=true")
	}
}

func TestLookAheadHardBreakFlagUnset(t *testing.T) {
	src := []byte(`.;`)
	result, err := Compile(src, CompileOptions{NumCategories: 1})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if result.LookAheadHardBreak {
		t.Fatal("expected LookAheadHardBreak=false when not declared")
	}
}

// ---------------------------------------------------------------------------
// 3. Lookahead (/) break-at-slash semantics
// ---------------------------------------------------------------------------
// The slash creates a marker position in the NFA. The DFA should mark states
// containing slash positions with LookAhead=true and LookAheadRuleIndex set
// to the rule that owns the slash.

func TestLookAheadSlashNFAPosition(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeSlash},
					{Kind: NodeCharClass, Classes: []uint16{1}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)

	var slashPos *Position
	for _, p := range nfa.Positions {
		if p.IsSlash {
			slashPos = p
			break
		}
	}
	if slashPos == nil {
		t.Fatal("expected a slash position in NFA")
	}
	if slashPos.RuleIndex != 0 {
		t.Fatalf("slash position should belong to rule 0, got %d", slashPos.RuleIndex)
	}

	if len(nfa.FollowPos[slashPos.ID]) == 0 {
		t.Fatal("slash position should have followpos (linking to post-context)")
	}
}

func TestLookAheadDFAState(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeSlash},
					{Kind: NodeCharClass, Classes: []uint16{1}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{NumCats: 2})

	var hasLookAhead bool
	for _, s := range dfa.States {
		if s.LookAhead {
			hasLookAhead = true
			if s.LookAheadRuleIndex != 0 {
				t.Fatalf("LookAheadRuleIndex should be 0, got %d", s.LookAheadRuleIndex)
			}
		}
	}
	if !hasLookAhead {
		t.Fatal("expected at least one DFA state with LookAhead=true")
	}

	start := dfa.States[dfa.StartState]
	s1, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on cat 0 from start")
	}
	afterA := dfa.States[s1]
	s2, ok := afterA.Trans[1]
	if !ok {
		t.Fatal("no transition on cat 1 after A (slash is transparent)")
	}
	if !dfa.States[s2].Accepting {
		t.Fatal("state after A / B should be accepting")
	}
}

func TestLookAheadRuleIndexMultiRule(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{
			{
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{0}},
						{Kind: NodeSlash},
						{Kind: NodeCharClass, Classes: []uint16{1}},
					},
				},
				Tag: -1,
			},
			{
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{2}},
						{Kind: NodeCharClass, Classes: []uint16{3}},
					},
				},
				Tag: -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{NumCats: 4})

	for _, s := range dfa.States {
		if s.LookAhead && s.LookAheadRuleIndex != 0 {
			t.Fatalf("only rule 0 has slash, but found LookAheadRuleIndex=%d", s.LookAheadRuleIndex)
		}
	}
}

func TestLookAheadPreservedAfterMinimize(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeSlash},
					{Kind: NodeCharClass, Classes: []uint16{1}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{NumCats: 2})
	min := Minimize(dfa)

	var hasLookAhead bool
	for _, s := range min.States {
		if s.LookAhead {
			hasLookAhead = true
			if s.LookAheadRuleIndex != 0 {
				t.Fatalf("after minimize: LookAheadRuleIndex should be 0, got %d", s.LookAheadRuleIndex)
			}
		}
	}
	if !hasLookAhead {
		t.Fatal("LookAhead should survive minimization")
	}
}

// ---------------------------------------------------------------------------
// 4. {bof}/{eof} anchor semantics
// ---------------------------------------------------------------------------
// {bof} and {eof} are virtual categories that only match at the beginning/end
// of the input. In the DFA, they produce transitions on the designated
// BOFCategory/EOFCategory IDs.

func TestBOFAnchorInNFA(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeBOF},
					{Kind: NodeCharClass, Classes: []uint16{0}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)

	var bofPos *Position
	for _, p := range nfa.Positions {
		if p.IsBOF {
			bofPos = p
			break
		}
	}
	if bofPos == nil {
		t.Fatal("expected a BOF position in NFA")
	}
	if !nfa.StartPos.Contains(bofPos.ID) {
		t.Fatal("BOF position should be in startpos")
	}
}

func TestEOFAnchorInNFA(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeEOF},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)

	var eofPos *Position
	for _, p := range nfa.Positions {
		if p.IsEOF {
			eofPos = p
			break
		}
	}
	if eofPos == nil {
		t.Fatal("expected an EOF position in NFA")
	}
}

func TestBOFAnchorInDFA(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeBOF},
					{Kind: NodeCharClass, Classes: []uint16{0}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{
		NumCats:     3,
		BOFCategory: 2,
		EOFCategory: -1,
	})

	start := dfa.States[dfa.StartState]
	bofNext, hasBOF := start.Trans[2]
	if !hasBOF {
		t.Fatal("expected transition on BOF category (2) from start state")
	}

	afterBOF := dfa.States[bofNext]
	catNext, hasCat := afterBOF.Trans[0]
	if !hasCat {
		t.Fatal("expected transition on cat 0 after BOF")
	}
	if !dfa.States[catNext].Accepting {
		t.Fatal("BOF followed by cat 0 should be accepting")
	}
}

func TestEOFAnchorInDFA(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeEOF},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{
		NumCats:     3,
		BOFCategory: -1,
		EOFCategory: 2,
	})

	start := dfa.States[dfa.StartState]
	catNext, hasCat := start.Trans[0]
	if !hasCat {
		t.Fatal("expected transition on cat 0 from start")
	}

	afterCat := dfa.States[catNext]
	eofNext, hasEOF := afterCat.Trans[2]
	if !hasEOF {
		t.Fatal("expected transition on EOF category (2) after cat 0")
	}
	if !dfa.States[eofNext].Accepting {
		t.Fatal("cat 0 followed by EOF should be accepting")
	}
}

func TestBOFIgnoredWhenCategoryMinusOne(t *testing.T) {
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeBOF},
					{Kind: NodeCharClass, Classes: []uint16{0}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{
		NumCats:     2,
		BOFCategory: -1,
		EOFCategory: -1,
	})

	start := dfa.States[dfa.StartState]
	if len(start.Trans) != 0 {
		t.Fatalf("with BOFCategory=-1, BOF position should produce no transitions, got %v", start.Trans)
	}
}

func TestBOFEOFEndToEndCompile(t *testing.T) {
	src := []byte(`
{bof} [\p{Lu}];
[\p{Ll}] {eof};
.;
`)
	resolver := func(expr string, negated bool) ([]uint16, error) {
		switch expr {
		case "Lu":
			return []uint16{0}, nil
		case "Ll":
			return []uint16{1}, nil
		}
		return nil, fmt.Errorf("unknown: %q", expr)
	}
	result, err := Compile(src, CompileOptions{
		NumCategories:    4,
		PropertyResolver: resolver,
		BOFCategory:      2,
		EOFCategory:      3,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dfa := result.DFA
	start := dfa.States[dfa.StartState]

	bofNext, hasBOF := start.Trans[2]
	if !hasBOF {
		t.Fatal("expected BOF transition from start")
	}
	afterBOF := dfa.States[bofNext]
	luNext, hasLu := afterBOF.Trans[0]
	if !hasLu {
		t.Fatal("expected Lu transition after BOF")
	}
	if !dfa.States[luNext].Accepting {
		t.Fatal("BOF Lu should be accepting")
	}

	llNext, hasLl := start.Trans[1]
	if !hasLl {
		t.Fatal("expected Ll transition from start")
	}
	afterLl := dfa.States[llNext]
	eofNext, hasEOF := afterLl.Trans[3]
	if !hasEOF {
		t.Fatal("expected EOF transition after Ll")
	}
	if !dfa.States[eofNext].Accepting {
		t.Fatal("Ll EOF should be accepting")
	}
}
