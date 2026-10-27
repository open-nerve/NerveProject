package domain

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Link is a live work-item link and its current preview metadata.
type Link struct {
	ID, WorkspaceID, ProjectID, IssueID uuid.UUID
	URL                                 string
	Title                               *string
	Metadata                            LinkMetadata
	CreatedByID                         *uuid.UUID
	CreatedAt, UpdatedAt                time.Time
}

// NewLink contains the caller's URL and optional title.
type NewLink struct {
	URL   string
	Title *string
}

// LinkPatch distinguishes an omitted title from an explicit null.
type LinkPatch struct {
	URL      *string
	SetTitle bool
	Title    *string
}

// CheckNewLink normalizes only surrounding URL whitespace and an omitted
// HTTP scheme, preserving the exact stored URL and title otherwise.
func CheckNewLink(n NewLink) (NewLink, error) {
	u, problem := checkLinkURL(n.URL)
	if err := invalid(problem, checkLinkTitle(n.Title)); err != nil {
		return NewLink{}, err
	}
	n.URL = u
	return n, nil
}

// CheckLinkPatch checks only supplied fields and preserves title presence.
func CheckLinkPatch(p LinkPatch) (LinkPatch, error) {
	var found []*shared.FieldError
	if p.URL != nil {
		u, problem := checkLinkURL(*p.URL)
		p.URL = &u
		found = append(found, problem)
	}
	if p.SetTitle {
		found = append(found, checkLinkTitle(p.Title))
	}
	if err := invalid(found...); err != nil {
		return LinkPatch{}, err
	}
	return p, nil
}

func checkLinkTitle(title *string) *shared.FieldError {
	if title == nil {
		return nil
	}
	if utf8.RuneCountInString(*title) > 255 {
		return &shared.FieldError{Field: "title", Code: shared.FieldTooLong, Message: "must be at most 255 characters"}
	}
	if strings.ContainsRune(*title, 0) {
		return &shared.FieldError{Field: "title", Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

func checkLinkURL(raw string) (string, *shared.FieldError) {
	refused := &shared.FieldError{Field: "url", Code: shared.FieldNotAllowed, Message: "must be an HTTP or HTTPS URL without credentials, zones or control characters, of at most 2048 characters"}
	value := strings.TrimSpace(raw)
	if len(value) >= 7 && strings.EqualFold(value[:7], "http://") || len(value) >= 8 && strings.EqualFold(value[:8], "https://") {
		// Parse a copy below; the caller's casing remains the uniqueness key.
	} else if prefix, _, found := strings.Cut(value, "://"); found && linkScheme(prefix) {
		return "", refused
	} else {
		value = "http://" + value
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 2048 || strings.ContainsFunc(value, unicode.IsControl) {
		return "", refused
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || strings.Contains(u.Hostname(), "%") {
		return "", refused
	}
	if strings.Contains(u.Hostname(), ":") || strings.HasPrefix(u.Host, "[") {
		ip, err := netip.ParseAddr(u.Hostname())
		if err != nil || !ip.Is6() || !strings.HasPrefix(u.Host, "[") {
			return "", refused
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return "", refused
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return "", refused
		}
	}
	return value, nil
}

// linkScheme recognizes only a leading RFC scheme; path/query :// is data.
func linkScheme(s string) bool {
	for n, c := range s {
		letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		rest := c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'
		if !letter && (n == 0 || !rest) {
			return false
		}
	}
	return s != ""
}
