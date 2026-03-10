package breakrules

import "fmt"

// CompileOptions controls the compilation pipeline.
type CompileOptions struct {
	NumCategories    int              // total number of input categories
	PropertyResolver PropertyResolver // maps property expressions to category ID sets
}

// CompileResult holds the output of the compilation pipeline.
type CompileResult struct {
	DFA     *DFA
	RuleSet *RuleSet
}

// Compile runs the full compilation pipeline: parse → resolve → NFA → chain → DFA → minimize.
func Compile(src []byte, opts CompileOptions) (*CompileResult, error) {
	rs, parseErrs := Parse(src)
	if len(parseErrs) > 0 {
		return nil, fmt.Errorf("parse errors: %v", parseErrs)
	}

	if resolveErrs := Resolve(rs); resolveErrs != nil {
		return nil, fmt.Errorf("resolve errors: %v", resolveErrs)
	}

	if opts.PropertyResolver != nil {
		if setErrs := ResolveUnicodeSets(rs, opts.NumCategories, opts.PropertyResolver); setErrs != nil {
			return nil, fmt.Errorf("unicode set errors: %v", setErrs)
		}
	}

	nfa := BuildNFA(rs)

	if rs.Controls["chain"] {
		CalcChainedFollowPos(nfa, rs)
	}

	dfa := BuildDFA(nfa, opts.NumCategories)
	dfa = Minimize(dfa)

	return &CompileResult{DFA: dfa, RuleSet: rs}, nil
}
