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
    "charm.land/xunicode/grapheme"
    "charm.land/xunicode/word"
    "charm.land/xunicode/sentence"
    "charm.land/xunicode/line"
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

## Contributing

See [contributing][contribute].

[contribute]: https://charm.land/xunicode/contribute

## Feedback

We’d love to hear your thoughts on this project. Feel free to drop us a note!

- [Twitter](https://twitter.com/charmcli)
- [The Fediverse](https://mastodon.social/@charmcli)
- [Discord](https://charm.sh/chat)

## Acknowledgments

XUnicode is inspired by and based on the work of the Go team, the Unicode
Consortium, and many Go libraries that have implemented Unicode text
segmentation. This includes but is not limited to:

- [github.com/blevesearch/segment](https://github.com/belvesearch/segment): A Go library for text segmentation that includes support for Unicode text segmentation algorithms.
- [github.com/clipperhouse/uax14](https://github.com/clipperhouse/uax14): A Go library that implements the Unicode Line Breaking Algorithm as defined in UAX#14.
- [github.com/clipperhouse/uax29](https://github.com/clipperhouse/uax29): A Go library that implements the Unicode Text Segmentation algorithms as defined in UAX#29.
- [github.com/rivo/uniseg](https://github.com/rivo/uniseg): A Go library for Unicode text segmentation that provides a simple API for segmenting text into grapheme clusters, words, sentences, and line breaks.
- [github.com/unicode-org/icu4x](https://github.com/unicode-org/icu4x): A project that provides a set of libraries for Unicode support in Rust, including text segmentation algorithms.
- [github.com/unicode-org/icu](https://github.com/unicode-org/icu): The International Components for Unicode (ICU) project, which provides a comprehensive set of libraries for Unicode support, including text segmentation algorithms.
- [golang.org/x/text](https://golang.org/x/text): The Go team's official text processing library, which includes Unicode segmentation algorithms.

## License

[BSD](LICENSE-BSD) and [Apache 2.0](LICENSE-APACHE)

---

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-banner-next.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source • نحنُ نحب المصادر المفتوحة
