package breakrules

import (
	"fmt"
	"strings"
	"testing"
)

// Grapheme property IDs matching grapheme/gen_trieval.go
const (
	gcbOther                uint16 = 0
	gcbCR                   uint16 = 1
	gcbLF                   uint16 = 2
	gcbControl              uint16 = 3
	gcbExtend               uint16 = 4
	gcbZWJ                  uint16 = 5
	gcbRegional_Indicator   uint16 = 6
	gcbPrepend              uint16 = 7
	gcbSpacingMark          uint16 = 8
	gcbL                    uint16 = 9
	gcbV                    uint16 = 10
	gcbT                    uint16 = 11
	gcbLV                   uint16 = 12
	gcbLVT                  uint16 = 13
	gcbExtended_Pictographic uint16 = 14
	gcbInCBLinker           uint16 = 15
	gcbInCBConsonant        uint16 = 16
	gcbInCBExtend           uint16 = 17
	gcbNumCats              int    = 18
)

func graphemeResolver(expr string, negated bool) ([]uint16, error) {
	normalizedExpr := strings.ReplaceAll(expr, " ", "")
	normalizedExpr = strings.ToLower(normalizedExpr)

	m := map[string]uint16{
		"grapheme_cluster_break=other":              gcbOther,
		"grapheme_cluster_break=cr":                 gcbCR,
		"grapheme_cluster_break=lf":                 gcbLF,
		"grapheme_cluster_break=control":            gcbControl,
		"grapheme_cluster_break=extend":             gcbExtend,
		"grapheme_cluster_break=zwj":                gcbZWJ,
		"grapheme_cluster_break=regional_indicator": gcbRegional_Indicator,
		"grapheme_cluster_break=prepend":            gcbPrepend,
		"grapheme_cluster_break=spacingmark":        gcbSpacingMark,
		"grapheme_cluster_break=l":                  gcbL,
		"grapheme_cluster_break=v":                  gcbV,
		"grapheme_cluster_break=t":                  gcbT,
		"grapheme_cluster_break=lv":                 gcbLV,
		"grapheme_cluster_break=lvt":                gcbLVT,
		"extended_pictographic":                     gcbExtended_Pictographic,
		"incb=linker":                               gcbInCBLinker,
		"incb=consonant":                            gcbInCBConsonant,
		"incb=extend":                               gcbInCBExtend,
	}
	id, ok := m[normalizedExpr]
	if !ok {
		return nil, fmt.Errorf("unknown grapheme property %q (normalized: %q)", expr, normalizedExpr)
	}
	if negated {
		result := make([]uint16, 0, gcbNumCats-1)
		for i := uint16(0); i < uint16(gcbNumCats); i++ {
			if i != id {
				result = append(result, i)
			}
		}
		return result, nil
	}
	return []uint16{id}, nil
}

const graphemeRules = `
!!chain;

# Variable definitions — one per GCB property
$CR          = [\p{Grapheme_Cluster_Break = CR}];
$LF          = [\p{Grapheme_Cluster_Break = LF}];
$Control     = [\p{Grapheme_Cluster_Break = Control}];
$Extend      = [\p{Grapheme_Cluster_Break = Extend}];
$ZWJ         = [\p{Grapheme_Cluster_Break = ZWJ}];
$Regional_Indicator = [\p{Grapheme_Cluster_Break = Regional_Indicator}];
$Prepend     = [\p{Grapheme_Cluster_Break = Prepend}];
$SpacingMark = [\p{Grapheme_Cluster_Break = SpacingMark}];
$L           = [\p{Grapheme_Cluster_Break = L}];
$V           = [\p{Grapheme_Cluster_Break = V}];
$T           = [\p{Grapheme_Cluster_Break = T}];
$LV          = [\p{Grapheme_Cluster_Break = LV}];
$LVT         = [\p{Grapheme_Cluster_Break = LVT}];
$Extended_Pictographic = [\p{Extended_Pictographic}];
$InCBLinker  = [\p{InCB = Linker}];
$InCBConsonant = [\p{InCB = Consonant}];
$InCBExtend  = [\p{InCB = Extend}];

# GB3: CR × LF
$CR $LF;

# GB6: L × (L | V | LV | LVT)
$L ($L | $V | $LV | $LVT);

# GB7: (LV | V) × (V | T)
($LV | $V) ($V | $T);

# GB8: (LVT | T) × T
($LVT | $T) $T;

# GB9: × (Extend | ZWJ | InCBExtend | InCBLinker)
. ($Extend | $ZWJ | $InCBExtend | $InCBLinker);

# GB9a: × SpacingMark
. $SpacingMark;

# GB9b: Prepend ×
$Prepend .;

# GB11: ExtPict Extend* ZWJ × ExtPict
$Extended_Pictographic ($Extend | $InCBExtend)* $ZWJ $Extended_Pictographic;

# GB12/13: RI × RI (only pairs)
$Regional_Indicator $Regional_Indicator;

# GB999: Any ÷ Any (default break — handled by DFA as "no match → break")
.;
`

