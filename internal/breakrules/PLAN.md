# ICU Break Rules Engine for xunicode

## Goal

Replace the hand-coded Go rule structs (`SimpleRule`, `ChainRule`, `IgnoreRule`,
`OverrideRule`) with a text-based rule language compatible with ICU's RBBI
(Rule-Based Break Iterator) syntax. Rules compile to the same N×N flat table at
gen-time, but the DFA intermediate representation is also usable at runtime for
custom/user-supplied rules.

```
.rules text
     │
     ▼
 ┌────────┐
 │ Parser │  tokenize → AST, resolve $variables, UnicodeSet expressions
 └───┬────┘
     │
     ▼
 ┌────────┐
 │  NFA   │  position-based (nullable/firstpos/lastpos/followpos)
 └───┬────┘  + chained followpos for !!chain
     │
     ▼
 ┌────────┐
 │  DFA   │  subset construction → minimize
 └───┬────┘
     │
     ├─────────────────────────┐
     ▼                         ▼
 ┌──────────┐          ┌───────────┐
 │ N×N flat │          │ DFA table │
 │ (gen)    │          │ (runtime) │
 └──────────┘          └───────────┘
 Compact table          For user-supplied
 checked into repo      rules at runtime
```

## Why

1. **Rules match the spec.** UAX #14/#29 define rules as ordered regex patterns
   with implicit fallthrough. Today we manually decompose these into Go structs,
   inventing combined states, managing priority, and writing manual re-add
   boilerplate. The rule text can be nearly identical to the UAX spec or ICU's
   rule files.

2. **No more ForceSet/priority.** The entire `Set`/`ForceSet`/priority problem
   exists because we manually decompose regex rules into imperative table writes.
   The NFA→DFA pipeline resolves rule interactions automatically via subset
   construction.

3. **Spec updates become text diffs.** When Unicode 18.0 modifies a break rule,
   the change is a text diff to a `.rules` file, not careful analysis of how new
   Go structs interact with existing combined-state logic.

4. **Combined states are auto-discovered.** The 50+ manually-enumerated combined
   states (`NU_Num`, `HL_HY`, `RI_RI`, `QU_SP`, etc.) are discovered
   automatically by the DFA compiler from the regex patterns.

5. **Runtime custom rules.** The DFA backend enables user-supplied rules at
   runtime (custom segmenters, locale tailoring beyond property remapping).

6. **CSS modes via rule files or property remapping.** ICU uses separate rule
   files (`line.txt`, `line_loose.txt`, etc.). We can do the same, or continue
   using `SetOverrideLookup` for simple property remapping (CJ→ID, AL→ID, etc.).
   Both approaches compose.

## Current Architecture (what we're replacing)

### Gen-time pipeline

```
gen_trieval.go   →  defines property enums + combined state indices (hand-written)
gen.go           →  builds []segmenter.Rule from Go structs, calls segmenter.Build()
table_builder.go →  iterates rules, each writes cells via Set/ForceSet into N×N table
rule.go          →  SimpleRule, ChainRule, IgnoreRule, OverrideRule implement Rule.Apply()
gen.go (segmenter) → serializes table to tables.go
```

### Runtime pipeline

```
tables.go        →  checked-in breakTable[...] + stride/sot/eot/lastCP constants
segmenter.go     →  Next() walks input: table[left*stride+right] → Break/Keep/NoMatch/combined-state
line.go          →  SetOverrideLookup for CSS tailoring (CJ→ID, etc.)
```

### Rule types today

| Type | Purpose | Example |
|------|---------|---------|
| `SimpleRule` | Wildcard keep/break for (left,right) pairs | `× SP` (LB7) |
| `ChainRule` | Multi-step lookahead creating combined states | `NU (SY\|IS)* CL` (LB25) |
| `IgnoreRule` | CM/ZWJ absorption (X Extend* → X) | LB9, WB4, SB5 |
| `OverrideRule` | Wipe combined-state row, set specific overrides | NU_Num, QU_SP, ATerm, etc. |

### Combined states per package

| Package | Base props | Combined states | Total (stride) |
|---------|-----------|-----------------|----------------|
| grapheme | 18 | 4 | ~24 |
| word | 22 | 15 | ~39 |
| sentence | 15 | 11 | ~28 |
| line | 54 | 58 (incl. _XX absorption) | ~114 |

### Table sizes

| Package | Stride | Table cells | Bytes |
|---------|--------|------------|-------|
| grapheme | ~24 | ~576 | 576 B |
| word | ~39 | ~1,521 | 1.5 KB |
| sentence | ~28 | ~784 | 784 B |
| line | ~114 | ~12,996 | 13 KB |

