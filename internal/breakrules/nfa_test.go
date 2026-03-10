package breakrules

import (
	"testing"
)

func TestBuildNFASimpleConcat(t *testing.T) {
	// $A $B; → two leaf positions + one end-mark
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeConcat,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{1}},
					{Kind: NodeCharClass, Classes: []uint16{2}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	if len(nfa.Positions) != 3 {
		t.Fatalf("expected 3 positions (2 leaves + 1 end-mark), got %d", len(nfa.Positions))
	}
	if !nfa.Positions[2].IsEndMark {
		t.Fatal("position 2 should be end-mark")
	}
	// firstpos should be {0} (position for class {1})
	if len(nfa.StartPos) != 1 || nfa.StartPos[0] != 0 {
		t.Fatalf("expected startPos={0}, got %v", nfa.StartPos)
	}
	// followpos(0) should include {1}
	if !nfa.FollowPos[0].Contains(1) {
		t.Fatalf("followpos(0) should contain 1, got %v", nfa.FollowPos[0])
	}
	// followpos(1) should include {2} (end-mark)
	if !nfa.FollowPos[1].Contains(2) {
		t.Fatalf("followpos(1) should contain 2, got %v", nfa.FollowPos[1])
	}
}

func TestBuildNFAAlternation(t *testing.T) {
	// $A | $B; → two leaf positions + one end-mark
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeAlt,
				Children: []*Node{
					{Kind: NodeCharClass, Classes: []uint16{1}},
					{Kind: NodeCharClass, Classes: []uint16{2}},
				},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	if len(nfa.Positions) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(nfa.Positions))
	}
	// firstpos should be {0, 1}
	if len(nfa.StartPos) != 2 {
		t.Fatalf("expected 2 start positions, got %v", nfa.StartPos)
	}
}

func TestBuildNFAStar(t *testing.T) {
	// $A*; → one leaf + end-mark, star makes it nullable
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind: NodeStar,
				Child: &Node{Kind: NodeCharClass, Classes: []uint16{1}},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	if len(nfa.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(nfa.Positions))
	}
	// star is nullable, so firstpos includes both the leaf and end-mark
	if len(nfa.StartPos) != 2 {
		t.Fatalf("expected startPos to have 2 elements (nullable star), got %v", nfa.StartPos)
	}
	// followpos(0) should include {0} (self-loop from star) and {1} (end-mark from concat with end-mark)
	if !nfa.FollowPos[0].Contains(0) {
		t.Fatalf("followpos(0) should contain self-loop, got %v", nfa.FollowPos[0])
	}
	if !nfa.FollowPos[0].Contains(1) {
		t.Fatalf("followpos(0) should contain end-mark, got %v", nfa.FollowPos[0])
	}
}

func TestBuildNFAMultipleRules(t *testing.T) {
	// Rule 0: $A $B;
	// Rule 1: $C;
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{
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
			{
				Expr: &Node{Kind: NodeCharClass, Classes: []uint16{3}},
				Tag:  -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	// Rule 0: positions 0,1,2(end). Rule 1: positions 3,4(end).
	if len(nfa.Positions) != 5 {
		t.Fatalf("expected 5 positions, got %d", len(nfa.Positions))
	}
	// startPos = firstpos(rule0) ∪ firstpos(rule1) = {0, 3}
	if len(nfa.StartPos) != 2 {
		t.Fatalf("expected 2 start positions, got %v", nfa.StartPos)
	}
	if !nfa.StartPos.Contains(0) || !nfa.StartPos.Contains(3) {
		t.Fatalf("expected {0,3} in startPos, got %v", nfa.StartPos)
	}
}

func TestBuildNFADot(t *testing.T) {
	// .; → dot position + end-mark
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{Kind: NodeDot},
			Tag:  -1,
		}},
	}
	nfa := BuildNFA(rs)
	if len(nfa.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(nfa.Positions))
	}
	if !nfa.Positions[0].IsDot {
		t.Fatal("position 0 should be dot")
	}
	if !nfa.Positions[1].IsEndMark {
		t.Fatal("position 1 should be end-mark")
	}
}

func TestBuildNFAPlus(t *testing.T) {
	// $A+; → non-nullable, has self-loop
	rs := &RuleSet{
		Controls: map[string]bool{},
		Rules: []*Rule{{
			Expr: &Node{
				Kind:  NodePlus,
				Child: &Node{Kind: NodeCharClass, Classes: []uint16{1}},
			},
			Tag: -1,
		}},
	}
	nfa := BuildNFA(rs)
	// plus is not nullable, so startPos should only be {0}
	if len(nfa.StartPos) != 1 || nfa.StartPos[0] != 0 {
		t.Fatalf("expected startPos={0}, got %v", nfa.StartPos)
	}
	// followpos(0) should include {0} (self-loop) and {1} (end-mark)
	if !nfa.FollowPos[0].Contains(0) {
		t.Fatal("expected self-loop in followpos(0)")
	}
	if !nfa.FollowPos[0].Contains(1) {
		t.Fatal("expected end-mark in followpos(0)")
	}
}
