//go:build ignore

package main

// Class is the line break property class.
// Each rune has a single class.
type Class uint8

// Line break property indices.
// XX is the zero value: the trie returns 0 for codepoints with no entry,
// which correctly maps to XX (resolved to AL by LB1).
//
// Base properties stored in the trie.
// The order does NOT need to match any external file — it only matters that
// these values are assigned consistently between gen.go (which writes the trie)
// and trieval.go (which reads it at runtime).
//
// AI and CJ are stored as distinct properties so CSS line-break modes can
// remap them at runtime. The default break rules treat AI identically to AL
// and CJ identically to NS.
const (
	XX         Class = iota // LB=XX (Unknown / default, zero value)
	AI                      // LB=AI (Ambiguous; default rules mirror AL)
	AK                      // LB=AK (Aksara)
	AL                      // LB=AL (Alphabetic)
	AL_DC                   // LB=AL with dotted circle (U+25CC)
	AP                      // LB=AP (Aksara Pre-base)
	AS                      // LB=AS (Aksara Start)
	B2                      // LB=B2 (Break Opportunity Before and After)
	BA                      // LB=BA (Break After)
	BB                      // LB=BB (Break Before)
	BK                      // LB=BK (Mandatory Break)
	CB                      // LB=CB (Contingent Break)
	CJ                      // LB=CJ (Conditional Japanese Starter; default rules mirror NS)
	CL                      // LB=CL (Close Punctuation)
	CM                      // LB=CM (Combining Mark)
	CP                      // LB=CP (Close Parenthesis)
	CR                      // LB=CR (Carriage Return)
	EB                      // LB=EB (Emoji Base)
	EM                      // LB=EM (Emoji Modifier)
	EX                      // LB=EX (Exclamation/Interrogation)
	GL                      // LB=GL (Non-breaking / Glue)
	H2                      // LB=H2 (Hangul LV Syllable)
	H3                      // LB=H3 (Hangul LVT Syllable)
	HL                      // LB=HL (Hebrew Letter)
	HY                      // LB=HY (Hyphen)
	ID                      // LB=ID (Ideographic)
	ID_ExtPict              // LB=ID with unassigned codepoints (GC=Cn + ExtPict)
	IN                      // LB=IN (Inseparable)
	IS                      // LB=IS (Infix Numeric Separator)
	JL                      // LB=JL (Hangul L Jamo)
	JT                      // LB=JT (Hangul T Jamo)
	JV                      // LB=JV (Hangul V Jamo)
	LF                      // LB=LF (Line Feed)
	NL                      // LB=NL (Next Line)
	NS                      // LB=NS (Nonstarter)
	NU                      // LB=NU (Numeric)
	OP_EA                   // LB=OP East Asian (Full/Half/Wide)
	OP                      // LB=OP non-East Asian
	PO                      // LB=PO (Postfix Numeric)
	PO_EA                   // LB=PO East Asian Width
	PR                      // LB=PR (Prefix Numeric)
	PR_EA                   // LB=PR East Asian Width
	QU                      // LB=QU (Quotation)
	QU_PF                   // LB=QU with GeneralCategory=Pf
	QU_PI                   // LB=QU with GeneralCategory=Pi
	RI                      // LB=RI (Regional Indicator)
	SA                      // LB=SA (Complex Context / South Asian)
	SP                      // LB=SP (Space)
	SY                      // LB=SY (Symbols Allowing Break After)
	VF                      // LB=VF (Virama Final)
	VI                      // LB=VI (Virama)
	WJ                      // LB=WJ (Word Joiner)
	ZW                      // LB=ZW (Zero Width Space)
	ZWJ                     // LB=ZWJ (U+200D)

	lastBaseProperty = uint8(ZWJ)
)

// LB9 absorption states. When base property B sees CM, it enters B_XX
// (same rule row as B, but self-loops on CM). When B or B_XX sees ZWJ,
// it enters ZWJ_absorb (LB8a: keep for everything).
//
// Only non-excluded base properties have absorption states.
// Excluded (BK, CR, LF, NL, SP, ZW, CM, ZWJ) do not absorb per LB9.
const (
	AI_XX = lastBaseProperty + 1 + iota
	AK_XX
	AL_XX
	AL_DC_XX
	AP_XX
	AS_XX
	B2_XX
	BA_XX
	BB_XX
	CB_XX
	CJ_XX
	CL_XX
	CP_XX
	EB_XX
	EM_XX
	EX_XX
	GL_XX
	H2_XX
	H3_XX
	HL_XX
	HY_XX
	ID_XX
	ID_ExtPict_XX
	IN_XX
	IS_XX
	JL_XX
	JT_XX
	JV_XX
	NS_XX
	NU_XX
	OP_EA_XX
	OP_XX
	PO_XX
	PO_EA_XX
	PR_XX
	PR_EA_XX
	QU_XX
	QU_PF_XX
	QU_PI_XX
	RI_XX
	SA_XX
	SY_XX
	VF_XX
	VI_XX
	WJ_XX
	XX_XX

	ZWJ_absorb // universal LB8a state: keep for everything after ZWJ

	lastCP = ZWJ_absorb
)

// Chain/combined states (not codepoint-advancing).
const (
	OP_SP    = lastCP + 1 + iota // OP SP*
	QU_SP                        // QU_PI SP* (× OP)
	SP_QU                        // SP × QU_PF (keep)
	CB_QU                        // CB × QU_PF (keep, LB20 exception)
	CL_CP_SP                     // (CL|CP) SP*
	B2_SP                        // B2 SP*
	HL_HY                        // HL × (HY|BA)
	AK_VI                        // (AK|AL_DC|AS) × VI
	AK_AK                        // Aksara chain
	AK_DC                        // aksara chain, last was [◌] (U+25CC)
	RI_RI                        // RI × RI pair

	NU_OP       // (PR|PO) × OP, awaiting NU
	NU_Num      // numeric context (NU seen)
	NU_Close_CL // NU (SY|IS|NU)* CL
	NU_Close_CP // NU (SY|IS|NU)* CP
	NU_Post     // NU ... (PR|PO) postfix

	sot
	eot
	stride
)
