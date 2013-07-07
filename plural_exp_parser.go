// This file is part of monsti/gettext.
// Copyright 2013 Christian Neumann

// monsti/gettext is free software: you can redistribute it and/or modify it
// under the terms of the GNU Lesser General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.

// monsti/gettext is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the GNU Lesser General Public License for more
// details.

// You should have received a copy of the GNU Lesser General Public License
// along with monsti/gettext. If not, see <http://www.gnu.org/licenses/>.

package gettext

import (
	"bytes"
	"fmt"
	"strconv"
)

// peParser parses plural form expression used in gettext catalogs.
type peParser struct {
	// Unparesed expression
	exp []byte
	// Last parsed symbol
	sym []byte
}

// pluralForm maps an n to an index.
type pluralForm func(n int) int

// A parser Error
type pError struct {
	err string
	exp []byte
}

func (p pError) Error() string {
	return fmt.Sprintf("%v. Remaining: %q", p.err, p.exp)
}

// errror emits a parser error.
func (p peParser) error(msg string, args ...interface{}) {
	err := pError{fmt.Sprintf(msg, args...), p.exp}
	panic(err)
}

// dewhitespace removes any whitespace
func (p *peParser) dewhitespace() {
	for len(p.exp) > 0 && string(p.exp[0]) == " " {
		p.exp = p.exp[1:]
	}
}

// accept tries to parse the given symbol in front of the expression. Iff it
// succeeds, it returns true.
func (p *peParser) accept(sym string) bool {
	p.dewhitespace()
	if len(p.exp) >= len(sym) && bytes.Equal(p.exp[:len(sym)], []byte(sym)) {
		p.sym = []byte(sym)
		p.exp = p.exp[len(sym):len(p.exp)]
		return true
	}
	return false
}

// expect tries to parse the given symbol in front of the expression or errors
// if it's not there.
func (p *peParser) expect(sym string) {
	p.dewhitespace()
	if bytes.Equal(p.exp[:len(sym)], []byte(sym)) {
		p.sym = []byte(sym)
		p.exp = p.exp[len(sym):len(p.exp)]
		return
	}
	p.error("Expected %v, got %v", []byte(sym), p.exp[:len(sym)])
}

// Parse parses the given expression and returns a pluralForm function.
func (p *peParser) Parse(exp []byte) (retForm pluralForm, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(pError); ok {
				retForm = nil
				retErr = err
				return
			}
			panic(r)
		}
	}()
	p.exp = exp
	p.sym = []byte("")
	pF := p.pExpression()
	p.dewhitespace()
	if len(p.exp) != 0 {
		p.error("Trailing characters")
	}
	return pF, nil
}

// pExpression tries to parse an expression.
func (p *peParser) pExpression() pluralForm {
	pri := p.pOred()
	if p.accept("?") {
		sec := p.pExpression()
		p.expect(":")
		ter := p.pExpression()
		return func(n int) int {
			if pri(n) > 0 {
				return sec(n)
			} else {
				return ter(n)
			}
		}
	}
	return pri
}

// pOred tries to parse an ored expression.
func (p *peParser) pOred() pluralForm {
	fst := p.pAnded()
	if p.accept("||") {
		snd := p.pAnded()
		return func(n int) int {
			if fst(n) > 0 || snd(n) > 0 {
				return 1
			}
			return 0
		}
	}
	return fst
}

// pAnded tries to parse an anded expression.
func (p *peParser) pAnded() pluralForm {
	fst := p.pEquality()
	if p.accept("&&") {
		snd := p.pEquality()
		return func(n int) int {
			if fst(n) > 0 && snd(n) > 0 {
				return 1
			}
			return 0
		}
	}
	return fst
}

// pEquality tries to parse an equality expression.
func (p *peParser) pEquality() pluralForm {
	fst := p.pInEquality()
	if p.accept("==") || p.accept("!=") {
		sym := p.sym
		snd := p.pInEquality()
		return func(n int) int {
			if string(sym) == "==" {
				if fst(n) == snd(n) {
					return 1
				}
			} else if fst(n) != snd(n) {
				return 1
			}
			return 0
		}
	}
	return fst
}

// pInEquality tries to parse an inequality expression.
func (p *peParser) pInEquality() pluralForm {
	fst := p.pProduct()
	if p.accept("<=") || p.accept(">=") || p.accept(">") || p.accept("<") {
		sym := p.sym
		snd := p.pProduct()
		return func(n int) int {
			switch string(sym) {
			case ">=":
				if fst(n) >= snd(n) {
					return 1
				}
			case "<=":
				if fst(n) <= snd(n) {
					return 1
				}
			case ">":
				if fst(n) > snd(n) {
					return 1
				}
			case "<":
				if fst(n) < snd(n) {
					return 1
				}
			}
			return 0
		}
	}
	return fst
}

// pProduct tries to parse a product expression.
func (p *peParser) pProduct() pluralForm {
	fst := p.pFactor()
	if p.accept("%") {
		snd := p.pFactor()
		return func(n int) int {
			return fst(n) % snd(n)
		}
	}
	return fst
}

// pFactor tries to parse a factor expression.
func (p *peParser) pFactor() pluralForm {
	switch {
	case p.accept("n"):
		return func(n int) int { return n }
	case p.accept("("):
		exp := p.pExpression()
		p.expect(")")
		return exp
	default:
		return p.pNumber()
	}
}

// pNumber tries to parse a number expression.
func (p *peParser) pNumber() pluralForm {
	number := make([]byte, 0)
	any := true
	for any {
		any = false
		for c := 0; c < 10; c++ {
			digit := strconv.Itoa(c)
			if p.accept(digit) {
				any = true
				number = append(number, digit[0])
				break
			}
		}
	}
	r, err := strconv.Atoi(string(number))
	if err != nil {
		p.error("Could not parse number %q: %v", number, err)
	}
	return func(n int) int {
		return r
	}
}
