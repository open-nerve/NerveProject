package domain

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SearchFor limits the relationship picker; the empty value means omitted.
type SearchFor string

// SearchQuery is a validated search input, independent of directories and SQL.
type SearchQuery struct {
	Text                             string
	ProjectID, IssueID               *uuid.UUID
	For                              SearchFor
	Limit                            int
	StartDate, TargetDate            []DateFilter
	Identifier                       string
	IdentifierNumber, SequenceNumber *int
}

// ParseSearchQuery preserves parameter presence and multiplicity. It validates
// syntax only; visibility and source/project relationships belong to the reader.
func ParseSearchQuery(raw map[string][]string) (SearchQuery, error) {
	q := SearchQuery{Limit: shared.DefaultPageSize}
	var fields []shared.FieldError
	for _, key := range slices.Sorted(maps.Keys(raw)) {
		values := raw[key]
		switch key {
		case "q", "project_id", "for", "issue_id", "limit", "start_date", "target_date":
		default:
			fields = append(fields, searchField(key, shared.FieldNotAllowed, "is not allowed for this search"))
			continue
		}
		if key == "start_date" || key == "target_date" {
			conditions, invalid := parseSearchDates(key, values)
			fields = append(fields, invalid...)
			if key == "start_date" {
				q.StartDate = conditions
			} else {
				q.TargetDate = conditions
			}
			continue
		}
		if len(values) != 1 {
			fields = append(fields, searchField(key, shared.FieldInvalidFormat, "must be given once"))
			continue
		}
		value := values[0]
		switch key {
		case "q":
			q.Text = strings.TrimSpace(value)
			if invalid := validateSearchText(q.Text); invalid != nil {
				fields = append(fields, *invalid)
			}
		case "project_id", "issue_id":
			id, err := uuid.Parse(value)
			if err != nil {
				fields = append(fields, searchField(key, shared.FieldInvalidFormat, "must be a UUID"))
			} else if key == "project_id" {
				q.ProjectID = &id
			} else {
				q.IssueID = &id
			}
		case "for":
			switch SearchFor(value) {
			case "parent", "sub_issue", "relation":
				q.For = SearchFor(value)
			default:
				fields = append(fields, searchField(key, shared.FieldNotAllowed, "must be one of parent, sub_issue, relation"))
			}
		case "limit":
			limit, err := strconv.Atoi(value)
			if err != nil {
				return SearchQuery{}, &shared.Error{
					Kind: shared.KindBadRequest, Code: shared.CodeBadRequest, Detail: "The search parameter is malformed.",
					Fields: []shared.FieldError{searchField(key, shared.FieldInvalidFormat, "must have the required parameter format")},
				}
			}
			q.Limit, err = shared.PageSize(&limit)
			if err != nil {
				var problem *shared.Error
				if !errors.As(err, &problem) {
					return SearchQuery{}, err
				}
				fields = append(fields, problem.Fields...)
			}
		}
	}
	_, hasFor := raw["for"]
	_, hasIssue := raw["issue_id"]
	if q.For != "" && !hasIssue {
		fields = append(fields, searchField("issue_id", shared.FieldRequired, "is required when for is provided"))
	}
	if !hasFor && q.IssueID != nil {
		fields = append(fields, searchField("issue_id", shared.FieldNotAllowed, "requires for"))
	}
	if len(fields) > 0 {
		return SearchQuery{}, shared.Invalid(fields...)
	}
	if project, number, ok := ParseIdentifier(q.Text); ok {
		q.Identifier, q.IdentifierNumber = project, &number
	}
	q.SequenceNumber = searchSequence(q.Text)
	return q, nil
}

func searchField(field, code, message string) shared.FieldError {
	return shared.FieldError{Field: field, Code: code, Message: message}
}

func validateSearchText(text string) *shared.FieldError {
	var invalid shared.FieldError
	switch {
	case !utf8.ValidString(text):
		invalid = searchField("q", shared.FieldInvalidFormat, "must be valid UTF-8")
	case strings.ContainsRune(text, 0):
		invalid = searchField("q", shared.FieldInvalidFormat, "must not contain a NUL character")
	case utf8.RuneCountInString(text) > 100:
		invalid = searchField("q", shared.FieldTooLong, "must be at most 100 characters")
	default:
		return nil
	}
	return &invalid
}

func parseSearchDates(key string, values []string) ([]DateFilter, []shared.FieldError) {
	if len(values) == 0 {
		return nil, []shared.FieldError{searchField(key, shared.FieldInvalidFormat, "must have at least one condition")}
	}
	// Report the first none even if an earlier condition has bad syntax.
	if len(values) > 1 {
		if index := slices.Index(values, "none"); index >= 0 {
			return nil, []shared.FieldError{searchField(fmt.Sprintf("%s[%d]", key, index), shared.FieldNotAllowed, "must not be combined with another date condition")}
		}
	}
	conditions := make([]DateFilter, 0, len(values))
	var fields []shared.FieldError
	for i, value := range values {
		condition, err := ParseListDateFilter(value)
		if err != nil {
			fields = append(fields, searchField(fmt.Sprintf("%s[%d]", key, i), shared.FieldInvalidFormat, "must be a date, date range, or none"))
		} else {
			conditions = append(conditions, condition)
		}
	}
	return conditions, fields
}

// Pure sequence terms allow leading zeroes, unlike complete identifiers.
// Invalid or overflowing terms remain available for literal title matching.
func searchSequence(text string) *int {
	if text == "" || strings.TrimLeft(text, "0123456789") != "" {
		return nil
	}
	number, err := strconv.ParseInt(text, 10, 32)
	if err != nil || number < 1 {
		return nil
	}
	sequence := int(number)
	return &sequence
}
