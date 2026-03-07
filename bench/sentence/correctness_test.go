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
)

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func icu4xBinary() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "icu4x", "target", "release", "icu4x-sentence-segment")
}

func extractICU4X(t *testing.T, text string) []sentResult {
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

	var segs []sentResult
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
		segs = append(segs, sentResult{
			start: start,
			end:   start + len(raw),
			bytes: raw,
			text:  string(raw),
		})
	}
	return segs
}

func countMismatches(a, b []sentResult) int {
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
	fmt.Fprintf(os.Stderr, "\n%-14s  %10s %10s %10s  %s  %s\n",
		"Corpus", "x/text", "uniseg", "icu4x",
		"x/text≠uniseg", "x/text≠icu4x")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 84))

	for _, name := range corpusNames {
		data := []byte(corpora[name])

		xt := extractXTextSentences(data)
		uni := extractUnisegSentences(data)
		icu := extractICU4X(t, corpora[name])

		xuMismatch := countMismatches(xt, uni)
		xiMismatch := countMismatches(xt, icu)

		fmt.Fprintf(os.Stderr, "%-14s  %10d %10d %10d  %13d  %12d\n",
			name, len(xt), len(uni), len(icu), xuMismatch, xiMismatch)
	}
	fmt.Fprintln(os.Stderr)
}
