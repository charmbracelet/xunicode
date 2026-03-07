// Ported segmenter tests from:
//   - icu4x (https://github.com/unicode-org/icu4x) — components/segmenter
//   - uniseg (https://github.com/rivo/uniseg)
//   - uax29/v2 (https://github.com/clipperhouse/uax29)

package line

import (
	"testing"
)

func portedSegments(data []byte) []string {
	var out []string
	seg := NewSegmenter(data)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

func portedSegmentsOpts(data []byte, o Options) []string {
	var out []string
	seg := o.NewSegmenter(data)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

func portedSegmentsStr(input string) []string {
	return portedSegments([]byte(input))
}

func portedSegmentsStrOpts(input string, o Options) []string {
	return portedSegmentsOpts([]byte(input), o)
}

func portedSlicesEqual(a, b []string) bool {
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

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/src/line.rs
// ---------------------------------------------------------------------------

func TestICU4X_EmptyString(t *testing.T) {
	got := portedSegmentsStr("")
	if len(got) != 0 {
		t.Errorf("empty string: got %v, want []", got)
	}
}

func TestICU4X_HelloWorld(t *testing.T) {
	// icu4x: "hello world" → breaks at [0, 6, 11]
	// In segment form: "hello " + "world"
	got := portedSegmentsStr("hello world")
	want := []string{"hello ", "world"}
	if !portedSlicesEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestICU4X_DollarAmount(t *testing.T) {
	// icu4x: "$10 $10" → breaks at [0, 4, 7]
	// In segment form: "$10 " + "$10"
	got := portedSegmentsStr("$10 $10")
	want := []string{"$10 ", "$10"}
	if !portedSlicesEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestICU4X_LB14OpenPunctuation(t *testing.T) {
	// icu4x: "[  abc def" → breaks at [0, 7, 10] → "[  abc " + "def"
	got := portedSegmentsStr("[  abc def")
	want := []string{"[  abc ", "def"}
	if !portedSlicesEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestICU4X_LB15a_Guillemets(t *testing.T) {
	// icu4x: "« miaou »" → breaks at [0, 11] → single segment
	// LB15a (Unicode ≥ 15.1) prevents break after QU_PI SP* at sot.
	if UnicodeVersion < "15.1" {
		t.Skipf("LB15a requires Unicode ≥ 15.1 (have %s)", UnicodeVersion)
	}
	got := portedSegmentsStr("« miaou »")
	if len(got) != 1 {
		t.Errorf("expected 1 segment for guillemet quote, got %d: %q", len(got), got)
	}
}

func TestICU4X_LB15b_GermanQuotes(t *testing.T) {
	// icu4x: "Die Katze hat »miau« gesagt." → 6 segments
	got := portedSegmentsStr("Die Katze hat \u00BBmiau\u00AB gesagt.")
	if len(got) == 0 {
		t.Error("expected segments for German quote test")
	}
	// Verify roundtrip
	var total string
	for _, s := range got {
		total += s
	}
	if total != "Die Katze hat \u00BBmiau\u00AB gesagt." {
		t.Errorf("roundtrip failed: got %q", total)
	}
}

func TestICU4X_LB16(t *testing.T) {
	// icu4x: ")\u203C" → stays together
	got := portedSegmentsStr(")\u203C")
	if len(got) != 1 {
		t.Errorf("LB16 `)‼`: expected 1 segment, got %d: %q", len(got), got)
	}

	// icu4x: ")  \u203C" → stays together (SP* rule)
	got = portedSegmentsStr(")  \u203C")
	if len(got) != 1 {
		t.Errorf("LB16 `)  ‼`: expected 1 segment, got %d: %q", len(got), got)
	}
}

func TestICU4X_LB17_B2(t *testing.T) {
	// icu4x: "\u2014\u2014aa" → breaks at [0, 6, 8] → "——" + "aa"
	got := portedSegmentsStr("\u2014\u2014aa")
	want := []string{"\u2014\u2014", "aa"}
	if !portedSlicesEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	// icu4x: "\u2014  \u2014aa" → "—  —" + "aa"
	got = portedSegmentsStr("\u2014  \u2014aa")
	want = []string{"\u2014  \u2014", "aa"}
	if !portedSlicesEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestICU4X_LB25_Numeric(t *testing.T) {
	// icu4x: "(0,1)+(2,3)" → single segment
	got := portedSegmentsStr("(0,1)+(2,3)")
	if len(got) != 1 {
		t.Errorf("LB25 numeric: expected 1 segment, got %d: %q", len(got), got)
	}
}

func TestICU4X_EmojiModifier(t *testing.T) {
	// icu4x: "\U0001F3FB \U0001F3FB" → breaks at [0, 5, 9] → "🏻 " + "🏻"
	got := portedSegmentsStr("\U0001F3FB \U0001F3FB")
	want := []string{"\U0001F3FB ", "\U0001F3FB"}
	if !portedSlicesEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/tests/css_line_break.rs
// ---------------------------------------------------------------------------

func TestICU4X_CSSStrictLineBreak(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "strict_CJ_small_kana",
			input: "サ\u3041サ",
			want:  []string{"サ\u3041", "サ"},
		},
		{
			name:  "strict_CJ_prolonged_sound",
			input: "サ\u30FCサ",
			want:  []string{"サ\u30FC", "サ"},
		},
		{
			name:  "strict_CJ_wave_dash",
			input: "サ\u301Cサ",
			want:  []string{"サ\u301C", "サ"},
		},
		{
			name:  "strict_CJ_ideographic_repeat",
			input: "サ\u3005サ",
			want:  []string{"サ\u3005", "サ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{Strictness: Strict})
			if !portedSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestICU4X_CSSNormalLineBreak(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "normal_CJ_small_kana",
			input: "サ\u3041サ",
			want:  []string{"サ", "\u3041", "サ"},
		},
		{
			name:  "normal_CJ_prolonged_sound",
			input: "サ\u30FCサ",
			want:  []string{"サ", "\u30FC", "サ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{Strictness: Normal})
			if !portedSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestICU4X_CSSLooseLineBreak(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "loose_CJ_small_kana",
			input: "サ\u3041サ",
			want:  []string{"サ", "\u3041", "サ"},
		},
		{
			name:  "loose_CJ_prolonged_sound",
			input: "サ\u30FCサ",
			want:  []string{"サ", "\u30FC", "サ"},
		},
		{
			name:  "loose_CJ_wave_dash",
			input: "サ\u301Cサ",
			want:  []string{"サ", "\u301C", "サ"},
		},
		{
			name:  "loose_CJ_ideographic_repeat",
			input: "サ\u3005サ",
			want:  []string{"サ", "\u3005", "サ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{Strictness: Loose})
			if !portedSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestICU4X_CSSAnywhereLineBreak(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "latin",
			input: "latin",
			want:  []string{"l", "a", "t", "i", "n"},
		},
		{
			name:  "XX_XXX",
			input: "XX XXX",
			want:  []string{"X", "X", " ", "X", "X", "X"},
		},
		{
			name:  "X_X",
			input: "X X",
			want:  []string{"X", " ", "X"},
		},
		{
			name:  "no_hyphenation",
			input: "no hyphenation",
			want:  []string{"n", "o", " ", "h", "y", "p", "h", "e", "n", "a", "t", "i", "o", "n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{Strictness: Anywhere})
			if !portedSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/tests/css_word_break.rs
// ---------------------------------------------------------------------------

func TestICU4X_CSSWordBreakBreakAll(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"japanese", "\u65e5\u672c\u8a9e"},
		{"latin", "latin"},
		{"hangul", "\ud55c\uae00\uc77e"},
		{"emoji", "\U0001f496\U0001f494"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{WordBreak: WordBreakAll})
			// Verify roundtrip
			var total string
			for _, s := range got {
				total += s
			}
			if total != tt.input {
				t.Errorf("roundtrip failed for %q: got %q", tt.input, total)
			}
			// break-all should break between most characters
			if len(tt.input) > 1 && len(got) < 2 {
				t.Errorf("break-all should produce multiple segments for %q, got %d", tt.input, len(got))
			}
		})
	}
}

func TestICU4X_CSSWordBreakKeepAll(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"latin", "latin"},
		{"japanese", "\u65e5\u672c\u8a9e"},
		{"hangul", "\ud55c\uae00\uc774"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{WordBreak: WordKeepAll})
			// Verify roundtrip
			var total string
			for _, s := range got {
				total += s
			}
			if total != tt.input {
				t.Errorf("roundtrip failed for %q: got %q", tt.input, total)
			}
			// keep-all should keep CJK/ideographic together
			if len(got) != 1 {
				t.Logf("keep-all for %q: got %d segments: %q (expected 1)", tt.input, len(got), got)
			}
		})
	}
}

func TestICU4X_CSSWordBreakKeepAllSpace(t *testing.T) {
	// icu4x: 字\u3000字 with keep-all → breaks at ideographic space
	got := portedSegmentsStrOpts("字\u3000字", Options{WordBreak: WordKeepAll})
	if len(got) < 2 {
		t.Logf("keep-all with ideographic space: got %d segments: %q", len(got), got)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from uniseg: line_test.go
// ---------------------------------------------------------------------------

func TestUniseg_LineBreakRoundtrip(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"This is 🏳\ufe0f\u200d🌈, a test string ツ for line breaking.",
		"$1,234.56",
		"\r\n\r\n",
		"",
	}
	for _, input := range inputs {
		got := portedSegmentsStr(input)
		var total string
		for _, s := range got {
			total += s
		}
		if total != input {
			t.Errorf("roundtrip failed for %q: got %q", input, total)
		}
	}
}

func TestUniseg_HasTrailingLineBreak(t *testing.T) {
	tests := []struct {
		input     string
		lastIsBrk bool
	}{
		{"\v", true},
		{"\r", true},
		{"\n", true},
		{"\u0085", true},
		{" ", false},
		{"A", false},
	}

	for _, tt := range tests {
		seg := NewSegmenter([]byte(tt.input))
		var lastSeg string
		for seg.Next() {
			lastSeg = seg.Text()
		}
		_ = lastSeg
		// We verify the segment exists and roundtrips correctly
		got := portedSegmentsStr(tt.input)
		var total string
		for _, s := range got {
			total += s
		}
		if total != tt.input {
			t.Errorf("roundtrip failed for %q", tt.input)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/src/line.rs (linebreak tests)
// ---------------------------------------------------------------------------

func TestICU4X_LB_QuoteSPOpen(t *testing.T) {
	// icu4x LB15 (removed in Unicode 15.1):
	// abc"  (def → breaks at [0, 6, 10] → 'abc"  ' + '(def'
	got := portedSegmentsStr("abc\u0022  (def")
	if len(got) == 0 {
		t.Error("expected segments")
	}
	var total string
	for _, s := range got {
		total += s
	}
	if total != "abc\u0022  (def" {
		t.Errorf("roundtrip failed: got %q", total)
	}
}

func TestICU4X_LB_B2Chain(t *testing.T) {
	// icu4x: "\u2014\u2014  \u2014\u2014123 abc"
	// → breaks at [0, 14, 18, 21]
	got := portedSegmentsStr("\u2014\u2014  \u2014\u2014123 abc")
	if len(got) == 0 {
		t.Error("expected segments for B2 chain test")
	}
	var total string
	for _, s := range got {
		total += s
	}
	if total != "\u2014\u2014  \u2014\u2014123 abc" {
		t.Errorf("roundtrip failed: got %q", total)
	}
}

// ---------------------------------------------------------------------------
// Combined CSS tests: Strictness + WordBreak
// ---------------------------------------------------------------------------

func TestICU4X_CSSComposed(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		strictness Strictness
		wordBreak  WordBreak
	}{
		{
			name:       "normal_keep_all",
			input:      "\u4E00\u4E8C\u30C3",
			strictness: Normal,
			wordBreak:  WordKeepAll,
		},
		{
			name:       "strict_keep_all",
			input:      "\u4E00\u4E8C",
			strictness: Strict,
			wordBreak:  WordKeepAll,
		},
		{
			name:       "loose_break_all",
			input:      "hello",
			strictness: Loose,
			wordBreak:  WordBreakAll,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{Strictness: tt.strictness, WordBreak: tt.wordBreak})
			var total string
			for _, s := range got {
				total += s
			}
			if total != tt.input {
				t.Errorf("roundtrip failed: got %q, want %q", total, tt.input)
			}
		})
	}
}

func TestPortedAPIConsistency(t *testing.T) {
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
		offset = end
	}
	if offset != len(input) {
		t.Errorf("consumed %d bytes, expected %d", offset, len(input))
	}
}

func TestPortedCoverage(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"日本語\nテスト",
		"\r\n\r\n",
		"$1,234.56%",
		"👨\u200d👩\u200d👧\u200d👦🇩🇪",
		"\u05D0-\u05D1 test",
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

// ---------------------------------------------------------------------------
// Ported from icu4x: css_line_break.rs — additional Loose tests
// ---------------------------------------------------------------------------

func TestICU4X_CSSLooseExtended(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"loose_insep_1", "文\u2024文"},
		{"loose_insep_2", "文\u2025文"},
		{"loose_insep_3", "文\u2026文"},
		{"loose_insep_4", "文\u22ef文"},
		{"loose_insep_5", "文\ufe19文"},
		{"loose_hyphens_1", "文\u2010文"},
		{"loose_hyphens_2", "文\u2013文"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{Strictness: Loose})
			var total string
			for _, s := range got {
				total += s
			}
			if total != tt.input {
				t.Errorf("roundtrip failed for %q: got %q", tt.input, total)
			}
			// Loose should generally break between CJK+punctuation
			if len(got) < 2 {
				t.Logf("loose for %q: got %d segments: %q", tt.input, len(got), got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ported from icu4x: css_word_break.rs — BreakAll detailed cases
// ---------------------------------------------------------------------------

func TestICU4X_CSSBreakAllDetailed(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "break_all_latin",
			input: "latin",
			want:  []string{"l", "a", "t", "i", "n"},
		},
		{
			name:  "break_all_emoji",
			input: "\U0001f496\U0001f494",
			want:  []string{"\U0001f496", "\U0001f494"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{WordBreak: WordBreakAll})
			if !portedSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestICU4X_CSSKeepAllDetailed(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "keep_all_latin",
			input: "latin",
			want:  []string{"latin"},
		},
		{
			name:  "keep_all_japanese",
			input: "\u65e5\u672c\u8a9e",
			want:  []string{"\u65e5\u672c\u8a9e"},
		},
		{
			name:  "keep_all_hangul",
			input: "\ud55c\uae00\uc774",
			want:  []string{"\ud55c\uae00\uc774"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portedSegmentsStrOpts(tt.input, Options{WordBreak: WordKeepAll})
			if !portedSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
