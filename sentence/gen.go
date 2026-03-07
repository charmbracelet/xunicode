//go:build ignore

package main

import (
	"fmt"
	"log"
	"unicode"

	"github.com/charmbracelet/xunicode/internal/gen"
	"github.com/charmbracelet/xunicode/internal/segmenter"
	"github.com/charmbracelet/xunicode/internal/triegen"
	"github.com/charmbracelet/xunicode/internal/ucd"
)

var sbMap = map[string]Class{
	"Other":     Other,
	"CR":        CR,
	"LF":        LF,
	"Sep":       Sep,
	"Extend":    Extend,
	"Format":    Format,
	"Sp":        Sp,
	"Lower":     Lower,
	"Upper":     Upper,
	"OLetter":   OLetter,
	"Numeric":   Numeric,
	"ATerm":     ATerm,
	"STerm":     STerm,
	"SContinue": SContinue,
	"Close":     Close,
}

func p(v ...uint8) []uint8 {
	return v
}

func main() {
	gen.Init()
	genTables()
}

func genTables() {
	gen.Repackage("gen_trieval.go", "trieval.go", "sentence")

	props := make([]Class, unicode.MaxRune+1)

	ucd.Parse(gen.OpenUCDFile("auxiliary/SentenceBreakProperty.txt"), func(parser *ucd.Parser) {
		r := parser.Rune(0)
		val := parser.String(1)
		cls, ok := sbMap[val]
		if !ok {
			log.Fatalf("U+%04X: unknown Sentence_Break value %q", r, val)
		}
		props[r] = cls
	})

	w := gen.NewCodeWriter()
	defer w.WriteVersionedGoFile("tables.go", "sentence")

	fmt.Fprintf(w, "import %q\n\n", "github.com/charmbracelet/xunicode/internal/segmenter")
	gen.WriteUnicodeVersion(w)

	t := triegen.NewTrie("sentence")
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if props[r] != 0 {
			t.Insert(r, uint64(props[r]))
		}
	}
	sz, err := t.Gen(w)
	if err != nil {
		log.Fatal(err)
	}
	w.Size += sz

	rules := buildRules()
	bt := segmenter.Build(rules, uint8(stride), uint8(sot), uint8(eot), uint8(lastCP))
	segmenter.WriteBreakTable(w, "ruleData", bt, "sentenceTrie", 0)
}

