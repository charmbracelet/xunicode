package segmenter

import (
	"fmt"
	"io"
)

// CodeWriter is the interface expected by the Write* helpers. It is
// satisfied by gen.CodeWriter.
type CodeWriter interface {
	io.Writer
	WriteComment(string, ...any)
}

// WriteBreakTable writes the breakTable array and a RuleBreakData struct
// literal to w, using the given variable name and trie type name.
func WriteBreakTable(w CodeWriter, varName string, bt *BreakTable, trieType string, complexProp uint8) {
	n := int(bt.Stride)
	w.WriteComment(
		`breakTable is the break state table.
	breakTable[left*stride + right] encodes the action for (left, right).
	See segmenter.BreakState for the action encoding.`)
	fmt.Fprintf(w, "var breakTable = [...]uint8{")
	for i, v := range bt.Table {
		if i%n == 0 {
			fmt.Fprintf(w, "\n\t")
		}
		fmt.Fprintf(w, "%d, ", v)
	}
	fmt.Fprintf(w, "\n}\n\n")

	fmt.Fprintf(w, "var trie = %s{}\n", trieType)
	fmt.Fprintf(w, "var %s = segmenter.RuleBreakData{\n", varName)
	fmt.Fprintf(w, "\tPropertyLookup:        trie.lookup,\n")
	fmt.Fprintf(w, "\tBreakStateTable:       breakTable[:],\n")
	fmt.Fprintf(w, "\tPropertyCount:         %d,\n", bt.Stride)
	fmt.Fprintf(w, "\tLastCodepointProperty: %d,\n", bt.LastCP)
	fmt.Fprintf(w, "\tSOTProperty:           %d,\n", bt.SOT)
	fmt.Fprintf(w, "\tEOTProperty:           %d,\n", bt.EOT)
	if complexProp != 0 {
		fmt.Fprintf(w, "\tComplexProp:           %d,\n", complexProp)
	}
	fmt.Fprintf(w, "}\n\n")
}