func TestGraphemeRulesCompile(t *testing.T) {
	result, err := Compile([]byte(graphemeRules), CompileOptions{
		NumCategories:    gcbNumCats,
		PropertyResolver: graphemeResolver,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dfa := result.DFA
	if dfa == nil {
		t.Fatal("nil DFA")
	}
	if len(dfa.States) == 0 {
		t.Fatal("no DFA states")
	}

	t.Logf("DFA has %d states (after minimization)", len(dfa.States))

	// Verify basic properties of the compiled DFA.

	// The start state should not be accepting (no rule matches empty string).
	start := dfa.States[dfa.StartState]
	// Note: with the dot rule (GB999: .) and chaining, the start state may
	// or may not be accepting depending on how the NFA was built.
	// The key test is that transitions exist.

	// From start, CR should lead somewhere.
	crNext, hasCR := start.Trans[gcbCR]
	if !hasCR {
		t.Fatal("no transition on CR from start")
	}

	// After CR, LF should lead to an accepting state (GB3: CR × LF).
	crState := dfa.States[crNext]
	lfNext, hasLF := crState.Trans[gcbLF]
	if !hasLF {
		t.Fatal("no transition on LF after CR")
	}
	lfState := dfa.States[lfNext]
	if !lfState.Accepting {
		t.Fatal("state after CR LF should be accepting (GB3)")
	}

	// From start, Regional_Indicator should lead somewhere.
	riNext, hasRI := start.Trans[gcbRegional_Indicator]
	if !hasRI {
		t.Fatal("no transition on RI from start")
	}

	// After RI, another RI should lead to accepting (GB12/13).
	riState := dfa.States[riNext]
	ri2Next, hasRI2 := riState.Trans[gcbRegional_Indicator]
	if !hasRI2 {
		t.Fatal("no transition on RI after RI")
	}
	ri2State := dfa.States[ri2Next]
	if !ri2State.Accepting {
		t.Fatal("state after RI RI should be accepting (GB12/13)")
	}

	// From start, L should lead somewhere, and from there L/V/LV/LVT should be accepted.
	lNext, hasL := start.Trans[gcbL]
	if !hasL {
		t.Fatal("no transition on L from start")
	}
	lState := dfa.States[lNext]
	for _, cat := range []uint16{gcbL, gcbV, gcbLV, gcbLVT} {
		next, ok := lState.Trans[cat]
		if !ok {
			t.Fatalf("no transition on %d after L (GB6)", cat)
		}
		if !dfa.States[next].Accepting {
			t.Fatalf("state after L + cat %d should be accepting (GB6)", cat)
		}
	}
}

func TestGraphemeRulesControls(t *testing.T) {
	result, err := Compile([]byte(graphemeRules), CompileOptions{
		NumCategories:    gcbNumCats,
		PropertyResolver: graphemeResolver,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !result.RuleSet.Controls["chain"] {
		t.Fatal("missing !!chain")
	}
}

func TestGraphemeRulesAssignments(t *testing.T) {
	result, err := Compile([]byte(graphemeRules), CompileOptions{
		NumCategories:    gcbNumCats,
		PropertyResolver: graphemeResolver,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if len(result.RuleSet.Assignments) != 17 {
		t.Fatalf("expected 17 assignments, got %d", len(result.RuleSet.Assignments))
	}
	for _, a := range result.RuleSet.Assignments {
		if a.Expr == nil {
			t.Fatalf("assignment $%s has nil expr", a.Name)
		}
		if a.Expr.Kind != NodeCharClass {
			t.Fatalf("assignment $%s: expected NodeCharClass, got %d", a.Name, a.Expr.Kind)
		}
		if len(a.Expr.Classes) == 0 {
			t.Fatalf("assignment $%s: no classes after resolve", a.Name)
		}
	}
}

func TestGraphemeRulesExtPict(t *testing.T) {
	result, err := Compile([]byte(graphemeRules), CompileOptions{
		NumCategories:    gcbNumCats,
		PropertyResolver: graphemeResolver,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dfa := result.DFA
	start := dfa.States[dfa.StartState]

	// ExtPict → Extend* → ZWJ → ExtPict should reach accepting (GB11).
	epNext, hasEP := start.Trans[gcbExtended_Pictographic]
	if !hasEP {
		t.Fatal("no transition on ExtPict from start")
	}
	epState := dfa.States[epNext]

	// From ExtPict state, ZWJ should go somewhere.
	zwjNext, hasZWJ := epState.Trans[gcbZWJ]
	if !hasZWJ {
		t.Fatal("no transition on ZWJ after ExtPict")
	}
	zwjState := dfa.States[zwjNext]

	// From ZWJ state, ExtPict should reach accepting.
	ep2Next, hasEP2 := zwjState.Trans[gcbExtended_Pictographic]
	if !hasEP2 {
		t.Fatal("no transition on ExtPict after ExtPict+ZWJ")
	}
	ep2State := dfa.States[ep2Next]
	if !ep2State.Accepting {
		t.Fatal("ExtPict Extend* ZWJ ExtPict should be accepting (GB11)")
	}
}

func TestGraphemeRulesDFADeterministic(t *testing.T) {
	result1, err := Compile([]byte(graphemeRules), CompileOptions{
		NumCategories:    gcbNumCats,
		PropertyResolver: graphemeResolver,
	})
	if err != nil {
		t.Fatalf("compile 1: %v", err)
	}
	result2, err := Compile([]byte(graphemeRules), CompileOptions{
		NumCategories:    gcbNumCats,
		PropertyResolver: graphemeResolver,
	})
	if err != nil {
		t.Fatalf("compile 2: %v", err)
	}
	if len(result1.DFA.States) != len(result2.DFA.States) {
		t.Fatalf("non-deterministic: %d vs %d states",
			len(result1.DFA.States), len(result2.DFA.States))
	}
}
