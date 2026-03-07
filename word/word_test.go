package word

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/xunicode/internal/gen"
	"github.com/charmbracelet/xunicode/internal/testtext"
	"golang.org/x/text/language"
)

func TestConformance(t *testing.T) {
	testtext.SkipIfNotLong(t)

	f := gen.OpenUCDFile("auxiliary/WordBreakTest.txt")
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

func TestFinnishWordBreak(t *testing.T) {
	input := []byte("EU:ssa on jäseniä")

	seg := NewSegmenter(input)
	var defaultWords []string
	for seg.Next() {
		defaultWords = append(defaultWords, seg.Text())
	}

	finnishOpts := Options{Locale: language.Finnish}
	seg = finnishOpts.NewSegmenter(input)
	var finnishWords []string
	for seg.Next() {
		finnishWords = append(finnishWords, seg.Text())
	}

	hasColon := false
	for _, w := range defaultWords {
		if w == "EU:ssa" {
			hasColon = true
		}
	}
	if !hasColon {
		t.Errorf("default segmenter: expected 'EU:ssa' as one word, got %v", defaultWords)
	}

	for _, w := range finnishWords {
		if w == "EU:ssa" {
			t.Errorf("Finnish segmenter: 'EU:ssa' should be split at colon, got %v", finnishWords)
			break
		}
	}

	hasEU := false
	for _, w := range finnishWords {
		if w == "EU" {
			hasEU = true
		}
	}
	if !hasEU {
		t.Errorf("Finnish segmenter: expected 'EU' as separate word, got %v", finnishWords)
	}
}

func TestSwedishWordBreak(t *testing.T) {
	input := []byte("kl:n")

	swedishOpts := Options{Locale: language.Swedish}
	seg := swedishOpts.NewSegmenter(input)
	var words []string
	for seg.Next() {
		words = append(words, seg.Text())
	}

	for _, w := range words {
		if w == "kl:n" {
			t.Errorf("Swedish segmenter: 'kl:n' should be split at colon, got %v", words)
			break
		}
	}
}
