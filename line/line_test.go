package line

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/xunicode/internal/gen"
	"charm.land/xunicode/internal/testtext"
)

func TestConformance(t *testing.T) {
	testtext.SkipIfNotLong(t)

	f := gen.OpenUCDFile("auxiliary/LineBreakTest.txt")
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	pass, fail := 0, 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		input, wantSegments := parseTestLine(t, lineNum, line)
		if input == nil {
			continue
		}

		var gotSegments []string
		seg := NewSegmenter(input)
		for seg.Next() {
			gotSegments = append(gotSegments, seg.Text())
		}

		if len(gotSegments) != len(wantSegments) {
			fail++
			t.Errorf("line %d: %s\ngot  %d segments: %v\nwant %d segments: %v",
				lineNum, line, len(gotSegments), fmtSegments(gotSegments),
				len(wantSegments), fmtSegments(wantSegments))
			continue
		}

		ok := true
		for i := range wantSegments {
			if gotSegments[i] != wantSegments[i] {
				ok = false
				break
			}
		}
		if !ok {
			fail++
			t.Errorf("line %d: %s\ngot  %v\nwant %v",
				lineNum, line, fmtSegments(gotSegments), fmtSegments(wantSegments))
		} else {
			pass++
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d tests passed, %d failed", pass, fail)
}

// TestMandatoryBreaks verifies mandatory line break rules (LB4, LB5).
func TestMandatoryBreaks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"BK", "a\x0Bb", []string{"a\x0B", "b"}},
		{"CR_LF", "a\r\nb", []string{"a\r\n", "b"}},
		{"CR_alone", "a\rb", []string{"a\r", "b"}},
		{"LF_alone", "a\nb", []string{"a\n", "b"}},
		{"NL", "a\u0085b", []string{"a\u0085", "b"}},
		{"multiple_LF", "a\n\nb", []string{"a\n", "\n", "b"}},
		{"CR_LF_no_split", "\r\n", []string{"\r\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSpaceHandling verifies LB7, LB14, LB18, and space-related chain rules.
func TestSpaceHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"SP_break", "a b", []string{"a ", "b"}},
		{"multiple_SP", "a   b", []string{"a   ", "b"}},
		{"ZW_break", "a\u200Bb", []string{"a\u200B", "b"}},
		{"ZW_SP_break", "a\u200B b", []string{"a\u200B ", "b"}},
		{"OP_SP_keep", "( a", []string{"( a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWordJoiner verifies LB11: × WJ, WJ ×.
func TestWordJoiner(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"WJ_keeps", "a\u2060b", []string{"a\u2060b"}},
		{"WJ_both_sides", "a\u2060 b", []string{"a\u2060 ", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGlue verifies LB12: GL × and LB12a: [^SP BA HY] × GL.
func TestGlue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"GL_keeps_right", "a\u00A0b", []string{"a\u00A0b"}},
		{"SP_GL_breaks", " \u00A0b", []string{" ", "\u00A0b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClosePunctuation verifies LB13 and close punctuation behavior.
func TestClosePunctuation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"no_break_before_CL", "a)", []string{"a)"}},
		{"no_break_before_EX", "a!", []string{"a!"}},
		{"no_break_before_IS", "a.", []string{"a."}},
		{"no_break_before_SY", "a/", []string{"a/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestQuotation verifies LB19: × QU, QU × and LB15 QU SP* × OP.
func TestQuotation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"QU_keeps_both_sides", "a\"b", []string{"a\"b"}},
		{"QU_SP_OP", "\" (a", []string{"\" (a"}},
		{"PI_keeps", "a\u00ABb", []string{"a\u00ABb"}},
		{"PF_keeps", "a\u00BBb", []string{"a\u00BBb"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNumericContext verifies tailored LB25.
func TestNumericContext(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple_number", "123", []string{"123"}},
		{"number_comma", "1,234", []string{"1,234"}},
		{"number_period", "3.14", []string{"3.14"}},
		{"prefix_number", "$123", []string{"$123"}},
		{"number_postfix", "100%", []string{"100%"}},
		{"prefix_op_number", "$-1", []string{"$-1"}},
		{"num_close_postfix", "(1)%", []string{"(1)%"}},
		{"chained_numeric", "(0,1)+(2,3)", []string{"(0,1)+(2,3)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHangul verifies LB26/LB27.
func TestHangul(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"LV_LV_break", "\uAC00\uAC00", []string{"\uAC00", "\uAC00"}},
		{"LVT_LVT_break", "\uAC01\uAC01", []string{"\uAC01", "\uAC01"}},
		{"JL_JV_keep", "\u1100\u1161", []string{"\u1100\u1161"}},
		{"JV_JT_keep", "\u1161\u11A8", []string{"\u1161\u11A8"}},
		{"hangul_break_AL", "\uAC00a", []string{"\uAC00", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRegionalIndicator verifies LB30a: RI × RI pairing.
func TestRegionalIndicator(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"one_pair", "\U0001F1E9\U0001F1EA", []string{"\U0001F1E9\U0001F1EA"}},
		{
			"two_pairs", "\U0001F1E9\U0001F1EA\U0001F1FA\U0001F1F8",
			[]string{"\U0001F1E9\U0001F1EA", "\U0001F1FA\U0001F1F8"},
		},
		{
			"three_RI", "\U0001F1E9\U0001F1EA\U0001F1FA",
			[]string{"\U0001F1E9\U0001F1EA", "\U0001F1FA"},
		},
		{
			"RI_pair_then_EB", "\U0001F1E9\U0001F1EA\U0001F3F3",
			[]string{"\U0001F1E9\U0001F1EA", "\U0001F3F3"},
		},
		{
			"RI_pair_then_AL", "\U0001F1E9\U0001F1EAa",
			[]string{"\U0001F1E9\U0001F1EA", "a"},
		},
		{
			"RI_pair_then_ID", "\U0001F1E9\U0001F1EA\u4E00",
			[]string{"\U0001F1E9\U0001F1EA", "\u4E00"},
		},
		{
			"EB_EM_RI_pair", "\U0001F44D\U0001F3FD\U0001F1E9\U0001F1EA",
			[]string{"\U0001F44D\U0001F3FD", "\U0001F1E9\U0001F1EA"},
		},
		{
			"RI_pair_then_rainbow_flag", "\U0001F1E9\U0001F1EA\U0001F3F3\uFE0F\u200D\U0001F308",
			[]string{"\U0001F1E9\U0001F1EA", "\U0001F3F3\uFE0F\u200D\U0001F308"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEmojiBase verifies LB30b: EB × EM.
func TestEmojiBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"EB_EM", "\U0001F466\U0001F3FB", []string{"\U0001F466\U0001F3FB"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHebrewLetter verifies LB21a: HL (HY|BA) × and LB21b: SY × HL.
func TestHebrewLetter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"HL_HY_keep", "\u05D0-\u05D1", []string{"\u05D0-\u05D1"}},
		{"SY_HL", "/\u05D0", []string{"/\u05D0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLB9Absorption verifies that Extend/ZWJ are absorbed (LB9).
func TestLB9Absorption(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"AL_extend", "a\u0300b", []string{"a\u0300b"}},
		{"AL_ZWJ", "a\u200Db", []string{"a\u200Db"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEmojiZWJ verifies that emoji ZWJ sequences are not broken across lines.
//
// Both icu4x and uniseg keep entire ZWJ sequences together by respecting
// extended grapheme cluster boundaries during line breaking. We should match.
func TestEmojiZWJ(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"family", "👨\u200d👩\u200d👧\u200d👦", []string{"👨\u200d👩\u200d👧\u200d👦"}},
		{"technologist", "🧑\u200d💻", []string{"🧑\u200d💻"}},
		{"couple_with_heart", "👩\u200d❤\ufe0f\u200d👨", []string{"👩\u200d❤\ufe0f\u200d👨"}},
		{"rainbow_flag", "🏳\ufe0f\u200d🌈", []string{"🏳\ufe0f\u200d🌈"}},
		{
			"two_families", "👨\u200d👩\u200d👧\u200d👦👨\u200d👩\u200d👧\u200d👦",
			[]string{"👨\u200d👩\u200d👧\u200d👦", "👨\u200d👩\u200d👧\u200d👦"},
		},
		{
			"family_then_space_text", "👨\u200d👩\u200d👧\u200d👦 hello",
			[]string{"👨\u200d👩\u200d👧\u200d👦 ", "hello"},
		},
		{
			"emoji_zwj_space_emoji_zwj", "🧑\u200d💻 🧑\u200d💻",
			[]string{"🧑\u200d💻 ", "🧑\u200d💻"},
		},
		{"thumbsup_skin_tone", "👍🏽", []string{"👍🏽"}},
		{"AL_ZWJ_AL_no_break", "a\u200db", []string{"a\u200db"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got  %v\nwant %v", fmtSegments(got), fmtSegments(tt.want))
			}
		})
	}
}

// TestCJK verifies ideographic break opportunities (LB31 default break).
func TestCJK(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"ID_breaks", "\u4E00\u4E8C\u4E09", []string{"\u4E00", "\u4E8C", "\u4E09"}},
		{"ID_no_break_before_CL", "\u4E00)", []string{"\u4E00)"}},
		{"AL_ID_break", "a\u4E00", []string{"a", "\u4E00"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestContingentBreak verifies LB20: ÷ CB, CB ÷.
func TestContingentBreak(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"CB_break_both", "a\uFFFCb", []string{"a", "\uFFFC", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEdgeCases covers boundary conditions.
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single_char", "a", []string{"a"}},
		{"single_SP", " ", []string{" "}},
		{"single_LF", "\n", []string{"\n"}},
		{"only_spaces", "   ", []string{"   "}},
		{"only_newlines", "\n\n\n", []string{"\n", "\n", "\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAPIConsistency verifies that Bytes/Text/Position are consistent.
func TestAPIConsistency(t *testing.T) {
	input := []byte("Hello, world!\nNew line.\r\n日本語テスト")
	seg := NewSegmenter(input)
	offset := 0
	for seg.Next() {
		b := seg.Bytes()
		text := seg.Text()
		start, end := seg.Position()

		if string(b) != text {
			t.Errorf("Bytes/Text mismatch at offset %d: %q vs %q", offset, b, text)
		}
		if start != offset {
			t.Errorf("start=%d, expected %d", start, offset)
		}
		if end != offset+len(b) {
			t.Errorf("end=%d, expected %d", end, offset+len(b))
		}
		if end-start != len(b) {
			t.Errorf("position length %d != bytes length %d", end-start, len(b))
		}
		offset = end
	}
	if offset != len(input) {
		t.Errorf("consumed %d bytes, expected %d", offset, len(input))
	}
}

// TestCoverage ensures all input bytes are covered by segments (no gaps or overlaps).
func TestCoverage(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"日本語\nテスト",
		"\r\n\r\n",
		"$1,234.56%",
		"👨‍👩‍👧‍👦🇩🇪",
		"\u05D0-\u05D1 test",
		"(a+b)×(c−d)",
		"",
	}
	for _, s := range inputs {
		data := []byte(s)
		seg := NewSegmenter(data)
		var total int
		prev := 0
		for seg.Next() {
			start, end := seg.Position()
			if start != prev {
				t.Errorf("gap at %d–%d in %q", prev, start, s)
			}
			total += end - start
			prev = end
		}
		if total != len(data) {
			t.Errorf("total segment bytes %d != input length %d for %q", total, len(data), s)
		}
	}
}

// TestLB30 verifies LB30: (AL|HL|NU) × OP (non-EA), CP (non-EA) × (AL|HL|NU).
func TestLB30(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"AL_OP", "a(b", []string{"a(b"}},
		{"CP_AL", ")a", []string{")a"}},
		{"NU_OP", "1(2", []string{"1(2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestB2 verifies LB17: B2 SP* × B2.
func TestB2(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"B2_B2", "\u2014\u2014", []string{"\u2014\u2014"}},
		{"B2_SP_B2", "\u2014 \u2014", []string{"\u2014 \u2014"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segments([]byte(tt.input))
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func segments(data []byte) []string {
	var out []string
	seg := NewSegmenter(data)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

func segmentsOpts(data []byte, o Options) []string {
	var out []string
	seg := o.NewSegmenter(data)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

// TestCSSStrictness verifies the CSS line-break strictness levels.
// CJ codepoints (e.g. small kana) are treated as NS under Strict (default)
// but as ID under Normal/Loose, allowing breaks before them.
func TestCSSStrictness(t *testing.T) {
	// U+30C3 (ッ) is a CJ codepoint (Katakana small tsu).
	// U+4E00 (一) is ID. Under Strict, CJ acts as NS (no break before).
	// Under Normal/Loose, CJ acts as ID (break before allowed).

	tests := []struct {
		name       string
		input      string
		strictness Strictness
		want       []string
	}{
		{
			name:       "strict_CJ_no_break",
			input:      "\u4E00\u30C3",
			strictness: Strict,
			want:       []string{"\u4E00\u30C3"},
		},
		{
			name:       "normal_CJ_break",
			input:      "\u4E00\u30C3",
			strictness: Normal,
			want:       []string{"\u4E00", "\u30C3"},
		},
		{
			name:       "loose_CJ_break",
			input:      "\u4E00\u30C3",
			strictness: Loose,
			want:       []string{"\u4E00", "\u30C3"},
		},
		{
			name:       "strict_multiple_CJ",
			input:      "\u4E00\u30C3\u30C3",
			strictness: Strict,
			want:       []string{"\u4E00\u30C3\u30C3"},
		},
		{
			name:       "normal_multiple_CJ",
			input:      "\u4E00\u30C3\u30C3",
			strictness: Normal,
			want:       []string{"\u4E00", "\u30C3", "\u30C3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsOpts([]byte(tt.input), Options{Strictness: tt.strictness})
			if !slicesEqual(got, tt.want) {
				t.Errorf("got  %v\nwant %v", fmtSegments(got), fmtSegments(tt.want))
			}
		})
	}
}

// TestCSSWordBreak verifies the CSS word-break property.
func TestCSSWordBreak(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wordBreak WordBreak
		want      []string
	}{
		{
			name:      "normal_AL_no_break",
			input:     "hello",
			wordBreak: WordNormal,
			want:      []string{"hello"},
		},
		{
			name:      "break_all_AL_becomes_ID",
			input:     "he",
			wordBreak: WordBreakAll,
			want:      []string{"h", "e"},
		},
		{
			name:      "break_all_latin_ideograph",
			input:     "a\u4E00",
			wordBreak: WordBreakAll,
			want:      []string{"a", "\u4E00"},
		},
		{
			name:      "keep_all_ID_no_break",
			input:     "\u4E00\u4E8C",
			wordBreak: WordKeepAll,
			want:      []string{"\u4E00\u4E8C"},
		},
		{
			name:      "keep_all_ID_three",
			input:     "\u4E00\u4E8C\u4E09",
			wordBreak: WordKeepAll,
			want:      []string{"\u4E00\u4E8C\u4E09"},
		},
		{
			name:      "normal_ID_breaks",
			input:     "\u4E00\u4E8C\u4E09",
			wordBreak: WordNormal,
			want:      []string{"\u4E00", "\u4E8C", "\u4E09"},
		},
		{
			name:      "keep_all_CJ_no_break",
			input:     "\u4E00\u30C3",
			wordBreak: WordKeepAll,
			want:      []string{"\u4E00\u30C3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsOpts([]byte(tt.input), Options{WordBreak: tt.wordBreak})
			if !slicesEqual(got, tt.want) {
				t.Errorf("got  %v\nwant %v", fmtSegments(got), fmtSegments(tt.want))
			}
		})
	}
}

// TestCSSComposed verifies combining Strictness and WordBreak options.
func TestCSSComposed(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		strictness Strictness
		wordBreak  WordBreak
		want       []string
	}{
		{
			name:       "normal_keep_all",
			input:      "\u4E00\u4E8C\u30C3",
			strictness: Normal,
			wordBreak:  WordKeepAll,
			want:       []string{"\u4E00\u4E8C\u30C3"},
		},
		{
			name:       "strict_keep_all",
			input:      "\u4E00\u4E8C",
			strictness: Strict,
			wordBreak:  WordKeepAll,
			want:       []string{"\u4E00\u4E8C"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsOpts([]byte(tt.input), Options{Strictness: tt.strictness, WordBreak: tt.wordBreak})
			if !slicesEqual(got, tt.want) {
				t.Errorf("got  %v\nwant %v", fmtSegments(got), fmtSegments(tt.want))
			}
		})
	}
}

// TestCSSAnywhere verifies that line-break: anywhere breaks after every
// extended grapheme cluster (typographic character unit).
func TestCSSAnywhere(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "ascii_letters",
			input: "abc",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "ideographs",
			input: "\u4E00\u4E8C\u4E09",
			want:  []string{"\u4E00", "\u4E8C", "\u4E09"},
		},
		{
			name:  "mixed_latin_cjk",
			input: "a\u4E00b",
			want:  []string{"a", "\u4E00", "b"},
		},
		{
			name:  "space_breaks",
			input: "a b",
			want:  []string{"a", " ", "b"},
		},
		{
			name:  "grapheme_cluster_preserved",
			input: "e\u0301",
			want:  []string{"e\u0301"},
		},
		{
			name:  "emoji_zwj_sequence",
			input: "\U0001F468\u200D\U0001F469\u200D\U0001F467",
			want:  []string{"\U0001F468\u200D\U0001F469\u200D\U0001F467"},
		},
		{
			name:  "regional_indicator_pair",
			input: "\U0001F1FA\U0001F1F8",
			want:  []string{"\U0001F1FA\U0001F1F8"},
		},
		{
			name:  "crlf_kept_together",
			input: "a\r\nb",
			want:  []string{"a", "\r\n", "b"},
		},
		{
			name:  "lf_alone",
			input: "a\nb",
			want:  []string{"a", "\n", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsOpts([]byte(tt.input), Options{Strictness: Anywhere})
			if !slicesEqual(got, tt.want) {
				t.Errorf("got  %v\nwant %v", fmtSegments(got), fmtSegments(tt.want))
			}
		})
	}
}

// TestCSSDefaultUnchanged verifies that default (Strict + WordNormal) produces
// the same results as calling NewSegmenter without options.
func TestCSSDefaultUnchanged(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"\u4E00\u4E8C\u4E09",
		"$1,234.56%",
		"\u05D0-\u05D1",
		"\r\n\r\n",
	}
	for _, s := range inputs {
		data := []byte(s)
		got := segmentsOpts(data, Options{Strictness: Strict, WordBreak: WordNormal})
		want := segments(data)
		if !slicesEqual(got, want) {
			t.Errorf("input %q: explicit defaults differ from no-opts\ngot  %v\nwant %v",
				s, fmtSegments(got), fmtSegments(want))
		}
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func parseTestLine(t *testing.T, lineNum int, line string) (input []byte, segments []string) {
	t.Helper()

	const (
		brk   = "÷"
		nobrk = "×"
	)

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil
	}

	var buf []byte
	var currentSeg []byte

	for _, f := range fields {
		switch f {
		case brk:
			if currentSeg != nil {
				segments = append(segments, string(currentSeg))
				currentSeg = nil
			}
		case nobrk:
		default:
			cp, err := strconv.ParseUint(f, 16, 32)
			if err != nil {
				t.Errorf("line %d: bad codepoint %q: %v", lineNum, f, err)
				return nil, nil
			}
			var enc [utf8.UTFMax]byte
			n := utf8.EncodeRune(enc[:], rune(cp))
			buf = append(buf, enc[:n]...)
			currentSeg = append(currentSeg, enc[:n]...)
		}
	}
	if currentSeg != nil {
		segments = append(segments, string(currentSeg))
	}

	return buf, segments
}

func fmtSegments(segs []string) []string {
	out := make([]string, len(segs))
	for i, s := range segs {
		var runes []string
		for _, r := range s {
			runes = append(runes, fmt.Sprintf("%04X", r))
		}
		out[i] = "[" + strings.Join(runes, " ") + "]"
	}
	return out
}
