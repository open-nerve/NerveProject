// Package linkpreview fetches a link's preview, the page title and one icon,
// through the outbound client's address policy (M4 design 3.11). Work-item
// and module links share it (M6 design 3.2). Like all of the platform it
// imports no module and not internal/shared, so it returns its own Metadata,
// which bootstrap hands to the modules as shared.LinkMetadata.
package linkpreview

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"mime"
	"net/url"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
)

// Metadata is a crawl's result: title, favicon, url and favicon_url when the
// page was fetched (a missing title or icon is nil), or error ("blocked" or
// "failed") and url when it was not.
type Metadata map[string]*string

// Crawler fetches a page and at most one icon through the same outbound policy.
type Crawler struct {
	client *outbound.Client
	logger *slog.Logger
}

func New(client *outbound.Client, logger *slog.Logger) *Crawler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Crawler{client: client, logger: logger}
}

// Crawl retains the task URL and shares one deadline across both resources.
func (c *Crawler) Crawl(parent context.Context, rawURL string) Metadata {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	page, err := c.client.Get(ctx, rawURL, limits(1<<20))
	if err != nil {
		category := errorCategory(err)
		c.diagnose(ctx, "page", category, err)
		public := "failed"
		if category == "blocked" {
			public = "blocked"
		}
		return Metadata{"error": &public, "url": &rawURL}
	}
	category := ""
	if page.StatusCode < 200 || page.StatusCode >= 300 {
		category = "http_status"
	} else if mediaType(page.ContentType) != "text/html" {
		category = "non_html"
	}
	if category != "" {
		c.diagnose(ctx, "page", category, nil)
		failed := "failed"
		return Metadata{"error": &failed, "url": &rawURL}
	}
	title, href, err := pageMetadata(page.Body, page.ContentType)
	if err != nil {
		c.diagnose(ctx, "page", "non_html", nil)
		failed := "failed"
		return Metadata{"error": &failed, "url": &rawURL}
	}
	result := Metadata{"title": title, "favicon": nil, "url": &rawURL, "favicon_url": nil}
	base, err := url.Parse(page.FinalURL)
	if err != nil {
		c.diagnose(ctx, "icon", "blocked", nil)
		return result
	}
	if href == "" {
		href = "/favicon.ico"
	}
	iconURL, err := base.Parse(href)
	if err != nil {
		c.diagnose(ctx, "icon", "blocked", nil)
		return result
	}
	icon, err := c.client.Get(ctx, iconURL.String(), limits(64<<10))
	if err != nil {
		c.diagnose(ctx, "icon", errorCategory(err), err)
		return result
	}
	if icon.StatusCode < 200 || icon.StatusCode >= 300 {
		c.diagnose(ctx, "icon", "http_status", nil)
		return result
	}
	contentType := mediaType(icon.ContentType)
	if !iconType(contentType) {
		c.diagnose(ctx, "icon", "invalid_icon_type", nil)
		return result
	}
	dataURL := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(icon.Body)
	result["favicon"], result["favicon_url"] = &dataURL, &icon.FinalURL
	return result
}

func limits(maxBytes int64) outbound.Limits {
	return outbound.Limits{ConnectTimeout: 2 * time.Second, Timeout: 5 * time.Second, MaxBytes: maxBytes}
}

func mediaType(raw string) string {
	parsed, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return ""
	}
	return parsed
}

func iconType(mime string) bool {
	switch mime {
	case "image/png", "image/x-icon", "image/vnd.microsoft.icon", "image/gif", "image/jpeg", "image/webp":
		return true
	}
	return false
}

func errorCategory(err error) string {
	switch {
	case errors.Is(err, outbound.ErrBlocked):
		return "blocked"
	case errors.Is(err, outbound.ErrTimeout):
		return "timeout"
	case errors.Is(err, outbound.ErrTooLarge):
		return "too_large"
	case errors.Is(err, outbound.ErrUnreachable):
		return "unreachable"
	default:
		return "unreachable"
	}
}

// diagnose logs only controlled fields; error text may contain credentials.
func (c *Crawler) diagnose(ctx context.Context, stage, category string, err error) {
	attrs := []slog.Attr{slog.String("stage", stage), slog.String("category", category)}
	if outbound.Reason(err) == "redirect_limit" {
		attrs = append(attrs, slog.String("reason", "redirect_limit"))
	}
	c.logger.LogAttrs(ctx, slog.LevelWarn, "link metadata fetch failed", attrs...)
}
