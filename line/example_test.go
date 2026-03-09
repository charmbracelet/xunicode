package line_test

import (
	"fmt"

	"charm.land/xunicode/line"
)

func ExampleNewSegmenter() {
	input := []byte("Hello, world! How are you?")
	seg := line.NewSegmenter(input)
	for seg.Next() {
		start, end := seg.Position()
		fmt.Printf("[%d:%d] %q\n", start, end, seg.Text())
	}

	// Output:
	// [0:7] "Hello, "
	// [7:14] "world! "
	// [14:18] "How "
	// [18:22] "are "
	// [22:26] "you?"
}
