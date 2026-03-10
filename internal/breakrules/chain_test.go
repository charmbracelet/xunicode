package breakrules

import (
	"testing"
)

func TestChainBasic(t *testing.T) {
	// Rule 0: A B  (classes {1} {2})
	// Rule 1: B C  (classes {2} {3})
	// With chaining, after rule 0 completes (A B #), the B position (pos 1)
	// should gain a followpos link to the B start position of rule 1 (pos 3).
	rs := &RuleSet{
		Controls: map[string]bool{"chain": true},
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
	CalcChainedFollowPos(nfa, rs)

	// pos 0: class {1} (rule 0)
	// pos 1: class {2} (rule 0)
	// pos 2: end-mark (rule 0)
	// pos 3: class {2} (rule 1)
	// pos 4: class {3} (rule 1)
	// pos 5: end-mark (rule 1)

	// followpos(1) should now include pos 3 (chaining: B overlaps with B)
	if !nfa.FollowPos[1].Contains(3) {
		t.Fatalf("expected chaining link from pos 1 to pos 3, followpos(1)=%v", nfa.FollowPos[1])
	}
}

func TestChainNoChanIn(t *testing.T) {
	// Rule 0: A B  (classes {1} {2})
	// Rule 1: ^B C  (classes {2} {3}) — no-chain-in
	// Chaining should NOT add link from pos 1 to pos 3.
	rs := &RuleSet{
		Controls: map[string]bool{"chain": true},
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
				Expr: &Node{
					Kind: NodeConcat,
					Children: []*Node{
						{Kind: NodeCharClass, Classes: []uint16{2}},
						{Kind: NodeCharClass, Classes: []uint16{3}},
					},
				},
				NoChanIn: true,
				Tag:      -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	CalcChainedFollowPos(nfa, rs)

	// followpos(1) should NOT include pos 3
	if nfa.FollowPos[1].Contains(3) {
		t.Fatalf("no-chain-in rule should not be chained into, followpos(1)=%v", nfa.FollowPos[1])
	}
}

func TestChainWithDot(t *testing.T) {
	// Rule 0: A B  (classes {1} {2})
	// Rule 1: .   (dot rule)
	// Dot overlaps with any class, so chaining should link pos 1 to dot position.
	rs := &RuleSet{
		Controls: map[string]bool{"chain": true},
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
				Expr: &Node{Kind: NodeDot},
				Tag:  -1,
			},
		},
	}
	nfa := BuildNFA(rs)
	CalcChainedFollowPos(nfa, rs)

	// pos 3 is the dot position (rule 1 start)
	if !nfa.FollowPos[1].Contains(3) {
		t.Fatalf("expected chaining link from pos 1 to dot pos 3, followpos(1)=%v", nfa.FollowPos[1])
	}
}
