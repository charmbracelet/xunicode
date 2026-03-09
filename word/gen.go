//go:build ignore

package main

import (
	"log"
	"unicode"

	"github.com/charmbracelet/xunicode/internal/gen"
	"github.com/charmbracelet/xunicode/internal/segmenter"
	"github.com/charmbracelet/xunicode/internal/triegen"
	"github.com/charmbracelet/xunicode/internal/ucd"
)

var wbMap = map[string]Class{
	"Other":              Other,
	"CR":                 CR,
	"LF":                 LF,
	"Newline":            Newline,
	"Extend":             Extend,
	"ZWJ":                ZWJ,
	"Regional_Indicator": Regional_Indicator,
	"Format":             Format,
	"Katakana":           Katakana,
	"Hebrew_Letter":      Hebrew_Letter,
	"ALetter":            ALetter,
	"Single_Quote":       Single_Quote,
	"Double_Quote":       Double_Quote,
	"MidNumLet":          MidNumLet,
	"MidLetter":          MidLetter,
	"MidNum":             MidNum,
	"Numeric":            Numeric,
	"ExtendNumLet":       ExtendNumLet,
	"WSegSpace":          WSegSpace,
}

func p(v ...uint8) []uint8 {
	return v
}

func main() {
	gen.Init()
	genTables()
}

func genTables() {
	gen.Repackage("gen_trieval.go", "trieval.go", "word")

	isALetter := make([]bool, unicode.MaxRune+1)
	props := make([]Class, unicode.MaxRune+1)

	ucd.Parse(gen.OpenUCDFile("auxiliary/WordBreakProperty.txt"), func(parser *ucd.Parser) {
		r := parser.Rune(0)
		val := parser.String(1)
		cls, ok := wbMap[val]
		if !ok {
			log.Fatalf("U+%04X: unknown Word_Break value %q", r, val)
		}
		props[r] = cls
		if cls == ALetter {
			isALetter[r] = true
		}
	})

	ucd.Parse(gen.OpenUCDFile("emoji/emoji-data.txt"), func(parser *ucd.Parser) {
		if parser.String(1) == "Extended_Pictographic" {
			r := parser.Rune(0)
			if isALetter[r] {
				props[r] = ALetter_Extended_Pictographic
			} else if props[r] == Other {
				props[r] = Extended_Pictographic
			}
		}
	})

	isNumeric := func(r rune) bool { return props[r] == Numeric }
	ucd.Parse(gen.OpenUCDFile("LineBreak.txt"), func(parser *ucd.Parser) {
		if parser.String(1) == "SA" {
			r := parser.Rune(0)
			if r == 0x19DA || isNumeric(r) || props[r] == Extend {
				return
			}
			props[r] = SA
		}
	})
	ucd.Parse(gen.OpenUCDFile("Scripts.txt"), func(parser *ucd.Parser) {
		sc := parser.String(1)
		if sc == "Han" || sc == "Hiragana" {
			r := parser.Rune(0)
			if !isNumeric(r) {
				props[r] = SA
			}
		}
	})

	w := gen.NewCodeWriter()
	defer w.WriteVersionedGoFile("tables.go", "word")

	gen.WriteUnicodeVersion(w)

	t := triegen.NewTrie("word")
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
	segmenter.WriteBreakTable(w, bt)
}

