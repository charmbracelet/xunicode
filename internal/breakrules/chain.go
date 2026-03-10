package breakrules

// CalcChainedFollowPos implements the !!chain mechanism for RBBI.
//
// When chaining is enabled, after a rule completes (reaches its end-mark),
// the last character consumed can serve as the first character of a new match
// from another rule. This is implemented by adding followpos links from
// positions that can reach an end-mark back to the firstpos of rules whose
// start positions match the same character classes.
//
// Specifically, for each end-mark position e with ruleIndex R:
//   - Find all positions p where e ∈ followpos(p) (i.e., p can transition to e)
//   - For each such p, and for each rule R2 (where R2 may equal R):
//   - If R2 is not marked no-chain-in:
//   - For each start position s in firstpos(R2):
//   - If p and s match overlapping categories (or either is dot):
//   - Add s to followpos(p)
//
// This creates the "overlap" where the last character of one match
// starts a new match.
func CalcChainedFollowPos(nfa *NFA, rs *RuleSet) {
	noChanIn := make(map[int]bool)
	for i, r := range rs.Rules {
		if r.NoChanIn {
			noChanIn[i] = true
		}
	}

	// Collect start positions per rule.
	ruleStartPos := make(map[int]PosSet)
	for _, pid := range nfa.StartPos {
		p := nfa.Positions[pid]
		ruleStartPos[p.RuleIndex] = append(ruleStartPos[p.RuleIndex], pid)
	}

	// Find positions that can reach an end-mark (positions p where
	// followpos(p) contains an end-mark).
	type preEndInfo struct {
		posID     int
		ruleIndex int
	}
	var preEnd []preEndInfo
	for pid, fps := range nfa.FollowPos {
		for _, fid := range fps {
			if nfa.Positions[fid].IsEndMark {
				preEnd = append(preEnd, preEndInfo{pid, nfa.Positions[fid].RuleIndex})
				break
			}
		}
	}

	for _, pe := range preEnd {
		p := nfa.Positions[pe.posID]
		for ruleIdx, startPositions := range ruleStartPos {
			if noChanIn[ruleIdx] {
				continue
			}
			for _, sid := range startPositions {
				s := nfa.Positions[sid]
				if positionsOverlap(p, s) {
					if !nfa.FollowPos[pe.posID].Contains(sid) {
						nfa.FollowPos[pe.posID] = posSetUnion(nfa.FollowPos[pe.posID], PosSet{sid})
					}
				}
			}
		}
	}
}

// positionsOverlap returns true if two positions can match the same input.
func positionsOverlap(a, b *Position) bool {
	if a.IsDot || b.IsDot {
		return true
	}
	if a.IsBOF || a.IsEOF || a.IsEndMark || a.IsSlash {
		return false
	}
	if b.IsBOF || b.IsEOF || b.IsEndMark || b.IsSlash {
		return false
	}
	for _, ac := range a.Classes {
		for _, bc := range b.Classes {
			if ac == bc {
				return true
			}
		}
	}
	return false
}
