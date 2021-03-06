// This file is part of monsti/gettext.
// Copyright 2012-2013 Christian Neumann

// monsti/gettext is free software: you can redistribute it and/or modify it under
// the terms of the GNU Lesser General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.

// monsti/gettext is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the GNU Lesser General Public License for more
// details.

// You should have received a copy of the GNU Lesser General Public License
// along with monsti/util. If not, see <http://www.gnu.org/licenses/>.

package gettext

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type message struct {
	Singular, Plural string
}

type translation struct {
	msgs map[message][][]byte
	pf   pluralForm
}

func (t *translation) Singular(msg string) string {
	if t != nil {
		if ret, ok := t.msgs[message{msg, ""}]; ok {
			return string(ret[0])
		}
	}
	return msg
}

func (t *translation) Plural(msg, plural string, n int) string {
	if t != nil {
		if ret, ok := t.msgs[message{msg, plural}]; ok {
			return string(ret[t.pf(n)])
		}
		if n == 1 {
			return msg
		}
	}
	return plural
}

type parseError string

func (p parseError) Error() string {
	return string(p)
}

// parseMO parses GetText MO files
func parseMO(dir, domain, locale string) (retTr *translation, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(parseError); ok {
				retTr = nil
				retErr = e
			} else {
				panic(r)
			}
		}
	}()
	path := filepath.Join(dir, locale, "LC_MESSAGES", domain+".mo")
	error := func(msg string, args ...interface{}) {
		panic(parseError(fmt.Sprintf(msg, args...)))
	}

	f, err := os.Open(path)
	if err != nil {
		error("Could not open message file: %v", err)
	}

	// Determine byte ordering
	var magic [4]byte
	if _, err := f.Read(magic[:]); err != nil {
		error("Could not read magic number: %v", err)
	}
	var bo binary.ByteOrder
	switch magic {
	case [...]byte{0x95, 0x04, 0x12, 0xde}:
		bo = binary.BigEndian
	case [...]byte{0xde, 0x12, 0x04, 0x95}:
		bo = binary.LittleEndian
	default:
		error("Unknown file format: magic 0x%x", magic)
	}
	try := func(target interface{}, msg string) {
		if err := binary.Read(f, bo, target); err != nil {
			error(msg+": %v", err)
		}
	}

	// Parse main variables (n, offsets, major and minor version)
	var major, minor uint16
	try(&major, "Could not parse major version")
	try(&minor, "Could not parse minor version")
	if major > 1 || minor > 1 {
		error("Unknown file format: major %d, minor %d",
			magic, major, minor)
	}
	var n, msgOff, transOff uint32
	try(&n, "Could not parse number of strings")
	try(&msgOff, "Could not parse message offset")
	try(&transOff, "Could not parse translation offset")

	// Parse the messages and their translations
	msgs := make([]struct {
		Length, Offset, TrLength, TrOffset uint32
		Messages, Translations             [][]byte
	}, n)
	if _, err := f.Seek(int64(msgOff), 0); err != nil {
		error("Could not seek message offset: %v", err)
	}
	for i := 0; i < int(n); i++ {
		try(&msgs[i].Length, "Could not parse string length")
		try(&msgs[i].Offset, "Could not parse string offset")
	}
	if _, err := f.Seek(int64(transOff), 0); err != nil {
		error("Could not seek translation offset: %v", err)
	}
	for i := 0; i < int(n); i++ {
		try(&msgs[i].TrLength, "Could not parse translation length")
		try(&msgs[i].TrOffset, "Could not parse translation offset")
	}
	for i := range msgs {
		msg := &msgs[i]
		msg.Messages = make([][]byte, 2)
		buffer := make([]byte, msg.Length)
		if _, err := f.Seek(int64(msg.Offset), 0); err != nil {
			error("Could not seek message offset: %v", err)
		}
		try(&buffer, "Could not read messages")
		msg.Messages = bytes.Split(buffer, []byte{0})
		if len(msg.Messages) == 0 {
			msg.Messages = append(msg.Messages, []byte(""))
		}
	}
	for i := range msgs {
		msg := &msgs[i]
		msg.Translations = make([][]byte, 2)
		buffer := make([]byte, msg.TrLength)
		if _, err := f.Seek(int64(msg.TrOffset), 0); err != nil {
			error("Could not seek translation offset: %v", err)
		}
		try(&buffer, "Could not read translations")
		msg.Translations = bytes.Split(buffer, []byte{0})
	}
	var translation translation
	translation.msgs = make(map[message][][]byte)
	for _, msg := range msgs {
		var plural string
		switch len(msg.Messages) {
		case 1:
		case 2:
			plural = string(msg.Messages[1])
		default:
			error("There shold be one or to messages.")
		}
		translation.msgs[message{string(msg.Messages[0]), plural}] = msg.Translations
	}

	// Get plural forms function
	meta := translation.Singular("")
	begin := strings.Index(meta, "plural= ") + 8
	if begin != 7 {
		end := begin + strings.Index(meta[begin:], ";")
		if end != -1 {
			exp := meta[begin:end]
			var parser peParser
			translation.pf, err = parser.Parse([]byte(exp))
			if err != nil {
				return nil, err
			}
		}
	}
	if translation.pf == nil {
		translation.pf = func(n int) int {
			if n == 1 {
				return 0
			}
			return 1
		}
	}
	return &translation, nil
}

