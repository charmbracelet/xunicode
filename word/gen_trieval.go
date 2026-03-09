//go:build ignore

package main

// Class is the word break property class.
// Each rune has a single class.
type Class uint8

// Word break property indices.
// The zero value is Other (the default for codepoints with no specific class).
// These values are stored directly in the trie.

// Base properties stored in the trie.
const (
	Other                         Class = iota // WB=Other
	CR                                         // WB=CR
	LF                                         // WB=LF
	Newline                                    // WB=Newline
	Extend                                     // WB=Extend
	ZWJ                                        // WB=ZWJ
	Regional_Indicator                         // WB=Regional_Indicator
	Format                                     // WB=Format
	Katakana                                   // WB=Katakana
	Hebrew_Letter                              // WB=Hebrew_Letter
	ALetter                                    // WB=ALetter (excluding ExtPict)
	Single_Quote                               // WB=Single_Quote
	Double_Quote                               // WB=Double_Quote
	MidNumLet                                  // WB=MidNumLet
	MidLetter                                  // WB=MidLetter
	MidNum                                     // WB=MidNum
	Numeric                                    // WB=Numeric
	ExtendNumLet                               // WB=ExtendNumLet
	WSegSpace                                  // WB=WSegSpace
	Extended_Pictographic                      // Extended_Pictographic (excluding ALetter)
	ALetter_Extended_Pictographic              // ALetter AND Extended_Pictographic
	SA                                         // Complex/dictionary characters

	lastBase = uint8(SA)
)

// WB4 absorption combined states. These track what base property
// was seen before absorbing Extend/Format/ZWJ.
const (
	WSegSpace_XX      = lastBase + 1 + iota // WSegSpace absorbing Extend/Format
	ALetter_ZWJ                             // ALetter absorbing ZWJ
	Hebrew_Letter_ZWJ                       // Hebrew_Letter absorbing ZWJ
	Numeric_ZWJ                             // Numeric absorbing ZWJ
	Katakana_ZWJ                            // Katakana absorbing ZWJ
	ExtendNumLet_ZWJ                        // ExtendNumLet absorbing ZWJ
	RI_ZWJ                                  // Regional_Indicator absorbing ZWJ
	ExtPict_ZWJ                             // Extended_Pictographic absorbing ZWJ
	WSegSpace_ZWJ                           // WSegSpace absorbing ZWJ
	ALetterEP_ZWJ                           // ALetter_ExtPict absorbing ZWJ

	lastCP = ALetterEP_ZWJ
)

// Lookahead combined states for mid-word and RI pairing.
const (
	AHL_MidLetter = lastCP + 1 + iota // AHLetter × MidLetter/MidNumLetQ
	HL_MidLetter                      // HebrewLetter × MidLetter/MidNumLet
	Num_MidNum                        // Numeric × MidNum/MidNumLetQ
	HL_DQ                             // HebrewLetter × Double_Quote
	RI_RI                             // RI × RI pair

	sot    // start of text
	eot    // end of text
	stride // total table dimension
)
