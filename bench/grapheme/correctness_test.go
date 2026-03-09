package bench

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/xunicode/grapheme"
	ugraphemes "github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/rivo/uniseg"
)

type graphemeResult struct {
	start, end int
	bytes      []byte
	text       string
}

func extractXTextGraphemes(data []byte) []graphemeResult {
	var segs []graphemeResult
	seg := grapheme.NewSegmenter(data)
	for seg.Next() {
		start, end := seg.Position()
		segs = append(segs, graphemeResult{
			start: start,
			end:   end,
			bytes: seg.Bytes(),
			text:  seg.Text(),
		})
	}
	return segs
}

func extractUnisegGraphemes(data []byte) []graphemeResult {
	var segs []graphemeResult
	rest := data
	state := -1
	pos := 0
	for len(rest) > 0 {
		var cluster []byte
		cluster, rest, _, state = uniseg.Step(rest, state)
		start := pos
		end := pos + len(cluster)
		segs = append(segs, graphemeResult{
			start: start,
			end:   end,
			bytes: cluster,
			text:  string(cluster),
		})
		pos = end
	}
	return segs
}

func extractUax29Graphemes(data []byte) []graphemeResult {
	var segs []graphemeResult
	it := ugraphemes.FromBytes(data)
	for it.Next() {
		segs = append(segs, graphemeResult{
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
			text := corpora[name]
			data := []byte(text)

			xt := extractXTextGraphemes(data)
			uni := extractUnisegGraphemes(data)
			uax := extractUax29Graphemes(data)

			// Compare counts
			if len(xt) != len(uni) {
				t.Errorf("x_text: %d graphemes, uniseg: %d graphemes", len(xt), len(uni))
			}
			if len(xt) != len(uax) {
				t.Errorf("x_text: %d graphemes, uax29: %d graphemes", len(xt), len(uax))
			}

			// Compare boundaries
			minLen := min(len(xt), min(len(uni), len(uax)))
			for i := 0; i < minLen; i++ {
				if xt[i].start != uni[i].start || xt[i].end != uni[i].end {
					t.Errorf("grapheme %d: x_text=[%d,%d), uniseg=[%d,%d)",
						i, xt[i].start, xt[i].end, uni[i].start, uni[i].end)
				}
				if xt[i].start != uax[i].start || xt[i].end != uax[i].end {
					t.Errorf("grapheme %d: x_text=[%d,%d), uax29=[%d,%d)",
						i, xt[i].start, xt[i].end, uax[i].start, uax[i].end)
				}
				if !bytes.Equal(xt[i].bytes, uni[i].bytes) {
					t.Errorf("grapheme %d: x_text and uniseg content differ: x=%q, uni=%q",
						i, xt[i].text, uni[i].text)
				}
				if !bytes.Equal(xt[i].bytes, uax[i].bytes) {
					t.Errorf("grapheme %d: x_text and uax29 content differ: x=%q, uax=%q",
						i, xt[i].text, uax[i].text)
				}
			}
		})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func icu4xBinary() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "icu4x", "target", "release", "icu4x-grapheme-segment")
}

func extractICU4X(t *testing.T, text string) []graphemeResult {
	t.Helper()

	bin := icu4xBinary()
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("icu4x binary not found at %s (run: cd icu4x && cargo build --release)", bin)
	}

	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("icu4x binary failed: %v", err)
	}

	var segs []graphemeResult
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			t.Fatalf("unexpected icu4x output line: %q", line)
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			t.Fatalf("bad offset %q: %v", parts[0], err)
		}
		raw, err := hex.DecodeString(parts[1])
		if err != nil {
			t.Fatalf("bad hex %q: %v", parts[1], err)
		}
		segs = append(segs, graphemeResult{
			start: start,
			end:   start + len(raw),
			bytes: raw,
			text:  string(raw),
		})
	}
	return segs
}

func countMismatches(a, b []graphemeResult) int {
	minLen := min(len(a), len(b))
	n := 0
	for i := range minLen {
		if a[i].start != b[i].start || a[i].end != b[i].end {
			n++
		}
	}
	n += abs(len(a) - len(b))
	return n
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var threeWayCorpusNames = []string{
	"ASCII", "Latin", "CJK", "Hangul", "Emoji",
	"Arabic", "Devanagari", "Mixed",
}

func TestCorrectnessAllThree(t *testing.T) {
	fmt.Fprintf(os.Stderr, "\n%-12s  %10s %10s %10s  %s  %s\n",
		"Corpus", "x/text", "uniseg", "icu4x",
		"x/text≠uniseg", "x/text≠icu4x")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 80))

	for _, name := range threeWayCorpusNames {
		text := corpora[name]
		data := []byte(text)

		xt := extractXTextGraphemes(data)
		uni := extractUnisegGraphemes(data)
		icu := extractICU4X(t, text)

		xuMismatch := countMismatches(xt, uni)
		xiMismatch := countMismatches(xt, icu)

		fmt.Fprintf(os.Stderr, "%-12s  %10d %10d %10d  %13d  %12d\n",
			name, len(xt), len(uni), len(icu), xuMismatch, xiMismatch)
	}
	fmt.Fprintln(os.Stderr)
}
