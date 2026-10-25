package shared

// Page sizes of a list (api/common.yaml Limit).
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// PageSize returns the page size that limit asks for: DefaultPageSize when
// it is nil. Outside 1–MaxPageSize it is 422 validation_failed on limit
// (M2 design 3.11); a limit that is no integer never gets here, the
// parameter binding answers 400.
func PageSize(limit *int) (int, error) {
	switch {
	case limit == nil:
		return DefaultPageSize, nil
	case *limit < 1 || *limit > MaxPageSize:
		return 0, Invalid(FieldError{Field: "limit", Code: FieldOutOfRange, Message: "must be between 1 and 100"})
	}
	return *limit, nil
}
