package app

import "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"

// HTMLSanitizer produces safe editor content without rejecting a write.
type HTMLSanitizer interface {
	Clean(input string) domain.RichText
}
