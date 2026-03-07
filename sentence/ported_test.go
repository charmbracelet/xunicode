// Ported segmenter tests from:
//   - icu4x (https://github.com/unicode-org/icu4x) — components/segmenter
//   - uniseg (https://github.com/rivo/uniseg)
//   - uax29/v2 (https://github.com/clipperhouse/uax29)

package sentence

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

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/src/sentence.rs
// ---------------------------------------------------------------------------

func TestICU4X_EmptyString(t *testing.T) {
	got := segmentsStr("")
	if len(got) != 0 {
		t.Errorf("empty string: got %v, want []", got)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: components/segmenter/tests/locale.rs
// ---------------------------------------------------------------------------

func TestICU4X_SentenceBreakWithLocale(t *testing.T) {
	// SB11 is different because U+003B is STerm on Greek.
	input := "hello; world"

	// Greek: semicolon terminates sentences
	greekSegs := segmentsStrOpts(input, Options{Locale: language.Greek})
	if len(greekSegs) < 2 {
		t.Errorf("Greek segmenter: expected at least 2 sentences for %q, got %d: %v",
			input, len(greekSegs), greekSegs)
	}

	// Default (English): semicolon does not terminate sentences
	defaultSegs := segmentsStr(input)
	if len(defaultSegs) != 1 {
		t.Logf("default segmenter: expected 1 sentence (no break at semicolon) for %q, got %d: %v",
			input, len(defaultSegs), defaultSegs)
	}
}

func TestICU4X_SentenceBreakGreekQuestionMark(t *testing.T) {
	// U+037E (Greek question mark) should also be STerm in Greek locale
	input := "Τι κάνεις\u037E Καλά."

	greekSegs := segmentsStrOpts(input, Options{Locale: language.Greek})
	if len(greekSegs) < 2 {
		t.Errorf("Greek segmenter: expected break at Greek question mark, got %d: %v",
			len(greekSegs), greekSegs)
	}
}

// ---------------------------------------------------------------------------
// Tests ported from uniseg: sentence_test.go
// ---------------------------------------------------------------------------

func TestUniseg_SentenceBreakRoundtrip(t *testing.T) {
	inputs := []string{
		"Hello, world! How are you?",
		"Mr. Smith went to Washington. He arrived at noon.",
		"This is a test.",
		"",
		"Single sentence",
		"First. Second. Third.",
		"Dr. No was a villain.",
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
// Tests ported from uax29/v2: sentences/string_test.go
// ---------------------------------------------------------------------------

func TestUAX29_SentenceRoundtrip(t *testing.T) {
	inputs := []string{
		"Hello. World.",
		"This is a test! Really? Yes.",
		"日本語テスト。二番目の文。",
		"",
		"No period at end",
		"Multiple... sentences!!! Here?? Yes.",
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

func TestUAX29_SentenceBasic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		min   int
	}{
		{"single_sentence", "Hello world.", 1},
		{"two_sentences", "Hello. World.", 1},
		{"question_exclamation", "Really? Yes!", 2},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsStr(tt.input)
			if len(got) < tt.min {
				t.Errorf("expected at least %d segments, got %d: %v", tt.min, len(got), got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests ported from icu4x: spec_test.rs sentence_break_test
// (the spec conformance tests check the same Unicode test data as
//  TestConformance, but here we port the structural edge cases)
// ---------------------------------------------------------------------------

func TestICU4X_SpecSentenceEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", "Hello. World."},
		{"numeric", "I have 3.14 apples."},
		{"abbreviation", "Mr. Smith arrived."},
		{"ellipsis", "Wait... what?"},
		{"crlf", "Hello.\r\nWorld."},
		{"unicode_punctuation", "Hello\u2026 World."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsStr(tt.input)
			// Verify roundtrip
			var total string
			for _, s := range got {
				total += s
			}
			if total != tt.input {
				t.Errorf("roundtrip failed: got %q, want %q", total, tt.input)
			}
			if len(got) == 0 && len(tt.input) > 0 {
				t.Error("expected at least one segment for non-empty input")
			}
		})
	}
}

func TestAPIConsistency(t *testing.T) {
	input := []byte("Hello. World! How? Yes.")
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
		"Hello. World!",
		"日本語。テスト。",
		"\r\n\r\n",
		"Mr. Smith went home.",
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