## New Architecture

### Gen-time pipeline

```
line.rules       →  ICU RBBI syntax rule text (checked into repo)
breakrules.Parse →  tokenize + parse → AST with $variable resolution
breakrules.NFA   →  position-based NFA (firstpos/lastpos/followpos + chaining)
breakrules.DFA   →  subset construction + Hopcroft minimization
breakrules.Flat  →  DFA → N×N table (constrained minimization mapping DFA states to properties)
gen.go           →  reads .rules, compiles, writes tables.go (same format as today)
```

### Runtime pipeline (unchanged for standard algorithms)

```
tables.go        →  same checked-in breakTable[...] format
segmenter.go     →  same Next() state machine
line.go          →  same SetOverrideLookup for CSS tailoring
```

### Runtime pipeline (new: custom rules)

```
breakrules.Parse →  parse user-supplied rule text
breakrules.DFA   →  compile to DFA
segmenter.NewFromDFA(dfa, input) →  DFA-based Next() (states × categories table)
```

### API surface

```go
// Gen-time (replaces gen.go rule structs):
rules, err := breakrules.ParseFile("line.rules")
dfa, err := breakrules.Compile(rules, breakrules.Options{...})
flat := dfa.FlatTable(stride, sot, eot, lastCP)  // N×N for checked-in tables
dfaTable := dfa.Table()                           // DFA format for runtime use

// Runtime (new capability):
rules, _ := breakrules.Parse(customRulesText)
dfa, _ := breakrules.Compile(rules)
seg := segmenter.NewFromDFA(dfa, input)
seg.SetOverrideLookup(myOverride)  // composes with custom rules
for seg.Next() { ... }
```

## ICU RBBI Syntax Reference

### Grammar (EBNF)

```
rules         ::= statement+
statement     ::= assignment | rule | control
control       ::= ('!!forward' | '!!reverse' | '!!safe_forward'
                  | '!!safe_reverse' | '!!chain' | '!!lookAheadHardBreak'
                  | '!!quoted_literals_only') ';'
assignment    ::= variable '=' expr ';'
rule          ::= '^'? expr ('/' expr)? ('{' number '}')? ';'
number        ::= [0-9]+
expr          ::= term | expr '|' expr | expr expr    (concat > alternation)
term          ::= atom | atom '*' | atom '?' | atom '+'
atom          ::= rule-char | unicode-set | variable | '(' expr ')' | '.'
variable      ::= '$' name-start-char name-char*
unicode-set   ::= '[' set-body ']' | '[:' prop-expr ':]' | '\p{' prop-expr '}'
```

### Operators

| Syntax | Meaning |
|--------|---------|
| `expr expr` | Concatenation (implicit, higher precedence than `\|`) |
| `expr \| expr` | Alternation |
| `expr*` | Zero or more |
| `expr+` | One or more |
| `expr?` | Zero or one |
| `^` | At rule start: prevent chaining into this rule |
| `/` | Lookahead break point (break happens here, but post-context must match) |
| `{N}` | Rule status tag value |
| `.` | Match any single character |
| `#` | Comment to end of line |
| `;` | Statement terminator |

### Key controls

| Control | Effect |
|---------|--------|
| `!!chain` | Enable chained matching (match end → next match start overlap) |
| `!!quoted_literals_only` | Require `'...'` for literal characters |
| `!!lookAheadHardBreak` | Lookahead causes immediate break (no longest match) |
| `!!forward` | Rules apply to forward iteration (only type still used) |

### UnicodeSet syntax (subset we need)

```
[abc]              literal characters
[a-z]              range
[^abc]             complement
[\p{prop=value}]   Unicode property
[:prop=value:]     POSIX-style Unicode property (within [...])
[\p{prop}]         Binary property or General_Category shorthand
[set1 set2]        union (implicit)
[set1 & set2]      intersection
[set1 - set2]      difference
```

Property types we need:
- `\p{Grapheme_Cluster_Break = CR}` — GCB properties
- `\p{Word_Break = ALetter}` — WB properties
- `\p{Sentence_Break = ATerm}` — SB properties
- `\p{LineBreak = Alphabetic}` — LB properties
- `\p{Extended_Pictographic}` — binary property
- `\p{InCB=Consonant}` — Indic_Conjunct_Break
- `[:Han:]` — Script property
- `\p{ea=F}`, `\p{ea=W}`, `\p{ea=H}` — East_Asian_Width
- `\p{Cn}`, `[:Mn:]`, `[:Mc:]` — General_Category

### Chaining mechanism

