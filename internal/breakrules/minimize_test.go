package breakrules

import (
	"testing"
)

func TestMinimizeIdentical(t *testing.T) {
	// Two states with identical transitions should merge.
	dfa := &DFA{
		NumCats: 2,
		States: []*DFAState{
			{ID: 0, Trans: map[uint16]int{0: 1, 1: 2}, RuleIndex: -1, Tag: -1, LookAheadRuleIndex: -1},
			{ID: 1, Trans: map[uint16]int{}, Accepting: true, RuleIndex: 0, Tag: -1, LookAheadRuleIndex: -1},
			{ID: 2, Trans: map[uint16]int{}, Accepting: true, RuleIndex: 0, Tag: -1, LookAheadRuleIndex: -1},
		},
		StartState: 0,
	}
	min := Minimize(dfa)
	// States 1 and 2 are equivalent (both accepting, same rule, no transitions)
	// Should merge to 2 states total.
	if len(min.States) != 2 {
		t.Fatalf("expected 2 states after minimization, got %d", len(min.States))
	}
}

func TestMinimizeDistinct(t *testing.T) {
	// Two accepting states with different rule indices should not merge.
	dfa := &DFA{
		NumCats: 2,
		States: []*DFAState{
			{ID: 0, Trans: map[uint16]int{0: 1, 1: 2}, RuleIndex: -1, Tag: -1, LookAheadRuleIndex: -1},
			{ID: 1, Trans: map[uint16]int{}, Accepting: true, RuleIndex: 0, Tag: -1, LookAheadRuleIndex: -1},
			{ID: 2, Trans: map[uint16]int{}, Accepting: true, RuleIndex: 1, Tag: -1, LookAheadRuleIndex: -1},
		},
		StartState: 0,
	}
	min := Minimize(dfa)
	if len(min.States) != 3 {
		t.Fatalf("expected 3 states (distinct rules), got %d", len(min.States))
	}
}

func TestMinimizePreservesLanguage(t *testing.T) {
	// Build a DFA for A B | A C, minimize, and verify the language is preserved.
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{
			{
				Expr: &Node{
					Kind: NodeAlt,
					Children: []*Node{
						{Kind: NodeConcat, Children: []*Node{
							{Kind: NodeCharClass, Classes: []uint16{0}},
							{Kind: NodeCharClass, Classes: []uint16{1}},
						}},
						{Kind: NodeConcat, Children: []*Node{
							{Kind: NodeCharClass, Classes: []uint16{0}},
							{Kind: NodeCharClass, Classes: []uint16{2}},
						}},
					},
				},
				Tag: -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	dfa := BuildDFA(nfa, DFAOptions{NumCats: 3})
	min := Minimize(dfa)

	// Should still accept A B and A C.
	// Minimized state count should be <= original.
	if len(min.States) > len(dfa.States) {
		t.Fatalf("minimized has more states (%d) than original (%d)", len(min.States), len(dfa.States))
	}

	// Walk A B: start → cat 0 → ? → cat 1 → accepting
	start := min.States[min.StartState]
	s1, ok := start.Trans[0]
	if !ok {
		t.Fatal("no transition on 0")
	}
	s2, ok := min.States[s1].Trans[1]
	if !ok {
		t.Fatal("no transition on 1 after A")
	}
	if !min.States[s2].Accepting {
		t.Fatal("A B should be accepted")
	}

	// Walk A C: start → cat 0 → ? → cat 2 → accepting
	s3, ok := min.States[s1].Trans[2]
	if !ok {
		t.Fatal("no transition on 2 after A")
	}
	if !min.States[s3].Accepting {
		t.Fatal("A C should be accepted")
	}
}

func TestMinimizeSingleState(t *testing.T) {
	dfa := &DFA{
		NumCats: 1,
		States: []*DFAState{
			{ID: 0, Trans: map[uint16]int{}, Accepting: true, RuleIndex: 0, Tag: -1, LookAheadRuleIndex: -1},
		},
		StartState: 0,
	}
	min := Minimize(dfa)
	if len(min.States) != 1 {
		t.Fatalf("expected 1 state, got %d", len(min.States))
	}
	if !min.States[0].Accepting {
		t.Fatal("single state should still be accepting")
	}
}
