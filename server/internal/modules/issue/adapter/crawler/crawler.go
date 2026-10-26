package crawler

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"mime"
	"net/url"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
)

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
func (c *Crawler) Crawl(parent context.Context, rawURL string) domain.LinkMetadata {
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
		return domain.LinkMetadata{"error": &public, "url": &rawURL}
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
		return domain.LinkMetadata{"error": &failed, "url": &rawURL}
	}
	title, href, err := pageMetadata(page.Body)
	if err != nil {
		c.diagnose(ctx, "page", "non_html", nil)
		failed := "failed"
		return domain.LinkMetadata{"error": &failed, "url": &rawURL}
	}
	result := domain.LinkMetadata{"title": title, "favicon": nil, "url": &rawURL, "favicon_url": nil}
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
