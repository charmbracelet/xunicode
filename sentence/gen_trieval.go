//go:build ignore

package main

// Class is the sentence break property class.
// Each rune has a single class.
type Class uint8

// Sentence break property indices.
// The zero value is Other (the default for codepoints with no specific class).
// These values are stored directly in the trie.

// Base properties stored in the trie.
const (
	Other     Class = iota // SB=Other
	CR                     // SB=CR
	LF                     // SB=LF
	Extend                 // SB=Extend
	Sep                    // SB=Sep
	Format                 // SB=Format
	Sp                     // SB=Sp
	Lower                  // SB=Lower
	Upper                  // SB=Upper
	OLetter                // SB=OLetter
	Numeric                // SB=Numeric
	ATerm                  // SB=ATerm
	SContinue              // SB=SContinue
	STerm                  // SB=STerm
	Close                  // SB=Close
)

// lastCP is the threshold for marker advancement in the segmenter engine.
// Properties with index > lastCP are combined states that should not advance
// the marker on Index state transitions.
const lastCP = uint8(Close)

// Combined states for sentence break chains and SB5 absorption.
const (
	UpperATerm       = lastCP + 1 + iota // Upper × ATerm (SB7)
	LowerATerm                           // Lower × ATerm (SB8)
	ATermClose                           // ATerm Close* (SB8/SB8a)
	ATermCloseSp                         // ATerm Close* Sp* (SB9)
	ATermCloseSpPSep                     // ATerm Close* Sp* (Sep|LF) — paragraph break
	ATermCloseSpCR                       // ATerm Close* Sp* CR — awaits LF
	ATermCloseSpSB8                      // SB8 scanning state
	STermClose                           // STerm Close* (SB8a)
	STermCloseSp                         // STerm Close* Sp* (SB9)
	STermCloseSpPSep                     // STerm Close* Sp* (Sep|LF) — paragraph break
	STermCloseSpCR                       // STerm Close* Sp* CR — awaits LF

	sot    // start of text
	eot    // end of text
	stride // total table dimension
)
