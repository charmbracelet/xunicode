package breakrules

// Position-based NFA construction (Aho et al. "Compilers" textbook).
//
// Each leaf in the parse tree is a "position". We compute:
//   - nullable(n): can the subtree match empty string?
//   - firstpos(n): set of positions that can begin a match of the subtree
//   - lastpos(n):  set of positions that can end a match of the subtree
//   - followpos(p): set of positions that can follow position p
//
// The end-mark position is a synthetic leaf appended to each rule.
// Accepting states in the DFA correspond to state sets containing an end-mark.

// Position represents a single leaf in the NFA.
type Position struct {
	ID        int      // unique position ID
	Classes   []uint16 // category IDs this position matches (empty for end-mark/dot)
	IsDot     bool     // matches any category
	IsEndMark bool     // synthetic end-of-rule marker
	IsBOF     bool     // {bof} anchor
	IsEOF     bool     // {eof} anchor
	IsSlash   bool     // lookahead break point marker
	RuleIndex int      // which rule this position belongs to
	Tag       int      // rule status tag, or -1
}

// NFA holds the position-based NFA: positions and followpos sets.
type NFA struct {
	Positions []*Position
	FollowPos []PosSet // followpos[posID] → set of position IDs
	StartPos  PosSet   // firstpos of the augmented start expression
}

// PosSet is a set of position IDs, implemented as a sorted slice.
type PosSet []int

func (s PosSet) Contains(id int) bool {
	for _, v := range s {
		if v == id {
			return true
		}
	}
	return false
}

