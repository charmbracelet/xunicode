package sentence_test

import (
	"fmt"

	"github.com/charmbracelet/xunicode/sentence"
)

func ExampleSegmenter() {
	input := []byte("Ceci tuera cela. Le livre tuera l'édifice.")

	seg := sentence.NewSegmenter(input)
	for seg.Next() {
		fmt.Printf("%q\n", seg.Text())
	}
	// Output:
	// "Ceci tuera cela. "
	// "Le livre tuera l'édifice."
}

func ExampleSegmenter_Position() {
	input := []byte("First. Second! Third?")

	seg := sentence.NewSegmenter(input)
	for seg.Next() {
		start, end := seg.Position()
		fmt.Printf("[%d:%d] %q\n", start, end, seg.Text())
	}
	// Output:
	// [0:7] "First. "
	// [7:15] "Second! "
	// [15:21] "Third?"
}

func ExampleSegmenter_abbreviation() {
	input := []byte("I live in the U.S.A. and work for 3.5 hrs.")

	seg := sentence.NewSegmenter(input)
	for seg.Next() {
		fmt.Printf("%q\n", seg.Text())
	}
	// Output:
	// "I live in the U.S.A. and work for 3.5 hrs."
}
