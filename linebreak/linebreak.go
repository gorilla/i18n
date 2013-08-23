// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// gorilla/i18n/linebreak implements the Unicode line breaking algorithm.
//
// Line breaking, also known as word wrapping, is the process of breaking a
// section of text into lines such that it will fit in the available width
// of a page, window or other display area.
//
// As simple as it sounds, this is not a trivial task when support for
// multilingual texts is required. The particular algorithm used in this
// package is based on best practices defined in UAX #14:
//
//     http://www.unicode.org/reports/tr14/
//
// A similar package that served as inspiration for this one is Bram Stein's
// Unicode Tokenizer (for Node.js):
//
//     https://github.com/bramstein/unicode-tokenizer
package linebreak

import (
	"io"
	"unicode"
)

type BreakAction int

// Line breaking actions.
const (
	// A line break opportunity exists between two adjacent characters of the
	// given line breaking classes.
	BreakDirect BreakAction = iota
	// A line break opportunity exists between two characters of the given
	// line breaking classes only if they are separated by one or more spaces.
	BreakIndirect
	BreakCombiningIndirect
	BreakCombiningProhibited
	// No line break opportunity exists between two characters of the given
	// line breaking classes, even if they are separated by one or more space
	// characters.
	BreakProhibited
	// A line must break following a character that has the mandatory break
	// property.
	BreakMandatory
)

// Pair table shortcuts.
const (
	di = BreakDirect
	in = BreakIndirect
	ci = BreakCombiningIndirect
	cp = BreakCombiningProhibited
	pr = BreakProhibited
	ex = BreakMandatory
)

// Pair table stores line breaking actions for adjacent line breaking classes.
//
//     PairTable[beforeClass][afterClass] = BreakAction
//
// Note: To determine a break it is generally not sufficient to just test the
// two adjacent characters. In any case, a custom table allows some degree of
// result tailoring.
type PairTable [][]BreakAction

// Action returns the line breaking action for the given class pair.
func (t PairTable) Action(before, after BreakClass) BreakAction {
	if int(before) < len(t) && int(after) < len(t[before]) {
		return t[before][after]
	}
	return BreakProhibited
}

