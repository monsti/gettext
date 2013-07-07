// This file is part of monsti/gettext.
// Copyright 2012-2013 Christian Neumann

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
	"testing"
)

func TestParse(t *testing.T) {
	parser := peParser{}
	tests := []struct {
		exp string
		n   []int
		ret []int
	}{
		{"0 ", []int{0, 1, 2}, []int{0, 0, 0}},
		{"  n ", []int{0, 1, 2}, []int{0, 1, 2}},
		{"n %  10", []int{2, 11, 32}, []int{2, 1, 2}},
		{"532 % n", []int{1, 10, 100, 1000}, []int{0, 2, 32, 532}},
		{"(n % (( 10 % 4 )))", []int{2, 11, 32}, []int{0, 1, 0}},
		{"11 == n", []int{2, 11, 32}, []int{0, 1, 0}},
		{"n != 11", []int{2, 11, 32}, []int{1, 0, 1}},
		{"n >= 11", []int{1, 11, 12}, []int{0, 1, 1}},
		{"n <= 11", []int{1, 11, 12}, []int{1, 1, 0}},
		{"n > 11", []int{1, 11, 12}, []int{0, 0, 1}},
		{"n < 11", []int{1, 11, 12}, []int{1, 0, 0}},
		{"n && 1", []int{1, 0, 12}, []int{1, 0, 1}},
		{"n || 1", []int{1, 0, 12}, []int{1, 1, 1}},
		{"0 || n", []int{1, 0, 12}, []int{1, 0, 1}},
		{"n ? 1 : 2", []int{1, 0}, []int{1, 2}},
		{"n ? 0 ? 1 : 3 : 2", []int{1, 0}, []int{3, 2}},
	}
	for _, test := range tests {
		pF, err := parser.Parse([]byte(test.exp))
		if err != nil {
			t.Fatalf("Could not parse %q: %v", test.exp, err)
		} else {
			for i := range test.n {
				ret := pF(test.n[i])
				if ret != test.ret[i] {
					t.Errorf("%q with n = %v should be %v, got %v",
						test.exp, test.n[i], test.ret[i], ret)
				}
			}
		}
	}
}
