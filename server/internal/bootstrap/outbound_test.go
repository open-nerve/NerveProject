package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
)

// Ignoring a parse error, dropping the loaded allowlist or version, and
// widening /32 to /8 each change a real constructor/GET result below.
func TestNewOutbound(t *testing.T) {
	t.Run("invalid direct config", func(t *testing.T) {
		for _, value := range []string{"broken", "127.0.0.1", "::1/129"} {
			t.Run(value, func(t *testing.T) {
				cfg := config.Config{Env: config.EnvTest, Outbound: config.OutboundConfig{AllowCIDRs: []string{"127.0.0.1/32", value}}}
				client, err := newOutbound(cfg, "test-construction-version")
				if err == nil || client != nil || errors.Unwrap(err) == nil {
					t.Fatalf("newOutbound() client is nil = %t, error = %v; want wrapped parse error and no client", client == nil, err)
				}
			})
		}
	})
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.UserAgent()))
	}))
	defer fixture.Close()
	limits := outbound.Limits{ConnectTimeout: time.Second, Timeout: 2 * time.Second, MaxBytes: 1024}
	t.Run("empty allowlist blocks local fixture", func(t *testing.T) {
		client, err := newOutbound(config.Config{Env: config.EnvTest}, "test-construction-version")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Get(t.Context(), fixture.URL, limits); !errors.Is(err, outbound.ErrBlocked) {
			t.Fatalf("Get() error = %v; want blocked with empty allowlist", err)
		}
	})
	t.Run("loaded test profile permits exactly one loopback address", func(t *testing.T) {
		cfg, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{
			"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://test",
		}})
		if err != nil {
			t.Fatal(err)
		}
		client, err := newOutbound(cfg, "test-construction-version")
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Get(t.Context(), fixture.URL, limits)
		if err != nil {
			t.Fatalf("Get(local fixture) = %v", err)
		}
		if response.StatusCode != http.StatusOK || string(response.Body) != "Nerve/test-construction-version (+link preview)" {
			t.Fatalf("fixture response = %+v; want successful request with supplied version in User-Agent", response)
		}
		other := strings.Replace(fixture.URL, "127.0.0.1", "127.0.0.2", 1)
		if _, err := client.Get(t.Context(), other, limits); !errors.Is(err, outbound.ErrBlocked) {
			t.Fatalf("Get(127.0.0.2) error = %v; want blocked before dialing", err)
		}
	})
}