// Locales loads and keeps message catalogs and provides translation functions.
// All methods belonging to Locales are thread safe.
type Locales struct {
	translations map[string]map[string]*translation
	// LocaleDir is the directory to search for message catalogs.
	LocaleDir string
	// Locale is the default locale to use.
	Locale string
	// Domain is the default domain to use.
	Domain string
	mutex  sync.RWMutex
}

// Singular returns the singular translation for the given domain, locale, and
// message.
//
// You have to load the corresponding message catalogs with Use before.
func (l *Locales) Singular(domain, locale, msg string) string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.translations[domain][locale].Singular(msg)
}

// Plural returns the plural translation for the given domain, locale, both
// singular and plural message, and the number n.
//
// You have to load the corresponding message catalogs with Use before.
func (l *Locales) Plural(domain, locale, singular, plural string,
	n int) string {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.translations[domain][locale].Plural(singular, plural, n)
}

// Singular is a function returning a singular translation for the given
// message.
type Singular func(msg string) string

// Plural is a function returning a plural translation for the given
// singular and plural message and the number n.
type Plural func(singular string, plural string, n int) string

// DomainSingular is like Singular but allows to specify a domain.
type DomainSingular func(domain string, message string) string

// DomainPlural is like Plural but allows to specify a domain.
type DomainPlural func(domain string, singular string, plural string,
	n int) string

// Use loads the the translation for the given domain and locale and returns
// translation functions.
// Uses the default domain or locale if the corresponding parameter is an empty
// string.
//
// If the given domain and locale has not been loaded before, Use tries to
// load the corresponding message catalog.
func (l *Locales) Use(domain, locale string) (Singular, Plural,
	DomainSingular, DomainPlural) {
	if len(domain) == 0 {
		domain = l.Domain
	}
	if len(locale) == 0 {
		locale = l.Locale
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.translations == nil {
		l.translations = make(map[string]map[string]*translation)
	}
	if _, ok := l.translations[domain]; !ok {
		l.translations[domain] = make(map[string]*translation)
	}
	if _, ok := l.translations[domain][locale]; !ok {
		ret, err := parseMO(l.LocaleDir, domain, locale)
		if err == nil {
			l.translations[domain][locale] = ret
		}
	}
	singular := func(msg string) string {
		return l.Singular(domain, locale, msg)
	}
	plural := func(msg1, msg2 string, n int) string {
		return l.Plural(domain, locale, msg1, msg2, n)
	}
	dSingular := func(domain, msg string) string {
		return l.Singular(domain, locale, msg)
	}
	dPlural := func(domain, msg1, msg2 string, n int) string {
		return l.Plural(domain, locale, msg1, msg2, n)
	}
	return singular, plural, dSingular, dPlural
}

// Use returns translation functions for the given locale dir, domain, and
// locale. It sets the default locale and domain of DefaultLocales.
func Use(localedir, domain, locale string) (Singular, Plural, DomainSingular, DomainPlural) {
	DefaultLocales.LocaleDir = localedir
	DefaultLocales.Locale = locale
	DefaultLocales.Domain = domain
	return DefaultLocales.Use("", "")
}

var (
	// DefaultLocales is the default locales object.
	DefaultLocales Locales
)
