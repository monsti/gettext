// This file is part of monsti/gettext.
// Copyright 2013 Christian Neumann <cneumann@datenkarussell.de>

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

/*
Package gettext provides a multi domain and multi language gettext parser.

It supports languages with different plural forms as specified in the language
catalog.

Basic usage:

	G, GN, GD, _ := gettext.Use("/usr/share/locale", "my-program",
		os.Getenv("LC_ALL"))
	
	fmt.Println(G("He: Hello World")
	fmt.Println(GN("World: Hey!", "She: What world, there are %d", n))
	fmt.Println(GD("gimp", "Refusing to modify this really nice painting."))

	Git, _, _, _ := gettext.DefaultLocales.Use("", "it")
	fmt.Println(Git("Hello World!")

*/
package gettext
