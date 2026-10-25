package domain

import (
	"fmt"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DateFilter is one closed date condition. None matches missing dates;
// nil bounds are open ends. Query parsers keep repeated conditions as AND.
type DateFilter struct {
	None     bool
	From, To *shared.Date
}

// ParseListDateFilter reads one day, closed or open-ended range, or none.
// It is the shared lexical parser for query and display-settings filters;
// callers map errors to their parameter or preference field paths.
func ParseListDateFilter(value string) (DateFilter, error) {
	if value == "none" {
		return DateFilter{None: true}, nil
	}
	from, to, ranged := strings.Cut(value, "..")
	if !ranged {
		day, err := shared.ParseDate(value)
		if err != nil {
			return DateFilter{}, err
		}
		return DateFilter{From: &day, To: &day}, nil
	}
	if from == "" && to == "" {
		return DateFilter{}, fmt.Errorf("a date range must have an endpoint")
	}
	var filter DateFilter
	if from != "" {
		day, err := shared.ParseDate(from)
		if err != nil {
			return DateFilter{}, err
		}
		filter.From = &day
	}
	if to != "" {
		day, err := shared.ParseDate(to)
		if err != nil {
			return DateFilter{}, err
		}
		filter.To = &day
	}
	if filter.From != nil && filter.To != nil && filter.From.Compare(*filter.To) > 0 {
		return DateFilter{}, fmt.Errorf("a date range must not end before it starts")
	}
	return filter, nil
}
