// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// linebreak implements the Unicode line breaking algorithm.
//
// Based on Unicode Standard Annex #14 - Revision 30
// http://www.unicode.org/reports/tr14/
package linebreak

import (
	"unicode"
)

type breakAction int

// Line breaking actions.
const (
	BreakDirect breakAction = iota
	BreakIndirect
	BreakCombiningIndirect
	BreakCombiningProhibited
	BreakProhibited
	BreakExplicit
)

// Table shortcuts.
const (
	bDI = BreakDirect
	bIN = BreakIndirect
	bCI = BreakCombiningIndirect
	bCP = BreakCombiningProhibited
	bPR = BreakProhibited
	bEX = BreakExplicit
)

// Pair Table.
//
// See: http://www.unicode.org/reports/tr14/#Table2
var pairTable = [][]breakType{
	{bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bPR, bCP, bPR, bPR, bPR, bPR, bPR, bPR, bPR},
	{bDI, bPR, bPR, bIN, bIN, bPR, bPR, bPR, bPR, bIN, bIN, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bPR, bPR, bPR, bPR, bIN, bIN, bIN, bIN, bIN, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bPR, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bPR, bCI, bPR, bIN, bIN, bIN, bIN, bIN, bIN},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bPR, bCI, bPR, bIN, bIN, bIN, bIN, bIN, bIN},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bIN, bIN, bIN, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bIN, bIN, bIN, bIN, bIN, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bIN, bIN, bIN, bIN, bIN, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bIN, bIN, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bIN, bIN, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bDI, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bPR, bPR, bIN, bDI, bIN, bPR, bPR, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bPR, bCI, bPR, bIN, bIN, bIN, bIN, bIN, bIN},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bPR, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bIN, bIN, bIN, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bDI},
	{bIN, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bIN, bPR, bCI, bPR, bIN, bIN, bIN, bIN, bIN, bIN},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bIN, bIN, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bIN, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bIN, bIN, bIN, bIN, bDI, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bIN, bIN, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bIN, bDI, bDI, bDI, bDI, bIN, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bIN, bDI},
	{bDI, bPR, bPR, bIN, bIN, bIN, bPR, bPR, bPR, bDI, bDI, bDI, bDI, bDI, bDI, bDI, bIN, bIN, bDI, bDI, bPR, bCI, bPR, bDI, bDI, bDI, bDI, bDI, bIN},
}
