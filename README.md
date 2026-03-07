# XUnicode

XUnicode is a library for handling Unicode text segmentation. It implements the
Unicode Text Segmentation algorithms as defined in UAX#29 and UAX#14. The
library provides functions for segmenting text into grapheme clusters, words,
sentences, and line breaks.

## Features

- Zero-allocation text segmentation
- Locale tailoring support
- CSS-style line break rules

## Usage

```go
package main

import (
    "fmt"
    "github.com/charmbracelet/xunicode/grapheme"
    "github.com/charmbracelet/xunicode/word"
    "github.com/charmbracelet/xunicode/sentence"
    "github.com/charmbracelet/xunicode/line"
)

func main() {
	input := []byte("Hello, 世界! 👨‍👩‍👧‍👦")

	seg := grapheme.NewSegmenter(input)
	for seg.Next() {
		start, end := seg.Position()
		fmt.Printf("[%d:%d] %q\n", start, end, seg.Text())
	}
	// Output:
	// [0:1] "H"
	// [1:2] "e"
	// [2:3] "l"
	// [3:4] "l"
	// [4:5] "o"
	// [5:6] ","
	// [6:7] " "
	// [7:10] "世"
	// [10:13] "界"
	// [13:14] "!"
	// [14:15] " "
	// [15:40] "👨\u200d👩\u200d👧\u200d👦"

	input = []byte("Hello, World!")

	seg = word.NewSegmenter(input)
	for seg.Next() {
		fmt.Printf("%q\n", seg.Text())
	}
	// Output:
	// "Hello"
	// ","
	// " "
	// "World"
	// "!"

	input = []byte("First. Second! Third?")

	seg = sentence.NewSegmenter(input)
	for seg.Next() {
		start, end := seg.Position()
		fmt.Printf("[%d:%d] %q\n", start, end, seg.Text())
	}
	// Output:
	// [0:7] "First. "
	// [7:15] "Second! "
	// [15:21] "Third?"

	input = []byte("Hello, world! How are you?")
	seg = line.NewSegmenter(input)
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
```

## License

BSD and MIT licensed. See [LICENSE-BSD](LICENSE-BSD) and [LICENSE-MIT](LICENSE-MIT) for details.
