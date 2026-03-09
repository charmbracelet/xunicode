package bench

import (
	"strings"
	"testing"

	"charm.land/xunicode/line"
	sckelemen_uax14 "github.com/SCKelemen/unicode/uax14"
	"github.com/clipperhouse/uax14"
	"github.com/rivo/uniseg"
)

var corpora = map[string]string{
	"ASCII":      strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100),
	"Latin":      strings.Repeat("Ré\u0301sume\u0301 naïve café über straße Ångström. ", 100),
	"CJK":        strings.Repeat("天地玄黃宇宙洪荒日月盈昃辰宿列張。", 100),
	"Hangul":     strings.Repeat("한국어 텍스트 분할 테스트입니다. ", 100),
	"Emoji":      strings.Repeat("👨‍👩‍👧‍👦👍🏽🇩🇪🏳️‍🌈🧑‍💻👩‍❤️‍👨", 50),
	"Arabic":     strings.Repeat("بِسْمِ ٱللَّهِ ٱلرَّحْمَـٰنِ ٱلرَّحِيمِ ", 100),
	"Devanagari": strings.Repeat("हिन्दी पाठ विभाजन परीक्षण है। ", 100),
	"Mixed":      strings.Repeat("Hello 世界! 🇺🇸 café 한국 हिन्दी عربي ", 100),
	"Numeric":    strings.Repeat("$1,234.56 + $7,890.12 = $9,124.68; ", 100),
	"Code":       strings.Repeat("func foo(x int) { return x + 1 } ", 100),
}

var corpusNames = []string{
	"ASCII", "Latin", "CJK", "Hangul", "Emoji",
	"Arabic", "Devanagari", "Mixed", "Numeric", "Code",
}

// ---------------------------------------------------------------------------
// Segment: iterate over all line break segments, consume bytes.
// ---------------------------------------------------------------------------

func BenchmarkSegment(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := line.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var seg []byte
					var mustBreak bool
					seg, rest, mustBreak, state = uniseg.FirstLineSegment(rest, state)
					_ = seg
					_ = mustBreak
				}
			}
		})

		b.Run(name+"/sckelemen_uax14", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = sckelemen_uax14.FindLineBreakOpportunities(text, sckelemen_uax14.HyphensManual)
			}
		})

		b.Run(name+"/clipperhouse_uax14", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				for len(rest) > 0 {
					adv, _ := uax14.NextBreak(rest)
					if adv == 0 {
						break
					}
					_ = rest[:adv]
					rest = rest[adv:]
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Count: count line break segments.
// ---------------------------------------------------------------------------

func BenchmarkCount(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := line.NewSegmenter(data)
				for seg.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/uniseg", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				rest := data
				state := -1
				for len(rest) > 0 {
					var seg []byte
					seg, rest, _, state = uniseg.FirstLineSegment(rest, state)
					_ = seg
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/sckelemen_uax14", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				breaks := sckelemen_uax14.FindLineBreakOpportunities(text, sckelemen_uax14.HyphensManual)
				_ = len(breaks)
			}
		})

		b.Run(name+"/clipperhouse_uax14", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				rest := data
				for len(rest) > 0 {
					adv, _ := uax14.NextBreak(rest)
					if adv == 0 {
						break
					}
					rest = rest[adv:]
					n++
				}
				_ = n
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Position: iterate and record start/end positions.
// ---------------------------------------------------------------------------

func BenchmarkPosition(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := line.NewSegmenter(data)
				for seg.Next() {
					s, e := seg.Position()
					_, _ = s, e
				}
			}
		})

		b.Run(name+"/uniseg", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				pos := 0
				for len(rest) > 0 {
					var seg []byte
					seg, rest, _, state = uniseg.FirstLineSegment(rest, state)
					start := pos
					pos += len(seg)
					_, _ = start, pos
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Text (string): iterate and extract string representation.
// ---------------------------------------------------------------------------

func BenchmarkText(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := line.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Text()
				}
			}
		})

		b.Run(name+"/uniseg_string", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := text
				state := -1
				for len(rest) > 0 {
					var seg string
					seg, rest, _, state = uniseg.FirstLineSegmentInString(rest, state)
					_ = seg
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Allocation: measure allocations per iteration.
// ---------------------------------------------------------------------------

func BenchmarkAllocs(b *testing.B) {
	for _, name := range corpusNames {
		if name == "TwoChar" || name == "SingleEmoji" {
			continue
		}
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				seg := line.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var seg []byte
					seg, rest, _, state = uniseg.FirstLineSegment(rest, state)
					_ = seg
				}
			}
		})

		b.Run(name+"/sckelemen_uax14", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = sckelemen_uax14.FindLineBreakOpportunities(text, sckelemen_uax14.HyphensManual)
			}
		})

		b.Run(name+"/clipperhouse_uax14", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				rest := data
				for len(rest) > 0 {
					adv, _ := uax14.NextBreak(rest)
					if adv == 0 {
						break
					}
					_ = rest[:adv]
					rest = rest[adv:]
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Short strings: overhead-dominated benchmark.
// ---------------------------------------------------------------------------

func BenchmarkShort(b *testing.B) {
	inputs := []struct {
		name string
		text string
	}{
		{"Empty", ""},
		{"1_ASCII", "A"},
		{"Word", "Hello"},
		{"SP_Break", "a b"},
		{"CJK_Pair", "天地"},
		{"Numeric", "$1.23"},
		{"Newline", "a\nb"},
		{"Emoji_Flag", "🇩🇪"},
	}

	for _, tc := range inputs {
		data := []byte(tc.text)

		b.Run(tc.name+"/x_text", func(b *testing.B) {
			for range b.N {
				seg := line.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(tc.name+"/uniseg", func(b *testing.B) {
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var seg []byte
					seg, rest, _, state = uniseg.FirstLineSegment(rest, state)
					_ = seg
				}
			}
		})

		b.Run(tc.name+"/sckelemen_uax14", func(b *testing.B) {
			for range b.N {
				_ = sckelemen_uax14.FindLineBreakOpportunities(tc.text, sckelemen_uax14.HyphensManual)
			}
		})

		b.Run(tc.name+"/clipperhouse_uax14", func(b *testing.B) {
			for range b.N {
				data := []byte(tc.text)
				rest := data
				for len(rest) > 0 {
					adv, _ := uax14.NextBreak(rest)
					if adv == 0 {
						break
					}
					_ = rest[:adv]
					rest = rest[adv:]
				}
			}
		})
	}
}