With `!!chain` enabled, when a rule completes a match, the last character of
that match can be the first character of a new match from a different rule. The
compiler implements this by adding followpos links from accepting states back to
start states of other rules when they share a character class.

Example: rule `A B` and rule `B C` with chaining means `A B C` is matched as
`A B` + `B C` with `B` overlapping. This is how word break rules WB5–WB13
compose.

### Lookahead (`/`)

The `/` operator marks the actual break position within a rule. The text before
`/` is the pre-context (break happens after it), and the text after `/` is the
post-context (must match for the break to fire, but is not consumed).

Example: `$ZW $SP* / [^$SP]` — break after ZW SP*, but only if followed by a
non-SP character.

## ICU Rule Files (reference copies)

### char.txt (grapheme) — ~40 rules

Simple. Uses `!!chain` and `!!lookAheadHardBreak`. Key rules:
- GB3: `$CR $LF`
- GB4/5: `[^$Control $CR $LF]` guards
- GB9: `[^$Control $CR $LF] ($Extend | $ZWJ)`
- GB9c: `$InCBConsonant [$InCBExtend $InCBLinker]* $InCBLinker ...`
- GB11: `$Extended_Pict $Extend* $ZWJ $Extended_Pict`
- GB12/13: `^$Prepend* $Regional_Indicator $Regional_Indicator / $Regional_Indicator`
- GB999: `.`

### word.txt — ~60 rules

Uses `!!chain`. Notable:
- WB4 as `$ExFm = [$Extend $Format $ZWJ]` threaded through every rule
- WB6/7: `($ALetterPlus | $Hebrew_Letter) $ExFm* ($MidLetter | $MidNumLet | $Single_Quote) $ExFm* ($ALetterPlus | $Hebrew_Letter)`
- Rule status tags `{100}`, `{200}`, `{400}` for word type identification
- Dictionary: `$dictionary = [$ComplexContext $dictionaryCJK]`
- Explicit `$Han`, `$Hiragana`, `$Katakana` for CJK dictionary triggering

### sent.txt (sentence) — ~20 rules

Uses `!!chain`. Compact because extended forms absorb Extend/Format:
- `$ATermEx = $ATerm ($Extend | $Format)*`
- SB6: `$ATermEx $NumericEx`
- SB8: `$ATermEx $CloseEx* $SpEx* $NotLettersEx* $Lower`
- SB9/10/11: `($STermEx | $ATermEx) $CloseEx* $SpEx* ($Sep | $CR | $LF)?`

### line.txt — ~120 rules (most complex)

Uses `!!chain`. The most complex due to:
- LB9 CM absorption: `$CAN_CM $CM*` repeated in nearly every rule
- LB25 numeric context: single massive regex
- LB15a/15b quotation rules: 18 lines handling Pf/Pi interactions
- LB19/19a: 6 lines with East Asian width checks
- LB20a: 8 lines for hyphen-letter non-breaking (Finnish tailoring promoted)
- LB30a: 3 lines for RI pairs with chaining control

ICU also maintains 10+ variants: `line_cj.txt`, `line_loose.txt`,
`line_normal.txt`, `line_loose_cj.txt`, `line_normal_cj.txt`,
`line_phrase_cj.txt`, etc.

## Table Size Analysis

The DFA is the intermediate representation. The output table format is our
choice:

### Option A: N×N flat (gen-time, current format)

Same format as today. The DFA→N×N collapse works because the N×N table is a DFA
where state identity equals property index. The minimizer merges DFA states that
produce identical transition rows for the same base property.

**Size: identical to today** (~13 KB for line break).

Auto-discovered combined states replace hand-enumerated ones. The minimizer may
discover slightly more or fewer states than we have today, but the difference is
bounded by the minimization quality.

### Option B: DFA state table (runtime, for custom rules)

`[][]uint8` where `table[state][category] → encoded_action`. Each action encodes
the next state and break/keep/nomatch.

**Estimated size**: ~500 states × 60 categories × 1 byte = ~30 KB for line
break (with 8-bit row optimization like ICU). Acceptable for runtime use where
startup cost matters more than binary size.

### Both from the same DFA

The DFA is compiled once. Two serialization backends produce the two formats:

```go
dfa := breakrules.Compile(rules, opts)
flat := dfa.FlatTable(...)   // N×N for gen-time (checked into repo)
rt := dfa.RuntimeTable()     // DFA state table for runtime custom rules
```

## Tailoring Model

### Current (preserved unchanged)

Three layers, all orthogonal:

| Layer | Mechanism | Example |
|-------|-----------|---------|
| **Property remapping** | `SetOverrideLookup(func(uint8, rune) uint8)` | CSS `line-break: loose` → CJ→ID |
| **Codepoint override** | Same mechanism, rune-specific | Finnish: `:` → `Other` for word break |
| **Full bypass** | Swap segmenter entirely | CSS `line-break: anywhere` → grapheme |