func buildRules() []segmenter.Rule {
	ahletter := p(uint8(ALetter), uint8(Hebrew_Letter), uint8(ALetter_Extended_Pictographic))
	ahletterPlusZWJ := p(uint8(ALetter), uint8(Hebrew_Letter), uint8(ALetter_Extended_Pictographic),
		ALetter_ZWJ, Hebrew_Letter_ZWJ, ALetterEP_ZWJ)
	hebrewPlusZWJ := p(uint8(Hebrew_Letter), Hebrew_Letter_ZWJ)
	numericPlusZWJ := p(uint8(Numeric), Numeric_ZWJ)
	katakanaPlusZWJ := p(uint8(Katakana), Katakana_ZWJ)
	extNumLetPlusZWJ := p(uint8(ExtendNumLet), ExtendNumLet_ZWJ)
	riPlusZWJ := p(uint8(Regional_Indicator), RI_ZWJ)
	allZWJ := p(uint8(ZWJ), ALetter_ZWJ, Hebrew_Letter_ZWJ, Numeric_ZWJ,
		Katakana_ZWJ, ExtendNumLet_ZWJ, RI_ZWJ, ExtPict_ZWJ,
		WSegSpace_ZWJ, ALetterEP_ZWJ)
	midLetterQ := p(uint8(MidLetter), uint8(MidNumLet), uint8(Single_Quote))
	midNumQ := p(uint8(MidNum), uint8(MidNumLet), uint8(Single_Quote))

	wb4Ignored := p(uint8(Extend), uint8(Format), uint8(ZWJ))
	wb13aLeft := make([]uint8, 0, len(ahletterPlusZWJ)+len(numericPlusZWJ)+len(katakanaPlusZWJ)+len(extNumLetPlusZWJ))
	wb13aLeft = append(wb13aLeft, ahletterPlusZWJ...)
	wb13aLeft = append(wb13aLeft, numericPlusZWJ...)
	wb13aLeft = append(wb13aLeft, katakanaPlusZWJ...)
	wb13aLeft = append(wb13aLeft, extNumLetPlusZWJ...)

	idx := segmenter.IndexState

	// WB4 absorption base definitions.
	wb4Bases := []struct {
		base uint8
		zwj  uint8
		ext  uint8
	}{
		{uint8(ALetter), ALetter_ZWJ, uint8(ALetter)},
		{uint8(Hebrew_Letter), Hebrew_Letter_ZWJ, uint8(Hebrew_Letter)},
		{uint8(Numeric), Numeric_ZWJ, uint8(Numeric)},
		{uint8(Katakana), Katakana_ZWJ, uint8(Katakana)},
		{uint8(ExtendNumLet), ExtendNumLet_ZWJ, uint8(ExtendNumLet)},
		{uint8(Regional_Indicator), RI_ZWJ, uint8(Regional_Indicator)},
		{uint8(Extended_Pictographic), ExtPict_ZWJ, uint8(Extended_Pictographic)},
		{uint8(WSegSpace), WSegSpace_ZWJ, WSegSpace_XX},
		{WSegSpace_XX, WSegSpace_ZWJ, WSegSpace_XX},
		{uint8(ALetter_Extended_Pictographic), ALetterEP_ZWJ, uint8(ALetter_Extended_Pictographic)},
	}

	var rules []segmenter.Rule

	// =========================================================================
	// Rules in spec order (WB1–WB999). First-write-wins: earlier rules
	// (higher priority) take precedence.
	// =========================================================================

	// WB1: sot ÷ Any
	rules = append(rules, segmenter.SimpleRule{Left: p(sot), Right: nil, Break: false})

	// WB2: Any ÷ eot
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: p(eot), Break: true})

	// WB3: CR × LF
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(CR)), Right: p(uint8(LF)), Break: false})

	// WB3a: (Newline | CR | LF) ÷
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(Newline), uint8(CR), uint8(LF)), Break: true})

	// WB3b: ÷ (Newline | CR | LF)
	rules = append(rules, segmenter.SimpleRule{Right: p(uint8(Newline), uint8(CR), uint8(LF)), Break: true})

	// WB3c: ZWJ × \p{Extended_Pictographic}
	rules = append(rules, segmenter.SimpleRule{
		Left: allZWJ, Right: p(uint8(Extended_Pictographic), uint8(ALetter_Extended_Pictographic)), Break: false,
	})

	// WB3d: WSegSpace × WSegSpace
	rules = append(rules, segmenter.SimpleRule{
		Left: p(uint8(WSegSpace)), Right: p(uint8(WSegSpace)), Break: false,
	})

	// WB4: X (Extend | Format | ZWJ)* → X
	rules = append(rules, segmenter.SimpleRule{Right: wb4Ignored, Break: false})
	for _, b := range wb4Bases {
		rules = append(rules, segmenter.IgnoreRule{
			Props:   []uint8{b.base, b.zwj},
			Ignored: wb4Ignored,
			Target: func(_, ign uint8) uint8 {
				if ign == uint8(ZWJ) {
					return b.zwj
				}
				return b.ext
			},
		})
	}

	// WB5: AHLetter × AHLetter
	rules = append(rules, segmenter.SimpleRule{Left: ahletterPlusZWJ, Right: ahletter, Break: false})

	// WB6: AHLetter × (MidLetter | MidNumLetQ) AHLetter
	rules = append(rules, segmenter.ChainRule{
		Entry: ahletterPlusZWJ,
		Steps: []segmenter.ChainStep{
			{Props: midLetterQ, State: AHL_MidLetter},
		},
	})

	// WB7: AHLetter (MidLetter | MidNumLetQ) × AHLetter
	ahlOverrides := map[uint8]uint8{
		uint8(Extend): idx(AHL_MidLetter),
		uint8(Format): idx(AHL_MidLetter),
		uint8(ZWJ):    idx(AHL_MidLetter),
	}
	for _, a := range ahletter {
		ahlOverrides[a] = segmenter.Keep
	}
	rules = append(rules, segmenter.OverrideRule{
		States:    p(AHL_MidLetter),
		Overrides: ahlOverrides,
		WipeValue: segmenter.NoMatch,
	})
	hlOverrides := map[uint8]uint8{
		uint8(Extend): idx(HL_MidLetter),
		uint8(Format): idx(HL_MidLetter),
		uint8(ZWJ):    idx(HL_MidLetter),
	}
	for _, a := range ahletter {
		hlOverrides[a] = segmenter.Keep
	}
	rules = append(rules, segmenter.OverrideRule{
		States:    p(HL_MidLetter),
		Overrides: hlOverrides,
		WipeValue: segmenter.NoMatch,
	})

	// WB7a: Hebrew_Letter × Single_Quote
	rules = append(rules, segmenter.SimpleRule{Left: hebrewPlusZWJ, Right: p(uint8(Single_Quote)), Break: false})
	rules = append(rules, segmenter.ChainRule{
		Entry: hebrewPlusZWJ,
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(Single_Quote)), State: AHL_MidLetter, Interm: segmenter.IntermTrue},
		},
		Interm: true,
	})

	// WB7b: Hebrew_Letter × Double_Quote Hebrew_Letter
	rules = append(rules, segmenter.ChainRule{
		Entry: hebrewPlusZWJ,
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(Double_Quote)), State: HL_DQ},
		},
	})

	// WB7c: Hebrew_Letter Double_Quote × Hebrew_Letter
	rules = append(rules, segmenter.OverrideRule{
		States: p(HL_DQ),
		Overrides: map[uint8]uint8{
			uint8(Hebrew_Letter): segmenter.Keep,
			uint8(Extend):        idx(HL_DQ),
			uint8(Format):        idx(HL_DQ),
			uint8(ZWJ):           idx(HL_DQ),
		},
		WipeValue: segmenter.NoMatch,
	})

	// WB8: Numeric × Numeric
	rules = append(rules, segmenter.SimpleRule{Left: numericPlusZWJ, Right: p(uint8(Numeric)), Break: false})

	// WB9: AHLetter × Numeric
	rules = append(rules, segmenter.SimpleRule{Left: ahletterPlusZWJ, Right: p(uint8(Numeric)), Break: false})

	// WB10: Numeric × AHLetter
	rules = append(rules, segmenter.SimpleRule{Left: numericPlusZWJ, Right: ahletter, Break: false})

	// WB11: Numeric (MidNum | MidNumLetQ) × Numeric
	rules = append(rules, segmenter.OverrideRule{
		States: p(Num_MidNum),
		Overrides: map[uint8]uint8{
			uint8(Numeric): segmenter.Keep,
			uint8(Extend):  idx(Num_MidNum),
			uint8(Format):  idx(Num_MidNum),
			uint8(ZWJ):     idx(Num_MidNum),
		},
		WipeValue: segmenter.NoMatch,
	})

	// WB12: Numeric × (MidNum | MidNumLetQ) Numeric
	rules = append(rules, segmenter.ChainRule{
		Entry: numericPlusZWJ,
		Steps: []segmenter.ChainStep{
			{Props: midNumQ, State: Num_MidNum},
		},
	})

	// WB13: Katakana × Katakana
	rules = append(rules, segmenter.SimpleRule{Left: katakanaPlusZWJ, Right: p(uint8(Katakana)), Break: false})

	// WB13a: (AHLetter | Numeric | Katakana | ExtendNumLet) × ExtendNumLet
	rules = append(rules, segmenter.SimpleRule{
		Left:  wb13aLeft,
		Right: p(uint8(ExtendNumLet)), Break: false,
	})

	// WB13b: ExtendNumLet × (AHLetter | Numeric | Katakana)
	rules = append(rules, segmenter.SimpleRule{
		Left:  extNumLetPlusZWJ,
		Right: p(uint8(ALetter), uint8(Hebrew_Letter), uint8(ALetter_Extended_Pictographic), uint8(Numeric), uint8(Katakana)),
		Break: false,
	})

	// WB15: sot (RI RI)* RI × RI
	// WB16: [^RI] (RI RI)* RI × RI
	rules = append(rules, segmenter.SimpleRule{Left: riPlusZWJ, Right: p(uint8(Regional_Indicator)), Break: false})
	rules = append(rules, segmenter.SimpleRule{Left: p(RI_RI), Right: p(uint8(Regional_Indicator)), Break: true})
	rules = append(rules, segmenter.ChainRule{
		Entry: riPlusZWJ,
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(Regional_Indicator)), State: RI_RI},
		},
	})
	rules = append(rules, segmenter.IgnoreRule{
		Props:   p(RI_RI),
		Ignored: wb4Ignored,
		Target:  func(base, _ uint8) uint8 { return base },
	})

	// WB999: Any ÷ Any
	rules = append(rules, segmenter.SimpleRule{Break: true})

	return rules
}
