package grapheme

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/xunicode/internal/gen"
	"github.com/charmbracelet/xunicode/internal/testtext"
)

func TestConformance(t *testing.T) {
	testtext.SkipIfNotLong(t)

	f := gen.OpenUCDFile("auxiliary/GraphemeBreakTest.txt")
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	pass, fail := 0, 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Strip comments.
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

// parseTestLine parses a GraphemeBreakTest.txt line into a byte input and
// expected segments. The format is:
//
//	÷ 0020 × 0308 ÷ 0020 ÷
//
// where ÷ indicates a break and × indicates no break. Each hex value is a
// Unicode codepoint.
func parseTestLine(t *testing.T, lineNum int, line string) (input []byte, segments []string) {
	t.Helper()

	// The ÷ character is U+00F7, encoded as 0xC3 0xB7 in UTF-8.
	// The × character is U+00D7, encoded as 0xC3 0x97 in UTF-8.
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
			// No break — continue accumulating into the current segment.
		default:
			// Hex codepoint.
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
	// The last ÷ closes the final segment.
	if currentSeg != nil {
		segments = append(segments, string(currentSeg))
	}

	return buf, segments
}

func TestProperties(t *testing.T) {
	tests := []struct {
		name  string
		input string
		class Class
	}{
		{"ASCII", "A", Other},
		{"CJK Wide", "世", Other},
		{"Ambiguous", "§", Other},
		{"Base+Combining", "e\u0301", Other},
		{"CR", "\r", CR},
		{"LF", "\n", LF},
		{"DEL", "0x7f", Other},
		{"Control 0", "\u0000", Control},
		{"Control 1", "\u009b", Control},
		{"Regional Indicator pair", "\U0001F1FA\U0001F1F8", Regional_Indicator},
		{"Extended Pictographic", "\U0001F600", Extended_Pictographic},
		{"Hangul LV", "\uAC00", LV},
		{"CJK followed by ASCII", "世A", Other},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := LookupString(tt.input)
			if got := p.Class(); got != tt.class {
				t.Errorf("Class() = %d, want %d", got, tt.class)
			}
		})
	}
}

// fmtSegments formats segments as a list of hex-encoded codepoint sequences
// for readable error messages.
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