func posSetUnion(a, b PosSet) PosSet {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	seen := make(map[int]bool, len(a)+len(b))
	for _, v := range a {
		seen[v] = true
	}
	for _, v := range b {
		seen[v] = true
	}
	result := make(PosSet, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	sortPosSet(result)
	return result
}

func sortPosSet(s PosSet) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// posSetEqual returns true if two PosSets contain the same elements.
func posSetEqual(a, b PosSet) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// nfaBuilder accumulates positions during tree traversal.
type nfaBuilder struct {
	positions []*Position
	nextID    int
}

func (b *nfaBuilder) newPos(ruleIdx int) *Position {
	p := &Position{ID: b.nextID, RuleIndex: ruleIdx, Tag: -1}
	b.nextID++
	b.positions = append(b.positions, p)
	return p
}

type treeInfo struct {
	nullable bool
	firstpos PosSet
	lastpos  PosSet
}

// BuildNFA constructs a position-based NFA from a resolved RuleSet.
// Each rule gets an end-mark appended: rule_expr . #endmark
// All rules are combined via alternation to form the start expression.
func BuildNFA(rs *RuleSet) *NFA {
	b := &nfaBuilder{}
	followPos := make(map[int]PosSet)

	var ruleInfos []treeInfo
	for ruleIdx, r := range rs.Rules {
		ruleInfo := b.calcTree(r.Expr, ruleIdx, followPos)

		endMark := b.newPos(ruleIdx)
		endMark.IsEndMark = true
		endMark.Tag = r.Tag

		endInfo := treeInfo{
			nullable: false,
			firstpos: PosSet{endMark.ID},
			lastpos:  PosSet{endMark.ID},
		}

		combined := concatInfo(ruleInfo, endInfo, followPos)
		ruleInfos = append(ruleInfos, combined)
	}

	var startPos PosSet
	for _, ri := range ruleInfos {
		startPos = posSetUnion(startPos, ri.firstpos)
	}

	nfa := &NFA{
		Positions: b.positions,
		FollowPos: make([]PosSet, len(b.positions)),
		StartPos:  startPos,
	}
	for id, fps := range followPos {
		nfa.FollowPos[id] = fps
	}
	return nfa
}

func (b *nfaBuilder) calcTree(n *Node, ruleIdx int, fp map[int]PosSet) treeInfo {
	if n == nil {
		return treeInfo{nullable: true}
	}
	switch n.Kind {
	case NodeCharClass:
		p := b.newPos(ruleIdx)
		p.Classes = n.Classes
		if n.Name != "" && len(n.Classes) == 0 {
			p.Classes = nil
		}
		return treeInfo{
			nullable: false,
			firstpos: PosSet{p.ID},
			lastpos:  PosSet{p.ID},
		}
	case NodeDot:
		p := b.newPos(ruleIdx)
		p.IsDot = true
		return treeInfo{
			nullable: false,
			firstpos: PosSet{p.ID},
			lastpos:  PosSet{p.ID},
		}
	case NodeBOF:
		p := b.newPos(ruleIdx)
		p.IsBOF = true
		return treeInfo{
			nullable: false,
			firstpos: PosSet{p.ID},
			lastpos:  PosSet{p.ID},
		}
	case NodeEOF:
		p := b.newPos(ruleIdx)
		p.IsEOF = true
		return treeInfo{
			nullable: false,
			firstpos: PosSet{p.ID},
			lastpos:  PosSet{p.ID},
		}
	case NodeSlash:
		p := b.newPos(ruleIdx)
		p.IsSlash = true
		return treeInfo{
			nullable: false,
			firstpos: PosSet{p.ID},
			lastpos:  PosSet{p.ID},
		}
	case NodeConcat:
		if len(n.Children) == 0 {
			return treeInfo{nullable: true}
		}
		result := b.calcTree(n.Children[0], ruleIdx, fp)
		for i := 1; i < len(n.Children); i++ {
			right := b.calcTree(n.Children[i], ruleIdx, fp)
			result = concatInfo(result, right, fp)
		}
		return result
	case NodeAlt:
		if len(n.Children) == 0 {
			return treeInfo{nullable: true}
		}
		result := b.calcTree(n.Children[0], ruleIdx, fp)
		for i := 1; i < len(n.Children); i++ {
			right := b.calcTree(n.Children[i], ruleIdx, fp)
			result = altInfo(result, right)
		}
		return result
	case NodeStar:
		child := b.calcTree(n.Child, ruleIdx, fp)
		for _, lp := range child.lastpos {
			fp[lp] = posSetUnion(fp[lp], child.firstpos)
		}
		return treeInfo{
			nullable: true,
			firstpos: child.firstpos,
			lastpos:  child.lastpos,
		}
	case NodePlus:
		child := b.calcTree(n.Child, ruleIdx, fp)
		for _, lp := range child.lastpos {
			fp[lp] = posSetUnion(fp[lp], child.firstpos)
		}
		return treeInfo{
			nullable: child.nullable,
			firstpos: child.firstpos,
			lastpos:  child.lastpos,
		}
	case NodeQuest:
		child := b.calcTree(n.Child, ruleIdx, fp)
		return treeInfo{
			nullable: true,
			firstpos: child.firstpos,
			lastpos:  child.lastpos,
		}
	case NodeVariable:
		return treeInfo{nullable: true}
	case NodeEndMark:
		p := b.newPos(ruleIdx)
		p.IsEndMark = true
		return treeInfo{
			nullable: false,
			firstpos: PosSet{p.ID},
			lastpos:  PosSet{p.ID},
		}
	default:
		return treeInfo{nullable: true}
	}
}

func concatInfo(left, right treeInfo, fp map[int]PosSet) treeInfo {
	for _, lp := range left.lastpos {
		fp[lp] = posSetUnion(fp[lp], right.firstpos)
	}
	var first, last PosSet
	if left.nullable {
		first = posSetUnion(left.firstpos, right.firstpos)
	} else {
		first = left.firstpos
	}
	if right.nullable {
		last = posSetUnion(left.lastpos, right.lastpos)
	} else {
		last = right.lastpos
	}
	return treeInfo{
		nullable: left.nullable && right.nullable,
		firstpos: first,
		lastpos:  last,
	}
}

func altInfo(left, right treeInfo) treeInfo {
	return treeInfo{
		nullable: left.nullable || right.nullable,
		firstpos: posSetUnion(left.firstpos, right.firstpos),
		lastpos:  posSetUnion(left.lastpos, right.lastpos),
	}
}
