// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"bytes"
	"strings"
	"testing"

	"github.com/blevesearch/segment"
	uwords "github.com/clipperhouse/uax29/v2/words"
	scuax29 "github.com/SCKelemen/unicode/uax29"
	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/word"
)

// Corpora covering different script and complexity profiles for word
// segmentation. Each corpus is sized to amortize per-call overhead while
// remaining representative of real workloads.
var corpora = map[string]string{
	"ASCII":      strings.Repeat("The quick brown fox jumps over the lazy dog. ", 200),
	"Latin":      strings.Repeat("Résumé naïve café über straße Ångström. ", 200),
	"CJK":        strings.Repeat("天地玄黃宇宙洪荒日月盈昃辰宿列張。", 200),
	"Hangul":     strings.Repeat("한국어 텍스트 분할 테스트입니다. ", 200),
	"Emoji":      strings.Repeat("👨‍👩‍👧‍👦👍🏽🇩🇪🏳️‍🌈🧑‍💻👩‍❤️‍👨 ", 100),
	"Arabic":     strings.Repeat("بِسْمِ ٱللَّهِ ٱلرَّحْمَـٰنِ ٱلرَّحِيمِ ", 200),
	"Devanagari": strings.Repeat("हिन्दी पाठ विभाजन परीक्षण है। ", 200),
	"Mixed":      strings.Repeat("Hello 世界! 🇺🇸 café 한국 हिन्दी عربي 42. ", 200),
	"Numbers":    strings.Repeat("3.14 1,000 $99.99 +1-555-0123 2024/01/15 ", 200),
	"Email":      strings.Repeat("user.name+tag@example.co.uk wrote: re: subject ", 200),
}

var corpusNames = []string{
	"ASCII", "Latin", "CJK", "Hangul", "Emoji",
	"Arabic", "Devanagari", "Mixed", "Numbers", "Email",
}

// ---------------------------------------------------------------------------
// Segment: iterate over all word segments, consume bytes.
// ---------------------------------------------------------------------------

func BenchmarkSegment(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_FirstWord", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var w []byte
					w, rest, state = uniseg.FirstWord(rest, state)
					_ = w
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := uwords.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Count: count word segments.
// ---------------------------------------------------------------------------

func BenchmarkCount(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := word.NewSegmenter(data)
				for seg.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/uniseg_FirstWord", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				rest := data
				state := -1
				for len(rest) > 0 {
					_, rest, state = uniseg.FirstWord(rest, state)
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				it := uwords.FromBytes(data)
				for it.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = len(scuax29.Words(text))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CountWords: count only word-like segments (letters/numbers).
// This is what most applications actually need (search indexing, word
// counting, etc.). Tests the overhead of word type classification.
// ---------------------------------------------------------------------------

func BenchmarkCountWords(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := word.NewSegmenter(data)
				for seg.Next() {
					if seg.IsWordLike() {
						n++
					}
				}
				_ = n
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					if seg.Type() != segment.None {
						n++
					}
				}
				_ = n
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				// SCKelemen doesn't provide word type filtering
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Position: iterate and record start/end byte offsets.
// ---------------------------------------------------------------------------

func BenchmarkPosition(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					s, e := seg.Position()
					_, _ = s, e
				}
			}
		})

		b.Run(name+"/uniseg_FirstWord", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				pos := 0
				for len(rest) > 0 {
					var w []byte
					w, rest, state = uniseg.FirstWord(rest, state)
					start := pos
					pos += len(w)
					_, _ = start, pos
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := uwords.FromBytes(data)
				for it.Next() {
					_, _ = it.Start(), it.End()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Text: iterate and extract string representation.
// ---------------------------------------------------------------------------

func BenchmarkText(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Text()
				}
			}
		})

		b.Run(name+"/uniseg_FirstWordInString", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := text
				state := -1
				for len(rest) > 0 {
					var w string
					w, rest, state = uniseg.FirstWordInString(rest, state)
					_ = w
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := uwords.FromString(text)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					_ = seg.Text()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Allocs: measure allocations per full iteration.
// ---------------------------------------------------------------------------

func BenchmarkAllocs(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_FirstWord", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var w []byte
					w, rest, state = uniseg.FirstWord(rest, state)
					_ = w
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				it := uwords.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Short: overhead-dominated benchmark for small inputs.
// ---------------------------------------------------------------------------

func BenchmarkShort(b *testing.B) {
	inputs := []struct {
		name string
		text string
	}{
		{"Empty", ""},
		{"1_ASCII", "A"},
		{"Word_Hello", "Hello"},
		{"Word_Number", "3.14"},
		{"Apostrophe", "don't"},
		{"Email_Short", "a@b.c"},
		{"Emoji_Flag", "🇩🇪"},
		{"CJK_Word", "中文"},
		{"Mixed_Short", "café42"},
	}

	for _, tc := range inputs {
		data := []byte(tc.text)

		b.Run(tc.name+"/x_text", func(b *testing.B) {
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(tc.name+"/uniseg_FirstWord", func(b *testing.B) {
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var w []byte
					w, rest, state = uniseg.FirstWord(rest, state)
					_ = w
				}
			}
		})

		b.Run(tc.name+"/uax29", func(b *testing.B) {
			for range b.N {
				it := uwords.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(tc.name+"/bleve", func(b *testing.B) {
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(tc.name+"/sckelemen_uax29", func(b *testing.B) {
			for range b.N {
				_ = scuax29.Words(tc.text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Throughput: raw throughput on large single-script inputs (1MB+).
// ---------------------------------------------------------------------------

func BenchmarkThroughput(b *testing.B) {
	large := map[string][]byte{
		"ASCII_1MB": []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 22000)),
		"CJK_1MB":   []byte(strings.Repeat("天地玄黃宇宙洪荒日月盈昃辰宿列張。", 20000)),
		"Mixed_1MB": []byte(strings.Repeat("Hello 世界! café 한국 42 हिन्दी عربي 🇺🇸 ", 15000)),
	}
	largeNames := []string{"ASCII_1MB", "CJK_1MB", "Mixed_1MB"}

	for _, name := range largeNames {
		data := large[name]
		text := string(data)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
				}
			}
		})

		b.Run(name+"/uniseg_FirstWord", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					_, rest, state = uniseg.FirstWord(rest, state)
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := uwords.FromBytes(data)
				for it.Next() {
				}
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scanner: compare bleve's bufio.Scanner-based API (io.Reader) with
// x/text's direct byte slice approach.
// ---------------------------------------------------------------------------

func BenchmarkScanner(b *testing.B) {
	for _, name := range corpusNames {
		data := []byte(corpora[name])
		text := corpora[name]

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/bleve_direct", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/bleve_reader", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := segment.NewWordSegmenter(bytes.NewReader(data))
				for seg.Segment() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Words(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WordType: compare word type classification between x/text and bleve.
// Both provide word type info; uniseg and uax29 do not.
// ---------------------------------------------------------------------------

func BenchmarkWordType(b *testing.B) {
	for _, name := range corpusNames {
		data := []byte(corpora[name])
		text := corpora[name]

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := word.NewSegmenter(data)
				for seg.Next() {
					_ = seg.WordType()
				}
			}
		})

		b.Run(name+"/bleve", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := segment.NewWordSegmenterDirect(data)
				for seg.Segment() {
					_ = seg.Type()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				// SCKelemen doesn't provide word type classification
				_ = scuax29.Words(text)
			}
		})
	}
}