// pairTable is the example PairTable defined in UAX #14:
//
//     http://www.unicode.org/reports/tr14/#Table2
var pairTable = PairTable{
//   after:
//   OP  CL  CP  QU  GL  NS  EX  SY  IS  PR  PO  NU  AL  HL  ID  IN  HY  BA  BB  B2  ZW  CM  WJ  H2  H3  JL  JV  JT  RI   // before:
	{pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, pr, cp, pr, pr, pr, pr, pr, pr, pr}, // OP
	{di, pr, pr, in, in, pr, pr, pr, pr, in, in, di, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // CL
	{di, pr, pr, in, in, pr, pr, pr, pr, in, in, in, in, in, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // CP
	{pr, pr, pr, in, in, in, pr, pr, pr, in, in, in, in, in, in, in, in, in, in, in, pr, ci, pr, in, in, in, in, in, in}, // QU
	{in, pr, pr, in, in, in, pr, pr, pr, in, in, in, in, in, in, in, in, in, in, in, pr, ci, pr, in, in, in, in, in, in}, // GL
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, di, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // NS
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, di, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // EX
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, in, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // SY
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, in, in, in, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // IS
	{in, pr, pr, in, in, in, pr, pr, pr, di, di, in, in, in, in, di, in, in, di, di, pr, ci, pr, in, in, in, in, in, di}, // PR
	{in, pr, pr, in, in, in, pr, pr, pr, di, di, in, in, in, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // PO
	{in, pr, pr, in, in, in, pr, pr, pr, in, in, in, in, in, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // NU
	{in, pr, pr, in, in, in, pr, pr, pr, di, di, in, in, in, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // AL
	{in, pr, pr, in, in, in, pr, pr, pr, di, di, in, in, in, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // HL
	{di, pr, pr, in, in, in, pr, pr, pr, di, in, di, di, di, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // ID
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, di, di, di, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // IN
	{di, pr, pr, in, di, in, pr, pr, pr, di, di, in, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // HY
	{di, pr, pr, in, di, in, pr, pr, pr, di, di, di, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // BA
	{in, pr, pr, in, in, in, pr, pr, pr, in, in, in, in, in, in, in, in, in, in, in, pr, ci, pr, in, in, in, in, in, in}, // BB
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, di, di, di, di, di, in, in, di, pr, pr, ci, pr, di, di, di, di, di, di}, // B2
	{di, di, di, di, di, di, di, di, di, di, di, di, di, di, di, di, di, di, di, di, pr, di, di, di, di, di, di, di, di}, // ZW
	{in, pr, pr, in, in, in, pr, pr, pr, di, di, in, in, in, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, di, di}, // CM
	{in, pr, pr, in, in, in, pr, pr, pr, in, in, in, in, in, in, in, in, in, in, in, pr, ci, pr, in, in, in, in, in, in}, // WJ
	{di, pr, pr, in, in, in, pr, pr, pr, di, in, di, di, di, di, in, in, in, di, di, pr, ci, pr, di, di, di, in, in, di}, // H2
	{di, pr, pr, in, in, in, pr, pr, pr, di, in, di, di, di, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, in, di}, // H3
	{di, pr, pr, in, in, in, pr, pr, pr, di, in, di, di, di, di, in, in, in, di, di, pr, ci, pr, in, in, in, in, di, di}, // JL
	{di, pr, pr, in, in, in, pr, pr, pr, di, in, di, di, di, di, in, in, in, di, di, pr, ci, pr, di, di, di, in, in, di}, // JV
	{di, pr, pr, in, in, in, pr, pr, pr, di, in, di, di, di, di, in, in, in, di, di, pr, ci, pr, di, di, di, di, in, di}, // JT
	{di, pr, pr, in, in, in, pr, pr, pr, di, di, di, di, di, di, di, in, in, di, di, pr, ci, pr, di, di, di, di, di, in}, // RI
}

// ClassResolver returns a line breaking class for the given rune.
type ClassResolver func(rune) BreakClass

// classResolver is the default ClassResolver.
func classResolver(r rune) BreakClass {
	cls := Class(r)
	// LB1: Resolve AI, CB, CJ, SA, SG, and XX into other classes.
	// We are using the generic resolution proposed in UAX #14.
	switch cls {
	case ClassAI, ClassSG, ClassXX:
		cls = ClassAL
	case ClassCJ:
		cls = ClassNS
	case ClassSA:
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) {
			cls = ClassCM
		} else {
			cls = ClassAL
		}
	case ClassCB:
		// TODO: CB should be left to be resolved later, according to
		// LB9, LB10 and LB20.
		// For now we are using a placeholder; maybe not the best one.
		cls = ClassID
	}
	return cls
}

// NewScanner returns a line breaking scanner to scan the given runes.
func NewScanner(r []rune) *Scanner {
	return &Scanner{
		Resolver: classResolver,
		Table:    pairTable,
		runes:    r,
	}
}

// Scanner scans a text looking for line breaking opportunities.
type Scanner struct {
	Resolver  ClassResolver // returns a line breaking class for a rune
	Table     PairTable     // returns an action for adjacent line breaking classes
	runes     []rune        // input
	pos       int           // position of the input when moving forward
	prevClass BreakClass    // previous rune class when moving forward
	err       error         // possible error; freezes the scanner
}

// Next finds the next line breaking action in the input.
//
// It can be called successively to find all actions until the end
// of the input, when it returns io.EOF as error.
func (s *Scanner) Next() (pos int, action BreakAction, err error) {
	var cls BreakClass

	// Read start of text and set prevClass.
	if s.pos == 0 {
		cls, err = s.nextClass()
		if err != nil {
			// Always break at the end.
			action = BreakMandatory
			return
		}
		s.prevClass = cls
		action = BreakProhibited
		return
	}

	// Now read the next rune and decide what to do.
	// We handle spaces manually, and anything else using PairTable.
	pos = s.pos
	cls, err = s.nextClass()
	if err != nil {
		// Always break at the end.
		action = BreakMandatory
		return
	}

	if !(s.prevClass != ClassBK && (s.prevClass != ClassCR || cls == ClassLF)) {
		return
	}

	switch cls {
	case ClassBK, ClassLF:
		// handle BK, NL and LF explicitly
		action = BreakProhibited
		s.prevClass = ClassBK
		return
	case ClassCR:
		// handle CR explicitly
		action = BreakProhibited
		s.prevClass = ClassCR
		return
	case ClassSP:
		// handle spaces explicitly
		// apply rule LB7: × SP
		// do not update s.prevClass
		action = BreakProhibited
		return
	}

	// Lookup pair table information in PairTable[before][after].
	action = s.Table.Action(s.prevClass, cls)

	switch action {
	case BreakIndirect:
		// resolve indirect break
		// if context is A SP + B
		//       break opportunity
		// else
		//       no break opportunity
		switch s.prevClass {
		case ClassSP:
			action = BreakIndirect
		default:
			action = BreakProhibited
		}
	case BreakCombiningIndirect:
		// resolve combining mark break
		switch s.prevClass {
		case ClassSP:
			// new: space is not a base
			// apply rule SP ÷
			action = BreakCombiningIndirect
		default:
			// do not break before CM
			action = BreakProhibited
			// apply rule LB9: X CM * -> X
			// do not update cls
			return
		}
	case BreakCombiningProhibited:
		// this is the case OP SP* CM
		// no break allowed
		action = BreakProhibited
		if s.prevClass == ClassSP {
			// apply rule LB9: X CM* -> X
			// do not update cls
			return
		}
	}

	// Save cls of "before" character.
	s.prevClass = cls
	return
}

// nextClass returns the next line breaking class in the input.
func (s *Scanner) nextClass() (cls BreakClass, err error) {
	if s.err != nil {
		err = s.err
		return
	}
	if s.pos >= len(s.runes) {
		s.err = io.EOF
		err = s.err
		return
	}
	sot := s.pos == 0
	cls = s.Resolver(s.runes[s.pos])
	s.pos += 1
	switch cls {
	case ClassNL:
		// Equivalent; simplifies code:
		// "The NL class acts like BK in all respects (there is a
		// mandatory break after any NEL character)."
		cls = ClassBK
	case ClassSP:
		// Special case for start of text.
		if sot {
			cls = ClassWJ
		}
	case ClassLF:
		// Special case for start of text.
		if sot {
			cls = ClassBK
		}
	}
	return
}

// last finds the last line breaking action in the input.
//
// It can be called successively to find all actions until the start
// of the input, when it returns io.EOF as error (really meaning SOF).
func (s *Scanner) last() (pos int, action BreakAction, err error) {
	// TODO
	return
}
