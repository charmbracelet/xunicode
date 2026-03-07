// Ported segmenter tests from:
//   - icu4x (https://github.com/unicode-org/icu4x) — components/segmenter
//   - uniseg (https://github.com/rivo/uniseg)
//   - uax29/v2 (https://github.com/clipperhouse/uax29)

package grapheme

import (
	"testing"
)

func segments(input []byte) []string {
	var out []string
	seg := NewSegmenter(input)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

func segmentsStr(input string) []string {
	return segments([]byte(input))
}

func runeSegments(input string) [][]rune {
	var out [][]rune
	seg := NewSegmenter([]byte(input))
	for seg.Next() {
		out = append(out, []rune(seg.Text()))
	}
	return out
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

func runeSlicesEqual(a, b [][]rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/src/grapheme.rs
// ---------------------------------------------------------------------------

func TestICU4X_EmptyString(t *testing.T) {
	got := segmentsStr("")
	if len(got) != 0 {
		t.Errorf("empty string: got %v, want []", got)
	}
}

func TestICU4X_EmojiFlags(t *testing.T) {
	// https://github.com/unicode-org/icu4x/issues/4780
	// 🇺🇸🏴󠁧󠁢󠁥󠁮󠁧󠁿
	input := "\U0001F1FA\U0001F1F8\U0001F3F4\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F"
	got := segmentsStr(input)
	want := []string{
		"\U0001F1FA\U0001F1F8",
		"\U0001F3F4\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F",
	}
	if !slicesEqual(got, want) {
		t.Errorf("emoji flags:\ngot  %q\nwant %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from uniseg: grapheme_test.go
// ---------------------------------------------------------------------------

func TestUniseg_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected [][]rune
	}{
		{"empty", "", nil},
		{"single_char", "x", [][]rune{{0x78}}},
		{"basic", "basic", [][]rune{{0x62}, {0x61}, {0x73}, {0x69}, {0x63}}},
		{"combining", "m\u00f6p", [][]rune{{0x6d}, {0xf6}, {0x70}}},
		{"crlf", "\r\n", [][]rune{{0xd, 0xa}}},
		{"lf_lf", "\n\n", [][]rune{{0xa}, {0xa}}},
		{"tab_star", "\t*", [][]rune{{0x9}, {0x2a}}},
		{"hangul", "\uB874", [][]rune{{0xB874}}},
		{"syriac_prepend", "ܐ܏ܒܓܕ", [][]rune{{0x710}, {0x70f, 0x712}, {0x713}, {0x715}}},
		{"thai_sara_am", "ำ", [][]rune{{0xe33}}},
		{"thai_sara_am_double", "ำำ", [][]rune{{0xe33, 0xe33}}},
		{"thai_vowel", "สระอำ", [][]rune{{0xe2a}, {0xe23}, {0xe30}, {0xe2d, 0xe33}}},
		{"star_hangul_star", "*\uB874*", [][]rune{{0x2a}, {0xB874}, {0x2a}}},
		{"kiss_emoji", "*👩\u200d❤\ufe0f\u200d💋\u200d👩*", [][]rune{
			{0x2a},
			{0x1f469, 0x200d, 0x2764, 0xfe0f, 0x200d, 0x1f48b, 0x200d, 0x1f469},
			{0x2a},
		}},
		{"kiss_emoji_alone", "👩\u200d❤\ufe0f\u200d💋\u200d👩", [][]rune{
			{0x1f469, 0x200d, 0x2764, 0xfe0f, 0x200d, 0x1f48b, 0x200d, 0x1f469},
		}},
		{"weightlifter", "🏋🏽\u200d♀\ufe0f", [][]rune{
			{0x1f3cb, 0x1f3fd, 0x200d, 0x2640, 0xfe0f},
		}},
		{"smiley", "🙂", [][]rune{{0x1f642}}},
		{"smiley_pair", "🙂🙂", [][]rune{{0x1f642}, {0x1f642}}},
		{"flag_de", "🇩🇪", [][]rune{{0x1f1e9, 0x1f1ea}}},
		{"rainbow_flag", "🏳\ufe0f\u200d🌈", [][]rune{
			{0x1f3f3, 0xfe0f, 0x200d, 0x1f308},
		}},
		{"tab_rainbow", "\t🏳\ufe0f\u200d🌈", [][]rune{
			{0x9},
			{0x1f3f3, 0xfe0f, 0x200d, 0x1f308},
		}},
		{"tab_rainbow_tab", "\t🏳\ufe0f\u200d🌈\t", [][]rune{
			{0x9},
			{0x1f3f3, 0xfe0f, 0x200d, 0x1f308},
			{0x9},
		}},
		{"crlf_vs", "\r\n\uFE0E", [][]rune{{13, 10}, {0xfe0e}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runeSegments(tt.input)
			if !runeSlicesEqual(got, tt.expected) {
				t.Errorf("input %q:\ngot  %v\nwant %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUniseg_GraphemeClusterCount(t *testing.T) {
	// 🇩🇪🏳️‍🌈 = 2 grapheme clusters
	input := "🇩🇪🏳\ufe0f\u200d🌈"
	got := runeSegments(input)
	if len(got) != 2 {
		t.Errorf("expected 2 grapheme clusters for %q, got %d: %v", input, len(got), got)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from uax29/v2: graphemes/string_test.go
// ---------------------------------------------------------------------------

func TestUAX29_First(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ASCII_start", "héllo world", "h"},
		{"combining_char", "É", "É"},
		{"empty", "", ""},
		{"single_ASCII", "a", "a"},
		{"pure_ASCII", "hello", "h"},
		{"emoji", "🎉 party", "🎉"},
		{"CJK", "日本語", "日"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := NewSegmenter([]byte(tt.input))
			if !seg.Next() {
				if tt.expected != "" {
					t.Errorf("expected %q, got no segments", tt.expected)
				}
				return
			}
			got := seg.Text()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestUAX29_FirstASCIIOptimization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single_printable_ASCII", "a", "a"},
		{"ASCII_followed_by_ASCII", "ab", "a"},
		{"ASCII_space", " hello", " "},
		{"ASCII_digit", "5abc", "5"},
		{"ASCII_punctuation", "!hello", "!"},
		{"ASCII_then_non_ASCII", "a日", "a"},
		{"ASCII_combining_mark", "e\u0301", "e\u0301"},
		{"non_ASCII_start", "日本", "日"},
		{"emoji_grapheme_cluster", "👨\u200d👩\u200d👧\u200d👦 family", "👨\u200d👩\u200d👧\u200d👦"},
		{"flag_emoji", "🇺🇸 USA", "🇺🇸"},
		{"control_char", "\t hello", "\t"},
		{"DEL_char", "\x7Fhello", "\x7F"},
		{"high_ASCII_combining", "n\u0303", "n\u0303"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := NewSegmenter([]byte(tt.input))
			if !seg.Next() {
				if tt.expected != "" {
					t.Errorf("expected %q, got no segments", tt.expected)
				}
				return
			}
			got := seg.Text()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestUAX29_Roundtrip(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"日本語テスト",
		"🏳\ufe0f\u200d🌈🇩🇪",
		"möp",
		"\r\n\r\n",
		"สระอำ",
		"",
	}

	for _, input := range inputs {
		seg := NewSegmenter([]byte(input))
		var output string
		for seg.Next() {
			output += seg.Text()
		}
		if output != input {
			t.Errorf("roundtrip failed for %q: got %q", input, output)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/tests/spec_test.rs
// (grapheme_break_test exercises the same data as TestConformance, but
//  here we port the structural pattern: ensure empty and basic cases work)
// ---------------------------------------------------------------------------

func TestICU4X_SpecBasicGrapheme(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"hello", "Hello", []string{"H", "e", "l", "l", "o"}},
		{"emoji_skin_tone", "👍🏽", []string{"👍🏽"}},
		{"family_zwj", "👨\u200d👩\u200d👧\u200d👦", []string{"👨\u200d👩\u200d👧\u200d👦"}},
		{"flag_pair", "🇩🇪🇫🇷", []string{"🇩🇪", "🇫🇷"}},
		{"three_RI", "\U0001F1E9\U0001F1EA\U0001F1FA", []string{"\U0001F1E9\U0001F1EA", "\U0001F1FA"}},
		{"crlf", "\r\n", []string{"\r\n"}},
		{"cr_alone", "\ra", []string{"\r", "a"}},
		{"lf_alone", "\na", []string{"\n", "a"}},
		{"combining_sequence", "a\u0300\u0301", []string{"a\u0300\u0301"}},
		{"hangul_lv", "\uAC00", []string{"\uAC00"}},
		{"hangul_lvt", "\uAC01", []string{"\uAC01"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsStr(tt.input)
			if !slicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIConsistency(t *testing.T) {
	input := []byte("Hello 🗺 world\r\ntest")
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
		offset = end
	}
	if offset != len(input) {
		t.Errorf("consumed %d bytes, expected %d", offset, len(input))
	}
}

func TestCoverage(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"日本語テスト",
		"\r\n\r\n",
		"👨\u200d👩\u200d👧\u200d👦🇩🇪",
		"möp",
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
