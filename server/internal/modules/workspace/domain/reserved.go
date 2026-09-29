package domain

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed reserved_slugs.txt
var reservedFile string

// ReservedSlugs is the reserved list's three sections (M3 design 3.10).
type ReservedSlugs struct {
	App      []string // the web app's top-level route segments and public/'s top-level directories
	Server   []string // the top-level paths the server answers itself, beside the web app's pages
	Reserved []string // names held for top-level pages to come
}

// All is the three sections together.
func (r ReservedSlugs) All() []string {
	return slices.Concat(r.App, r.Server, r.Reserved)
}

// reserved is the embedded list, parsed once. The file is part of the
// binary, so a list that does not parse stops every nerve at start.
var reserved = mustParseReserved(reservedFile)

// Reserved returns the reserved list.
func Reserved() ReservedSlugs {
	return ReservedSlugs{App: slices.Clone(reserved.App), Server: slices.Clone(reserved.Server), Reserved: slices.Clone(reserved.Reserved)}
}

func mustParseReserved(text string) ReservedSlugs {
	r, err := parseReserved(text)
	if err != nil {
		panic("workspace: reserved_slugs.txt: " + err.Error())
	}
	return r
}

// parseReserved reads the list: sections [app], [server] and [reserved],
// one name per line under each; blank lines and lines starting with # are
// comments. The list is strict, since nerve cannot start with a list that
// does not parse, and a typo must not become a name: a line starting with [
// is one of the three headers; a name is spelled as a slug, which leaves no
// room for a comment after it; no section is empty; no name is listed twice,
// in one section or in two.
func parseReserved(text string) (ReservedSlugs, error) {
	var r ReservedSlugs
	var section *[]string
	seen := map[string]bool{}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "":
			continue
		case "[app]":
			section = &r.App
		case "[server]":
			section = &r.Server
		case "[reserved]":
			section = &r.Reserved
		default:
			switch {
			case strings.HasPrefix(line, "#"):
				continue
			case strings.HasPrefix(line, "["):
				return ReservedSlugs{}, fmt.Errorf("line %d: %q is not [app], [server] or [reserved]", i+1, line)
			case section == nil:
				return ReservedSlugs{}, fmt.Errorf("line %d: %q is in no section", i+1, line)
			case !spelledAsSlug(line):
				return ReservedSlugs{}, fmt.Errorf("line %d: %q is not spelled as a slug", i+1, line)
			case seen[line]:
				return ReservedSlugs{}, fmt.Errorf("line %d: %q is listed twice", i+1, line)
			}
			seen[line] = true
			*section = append(*section, line)
		}
	}
	for _, s := range []struct {
		header string
		names  []string
	}{{"[app]", r.App}, {"[server]", r.Server}, {"[reserved]", r.Reserved}} {
		if len(s.names) == 0 {
			return ReservedSlugs{}, fmt.Errorf("section %s is empty", s.header)
		}
	}
	return r, nil
}

// isReserved reports whether slug is on the list, in any section. Its
// callers pass a slug CheckSlug has found spelled as one, lower case, so an
// exact match is the whole comparison.
func isReserved(slug string) bool {
	return slices.Contains(reserved.All(), slug)
}