func buildRules() []segmenter.Rule {
	sb5Ignored := p(uint8(Extend), uint8(Format))

	idx := segmenter.IndexState
	interm := segmenter.IntermediateState

	var rules []segmenter.Rule

	// =========================================================================
	// Rules in spec order (SB1–SB998). First-write-wins: earlier rules
	// (higher priority) take precedence.
	// =========================================================================

	// SB1: sot ÷ Any
	rules = append(rules, segmenter.SimpleRule{Left: p(sot), Right: nil, Break: false})

	// SB2: Any ÷ eot
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: p(eot), Break: true})

	// SB3: CR × LF
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(CR)), Right: p(uint8(LF)), Break: false})

	// SB4: ParaSep ÷
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(Sep), uint8(CR), uint8(LF)), Break: true})

	// SB5: X (Extend | Format)* → X
	for _, base := range p(uint8(Lower), uint8(Upper), uint8(OLetter), uint8(ATerm), uint8(STerm)) {
		rules = append(rules, segmenter.IgnoreRule{
			Props:   []uint8{base},
			Ignored: sb5Ignored,
			Target:  func(b, _ uint8) uint8 { return b },
		})
	}
	for _, cs := range p(UpperATerm, LowerATerm) {
		rules = append(rules, segmenter.IgnoreRule{
			Props:   []uint8{cs},
			Ignored: sb5Ignored,
			Target:  func(b, _ uint8) uint8 { return b },
		})
	}

	// SB6: ATerm × Numeric
	// SB7: (Upper | Lower) ATerm × Upper
	// SB8: ATerm Close* Sp* × (¬(OLetter|Upper|Lower|ParaSep|SATerm))* Lower
	// SB8a: SATerm Close* Sp* × (SContinue | SATerm)
	// SB9: SATerm Close* × (Close | Sp | ParaSep)
	// SB10: SATerm Close* Sp* × (Sp | ParaSep)
	// SB11: SATerm Close* Sp* ParaSep? ÷
	//
	// These rules interact via chain states. The ATerm/STerm base rows and
	// their Close/Sp/ParaSep chain states implement SB6–SB11 together.

	rules = append(rules, segmenter.ChainRule{
		Entry: p(uint8(Upper)),
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(ATerm)), State: uint8(UpperATerm)},
		},
	})
	rules = append(rules, segmenter.ChainRule{
		Entry: p(uint8(Lower)),
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(ATerm)), State: uint8(LowerATerm)},
		},
	})

	// ATerm row: SB6 + SB8 + SB8a + SB9 + SB11
	atermOverrides := map[uint8]uint8{
		uint8(Numeric):   segmenter.Keep,                   // SB6
		uint8(Close):     interm(uint8(ATermClose)),        // SB9
		uint8(Sp):        interm(uint8(ATermCloseSp)),      // SB9
		uint8(Sep):       interm(uint8(ATermCloseSpPSep)),  // SB9/SB11
		uint8(LF):        interm(uint8(ATermCloseSpPSep)),  // SB9/SB11
		uint8(CR):        interm(uint8(ATermCloseSpCR)),    // SB9/SB11
		uint8(Extend):    idx(uint8(ATerm)),                // SB5
		uint8(Format):    idx(uint8(ATerm)),                // SB5
		uint8(SContinue): segmenter.Keep,                   // SB8a
		uint8(ATerm):     segmenter.Keep,                   // SB8a
		uint8(STerm):     segmenter.Keep,                   // SB8a
		uint8(Other):     idx(uint8(ATermCloseSpSB8)),      // SB8
		uint8(Lower):     segmenter.Keep,                   // SB8
	}
	rules = append(rules, segmenter.OverrideRule{
		States:    p(uint8(ATerm)),
		Overrides: atermOverrides,
		WipeValue: segmenter.Break,
	})

	// UpperATerm row: ATerm + SB7 (× Upper → Keep)
	uaOverrides := make(map[uint8]uint8)
	for k, v := range atermOverrides {
		uaOverrides[k] = v
	}
	uaOverrides[uint8(Extend)] = idx(uint8(UpperATerm))
	uaOverrides[uint8(Format)] = idx(uint8(UpperATerm))
	uaOverrides[uint8(Upper)] = segmenter.Keep // SB7
	rules = append(rules, segmenter.OverrideRule{
		States:    p(UpperATerm),
		Overrides: uaOverrides,
		WipeValue: segmenter.Break,
	})

	// LowerATerm row: ATerm + SB7 (× Upper → Keep)
	laOverrides := make(map[uint8]uint8)
	for k, v := range atermOverrides {
		laOverrides[k] = v
	}
	laOverrides[uint8(Extend)] = idx(uint8(LowerATerm))
	laOverrides[uint8(Format)] = idx(uint8(LowerATerm))
	laOverrides[uint8(Upper)] = segmenter.Keep // SB7
	rules = append(rules, segmenter.OverrideRule{
		States:    p(LowerATerm),
		Overrides: laOverrides,
		WipeValue: segmenter.Break,
	})

	// ATermClose row: SB8 + SB8a + SB9 + SB11
	rules = append(rules, segmenter.OverrideRule{
		States: p(ATermClose),
		Overrides: map[uint8]uint8{
			uint8(Close):     interm(uint8(ATermClose)),       // SB9
			uint8(Sp):        interm(uint8(ATermCloseSp)),     // SB9
			uint8(Sep):       interm(uint8(ATermCloseSpPSep)), // SB11
			uint8(LF):        interm(uint8(ATermCloseSpPSep)), // SB11
			uint8(CR):        interm(uint8(ATermCloseSpCR)),   // SB11
			uint8(Extend):    interm(uint8(ATermClose)),       // SB5
			uint8(Format):    interm(uint8(ATermClose)),       // SB5
			uint8(SContinue): segmenter.Keep,                  // SB8a
			uint8(ATerm):     segmenter.Keep,                  // SB8a
			uint8(STerm):     segmenter.Keep,                  // SB8a
			uint8(Lower):     segmenter.Keep,                  // SB8
			uint8(Numeric):   idx(uint8(ATermCloseSpSB8)),     // SB8
			uint8(Other):     idx(uint8(ATermCloseSpSB8)),     // SB8
		},
		WipeValue: segmenter.Break,
	})

	// ATermCloseSp row: SB8 + SB8a + SB10 + SB11
	rules = append(rules, segmenter.OverrideRule{
		States: p(ATermCloseSp),
		Overrides: map[uint8]uint8{
			uint8(Sp):        interm(uint8(ATermCloseSp)),     // SB10
			uint8(Sep):       interm(uint8(ATermCloseSpPSep)), // SB11
			uint8(LF):        interm(uint8(ATermCloseSpPSep)), // SB11
			uint8(CR):        interm(uint8(ATermCloseSpCR)),   // SB11
			uint8(Extend):    interm(uint8(ATermCloseSp)),     // SB5
			uint8(Format):    interm(uint8(ATermCloseSp)),     // SB5
			uint8(SContinue): segmenter.Keep,                  // SB8a
			uint8(ATerm):     segmenter.Keep,                  // SB8a
			uint8(STerm):     segmenter.Keep,                  // SB8a
			uint8(Lower):     segmenter.Keep,                  // SB8
			uint8(Close):     idx(uint8(ATermCloseSpSB8)),     // SB8
			uint8(Numeric):   idx(uint8(ATermCloseSpSB8)),     // SB8
			uint8(Other):     idx(uint8(ATermCloseSpSB8)),     // SB8
		},
		WipeValue: segmenter.Break,
	})

	// ATermCloseSpSB8 row: SB8 scanning for Lower
	rules = append(rules, segmenter.OverrideRule{
		States: p(ATermCloseSpSB8),
		Overrides: map[uint8]uint8{
			uint8(Lower):     segmenter.Keep,                // SB8 completion
			uint8(Close):     idx(uint8(ATermCloseSpSB8)),   // SB8 scan
			uint8(Sp):        idx(uint8(ATermCloseSpSB8)),   // SB8 scan
			uint8(Numeric):   idx(uint8(ATermCloseSpSB8)),   // SB8 scan
			uint8(Other):     idx(uint8(ATermCloseSpSB8)),   // SB8 scan
			uint8(SContinue): idx(uint8(ATermCloseSpSB8)),   // SB8 scan
			uint8(Extend):    idx(uint8(ATermCloseSpSB8)),   // SB5
			uint8(Format):    idx(uint8(ATermCloseSpSB8)),   // SB5
		},
		WipeValue: segmenter.NoMatch,
	})

	// ATermCloseSpPSep row: SB11 (break after ParaSep)
	rules = append(rules, segmenter.OverrideRule{
		States:    p(ATermCloseSpPSep),
		Overrides: nil,
		WipeValue: segmenter.Break,
	})

	// ATermCloseSpCR row: SB3 × LF, else SB11 break
	rules = append(rules, segmenter.OverrideRule{
		States: p(ATermCloseSpCR),
		Overrides: map[uint8]uint8{
			uint8(LF): segmenter.Keep, // SB3
		},
		WipeValue: segmenter.Break,
	})

	// STerm row: SB8a + SB9 + SB11
	rules = append(rules, segmenter.OverrideRule{
		States: p(uint8(STerm)),
		Overrides: map[uint8]uint8{
			uint8(Close):     interm(uint8(STermClose)),       // SB9
			uint8(Sp):        interm(uint8(STermCloseSp)),     // SB9
			uint8(Sep):       interm(uint8(STermCloseSpPSep)), // SB11
			uint8(LF):        interm(uint8(STermCloseSpPSep)), // SB11
			uint8(CR):        interm(uint8(STermCloseSpCR)),   // SB11
			uint8(Extend):    idx(uint8(STerm)),               // SB5
			uint8(Format):    idx(uint8(STerm)),               // SB5
			uint8(SContinue): segmenter.Keep,                  // SB8a
			uint8(ATerm):     segmenter.Keep,                  // SB8a
			uint8(STerm):     segmenter.Keep,                  // SB8a
		},
		WipeValue: segmenter.Break,
	})

	// STermClose row: SB8a + SB9 + SB11
	rules = append(rules, segmenter.OverrideRule{
		States: p(STermClose),
		Overrides: map[uint8]uint8{
			uint8(Close):     interm(uint8(STermClose)),       // SB9
			uint8(Sp):        interm(uint8(STermCloseSp)),     // SB10
			uint8(Sep):       interm(uint8(STermCloseSpPSep)), // SB11
			uint8(LF):        interm(uint8(STermCloseSpPSep)), // SB11
			uint8(CR):        interm(uint8(STermCloseSpCR)),   // SB11
			uint8(Extend):    idx(uint8(STermClose)),          // SB5
			uint8(Format):    idx(uint8(STermClose)),          // SB5
			uint8(SContinue): segmenter.Keep,                  // SB8a
			uint8(ATerm):     segmenter.Keep,                  // SB8a
			uint8(STerm):     segmenter.Keep,                  // SB8a
		},
		WipeValue: segmenter.Break,
	})

	// STermCloseSp row: SB8a + SB10 + SB11
	rules = append(rules, segmenter.OverrideRule{
		States: p(STermCloseSp),
		Overrides: map[uint8]uint8{
			uint8(Sp):        interm(uint8(STermCloseSp)),     // SB10
			uint8(Sep):       interm(uint8(STermCloseSpPSep)), // SB11
			uint8(LF):        interm(uint8(STermCloseSpPSep)), // SB11
			uint8(CR):        interm(uint8(STermCloseSpCR)),   // SB11
			uint8(Extend):    idx(uint8(STermCloseSp)),        // SB5
			uint8(Format):    idx(uint8(STermCloseSp)),        // SB5
			uint8(SContinue): segmenter.Keep,                  // SB8a
			uint8(ATerm):     segmenter.Keep,                  // SB8a
			uint8(STerm):     segmenter.Keep,                  // SB8a
		},
		WipeValue: segmenter.Break,
	})

	// STermCloseSpPSep row: SB11 (break after ParaSep)
	rules = append(rules, segmenter.OverrideRule{
		States:    p(STermCloseSpPSep),
		Overrides: nil,
		WipeValue: segmenter.Break,
	})

	// STermCloseSpCR row: SB3 × LF, else SB11 break
	rules = append(rules, segmenter.OverrideRule{
		States: p(STermCloseSpCR),
		Overrides: map[uint8]uint8{
			uint8(LF): segmenter.Keep, // SB3
		},
		WipeValue: segmenter.Break,
	})

	// SB998: Any × Any
	rules = append(rules, segmenter.SimpleRule{Break: false})

	return rules
}
