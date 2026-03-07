// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"bytes"
	"strings"
	"testing"

	usentences "github.com/clipperhouse/uax29/v2/sentences"
	"github.com/rivo/uniseg"
	scuax29 "github.com/SCKelemen/unicode/uax29"
	"golang.org/x/text/unicode/sentence"
)

// Corpora covering different script and complexity profiles for sentence
// segmentation. Each corpus is sized to amortize per-call overhead while
// remaining representative of real workloads.
var corpora = map[string]string{
	"ASCII": strings.Repeat("The quick brown fox jumps over the lazy dog. "+
		"This is a test. Sentences end here! Or do they? ", 200),
	"Latin": strings.Repeat("Résumé naïve café über straße Ångström. "+
		"Testé en français! ¿Qué tal? ", 200),
	"CJK": strings.Repeat("天地玄黃宇宙洪荒日月盈昃辰宿列張。"+
		"中文斷句測試。日文のテスト。 ", 200),
	"Hangul": strings.Repeat("한국어 문장 분할 테스트입니다. "+
		"다른 문장도 테스트합니다! ", 200),
	"Arabic": strings.Repeat("بِسْمِ ٱللَّهِ ٱلرَّحْمَـٰنِ ٱلرَّحِيمِ. "+
		"تجربة تقسيم الجمل! ", 200),
	"Devanagari": strings.Repeat("हिन्दी वाक्य विभाजन परीक्षण है। "+
		"दूसरा वाक्य! ", 200),
	"Mixed": strings.Repeat("Hello 世界! café 한국 42 हिन्दी عربي. "+
		"More mixed text here! ", 200),
	"Abbreviations": strings.Repeat("Dr. Smith lives in the U.S.A. "+
		"and works for 3.5 hrs. at 5 p.m. Mr. Jones, Mrs. Brown, Ms. Lee. "+
		"Ph.D., M.D., and B.S. degrees. ", 100),
	"Terminators": strings.Repeat("End here! Or maybe there? Yes, no. "+
		"How are you? I am fine. Wow! Amazing? ", 200),
	"Numbers": strings.Repeat("The year is 2024. The price is $99.99. "+
		"Section 3.14. Page 1.2.3. Version 5.0. ", 200),
	"Email": strings.Repeat("User@example.com wrote: Please reply ASAP. "+
		"Contact info@example.org for more details! ", 200),
}

var corpusNames = []string{
	"ASCII", "Latin", "CJK", "Hangul", "Arabic",
	"Devanagari", "Mixed", "Abbreviations", "Terminators",
	"Numbers", "Email",
}

// ---------------------------------------------------------------------------
// Segment: iterate over all sentence segments, consume bytes.
// ---------------------------------------------------------------------------

