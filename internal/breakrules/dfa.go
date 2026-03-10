package breakrules

// DFA subset construction from a position-based NFA.
//
// Each DFA state is a set of NFA positions. Transitions are computed by
// collecting followpos for all positions that match each input category.
//
// A DFA state is accepting if it contains an end-mark position.
// A DFA state has lookahead if it contains a slash position.

// DFA represents the deterministic finite automaton.
type DFA struct {
	States     []*DFAState
	StartState int
	NumCats    int // number of input categories
}

// DFAState is a single DFA state.
type DFAState struct {
	ID         int
	PosSet     PosSet       // the set of NFA positions in this state
	Trans      map[uint16]int // category → next state ID (-1 if none)
	Accepting  bool         // contains an end-mark position
	RuleIndex  int          // rule index of the accepting end-mark (-1 if not accepting, lowest if multiple)
	Tag        int          // status tag from the accepting rule (-1 if none)
	LookAhead  bool         // contains a slash (lookahead) position
}

// BuildDFA constructs a DFA from an NFA using subset construction.
// numCats is the total number of input categories.
func BuildDFA(nfa *NFA, numCats int) *DFA {
	dfa := &DFA{NumCats: numCats}

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

		// For each category, compute the union of followpos for all
		// positions in this state that match that category.
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

			if pos.IsDot {
				for cat := uint16(0); cat < uint16(numCats); cat++ {
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
		ID:        id,
		PosSet:    ps,
		Trans:     make(map[uint16]int),
		RuleIndex: -1,
		Tag:       -1,
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