This works identically with both N×N and DFA table formats because the override
sits between trie lookup and table lookup — it remaps property/category indices.

### New (enabled by DFA runtime)

| Layer | Mechanism | Example |
|-------|-----------|---------|
| **Rule-level override** | Compile separate .rules file | Custom line breaker for search indexing |
| **Rule composition** | (future) merge rule ASTs before compiling | Add rules to base algorithm |

ICU's approach to CSS modes: entirely separate rule files (`line_loose.txt`,
etc.). We can do this too, but our property-remapping approach is more
memory-efficient (one table, runtime remapping) and sufficient for CSS.

## Implementation Plan

### Phase 1: Core compiler (breakrules package)

New package: `internal/breakrules/`

| Component | File | Est. Lines | Description |
|-----------|------|-----------|-------------|
| Token types | `token.go` | ~80 | Token enum, Token struct |
| Lexer | `lexer.go` | ~350 | Tokenizes .rules text (comments, variables, sets, operators, controls) |
| UnicodeSet | `unicodeset.go` | ~500 | Parse `[...]` expressions with properties, ranges, set operations |
| AST nodes | `ast.go` | ~120 | Concat, Alt, Repeat, Dot, CharClass, Variable, LookAhead, ChainBreak |
| Parser | `parser.go` | ~450 | Tokens → AST with precedence (concat > alt), handle `^`, `/`, `{N}` |
| Variables | `resolve.go` | ~150 | Expand $variable references, validate no cycles |
| NFA | `nfa.go` | ~500 | Position-based: calcNullable, calcFirstPos, calcLastPos, calcFollowPos |
| Chaining | `chain.go` | ~200 | calcChainedFollowPos: link accepting → start positions via shared classes |
| DFA | `dfa.go` | ~400 | Subset construction, flag accepting/lookahead states |
| Minimize | `minimize.go` | ~250 | Hopcroft's algorithm for DFA state minimization |
| Compile | `compile.go` | ~200 | Orchestrator: parse → resolve → NFA → chain → DFA → minimize |
| **Total** | | **~3,200** | |

### Phase 2: Gen-time backend (DFA → N×N flat table)

| Component | File | Est. Lines | Description |
|-----------|------|-----------|-------------|
| Flat table | `flat.go` | ~300 | Map DFA states to property indices, emit N×N table |
| Property map | `propmap.go` | ~200 | Map UnicodeSet expressions → property indices from gen_trieval.go |
| **Total** | | **~500** | |

### Phase 3: Write .rules files

Translate current gen.go rule structs into .rules text files:

| File | Source gen.go | Est. Lines |
|------|--------------|-------------|
| `grapheme/grapheme.rules` | grapheme/gen.go | ~40 |
| `word/word.rules` | word/gen.go | ~60 |
| `sentence/sentence.rules` | sentence/gen.go | ~30 |
| `line/line.rules` | line/gen.go | ~120 |

These should closely follow the ICU rule files listed above, adapted for our
property splits (QU_PI/QU_PF, OP_EA, PR_EA, PO_EA, AL_DC, ID_ExtPict).

### Phase 4: Runtime backend (DFA table for custom rules)

| Component | File | Est. Lines | Description |
|-----------|------|-----------|-------------|
| Runtime table | `runtime.go` | ~150 | DFA → runtime state table format |
| DFA segmenter | `segmenter_dfa.go` | ~200 | NewFromDFA, Next() using DFA table |
| **Total** | | **~350** | |

### Phase 5: Replace gen.go files

Update each package's gen.go to:
1. Read `.rules` file instead of building Go rule structs
2. Call `breakrules.Compile` → `FlatTable` → `WriteBreakTable`
3. Delete `SimpleRule`, `ChainRule`, `IgnoreRule`, `OverrideRule`, `TableBuilder`

### Phase 6: Delete old infrastructure

Remove from `internal/segmenter/`:
- `rule.go` (SimpleRule, ChainRule, IgnoreRule, OverrideRule)
- `table_builder.go` (TableBuilder, Build, Set, ForceSet)

Keep:
- `segmenter.go` (runtime engine, unchanged)
- `gen.go` (WriteBreakTable serializer, unchanged)

## Verification Strategy

### Table equivalence

For each package (grapheme, word, sentence, line):

1. Compile `.rules` → DFA → N×N flat table
2. Compare byte-for-byte with current checked-in tables
3. Any difference means a rule translation error

