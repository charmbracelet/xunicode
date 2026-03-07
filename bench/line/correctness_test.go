// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bench

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/SCKelemen/unicode/uax14"
	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/line"
)

type segResult struct {
	start, end int
	text       string
}

func extractXText(data []byte) []segResult {
	var segs []segResult
	seg := line.NewSegmenter(data)
	for seg.Next() {
		start, end := seg.Position()
		segs = append(segs, segResult{start: start, end: end, text: seg.Text()})
	}
	return segs
}

func extractUniseg(data []byte) []segResult {
	var segs []segResult
	rest := data
	state := -1
	pos := 0
	for len(rest) > 0 {
		var seg []byte
		seg, rest, _, state = uniseg.FirstLineSegment(rest, state)
		start := pos
		end := pos + len(seg)
		segs = append(segs, segResult{start: start, end: end, text: string(seg)})
		pos = end
	}
	return segs
}

func extractUAX14(text string) []segResult {
	breaks := uax14.FindLineBreakOpportunities(text, uax14.HyphensManual)
	if len(breaks) == 0 {
		if len(text) > 0 {
			return []segResult{{start: 0, end: len(text), text: text}}
		}
		return nil
	}

	var segs []segResult
	prev := 0
	for _, bp := range breaks {
		bytePos := 0
		for i := 0; i < bp && bytePos < len(text); i++ {
			_, size := utf8.DecodeRuneInString(text[bytePos:])
			bytePos += size
		}
		if bytePos > prev {
			segs = append(segs, segResult{start: prev, end: bytePos, text: text[prev:bytePos]})
		}
		prev = bytePos
	}
	if prev < len(text) {
		segs = append(segs, segResult{start: prev, end: len(text), text: text[prev:]})
	}
	return segs
}

func TestCorrectness(t *testing.T) {
	for _, name := range corpusNames {
		t.Run(name, func(t *testing.T) {
			text := corpora[name]
			data := []byte(text)

			xt := extractXText(data)
			uni := extractUniseg(data)

			if len(xt) != len(uni) {
				t.Logf("x_text: %d segments, uniseg: %d segments", len(xt), len(uni))
			}

			minLen := min(len(xt), len(uni))
			mismatches := 0
			for i := 0; i < minLen; i++ {
				if xt[i].start != uni[i].start || xt[i].end != uni[i].end {
					if mismatches < 5 {
						t.Logf("segment %d: x_text=[%d,%d) %q, uniseg=[%d,%d) %q",
							i, xt[i].start, xt[i].end, truncate(xt[i].text, 20),
							uni[i].start, uni[i].end, truncate(uni[i].text, 20))
					}
					mismatches++
				}
			}
			if mismatches > 5 {
				t.Logf("... and %d more mismatches", mismatches-5)
			}
		})
	}
}

func TestCorrectnessUAX14(t *testing.T) {
	for _, name := range corpusNames {
		t.Run(name, func(t *testing.T) {
			text := corpora[name]
			data := []byte(text)

			xt := extractXText(data)
			u14 := extractUAX14(text)

			if len(xt) != len(u14) {
				t.Logf("x_text: %d segments, uax14: %d segments", len(xt), len(u14))
			}

			minLen := min(len(xt), len(u14))
			mismatches := 0
			for i := 0; i < minLen; i++ {
				if xt[i].start != u14[i].start || xt[i].end != u14[i].end {
					if mismatches < 5 {
						t.Logf("segment %d: x_text=[%d,%d) %q, uax14=[%d,%d) %q",
							i, xt[i].start, xt[i].end, truncate(xt[i].text, 20),
							u14[i].start, u14[i].end, truncate(u14[i].text, 20))
					}
					mismatches++
				}
			}
			if mismatches > 5 {
				t.Logf("... and %d more mismatches", mismatches-5)
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
	return filepath.Join(filepath.Dir(file), "icu4x", "target", "release", "icu4x-line-break")
}

func extractICU4X(t *testing.T, text string) []segResult {
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

	var segs []segResult
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
		segs = append(segs, segResult{
			start: start,
			end:   start + len(raw),
			text:  string(raw),
		})
	}
	return segs
}

func TestCorrectnessICU4X(t *testing.T) {
	for _, name := range corpusNames {
		t.Run(name, func(t *testing.T) {
			text := corpora[name]
			data := []byte(text)

			xt := extractXText(data)
			icu := extractICU4X(t, text)

			if len(xt) != len(icu) {
				t.Logf("x_text: %d segments, icu4x: %d segments", len(xt), len(icu))
			}

			minLen := min(len(xt), len(icu))
			mismatches := 0
			for i := range minLen {
				if xt[i].start != icu[i].start || xt[i].end != icu[i].end {
					if mismatches < 5 {
						t.Logf("segment %d: x_text=[%d,%d) %q, icu4x=[%d,%d) %q",
							i, xt[i].start, xt[i].end, truncate(xt[i].text, 20),
							icu[i].start, icu[i].end, truncate(icu[i].text, 20))
					}
					mismatches++
				}
			}
			if mismatches > 5 {
				t.Logf("... and %d more mismatches", mismatches-5)
			}
			if mismatches > 0 {
				t.Logf("total: %d mismatches out of %d compared segments", mismatches, minLen)
			}
		})
	}
}

func TestCorrectnessICU4XvsUniseg(t *testing.T) {
	for _, name := range corpusNames {
		t.Run(name, func(t *testing.T) {
			text := corpora[name]
			data := []byte(text)

			uni := extractUniseg(data)
			icu := extractICU4X(t, text)

			if len(uni) != len(icu) {
				t.Logf("uniseg: %d segments, icu4x: %d segments", len(uni), len(icu))
			}

			minLen := min(len(uni), len(icu))
			mismatches := 0
			for i := range minLen {
				if uni[i].start != icu[i].start || uni[i].end != icu[i].end {
					if mismatches < 5 {
						t.Logf("segment %d: uniseg=[%d,%d) %q, icu4x=[%d,%d) %q",
							i, uni[i].start, uni[i].end, truncate(uni[i].text, 20),
							icu[i].start, icu[i].end, truncate(icu[i].text, 20))
					}
					mismatches++
				}
			}
			if mismatches > 5 {
				t.Logf("... and %d more mismatches", mismatches-5)
			}
			if mismatches > 0 {
				t.Logf("total: %d mismatches out of %d compared segments", mismatches, minLen)
			}
		})
	}
}

func TestCorrectnessAllThree(t *testing.T) {
	fmt.Fprintf(os.Stderr, "\n%-12s  %10s %10s %10s  %s  %s\n",
		"Corpus", "x/text", "uniseg", "icu4x",
		"x/text≠uniseg", "x/text≠icu4x")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 80))

	for _, name := range corpusNames {
		text := corpora[name]
		data := []byte(text)

		xt := extractXText(data)
		uni := extractUniseg(data)
		icu := extractICU4X(t, text)

		xuMismatch := countMismatches(xt, uni)
		xiMismatch := countMismatches(xt, icu)

		fmt.Fprintf(os.Stderr, "%-12s  %10d %10d %10d  %13d  %12d\n",
			name, len(xt), len(uni), len(icu), xuMismatch, xiMismatch)
	}
	fmt.Fprintln(os.Stderr)
}

func countMismatches(a, b []segResult) int {
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
