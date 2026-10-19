package domain

import (
	"strconv"
	"strings"
)

// A work item's identifier is its project's identifier, a hyphen, and its
// number: WEB-12 (M4 design 3.7). The project's identifier is compared
// folded (foldIdentifier), so that one asked for in any case, Turkish
// letters too, finds its project.

// ParseIdentifier splits the identifier s of a work item into its
// project's identifier and its number: at its last hyphen, the project's
// identifier not empty, the number in decimal digits without a sign or a
// leading zero, from 1 to 2147483647, the range of a work item's number.
// ok is false for anything else.
func ParseIdentifier(s string) (project string, number int, ok bool) {
	project, digits, found := strings.CutLast(s, "-")
	if !found || project == "" || digits == "" || digits[0] == '0' || strings.TrimLeft(digits, "0123456789") != "" {
		return "", 0, false
	}
	n, err := strconv.ParseInt(digits, 10, 32)
	if err != nil {
		return "", 0, false
	}
	return project, int(n), true
}

// MatchIdentifier is the index among identifiers, a workspace's projects'
// identifiers, of the project's identifier asked, compared folded
// (foldIdentifier): the only one that folds as asked does; of two or more,
// the one that is asked in upper case letter for letter (strings.ToUpper),
// as a project's identifier is stored. ok is false when none folds as
// asked does, or when two or more do and none is asked in upper case: the
// identifiers I and İ tell apart, which strings.ToUpper keeps apart (i is
// I, ı is I, İ is İ) and folding does not (M4 design 3.7).
func MatchIdentifier(asked string, identifiers []string) (index int, ok bool) {
	folded, upper := foldIdentifier(asked), strings.ToUpper(asked)
	index, matches := -1, 0
	for i, id := range identifiers {
		if foldIdentifier(id) != folded {
			continue
		}
		matches++
		if index < 0 || id == upper {
			index = i
		}
	}
	if matches > 1 && identifiers[index] != upper {
		return 0, false
	}
	return index, matches > 0
}

// foldIdentifier is s as identifiers compare: a–z as A–Z, çşğöü as
// ÇŞĞÖÜ, and i, ı, I and İ as one letter, I; every other character as it
// is. A project's identifier holds A–Z, 0–9 and ÇŞĞİÖÜ (M3 design 3.19).
func foldIdentifier(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		case r == 'ı' || r == 'İ':
			return 'I'
		}
		switch r {
		case 'ç':
			return 'Ç'
		case 'ş':
			return 'Ş'
		case 'ğ':
			return 'Ğ'
		case 'ö':
			return 'Ö'
		case 'ü':
			return 'Ü'
		}
		return r
	}, s)
}
