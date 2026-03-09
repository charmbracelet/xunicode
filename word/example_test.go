package word_test

import (
	"fmt"

	"charm.land/xunicode/word"
)

func ExampleSegmenter() {
	input := []byte("Hello, World!")

	seg := word.NewSegmenter(input)
	for seg.Next() {
		fmt.Printf("%q\n", seg.Text())
	}
	// Output:
	// "Hello"
	// ","
	// " "
	// "World"
	// "!"
}

func ExampleSegmenter_IsWordLike() {
	input := []byte("Hello, World! 123")

	seg := word.NewSegmenter(input)
	for seg.Next() {
		if seg.IsWordLike() {
			fmt.Printf("%q\n", seg.Text())
		}
	}
	// Output:
	// "Hello"
	// "World"
	// "123"
}
