// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package linebreak

import (
	"testing"
)

func getActions(r []rune) (actions []BreakAction, pos int) {
	var err error
	var action BreakAction
	s := NewScanner(r)
	for {
		pos, action, err = s.Next()
		actions = append(actions, action)
		if err != nil {
			break
		}
	}
	return
}

func breakToInts(breaks []BreakAction) (bi []int) {
	for _, v := range breaks {
		brk := 0
		switch v {
		case BreakDirect, BreakIndirect, BreakCombiningIndirect, BreakMandatory:
			brk = 1
		}
		bi = append(bi, brk)
	}
	return
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestScanner(t *testing.T) {
	bad := 0
	for _, v := range lineBreakTests {
		breaks, _ := getActions([]rune(v.text))
		breakInts := breakToInts(breaks)
		if !equalIntSlice(v.breaks, breakInts) {
			t.Errorf("Failed test %d (%+q): expected %#v, got %#v", v.id, v.text, v.breaks, breakInts)
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("Failed: %d - Succeeded: %d - Total: %d tests", bad, len(lineBreakTests)-bad, len(lineBreakTests))
	}
}
