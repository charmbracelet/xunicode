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

var gcbMap = map[string]Class{
	"Other":              Other,
	"CR":                 CR,
	"LF":                 LF,
	"Control":            Control,
	"Extend":             Extend,
	"ZWJ":                ZWJ,
	"Regional_Indicator": Regional_Indicator,
	"Prepend":            Prepend,
	"SpacingMark":        SpacingMark,
	"L":                  L,
	"V":                  V,
	"T":                  T,
	"LV":                 LV,
	"LVT":                LVT,
}

func p(v ...uint8) []uint8 {
	return v
}

func main() {
	gen.Init()
	genTables()
}

func genTables() {
	gen.Repackage("gen_trieval.go", "trieval.go", "grapheme")

	props := make([]Class, unicode.MaxRune+1)

	ucd.Parse(gen.OpenUCDFile("auxiliary/GraphemeBreakProperty.txt"), func(p *ucd.Parser) {
		r := p.Rune(0)
		val := p.String(1)
		cls, ok := gcbMap[val]
		if !ok {
			log.Fatalf("U+%04X: unknown Grapheme_Cluster_Break value %q", r, val)
		}
		props[r] = cls
	})

	ucd.Parse(gen.OpenUCDFile("emoji/emoji-data.txt"), func(p *ucd.Parser) {
		if p.String(1) == "Extended_Pictographic" {
			props[p.Rune(0)] = Extended_Pictographic
		}
	})

	ucd.Parse(gen.OpenUCDFile("DerivedCoreProperties.txt"), func(p *ucd.Parser) {
		if p.String(1) != "InCB" {
			return
		}
		r := p.Rune(0)
		switch p.String(2) {
		case "Linker":
			props[r] = InCBLinker
		case "Consonant":
			props[r] = InCBConsonant
		case "Extend":
			if r != 0x200D {
				props[r] = InCBExtend
			}
		}
	})

	w := gen.NewCodeWriter()
	defer w.WriteVersionedGoFile("tables.go", "grapheme")

	fmt.Fprintf(w, "import %q\n\n", "github.com/charmbracelet/xunicode/internal/segmenter")
	gen.WriteUnicodeVersion(w)

	t := triegen.NewTrie("grapheme")
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
	segmenter.WriteBreakTable(w, "ruleData", bt, "graphemeTrie", 0)
}

func buildRules() []segmenter.Rule {
	var rules []segmenter.Rule

	// Rules are listed in spec order (GB1–GB999). The table builder uses
	// first-write-wins, so higher-priority rules (lower numbers) that
	// appear earlier take precedence over lower-priority ones.

	// GB1: sot ÷ (implicit — segmenter starts at position 0)
	// SOT consumes the first character; keep transitions so the first
	// segment isn't artificially split.
	rules = append(rules, segmenter.SimpleRule{Left: p(sot), Right: nil, Break: false})
	// GB2: ÷ eot
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: p(eot), Break: true})

	// GB3: CR × LF
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(CR)), Right: p(uint8(LF)), Break: false})

	// GB4: (Control|CR|LF) ÷
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(Control), uint8(CR), uint8(LF)), Right: nil, Break: true})
	// GB5: ÷ (Control|CR|LF)
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: p(uint8(Control), uint8(CR), uint8(LF)), Break: true})

	// GB6: L × (L|V|LV|LVT)
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(L)), Right: p(uint8(L), uint8(V), uint8(LV), uint8(LVT)), Break: false})
	// GB7: (LV|V) × (V|T)
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(LV), uint8(V)), Right: p(uint8(V), uint8(T)), Break: false})
	// GB8: (LVT|T) × T
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(LVT), uint8(T)), Right: p(uint8(T)), Break: false})

	// GB9: × (Extend|ZWJ|InCBExtend|InCBLinker)
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: p(uint8(Extend), uint8(ZWJ), uint8(InCBExtend), uint8(InCBLinker)), Break: false})
	// GB9a: × SpacingMark
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: p(uint8(SpacingMark)), Break: false})
	// GB9b: Prepend ×
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(Prepend)), Right: nil, Break: false})

	// GB9c: Consonant [{Extend|InCBExtend} {Linker} {Extend|InCBExtend}]+ Consonant
	rules = append(rules, segmenter.SimpleRule{Left: p(InCB_Linker), Right: p(uint8(InCBConsonant)), Break: false})
	rules = append(rules, segmenter.ChainRule{
		Entry: p(uint8(InCBConsonant)),
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(InCBLinker)), State: uint8(InCB_Linker)},
		},
		SelfLoop: p(uint8(InCBExtend), uint8(InCBLinker)),
		Interm:   false,
	})

	// GB11: ExtPict Extend* ZWJ × ExtPict
	rules = append(rules, segmenter.SimpleRule{Left: p(ExtPict_ZWJ), Right: p(uint8(Extended_Pictographic)), Break: false})
	rules = append(rules, segmenter.ChainRule{
		Entry: p(uint8(Extended_Pictographic)),
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(Extend), uint8(InCBExtend)), State: uint8(ExtPict_Ext)},
			{Props: p(uint8(ZWJ)), State: uint8(ExtPict_ZWJ)},
		},
		SelfLoop: p(uint8(Extend), uint8(InCBExtend)),
	})
	rules = append(rules, segmenter.ChainRule{
		Entry: p(uint8(Extended_Pictographic)),
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(ZWJ)), State: uint8(ExtPict_ZWJ)},
		},
	})

	// GB12/13: RI × RI (pair, then break on next RI)
	rules = append(rules, segmenter.SimpleRule{Left: p(uint8(Regional_Indicator)), Right: p(uint8(Regional_Indicator)), Break: false})
	rules = append(rules, segmenter.SimpleRule{Left: p(RI_RI), Right: p(uint8(Regional_Indicator)), Break: true})
	rules = append(rules, segmenter.ChainRule{
		Entry: p(uint8(Regional_Indicator)),
		Steps: []segmenter.ChainStep{
			{Props: p(uint8(Regional_Indicator)), State: uint8(RI_RI)},
		},
	})

	// GB999: Any ÷ Any (default break — lowest priority, last)
	rules = append(rules, segmenter.SimpleRule{Left: nil, Right: nil, Break: true})

	return rules
}