Command:
```sh
cd line && go run ./gen.go gen_trieval.go -unicode=15.0.0
cd line && go run -tags=go1.27 ./gen.go gen_trieval.go -unicode=17.0.0
```

Repeat for all four packages.

### Conformance tests

```sh
go test -count=1 ./...
```

Tests parse `auxiliary/*BreakTest.txt` from UCD and compare segmenter output.
These validate runtime correctness against the Unicode conformance suite.

### DFA runtime tests

New tests comparing `segmenter.NewFromDFA(dfa, input)` output against
`segmenter.New(&ruleData, input)` for the same input.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| DFA→N×N collapse produces different combined states than hand-enumerated | Byte-for-byte table comparison; if states differ but behavior is equivalent, conformance tests catch any real bugs |
| ICU rule syntax has features we don't need (reverse rules, dictionary) | Implement the subset we need; `!!reverse`, `!!safe_*` are already deprecated in ICU |
| LB9 CM absorption is verbose in regex form (`$CAN_CM $CM*` everywhere) | Accept the verbosity in .rules (matches ICU); or add a syntactic sugar extension |
| Runtime DFA table too large for constrained environments | N×N flat table remains default; DFA is opt-in for custom rules |
| Performance regression in gen-time compilation | Compilation runs once at `go generate`; seconds vs milliseconds doesn't matter |
| UnicodeSet parser complexity | We need a subset: properties, ranges, complement, union, intersection, difference. No string literals, no `{bof}`/`{eof}` |

## Open Questions

1. **Should we support `{bof}` and `{eof}`?** ICU's sent.txt uses `{bof}` for
   beginning-of-file matching. We currently handle sot/eot as pseudo-properties.
   May be simpler to keep our sot/eot model.

2. **Should we support rule status tags `{N}`?** ICU word.txt uses `{100}`,
   `{200}`, `{400}` for word type identification. We don't currently expose word
   types. Could add later.

3. **Should .rules files be embedded via `//go:embed`?** Cleaner than reading
   from disk at gen-time, but gen.go runs as a standalone program.

4. **East Asian width splits.** Our OP_EA, PR_EA, PO_EA, QU_PI, QU_PF, AL_DC,
   ID_ExtPict are not standard UAX properties — they're splits we compute from
   multiple UCD files. The .rules files need a way to reference them. Options:
   (a) custom property names, (b) UnicodeSet expressions like
   `[$OP & [\p{ea=F}\p{ea=W}\p{ea=H}]]`, (c) pre-computed variables.

5. **How much ICU compatibility do we target?** Full syntax compatibility means
   we could consume ICU's rule files directly. Subset compatibility means we
   write our own .rules with a compatible but smaller grammar.

## Dependencies

- No new external dependencies. The compiler is pure Go.
- `golang.org/x/text` (existing) for locale support in tailoring.
- Unicode property data from UCD files (already downloaded by gen infrastructure).

## File Inventory (what gets created/modified/deleted)

### New files

```
internal/breakrules/
  token.go          ~80 lines
  lexer.go          ~350 lines
  unicodeset.go     ~500 lines
  ast.go            ~120 lines
  parser.go         ~450 lines
  resolve.go        ~150 lines
  nfa.go            ~500 lines
  chain.go          ~200 lines
  dfa.go            ~400 lines
  minimize.go       ~250 lines
  compile.go        ~200 lines
  flat.go           ~300 lines
  propmap.go        ~200 lines
  runtime.go        ~150 lines
  *_test.go         ~1000 lines (tests for each component)

grapheme/grapheme.rules   ~40 lines
word/word.rules           ~60 lines
sentence/sentence.rules   ~30 lines
line/line.rules           ~120 lines
```

### Modified files

```
grapheme/gen.go     replace rule structs with .rules compilation
word/gen.go         replace rule structs with .rules compilation
sentence/gen.go     replace rule structs with .rules compilation
line/gen.go         replace rule structs with .rules compilation
internal/segmenter/segmenter.go   add NewFromDFA (Phase 4)
```

### Deleted files (Phase 6)

```
internal/segmenter/rule.go           SimpleRule, ChainRule, IgnoreRule, OverrideRule
internal/segmenter/table_builder.go  TableBuilder, Build, Set, ForceSet
```

### Unchanged files

```
internal/segmenter/segmenter.go   Next() runtime engine (N×N path)
internal/segmenter/gen.go         WriteBreakTable serializer
{package}/trieval.go              runtime property enums
{package}/gen_trieval.go          gen-time property enums (may shrink if combined states auto-generated)
{package}/tables*.go              regenerated but same format
{package}/line.go, word.go, ...   runtime API + SetOverrideLookup (unchanged)
```
