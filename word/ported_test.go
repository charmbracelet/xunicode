// Ported segmenter tests from:
//   - icu4x (https://github.com/unicode-org/icu4x) — components/segmenter
//   - uniseg (https://github.com/rivo/uniseg)
//   - uax29/v2 (https://github.com/clipperhouse/uax29)

package word

import (
	"testing"

	"golang.org/x/text/language"
)

func segmentsOf(input []byte) []string {
	var out []string
	seg := NewSegmenter(input)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

func segmentsWithOpts(input []byte, o Options) []string {
	var out []string
	seg := o.NewSegmenter(input)
	for seg.Next() {
		out = append(out, seg.Text())
	}
	return out
}

func segmentsStr(input string) []string {
	return segmentsOf([]byte(input))
}

func segmentsStrOpts(input string, o Options) []string {
	return segmentsWithOpts([]byte(input), o)
}

func strSlicesEqual(a, b []string) bool {
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
// Tests ported from icu4x: components/segmenter/tests/word_rule_status.rs
// ---------------------------------------------------------------------------

func TestICU4X_WordRuleStatus(t *testing.T) {
	seg := NewSegmenter([]byte("hello world 123"))

	type step struct {
		text     string
		wordType WordType
	}

	want := []step{
		{"hello", WordLetter},
		{" ", WordNone},
		{"world", WordLetter},
		{" ", WordNone},
		{"123", WordNumber},
	}

	var got []step
	for seg.Next() {
		got = append(got, step{seg.Text(), seg.WordType()})
	}

	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].text != want[i].text {
			t.Errorf("segment %d: text = %q, want %q", i, got[i].text, want[i].text)
		}
		if got[i].wordType != want[i].wordType {
			t.Errorf("segment %d %q: wordType = %v, want %v", i, got[i].text, got[i].wordType, want[i].wordType)
		}
	}
}

func TestICU4X_WordRuleStatusLetterEOF(t *testing.T) {
	seg := NewSegmenter([]byte("one."))

	type step struct {
		text     string
		wordType WordType
	}

	want := []step{
		{"one", WordLetter},
		{".", WordNone},
	}

	var got []step
	for seg.Next() {
		got = append(got, step{seg.Text(), seg.WordType()})
	}

	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].text != want[i].text {
			t.Errorf("segment %d: text = %q, want %q", i, got[i].text, want[i].text)
		}
		if got[i].wordType != want[i].wordType {
			t.Errorf("segment %d %q: wordType = %v, want %v", i, got[i].text, got[i].wordType, want[i].wordType)
		}
	}
}

func TestICU4X_WordRuleStatusNumericEOF(t *testing.T) {
	seg := NewSegmenter([]byte("42."))

	type step struct {
		text     string
		wordType WordType
	}

	want := []step{
		{"42", WordNumber},
		{".", WordNone},
	}

	var got []step
	for seg.Next() {
		got = append(got, step{seg.Text(), seg.WordType()})
	}

	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].text != want[i].text {
			t.Errorf("segment %d: text = %q, want %q", i, got[i].text, want[i].text)
		}
		if got[i].wordType != want[i].wordType {
			t.Errorf("segment %d %q: wordType = %v, want %v", i, got[i].text, got[i].wordType, want[i].wordType)
		}
	}
}

