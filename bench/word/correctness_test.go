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

	"charm.land/xunicode/word"
	"github.com/blevesearch/segment"
	uwords "github.com/clipperhouse/uax29/v2/words"
	"github.com/rivo/uniseg"
)

type wordResult struct {
	start, end int
	bytes      []byte
	text       string
}

func extractXTextWords(data []byte) []wordResult {
	var segs []wordResult
	seg := word.NewSegmenter(data)
	for seg.Next() {
		start, end := seg.Position()
		segs = append(segs, wordResult{
			start: start,
			end:   end,
			bytes: seg.Bytes(),
			text:  seg.Text(),
		})
	}
	return segs
}

func extractUnisegWords(data []byte) []wordResult {
	var segs []wordResult
	rest := data
	state := -1
	pos := 0
	for len(rest) > 0 {
		var w []byte
		w, rest, state = uniseg.FirstWord(rest, state)
		start := pos
		end := pos + len(w)
		segs = append(segs, wordResult{
			start: start,
			end:   end,
			bytes: w,
			text:  string(w),
		})
		pos = end
	}
	return segs
}

func extractUax29Words(data []byte) []wordResult {
	var segs []wordResult
	it := uwords.FromBytes(data)
	for it.Next() {
		segs = append(segs, wordResult{
			start: it.Start(),
			end:   it.End(),
			bytes: it.Value(),
			text:  string(it.Value()),
		})
	}
	return segs
}

func extractBleveWords(data []byte) []wordResult {
	var segs []wordResult
	seg := segment.NewWordSegmenterDirect(data)
	pos := 0
	for seg.Segment() {
		token := seg.Bytes()
		start := pos
		end := pos + len(token)
		segs = append(segs, wordResult{
			start: start,
			end:   end,
			bytes: token,
			text:  seg.Text(),
		})
		pos = end
	}
	return segs
}

func TestCorrectness(t *testing.T) {
	for _, name := range corpusNames {
		t.Run(name, func(t *testing.T) {
			data := []byte(corpora[name])

			xt := extractXTextWords(data)
			uni := extractUnisegWords(data)
			uax := extractUax29Words(data)
			bleve := extractBleveWords(data)

			// Compare counts
			if len(xt) != len(uni) {
				t.Errorf("x_text: %d words, uniseg: %d words", len(xt), len(uni))
			}
			if len(xt) != len(uax) {
				t.Errorf("x_text: %d words, uax29: %d words", len(xt), len(uax))
			}
			if len(xt) != len(bleve) {
				t.Errorf("x_text: %d words, bleve: %d words", len(xt), len(bleve))
			}

			// Compare boundaries
			minLen := min(len(xt), min(len(uni), min(len(uax), len(bleve))))
			for i := 0; i < minLen; i++ {
				if xt[i].start != uni[i].start || xt[i].end != uni[i].end {
					t.Errorf("word %d: x_text=[%d,%d), uniseg=[%d,%d)",
						i, xt[i].start, xt[i].end, uni[i].start, uni[i].end)
				}
				if xt[i].start != uax[i].start || xt[i].end != uax[i].end {
					t.Errorf("word %d: x_text=[%d,%d), uax29=[%d,%d)",
						i, xt[i].start, xt[i].end, uax[i].start, uax[i].end)
				}
				if xt[i].start != bleve[i].start || xt[i].end != bleve[i].end {
					t.Errorf("word %d: x_text=[%d,%d), bleve=[%d,%d)",
						i, xt[i].start, xt[i].end, bleve[i].start, bleve[i].end)
				}
				if !bytes.Equal(xt[i].bytes, uni[i].bytes) {
					t.Errorf("word %d: x_text and uniseg content differ: x=%q, uni=%q",
						i, xt[i].text, uni[i].text)
				}
				if !bytes.Equal(xt[i].bytes, uax[i].bytes) {
					t.Errorf("word %d: x_text and uax29 content differ: x=%q, uax=%q",
						i, xt[i].text, uax[i].text)
				}
				if !bytes.Equal(xt[i].bytes, bleve[i].bytes) {
					t.Errorf("word %d: x_text and bleve content differ: x=%q, bleve=%q",
						i, xt[i].text, bleve[i].text)
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
	return filepath.Join(filepath.Dir(file), "icu4x", "target", "release", "icu4x-word-segment")
}

func extractICU4X(t *testing.T, text string) []wordResult {
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

	var segs []wordResult
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
		segs = append(segs, wordResult{
			start: start,
			end:   start + len(raw),
			bytes: raw,
			text:  string(raw),
		})
	}
	return segs
}

func countMismatches(a, b []wordResult) int {
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

func TestCorrectnessAllThree(t *testing.T) {
	fmt.Fprintf(os.Stderr, "\n%-12s  %10s %10s %10s  %s  %s\n",
		"Corpus", "x/text", "uniseg", "icu4x",
		"x/text≠uniseg", "x/text≠icu4x")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 80))

	for _, name := range corpusNames {
		data := []byte(corpora[name])

		xt := extractXTextWords(data)
		uni := extractUnisegWords(data)
		icu := extractICU4X(t, corpora[name])

		xuMismatch := countMismatches(xt, uni)
		xiMismatch := countMismatches(xt, icu)

		fmt.Fprintf(os.Stderr, "%-12s  %10d %10d %10d  %13d  %12d\n",
			name, len(xt), len(uni), len(icu), xuMismatch, xiMismatch)
	}
	fmt.Fprintln(os.Stderr)
}
