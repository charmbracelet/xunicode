// Package grapheme implements Unicode grapheme cluster segmentation
// as defined by UAX #29.
package grapheme

import (
	"github.com/charmbracelet/xunicode/internal/segmenter"
)

// Segmenter iterates over the grapheme clusters in a byte slice.
// The usage pattern is:
//
//	seg := grapheme.NewSegmenter(input)
//	for seg.Next() {
//	    fmt.Println(seg.Bytes())
//	}
type Segmenter struct {
	s *segmenter.Segmenter
}

// NewSegmenter returns a Segmenter that iterates over the grapheme clusters
// in the given input.
func NewSegmenter(input []byte) *Segmenter {
	return &Segmenter{s: segmenter.New(&ruleData, input)}
}

// Next advances to the next grapheme cluster. It returns false when the
// end of input has been reached.
func (g *Segmenter) Next() bool {
	input := g.s.Input()
	pos := g.s.End()

	// ASCII fast path: non-CR/LF ASCII bytes are each their own grapheme
	// cluster (property Other). Emit one directly, skipping the state machine.
	// The next byte must also be ASCII to avoid splitting a base+combining-mark
	// sequence (a non-ASCII follower could be Extend/ZWJ/SpacingMark).
	if pos < len(input) {
		b := input[pos]
		if b < 0x80 && b != '\r' && b != '\n' {
			if pos+1 >= len(input) || input[pos+1] < 0x80 {
				g.s.FastForward(pos+1, 0)
				return true
			}
		}
	}
	return g.s.Next()
}

// Bytes returns the current grapheme cluster as a byte slice.
func (g *Segmenter) Bytes() []byte { return g.s.Bytes() }

// Text returns the current grapheme cluster as a string.
func (g *Segmenter) Text() string { return g.s.Text() }

// Position returns the byte offsets [start, end) of the current grapheme
// cluster.
func (g *Segmenter) Position() (start, end int) { return g.s.Position() }
