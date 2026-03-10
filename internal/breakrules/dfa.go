package breakrules

// DFA subset construction from a position-based NFA.
//
// Each DFA state is a set of NFA positions. Transitions are computed by
// collecting followpos for all positions that match each input category.
//
// A DFA state is accepting if it contains an end-mark position.
// A DFA state has lookahead if it contains a slash position.
//
// {bof} and {eof} positions are treated as virtual categories: during
// transition computation they match only their designated category ID.
//
// Slash (lookahead) positions track which rule's lookahead is active via
// LookAheadRuleIndex, so the runtime can backtrack to the break point.

// DFA represents the deterministic finite automaton.
type DFA struct {
	States     []*DFAState
	StartState int
	NumCats    int // number of input categories
}

// DFAState is a single DFA state.
type DFAState struct {
	ID                  int
	PosSet              PosSet         // the set of NFA positions in this state
	Trans               map[uint16]int // category → next state ID (-1 if none)
	Accepting           bool           // contains an end-mark position
	RuleIndex           int            // rule index of the accepting end-mark (-1 if not accepting, lowest if multiple)
	Tag                 int            // status tag from the accepting rule (-1 if none)
	LookAhead           bool           // contains a slash (lookahead) position
	LookAheadRuleIndex  int            // rule whose lookahead is active (-1 if none)
}

// DFAOptions controls DFA construction.
type DFAOptions struct {
	NumCats     int // total number of input categories
	BOFCategory int // category ID for {bof} (-1 if unused)
	EOFCategory int // category ID for {eof} (-1 if unused)
}

// BuildDFA constructs a DFA from an NFA using subset construction.
func BuildDFA(nfa *NFA, opts DFAOptions) *DFA {
	dfa := &DFA{NumCats: opts.NumCats}

	stateMap := make(map[string]int) // canonical PosSet string → state ID

	startSet := nfa.StartPos
	startState := newDFAState(0, startSet, nfa)
	dfa.States = append(dfa.States, startState)
	stateMap[posSetKey(startSet)] = 0
	dfa.StartState = 0

	worklist := []int{0}

	for len(worklist) > 0 {
		sid := worklist[0]
		worklist = worklist[1:]
		state := dfa.States[sid]

		catTargets := make(map[uint16]PosSet)

		for _, pid := range state.PosSet {
			pos := nfa.Positions[pid]
			if pos.IsEndMark || pos.IsSlash {
				continue
			}
			fp := nfa.FollowPos[pid]
			if len(fp) == 0 {
				continue
			}

			if pos.IsBOF {
				if opts.BOFCategory >= 0 {
					cat := uint16(opts.BOFCategory)
					catTargets[cat] = posSetUnion(catTargets[cat], fp)
				}
			} else if pos.IsEOF {
				if opts.EOFCategory >= 0 {
					cat := uint16(opts.EOFCategory)
					catTargets[cat] = posSetUnion(catTargets[cat], fp)
				}
			} else if pos.IsDot {
				for cat := uint16(0); cat < uint16(opts.NumCats); cat++ {
					catTargets[cat] = posSetUnion(catTargets[cat], fp)
				}
			} else {
				for _, cat := range pos.Classes {
					catTargets[cat] = posSetUnion(catTargets[cat], fp)
				}
			}
		}

		for cat, targetSet := range catTargets {
			if len(targetSet) == 0 {
				continue
			}
			key := posSetKey(targetSet)
			nextID, exists := stateMap[key]
			if !exists {
				nextID = len(dfa.States)
				ns := newDFAState(nextID, targetSet, nfa)
				dfa.States = append(dfa.States, ns)
				stateMap[key] = nextID
				worklist = append(worklist, nextID)
			}
			state.Trans[cat] = nextID
		}
	}

	return dfa
}

func newDFAState(id int, ps PosSet, nfa *NFA) *DFAState {
	s := &DFAState{
		ID:                 id,
		PosSet:             ps,
		Trans:              make(map[uint16]int),
		RuleIndex:          -1,
		Tag:                -1,
		LookAheadRuleIndex: -1,
	}
	for _, pid := range ps {
		pos := nfa.Positions[pid]
		if pos.IsEndMark {
			if !s.Accepting || pos.RuleIndex < s.RuleIndex {
				s.Accepting = true
				s.RuleIndex = pos.RuleIndex
				s.Tag = pos.Tag
			}
		}
		if pos.IsSlash {
			s.LookAhead = true
			if s.LookAheadRuleIndex == -1 || pos.RuleIndex < s.LookAheadRuleIndex {
				s.LookAheadRuleIndex = pos.RuleIndex
			}
		}
	}
	return s
}

func posSetKey(ps PosSet) string {
	sorted := make(PosSet, len(ps))
	copy(sorted, ps)
	sortPosSet(sorted)
	buf := make([]byte, 0, len(sorted)*4)
	for _, v := range sorted {
		buf = append(buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
	return string(buf)
}
