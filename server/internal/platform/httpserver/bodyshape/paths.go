package bodyshape

import (
	"cmp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The paths of the problems: kept while a body is read, cut short when
// they are long, each problem reported once, and the problems put in order.

// maxProblems bounds the problems of one body. A body can have a problem
// every few bytes: listing them all could cost the square of the body (M2
// design 3.11).
const maxProblems = 16

// maxPath bounds a reported path, in bytes. A name is the client's and can be
// as long as the body, and every problem under it repeats it: a path longer
// than this is cut short and ends with an ellipsis, so the paths of one
// answer are at most maxProblems × maxPath bytes, whatever the body (M2
// design 3.11). The contract's paths are tens of bytes.
const maxPath = 256

// ellipsis ends a path that was cut short.
const ellipsis = "…"

// problems collects the problems of one body while it is read. It keeps the
// path of the value being read as text, cut after maxPath+1 bytes, and where
// each segment began. Writing a path for every value would cost the square of
// the depth, and the whole path for every problem the problems times the
// name. This way a report copies at most maxPath bytes and compares them with
// the paths already reported, at most maxProblems of them, however long or
// deep the path, so a body can repeat one problem as often as it likes.
type problems struct {
	path  []byte // names joined by dots, indexes in brackets, e.g. tags[1].name
	marks []int  // len(path) before each segment, for leave
	shown []byte // at's buffer for a path cut short
	errs  []FieldError
}

// enter adds a member's name to the path.
func (p *problems) enter(name string) {
	p.marks = append(p.marks, len(p.path))
	if len(p.path) > 0 {
		p.path = append(p.path, '.')
	}
	p.path = append(p.path, name[:min(len(name), maxPath+1)]...) // no more than clip keeps
	p.clip()
}

// enterItem adds an item's index to the path.
func (p *problems) enterItem(i int) {
	p.marks = append(p.marks, len(p.path))
	p.path = append(strconv.AppendInt(append(p.path, '['), int64(i), 10), ']')
	p.clip()
}

// leave takes the last segment off the path.
func (p *problems) leave() {
	p.path = p.path[:p.marks[len(p.marks)-1]]
	p.marks = p.marks[:len(p.marks)-1]
}

// clip keeps the first maxPath+1 bytes of the path: one more than a reported
// path has, which tells at that the path is longer.
func (p *problems) clip() { p.path = p.path[:min(len(p.path), maxPath+1)] }

// full reports whether maxProblems have been reported: the rest is not read.
func (p *problems) full() bool { return len(p.errs) >= maxProblems }

// report adds a problem at the current path, unless p is full or has it
// already. A problem is reported once: a name repeated in its object can
// break the same rule each time, and two long paths can be cut short to the
// same text, which a client could not tell apart anyway. A repeat does not
// count against maxProblems.
func (p *problems) report(code string) {
	if p.full() {
		return
	}
	path := p.at()
	for _, f := range p.errs {
		if f.Field == string(path) && f.Code == code {
			return
		}
	}
	p.errs = append(p.errs, FieldError{string(path), code})
}

// at returns the current path as it is reported; the body itself is the
// empty path. A path longer than maxPath is cut short: its first bytes, cut
// at a rune boundary, then an ellipsis, maxPath bytes in all at most. The
// path's names are decoded, so it is UTF-8 up to the cut. The result is valid
// until the path changes.
func (p *problems) at() []byte {
	if len(p.path) <= maxPath {
		return p.path
	}
	cut := maxPath - len(ellipsis)
	for !utf8.RuneStart(p.path[cut]) {
		cut--
	}
	p.shown = append(append(p.shown[:0], p.path[:cut]...), ellipsis...)
	return p.shown
}

// comparePaths orders two reported paths as a reader counts: byte by byte,
// but an index against an index by its number, so tags[2] comes before
// tags[10] (M2 handoff 4). Each '[' starts an index, the digits after it:
// a member name with a '[' and digits in it is ordered the same way, which
// keeps the order total.
func comparePaths(a, b string) int {
	for a != "" && b != "" {
		if a[0] == '[' && b[0] == '[' {
			var x, y string
			x, a = cutDigits(a[1:])
			y, b = cutDigits(b[1:])
			if c := cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y)); c != 0 {
				return c
			}
			continue
		}
		if a[0] != b[0] {
			return cmp.Compare(a[0], b[0])
		}
		a, b = a[1:], b[1:]
	}
	return cmp.Compare(len(a), len(b))
}

// cutDigits is the digits s starts with, and the rest of s.
func cutDigits(s string) (digits, rest string) {
	i := 0
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:]
}
