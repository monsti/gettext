// This file is part of monsti/gettext.
// Copyright 2013 Christian Neumann

// monsti/gettext is free software: you can redistribute it and/or modify it under
// the terms of the GNU Lesser General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.

// monsti/gettext is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the GNU Lesser General Public License for more
// details.

// You should have received a copy of the GNU Lesser General Public License
// along with monsti/gettext. If not, see <http://www.gnu.org/licenses/>.

package gettext

import (
	"os"
	"path/filepath"
	"testing"
)

func setupLocales(t *testing.T) *Locales {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get working directory.")
	}

	locales := Locales{
		LocaleDir: filepath.Join(pwd, "test_locale")}
	return &locales
}

func TestNull(t *testing.T) {
	var locales Locales
	G, GN, _, _ := locales.Use("test", "de")
	translated := G("Message")
	if translated != "Message" {
		t.Errorf(`Translation of "Message" should be "Message", got %q`,
			translated)
	}
	translated = GN("Singular", "Plural", 2)
	if translated != "Plural" {
		t.Errorf(`Translation of "Singular", "Plural", 2 should be "Plural", got %q`,
			translated)
	}
}

func TestSingular(t *testing.T) {
	G, _, _, _ := setupLocales(t).Use("test", "de")
	translated := G("Message")
	if translated != "Translated Message" {
		t.Errorf(`Translation of "Message" should be "Translated", got %q`,
			translated)
	}
	translated = G("Unknown Message")
	if translated != "Unknown Message" {
		t.Errorf(`Translation of "Unknown Message" should be "Unknown Message", got %q`,
			translated)
	}
}

func TestPlural(t *testing.T) {
	_, GN, _, _ := setupLocales(t).Use("test", "de")
	tests := []struct {
		Singular, Plural string
		N                int
		Translated       string
	}{
		{"Unknown Singular", "Plural", 1, "Unknown Singular"},
		{"Unknown Singular", "Plural", 2, "Plural"},
		{"Singular", "Plural", 0, "Translated Plural"},
		{"Singular", "Plural", 1, "Translated Singular"},
		{"Singular", "Plural", 3, "Translated Second Plural"},
		{"Singular", "Plural", 5, "Translated Plural"},
	}
	for _, test := range tests {
		ret := GN(test.Singular, test.Plural, test.N)
		if ret != test.Translated {
			t.Errorf(`Translation of (%q, %q, %v) should be %q, got %q`,
				test.Singular, test.Plural, test.N, test.Translated, ret)
		}
	}
}