func BenchmarkSegment(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_FirstSentence", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var s []byte
					s, rest, state = uniseg.FirstSentence(rest, state)
					_ = s
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := usentences.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Sentences(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Count: count sentence segments.
// ---------------------------------------------------------------------------

func BenchmarkCount(b *testing.B) {
	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/uniseg_FirstSentence", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				rest := data
				state := -1
				for len(rest) > 0 {
					_, rest, state = uniseg.FirstSentence(rest, state)
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				n := 0
				it := usentences.FromBytes(data)
				for it.Next() {
					n++
				}
				_ = n
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = len(scuax29.Sentences(text))
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
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					s, e := seg.Position()
					_, _ = s, e
				}
			}
		})

		b.Run(name+"/uniseg_FirstSentence", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				pos := 0
				for len(rest) > 0 {
					var s []byte
					s, rest, state = uniseg.FirstSentence(rest, state)
					start := pos
					pos += len(s)
					_, _ = start, pos
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := usentences.FromBytes(data)
				for it.Next() {
					_, _ = it.Start(), it.End()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Sentences(text)
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
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Text()
				}
			}
		})

		b.Run(name+"/uniseg_FirstSentenceInString", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := text
				state := -1
				for len(rest) > 0 {
					var s string
					s, rest, state = uniseg.FirstSentenceInString(rest, state)
					_ = s
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := usentences.FromString(text)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Sentences(text)
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
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uniseg_FirstSentence", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var s []byte
					s, rest, state = uniseg.FirstSentence(rest, state)
					_ = s
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				it := usentences.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_ = scuax29.Sentences(text)
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
		{"Sentence_Short", "Hello!"},
		{"Terminator_Dot", "Yes."},
		{"Terminator_Question", "No?"},
		{"Terminator_Exclaim", "Wow!"},
		{"Abbreviation_Dr", "Dr. Smith"},
		{"CJK_Sentence", "中文。"},
		{"Mixed_Short", "Hello 世界!"},
	}

	for _, tc := range inputs {
		data := []byte(tc.text)

		b.Run(tc.name+"/x_text", func(b *testing.B) {
			for range b.N {
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(tc.name+"/uniseg_FirstSentence", func(b *testing.B) {
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					var s []byte
					s, rest, state = uniseg.FirstSentence(rest, state)
					_ = s
				}
			}
		})

		b.Run(tc.name+"/uax29", func(b *testing.B) {
			for range b.N {
				it := usentences.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(tc.name+"/sckelemen_uax29", func(b *testing.B) {
			for range b.N {
				_ = scuax29.Sentences(tc.text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Throughput: raw throughput on large single-script inputs (1MB+).
// ---------------------------------------------------------------------------

func BenchmarkThroughput(b *testing.B) {
	large := map[string][]byte{
		"ASCII_1MB": []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. "+
			"This is a test sentence with more content. Here is another one! ", 22000)),
		"CJK_1MB": []byte(strings.Repeat("天地玄黃宇宙洪荒日月盈昃辰宿列張。"+
			"中文斷句測試。日文のテストです。 ", 20000)),
		"Mixed_1MB": []byte(strings.Repeat("Hello 世界! café 한국 42 हिन्दी عربي. "+
			"More mixed text content here! ", 15000)),
	}
	largeNames := []string{"ASCII_1MB", "CJK_1MB", "Mixed_1MB"}

	for _, name := range largeNames {
		data := large[name]

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
				}
			}
		})

		b.Run(name+"/uniseg_FirstSentence", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				rest := data
				state := -1
				for len(rest) > 0 {
					_, rest, state = uniseg.FirstSentence(rest, state)
				}
			}
		})

		b.Run(name+"/uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := usentences.FromBytes(data)
				for it.Next() {
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Reader: compare x/text's direct byte slice approach with io.Reader API
// (uax29 supports both).
// ---------------------------------------------------------------------------

func BenchmarkReader(b *testing.B) {
	for _, name := range corpusNames {
		data := []byte(corpora[name])
		text := corpora[name]

		b.Run(name+"/x_text", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				seg := sentence.NewSegmenter(data)
				for seg.Next() {
					_ = seg.Bytes()
				}
			}
		})

		b.Run(name+"/uax29_bytes", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := usentences.FromBytes(data)
				for it.Next() {
					_ = it.Value()
				}
			}
		})

		b.Run(name+"/uax29_reader", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				it := usentences.FromReader(bytes.NewReader(data))
				for it.Scan() {
					_ = it.Bytes()
				}
			}
		})

		b.Run(name+"/sckelemen_uax29", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for range b.N {
				_ = scuax29.Sentences(text)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Correctness: verify all libraries produce identical results.
// ---------------------------------------------------------------------------

type sentResult struct {
	start, end int
	bytes     []byte
	text      string
}

func extractXTextSentences(data []byte) []sentResult {
	var segs []sentResult
	seg := sentence.NewSegmenter(data)
	for seg.Next() {
		start, end := seg.Position()
		segs = append(segs, sentResult{
			start: start,
			end:   end,
			bytes: seg.Bytes(),
			text:  seg.Text(),
		})
	}
	return segs
}

func extractUnisegSentences(data []byte) []sentResult {
	var segs []sentResult
	rest := data
	state := -1
	pos := 0
	for len(rest) > 0 {
		s, r, st := uniseg.FirstSentence(rest, state)
		start := pos
		end := pos + len(s)
		segs = append(segs, sentResult{
			start: start,
			end:   end,
			bytes: s,
			text:  string(s),
		})
		rest = r
		state = st
		pos = end
	}
	return segs
}

func extractUax29Sentences(data []byte) []sentResult {
	var segs []sentResult
	it := usentences.FromBytes(data)
	for it.Next() {
		segs = append(segs, sentResult{
			start: it.Start(),
			end:   it.End(),
			bytes: it.Value(),
			text:  string(it.Value()),
		})
	}
	return segs
}

func TestCorrectness(t *testing.T) {
	for _, name := range corpusNames {
		t.Run(name, func(t *testing.T) {
			data := []byte(corpora[name])

			xt := extractXTextSentences(data)
			uni := extractUnisegSentences(data)
			uax := extractUax29Sentences(data)

			// Compare counts
			if len(xt) != len(uni) {
				t.Errorf("x_text: %d sentences, uniseg: %d sentences", len(xt), len(uni))
			}
			if len(xt) != len(uax) {
				t.Errorf("x_text: %d sentences, uax29: %d sentences", len(xt), len(uax))
			}

			// Compare boundaries
			minLen := min(len(xt), min(len(uni), len(uax)))
			for i := 0; i < minLen; i++ {
				if xt[i].start != uni[i].start || xt[i].end != uni[i].end {
					t.Errorf("sentence %d: x_text=[%d,%d), uniseg=[%d,%d)",
						i, xt[i].start, xt[i].end, uni[i].start, uni[i].end)
				}
				if xt[i].start != uax[i].start || xt[i].end != uax[i].end {
					t.Errorf("sentence %d: x_text=[%d,%d), uax29=[%d,%d)",
						i, xt[i].start, xt[i].end, uax[i].start, uax[i].end)
				}
				if !bytes.Equal(xt[i].bytes, uni[i].bytes) {
					t.Errorf("sentence %d: x_text and uniseg content differ: x=%q, uni=%q",
						i, xt[i].text, uni[i].text)
				}
				if !bytes.Equal(xt[i].bytes, uax[i].bytes) {
					t.Errorf("sentence %d: x_text and uax29 content differ: x=%q, uax=%q",
						i, xt[i].text, uax[i].text)
				}
			}
		})
	}
}
