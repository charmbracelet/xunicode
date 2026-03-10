package breakrules

// Minimize performs Hopcroft's DFA minimization algorithm.
// It partitions states into equivalence classes where states in the same
// class have identical behavior (same accepting/tag status and equivalent
// transitions).
func Minimize(dfa *DFA) *DFA {
	n := len(dfa.States)
	if n <= 1 {
		return dfa
	}

	// Collect all categories that appear in any transition.
	catSet := make(map[uint16]bool)
	for _, s := range dfa.States {
		for c := range s.Trans {
			catSet[c] = true
		}
	}
	cats := make([]uint16, 0, len(catSet))
	for c := range catSet {
		cats = append(cats, c)
	}

	// Initial partition: group by (accepting, ruleIndex, tag).
	type stateKey struct {
		accepting bool
		ruleIndex int
		tag       int
		lookAhead bool
	}
	keyGroups := make(map[stateKey][]int)
	for _, s := range dfa.States {
		k := stateKey{s.Accepting, s.RuleIndex, s.Tag, s.LookAhead}
		keyGroups[k] = append(keyGroups[k], s.ID)
	}

	partition := make([][]int, 0, len(keyGroups))
	for _, group := range keyGroups {
		partition = append(partition, group)
	}

	stateToGroup := make([]int, n)
	updateStateToGroup := func() {
		for gi, group := range partition {
			for _, sid := range group {
				stateToGroup[sid] = gi
			}
		}
	}
	updateStateToGroup()

	changed := true
	for changed {
		changed = false
		var newPartition [][]int
		for _, group := range partition {
			if len(group) <= 1 {
				newPartition = append(newPartition, group)
				continue
			}
			splits := splitGroup(group, cats, dfa, stateToGroup)
			if len(splits) > 1 {
				changed = true
			}
			newPartition = append(newPartition, splits...)
		}
		partition = newPartition
		updateStateToGroup()
	}

	// Build minimized DFA.
	minDFA := &DFA{NumCats: dfa.NumCats}
	groupID := make(map[int]int) // old group index → new state ID
	for gi := range partition {
		groupID[gi] = gi
	}

	for gi, group := range partition {
		representative := dfa.States[group[0]]
		ms := &DFAState{
			ID:        gi,
			Trans:     make(map[uint16]int),
			Accepting: representative.Accepting,
			RuleIndex: representative.RuleIndex,
			Tag:       representative.Tag,
			LookAhead: representative.LookAhead,
		}
		for cat, target := range representative.Trans {
			ms.Trans[cat] = groupID[stateToGroup[target]]
		}
		minDFA.States = append(minDFA.States, ms)
	}

	minDFA.StartState = groupID[stateToGroup[dfa.StartState]]
	return minDFA
}

func splitGroup(group []int, cats []uint16, dfa *DFA, stateToGroup []int) [][]int {
	// Build a signature for each state: for each category, which group does
	// the transition lead to? (-1 for no transition)
	type sig []int
	makeSig := func(sid int) sig {
		s := dfa.States[sid]
		result := make(sig, len(cats))
		for i, c := range cats {
			if target, ok := s.Trans[c]; ok {
				result[i] = stateToGroup[target]
			} else {
				result[i] = -1
			}
		}
		return result
	}

	sigKey := func(s sig) string {
		buf := make([]byte, len(s)*4)
		for i, v := range s {
			v32 := int32(v)
			buf[i*4] = byte(v32 >> 24)
			buf[i*4+1] = byte(v32 >> 16)
			buf[i*4+2] = byte(v32 >> 8)
			buf[i*4+3] = byte(v32)
		}
		return string(buf)
	}

	groups := make(map[string][]int)
	for _, sid := range group {
		k := sigKey(makeSig(sid))
		groups[k] = append(groups[k], sid)
	}

	result := make([][]int, 0, len(groups))
	for _, g := range groups {
		result = append(result, g)
	}
	return result
}
