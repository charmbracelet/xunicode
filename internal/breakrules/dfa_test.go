package breakrules

import (
	"testing"
)

func TestBuildDFASimple(t *testing.T) {
	// Rule: A B  (class 0 then class 1), 2 categories
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeCharClass, Classes: []uint16{1}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, 2)

	if len(dfa.States) < 2 {
		t.Fatalf("expected at least 2 states, got %d", len(dfa.States))
	}

	// Start state should not be accepting
	start := dfa.States[dfa.StartState]
	if start.Accepting {
		t.Fatal("start state should not be accepting")
	}

	// From start, category 0 should go somewhere
	next, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on category 0 from start")
	}

	// From that state, category 1 should lead to accepting
	s1 := dfa.States[next]
	next2, ok := s1.Trans[1]
	if !ok {
		t.Fatal("no transition on category 1 from state after A")
	}
	s2 := dfa.States[next2]
	if !s2.Accepting {
		t.Fatal("final state should be accepting")
	}
}

func TestBuildDFAAlternation(t *testing.T) {
	// Rule: A | B (class 0 or class 1), 2 categories
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeAlt,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{0}},
					{Kind: NodeCharClass, Classes: []uint16{1}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, 2)

	start := dfa.States[dfa.StartState]

	// Both categories 0 and 1 should lead to accepting states
	for _, cat := range []uint16{0, 1} {
		next, ok := start.Trans[cat]
		if !ok {
			t.Fatalf("no transition on category %d", cat)
		}
		if !dfa.States[next].Accepting {
			t.Fatalf("state after category %d should be accepting", cat)
		}
	}
}

func TestBuildDFAStar(t *testing.T) {
	// Rule: A* (class 0, zero or more), 1 category
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind:  NodeStar,
				Child: &Node{Kind: NodeCharClass, Classes: []uint16{0}},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, 1)

	// Start state should be accepting (star is nullable)
	start := dfa.States[dfa.StartState]
	if !start.Accepting {
		t.Fatal("start state should be accepting (nullable star)")
	}

	// Should have a transition on category 0 (self-loop or to accepting)
	_, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on category 0 from start")
	}
}

func TestBuildDFAMultiRule(t *testing.T) {
	// Rule 0: A B (classes 0, 1)
	// Rule 1: A C (classes 0, 2)
	// 3 categories
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{
			{
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{0}},
						{Kind: NodeCharClass, Classes: []uint16{1}},
					},
				},
				Tag: -1,
			},
			{
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{0}},
						{Kind: NodeCharClass, Classes: []uint16{2}},
					},
				},
				Tag: -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, 3)

	start := dfa.States[dfa.StartState]
	next, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on category 0 from start")
	}

	// After A, should have transitions on both 1 and 2
	afterA := dfa.States[next]
	_, has1 := afterA.Trans[1]
	_, has2 := afterA.Trans[2]
	if !has1 || !has2 {
		t.Fatalf("after A, expected transitions on 1 and 2, got trans=%v", afterA.Trans)
	}
}

func TestBuildDFAWithChaining(t *testing.T) {
	// Rule 0: A B (classes 0, 1)
	// Rule 1: B C (classes 1, 2)
	// With chaining, after A B matches, B should start rule 1.
	rs := &RuleSet{
		Controls: map[string]bool{"chain": true},
		Rules: []*Rule{
			{
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{0}},
						{Kind: NodeCharClass, Classes: []uint16{1}},
					},
				},
				Tag: -1,
			},
			{
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{1}},
						{Kind: NodeCharClass, Classes: []uint16{2}},
					},
				},
				Tag: -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	CalcChainedFollowPos(nfa, rs)
	dfa := BuildDFA(nfa, 3)

	// Walk: A(0) → B(1) → should reach accepting state
	start := dfa.States[dfa.StartState]
	s1, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on 0")
	}
	s2, ok := dfa.States[s1].Trans[1]
	if !ok {
		t.Fatal("no transition on 1")
	}
	state2 := dfa.States[s2]
	if !state2.Accepting {
		t.Fatal("state after A B should be accepting")
	}
	// With chaining, state2 should have pos 3 (B start of rule 1).
	// From state2, reading cat 1 (B again via chaining overlap) should
	// transition to a state containing pos 4 (C position of rule 1).
	s3, ok := state2.Trans[1]
	if !ok {
		t.Fatal("chained transition on cat 1 missing after A B")
	}
	// From that state, reading cat 2 (C) should reach accepting.
	s4, ok := dfa.States[s3].Trans[2]
	if !ok {
		t.Fatal("transition on cat 2 missing after chained B")
	}
	if !dfa.States[s4].Accepting {
		t.Fatal("state after chained B C should be accepting")
	}
}