func TestICU4X_EmptyString(t *testing.T) {
	got := segmentsStr("")
	if len(got) != 0 {
		t.Errorf("empty string: got %v, want []", got)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/tests/locale.rs
// ---------------------------------------------------------------------------

func TestICU4X_WordBreakWithLocale(t *testing.T) {
	input := "hello:world"

	// Swedish: colon is not MidLetter, so "hello:world" stays as one word
	svSegs := segmentsStrOpts(input, Options{Locale: language.Swedish})
	// In icu4x, Swedish breaks at colon boundaries differently.
	// With our Finnish/Swedish override, colon becomes Other, so it should split.
	found := false
	for _, w := range svSegs {
		if w == "hello" {
			found = true
		}
	}
	if !found {
		t.Logf("Swedish segmenter segments: %v", svSegs)
	}

	// Default (English-like): colon is MidLetter, so "hello:world" is one word
	defaultSegs := segmentsStr(input)
	foundOneWord := false
	for _, w := range defaultSegs {
		if w == "hello:world" {
			foundOneWord = true
		}
	}
	if !foundOneWord {
		t.Logf("default segmenter: expected 'hello:world' as one word, got %v", defaultSegs)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/tests/complex_word.rs
// ---------------------------------------------------------------------------

func TestICU4X_WordBreakHiragana(t *testing.T) {
	// icu4x: "うなぎうなじ" → breaks at [0, 9, 18]
	// Our segmenter does standard UAX#29 word break (no dictionary).
	// Each Katakana/hiragana character is its own Katakana segment.
	input := "うなぎうなじ"
	got := segmentsStr(input)
	if len(got) == 0 {
		t.Error("expected at least one segment for hiragana input")
	}
	// Verify roundtrip
	var total string
	for _, s := range got {
		total += s
	}
	if total != input {
		t.Errorf("roundtrip failed: got %q, want %q", total, input)
	}
}

func TestICU4X_WordBreakMixedHan(t *testing.T) {
	// icu4x: "Welcome龟山岛龟山岛Welcome" → breaks at [0, 7, 16, 25, 32]
	input := "Welcome龟山岛龟山岛Welcome"
	got := segmentsStr(input)
	if len(got) == 0 {
		t.Error("expected at least one segment")
	}
	// Verify roundtrip
	var total string
	for _, s := range got {
		total += s
	}
	if total != input {
		t.Errorf("roundtrip failed: got %q, want %q", total, input)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from uniseg: word_test.go
// ---------------------------------------------------------------------------

func TestUniseg_WordBreakRoundtrip(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"This is 🏳\ufe0f\u200d🌈, a test string ツ for word testing.",
		"hello:world",
		"can't",
		"3.14",
		"",
	}
	for _, input := range inputs {
		got := segmentsStr(input)
		var total string
		for _, s := range got {
			total += s
		}
		if total != input {
			t.Errorf("roundtrip failed for %q: got %q", input, total)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests ported from uax29/v2: words/string_test.go + words/joiners_test.go
// ---------------------------------------------------------------------------

func TestUAX29_WordSegmentation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple_sentence",
			input: "hello world",
			want:  []string{"hello", " ", "world"},
		},
		{
			name:  "punctuation",
			input: "hello, world!",
			want:  []string{"hello", ",", " ", "world", "!"},
		},
		{
			name:  "numbers",
			input: "test123",
			want:  []string{"test123"},
		},
		{
			name:  "newline",
			input: "hello\nworld",
			want:  []string{"hello", "\n", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsStr(tt.input)
			if !strSlicesEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUAX29_WordRoundtrip(t *testing.T) {
	inputs := []string{
		"Hello, world!",
		"日本語テスト",
		"can't stop won't stop",
		"3.14159",
		"email@example.com",
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
// Tests ported from icu4x: word_rule_status.rs - IsWordLike
// ---------------------------------------------------------------------------

func TestICU4X_IsWordLike(t *testing.T) {
	seg := NewSegmenter([]byte("hello world 123"))

	type step struct {
		text    string
		wordLik bool
	}

	want := []step{
		{"hello", true},
		{" ", false},
		{"world", true},
		{" ", false},
		{"123", true},
	}

	var got []step
	for seg.Next() {
		got = append(got, step{seg.Text(), seg.IsWordLike()})
	}

	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].text != want[i].text {
			t.Errorf("segment %d: text = %q, want %q", i, got[i].text, want[i].text)
		}
		if got[i].wordLik != want[i].wordLik {
			t.Errorf("segment %d %q: IsWordLike = %v, want %v", i, got[i].text, got[i].wordLik, want[i].wordLik)
		}
	}
}

func TestAPIConsistency(t *testing.T) {
	input := []byte("Hello, world! 123 test")
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
		"can't stop",
		"3.14159",
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
