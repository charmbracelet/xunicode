package bench

import (
	"strings"
	"testing"

	"charm.land/xunicode/grapheme"
	scuax29 "github.com/SCKelemen/unicode/uax29"
	ugraphemes "github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/rivo/uniseg"
)

// Corpora covering different script and complexity profiles.
var corpora = map[string]string{
	"ASCII":       strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100),
	"Latin":       strings.Repeat("Ré\u0301sume\u0301 naïve café über straße Ångström. ", 100),
	"CJK":         strings.Repeat("天地玄黃宇宙洪荒日月盈昃辰宿列張。", 100),
	"Hangul":      strings.Repeat("한국어 텍스트 분할 테스트입니다. ", 100),
	"Emoji":       strings.Repeat("👨‍👩‍👧‍👦👍🏽🇩🇪🏳️‍🌈🧑‍💻👩‍❤️‍👨", 50),
	"Arabic":      strings.Repeat("بِسْمِ ٱللَّهِ ٱلرَّحْمَـٰنِ ٱلرَّحِيمِ ", 100),
	"Devanagari":  strings.Repeat("हिन्दी पाठ विभाजन परीक्षण है। ", 100),
	"Mixed":       strings.Repeat("Hello 世界! 🇺🇸 café 한국 हिन्दी عربي ", 100),
	"TwoChar":     "ab",
	"SingleEmoji": "👨‍👩‍👧‍👦",
}

var corpusNames = []string{
	"ASCII", "Latin", "CJK", "Hangul", "Emoji",
	"Arabic", "Devanagari", "Mixed", "TwoChar", "SingleEmoji",
}

// ---------------------------------------------------------------------------
// Segment iteration: iterate over all grapheme clusters, consume bytes.
// ---------------------------------------------------------------------------

func BenchmarkSegment(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := grapheme.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_Graphemes", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				g := uniseg.NewGraphemes(text)
				for g.Next() {
					_ = g.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_Step", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var cluster []byte
					cluster, rest, _, state = uniseg.Step(rest, state)
					_ = cluster
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := ugraphemes.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Graphemes(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Count: count grapheme clusters.
// ---------------------------------------------------------------------------

func BenchmarkCount(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := grapheme.NewSegmenter(data)
				for seg.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/uniseg", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = uniseg.GraphemeClusterCount(text)
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				it := ugraphemes.FromBytes(data)
				for it.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = len(scuax29.Graphemes(text))
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
				seg := grapheme.NewSegmenter(data)
				for seg.Next() {
					s, e := seg.Position()
					_, _ = s, e
				}
			}
		})

		b.Run(name+"/uniseg_Step", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				pos := 0
				for len(rest) > 0 {
					var cluster []byte
					cluster, rest, _, state = uniseg.Step(rest, state)
					start := pos
					pos += len(cluster)
					_, _ = start, pos
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := ugraphemes.FromBytes(data)
				for it.Next() {
					_, _ = it.Start(), it.End()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Graphemes(text)
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
				seg := grapheme.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Text()
				}
			}
		})

		b.Run(name+"/uniseg_Graphemes", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				g := uniseg.NewGraphemes(text)
				for g.Next() {
					_ = g.Str()
				}
			}
		})

		b.Run(name+"/uniseg_StepString", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := text
				state := -1
				for len(rest) > 0 {
					var cluster string
					cluster, rest, _, state = uniseg.StepString(rest, state)
					_ = cluster
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := ugraphemes.FromString(text)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Graphemes(text)
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
				seg := grapheme.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_Step", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var cluster []byte
					cluster, rest, _, state = uniseg.Step(rest, state)
					_ = cluster
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				it := ugraphemes.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = scuax29.Graphemes(text)
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
		{"3_ASCII", "Dog"},
		{"Emoji_Flag", "🇩🇪"},
		{"Emoji_Family", "👨‍👩‍👧‍👦"},
		{"Devanagari_Cluster", "हिन्"},
	}

	for _, tc := range inputs {
		data := []byte(tc.text)

		b.Run(tc.name+"/x_text", func(b *testing.B) {
			for range b.N {
				seg := grapheme.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(tc.name+"/uniseg_Step", func(b *testing.B) {
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var cluster []byte
					cluster, rest, _, state = uniseg.Step(rest, state)
					_ = cluster
				}
			}
		})

		b.Run(tc.name+"/uax29", func(b *testing.B) {
			for range b.N {
				it := ugraphemes.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(tc.name+"/sckelemen_uax29", func(b *testing.B) {
			for range b.N {
				_ = scuax29.Graphemes(tc.text)
			}
		})
	}
}
