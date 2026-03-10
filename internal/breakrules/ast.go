package breakrules

// Node is a single node in the rule expression AST.
type Node struct {
	Kind     NodeKind
	Children []*Node  // for Concat, Alt
	Child    *Node    // for Star, Plus, Quest, Caret
	Classes  []uint16 // for CharClass — set of category IDs (leaf)
	Name     string   // for Variable (pre-resolution)
	Tag      int      // for rule-level status tag {N}
}

// NodeKind classifies AST nodes.
type NodeKind int

const (
	NodeConcat    NodeKind = iota // a b — sequence
	NodeAlt                      // a | b — alternation
	NodeStar                     // a* — zero or more
	NodePlus                     // a+ — one or more
	NodeQuest                    // a? — zero or one
	NodeCharClass                // [...] or \p{} — leaf: set of categories
	NodeDot                      // . — any character (all categories)
	NodeVariable                 // $name (pre-resolution; replaced during resolve)
	NodeCaret                    // ^ at rule start — suppress chain-in
	NodeSlash                    // / — lookahead break point
	NodeEndMark                  // synthetic end-of-rule marker (added by compiler)
	NodeBOF                      // {bof} pseudo-anchor
	NodeEOF                      // {eof} pseudo-anchor
)

// Rule is a parsed top-level rule statement.
type Rule struct {
	Expr      *Node // the rule expression (may contain NodeSlash for lookahead)
	NoChanIn  bool  // true if rule starts with ^
	Tag       int   // rule status tag from {N}, or -1 if none
	SourcePos int   // byte offset of the rule in the source
}

// Assignment is a parsed $variable = expr; statement.
type Assignment struct {
	Name      string
	Expr      *Node
	SourcePos int
}

// RuleSet is the complete parsed content of a .rules file.
type RuleSet struct {
	Controls    map[string]bool // !!chain, !!quoted_literals_only, etc.
	Assignments []*Assignment   // in order of appearance
	Rules       []*Rule         // in order of appearance
}
