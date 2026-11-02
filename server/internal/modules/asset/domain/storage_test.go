package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
	"uuid"
)

func TestObjectKeyAndGrantBounds(t *testing.T) {
	asset := uuid.MustParse("28fe241a-34a8-4d94-a769-116a4cd80d15")
	object := uuid.MustParse("a34c291b-e277-43e2-acce-78a2a7ff053e")
	u := ObjectKey("u-28fe241a34a84d94a769116a4cd80d15-a34c291be27743e2acce78a2a7ff053e")
	o := ObjectKey("o-28fe241a34a84d94a769116a4cd80d15-a34c291be27743e2acce78a2a7ff053e")
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	t.Run("canonical key preserves both UUIDs", func(t *testing.T) {
		for _, want := range []ObjectKey{u, o} {
			key, err := NewObjectKey(string(want)[0], asset, object)
			if err != nil || key != want {
				t.Fatalf("NewObjectKey = %q, %v; want %q", key, err, want)
			}
			parsed, err := ParseObjectKey(string(key))
			if err != nil || parsed != key || parsed.Kind() != string(want)[0] || parsed.AssetID() != asset || parsed.ObjectID() != object {
				t.Fatalf("key round trip = %q, %v", parsed, err)
			}
		}
	})
	t.Run("reject invalid keys and never echo them", func(t *testing.T) {
		for _, raw := range []string{
			"", "../secret", string(u) + "/secret", strings.ReplaceAll(string(u), "a", "A"),
			"x" + string(u)[1:], "u-00000000000000000000000000000000-" + string(u)[35:],
			string(u)[:35] + "00000000000000000000000000000000", string(u)[:34] + "%2f" + string(u)[35:],
			string(u)[:66], string(u) + "0", string(u)[:2] + asset.String() + "-" + object.String(),
			strings.Replace(string(u), "a", "g", 1),
		} {
			key, err := ParseObjectKey(raw)
			if !errors.Is(err, ErrInvalidObjectKey) || key != "" || err.Error() != ErrInvalidObjectKey.Error() {
				t.Errorf("ParseObjectKey(%q) = %q, %v", raw, key, err)
			}
			if ObjectKey(raw).Kind() != 0 || ObjectKey(raw).AssetID() != uuid.Nil() || ObjectKey(raw).ObjectID() != uuid.Nil() {
				t.Errorf("invalid key %q exposes a usable identity", raw)
			}
		}
		for _, args := range []struct {
			kind          byte
			asset, object uuid.UUID
		}{
			{'x', asset, object}, {'u', uuid.Nil(), object}, {'o', asset, uuid.Nil()},
		} {
			if key, err := NewObjectKey(args.kind, args.asset, args.object); key != "" || !errors.Is(err, ErrInvalidObjectKey) {
				t.Errorf("NewObjectKey invalid = %q, %v", key, err)
			}
		}
	})
	t.Run("upload grants", func(t *testing.T) {
		valid := UploadGrant{Key: u, Size: 5242880, ContentType: "application/octet-stream", ExpiresAt: now.Add(15 * time.Minute)}
		if err := ValidateUpload(valid, 5242880, now); err != nil {
			t.Fatal(err)
		}
		for _, size := range []int64{1, 104857600} {
			g := valid
			g.Size = size
			if err := ValidateUpload(g, 104857600, now); err != nil {
				t.Errorf("size %d: %v", size, err)
			}
		}
		for _, tc := range []struct {
			name   string
			change func(*UploadGrant)
			limit  int64
		}{
			{"empty key", func(g *UploadGrant) { g.Key = "" }, 5242880},
			{"sealed key", func(g *UploadGrant) { g.Key = o }, 5242880},
			{"zero size", func(g *UploadGrant) { g.Size = 0 }, 5242880},
			{"negative size", func(g *UploadGrant) { g.Size = -1 }, 5242880},
			{"past configured limit", func(g *UploadGrant) { g.Size++ }, 5242880},
			{"past hard limit", func(g *UploadGrant) { g.Size = 104857601 }, 104857601},
			{"zero configured limit", func(*UploadGrant) {}, 0},
			{"excess configured limit", func(*UploadGrant) {}, 104857601},
			{"expired exactly", func(g *UploadGrant) { g.ExpiresAt = now }, 5242880},
			{"expired before", func(g *UploadGrant) { g.ExpiresAt = now.Add(-time.Second) }, 5242880},
			{"beyond expiry window", func(g *UploadGrant) { g.ExpiresAt = now.Add(15*time.Minute + time.Nanosecond) }, 5242880},
			{"no type", func(g *UploadGrant) { g.ContentType = "" }, 5242880},
			{"parameter", func(g *UploadGrant) { g.ContentType = "text/plain; charset=utf-8" }, 5242880},
			{"noncanonical type", func(g *UploadGrant) { g.ContentType = "Image/PNG" }, 5242880},
			{"header injection", func(g *UploadGrant) { g.ContentType = "image/png\r\nX: evil" }, 5242880},
			{"wildcard", func(g *UploadGrant) { g.ContentType = "image/*" }, 5242880},
		} {
			t.Run(tc.name, func(t *testing.T) {
				g := valid
				tc.change(&g)
				if err := ValidateUpload(g, tc.limit, now); !errors.Is(err, ErrInvalidUpload) {
					t.Fatalf("error = %v", err)
				}
			})
		}
	})
	t.Run("read grants", func(t *testing.T) {
		valid := ReadGrant{Key: o, Method: "GET", Name: "项目 \"封面\".png", ContentType: "image/png", Inline: true, ExpiresAt: now.Add(65 * time.Minute)}
		for _, method := range []string{"GET", "HEAD"} {
			g := valid
			g.Method = method
			if err := ValidateRead(g, now); err != nil {
				t.Fatalf("%s: %v", method, err)
			}
		}
		for _, media := range []string{"image/jpeg", "image/png", "image/webp", "image/gif"} {
			g := valid
			g.ContentType = media
			if err := ValidateRead(g, now); err != nil {
				t.Errorf("inline %s: %v", media, err)
			}
		}
		g := valid
		g.Inline = false
		g.ContentType = "text/html"
		g.Name = strings.Repeat("图", 255)
		if err := ValidateRead(g, now); err != nil {
			t.Fatalf("attachment: %v", err)
		}
		for _, tc := range []struct {
			name   string
			change func(*ReadGrant)
		}{
			{"staging key", func(g *ReadGrant) { g.Key = u }},
			{"invalid key", func(g *ReadGrant) { g.Key = "../secret" }},
			{"PUT", func(g *ReadGrant) { g.Method = "PUT" }},
			{"lowercase method", func(g *ReadGrant) { g.Method = "get" }},
			{"expired exactly", func(g *ReadGrant) { g.ExpiresAt = now }},
			{"beyond expiry window", func(g *ReadGrant) { g.ExpiresAt = now.Add(65*time.Minute + time.Nanosecond) }},
			{"empty filename", func(g *ReadGrant) { g.Name = "" }},
			{"long filename", func(g *ReadGrant) { g.Name = strings.Repeat("图", 256) }},
			{"invalid UTF8", func(g *ReadGrant) { g.Name = "bad\xff" }},
			{"CRLF filename", func(g *ReadGrant) { g.Name = "a\r\nX: evil" }},
			{"control filename", func(g *ReadGrant) { g.Name = "a\u0085b" }},
			{"slash", func(g *ReadGrant) { g.Name = "a/b" }},
			{"backslash", func(g *ReadGrant) { g.Name = "a\\b" }},
			{"active inline type", func(g *ReadGrant) { g.ContentType = "text/html" }},
			{"SVG inline type", func(g *ReadGrant) { g.ContentType = "image/svg+xml" }},
			{"type parameter", func(g *ReadGrant) { g.ContentType = "image/png; charset=utf-8" }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				g := valid
				tc.change(&g)
				if err := ValidateRead(g, now); !errors.Is(err, ErrInvalidRead) {
					t.Fatalf("error = %v", err)
				}
			})
		}
	})
}
