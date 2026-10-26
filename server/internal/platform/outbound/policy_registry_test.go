package outbound

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestPolicyRegistryCoverage(t *testing.T) {
	expected := registryPrefixes(t)
	p := mustPolicy(t, nil, nil)
	actual := make(map[netip.Prefix]bool)
	for _, prefix := range p.deny {
		if actual[prefix] {
			t.Errorf("duplicate production prefix %s", prefix)
		}
		actual[prefix] = true
	}
	for _, prefix := range expected {
		if !actual[prefix] {
			t.Errorf("missing production prefix %s", prefix)
		}
		delete(actual, prefix)
	}
	for prefix := range actual {
		t.Errorf("production prefix absent from independent registry and extras: %s", prefix)
	}
}

func TestPolicyEndpoints(t *testing.T) {
	p := mustPolicy(t, nil, nil)
	for _, prefix := range registryPrefixes(t) {
		t.Run(prefix.String(), func(t *testing.T) {
			first, last := prefixEndpoints(t, prefix)
			if prefix == netip.MustParsePrefix("::ffff:0:0/96") && (!first.Is4In6() || !last.Is4In6()) {
				t.Fatal("mapped endpoints lost their original IPv6 representation")
			}
			t.Run("first", func(t *testing.T) { wantBlocked(t, p, first) })
			t.Run("last", func(t *testing.T) { wantBlocked(t, p, last) })
		})
	}
}

// registryPrefixes reads only the independently downloaded official CSVs.
// Keep every row, including globally reachable, terminated and overlapping rows.
func registryPrefixes(t *testing.T) []netip.Prefix {
	t.Helper()
	type source struct {
		File         string `json:"file"`
		URL          string `json:"url"`
		AccessedAt   string `json:"accessed_at"`
		EffectiveURL string `json:"effective_url"`
		Status       int    `json:"status"`
		Size         int    `json:"size"`
		SHA256       string `json:"sha256"`
	}
	data, err := os.ReadFile("testdata/sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var sources []source
	if err := json.Unmarshal(data, &sources); err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Fatalf("source count = %d, want 2", len(sources))
	}
	footnote := regexp.MustCompile(`\[[0-9]+\]`)
	var prefixes []netip.Prefix
	seen := make(map[netip.Prefix]bool)
	seenFiles := make(map[string]bool)
	for _, source := range sources {
		version := "ipv4"
		wantPrefixes := 26
		switch source.File {
		case "iana-ipv4-special-registry-1.csv":
		case "iana-ipv6-special-registry-1.csv":
			version, wantPrefixes = "ipv6", 25
		default:
			t.Fatalf("unexpected source file %q", source.File)
		}
		if seenFiles[source.File] {
			t.Fatalf("duplicate source file %q", source.File)
		}
		seenFiles[source.File] = true
		wantURL := "https://www.iana.org/assignments/iana-" + version + "-special-registry/" + source.File
		if source.URL != wantURL || source.EffectiveURL != wantURL || source.Status != 200 {
			t.Fatalf("invalid official provenance: %+v", source)
		}
		if _, err := time.Parse(time.RFC3339Nano, source.AccessedAt); err != nil {
			t.Fatalf("invalid source access time: %v", err)
		}
		raw, err := os.ReadFile("testdata/" + source.File)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if len(raw) != source.Size || hex.EncodeToString(sum[:]) != source.SHA256 {
			t.Fatalf("source bytes differ from adopted snapshot: %s", source.File)
		}
		records, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 26 || records[0][0] != "Address Block" {
			t.Fatalf("unexpected CSV record count/header: %s", source.File)
		}
		before := len(prefixes)
		for _, record := range records[1:] {
			block := footnote.ReplaceAllString(record[0], "")
			for _, token := range strings.Split(block, ",") {
				prefix, err := netip.ParsePrefix(strings.TrimSpace(token))
				if err != nil || prefix != prefix.Masked() {
					t.Fatalf("invalid registry CIDR %q: %v", token, err)
				}
				if seen[prefix] {
					t.Fatalf("duplicate independent CIDR %s", prefix)
				}
				seen[prefix] = true
				prefixes = append(prefixes, prefix)
			}
		}
		if got := len(prefixes) - before; got != wantPrefixes {
			t.Fatalf("%s expanded prefixes = %d, want %d", source.File, got, wantPrefixes)
		}
	}
	// The spec adds multicast and deprecated IPv6 site-local to the IANA rows.
	for _, extra := range []string{"224.0.0.0/4", "ff00::/8", "fec0::/10"} {
		prefix := netip.MustParsePrefix(extra)
		if seen[prefix] {
			t.Fatalf("extra unexpectedly present in IANA rows: %s", prefix)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

// Compute both endpoints from the independent network bits, retaining raw
// mapped IPv6 instead of unmapping it before the policy gets to inspect it.
func prefixEndpoints(t *testing.T, prefix netip.Prefix) (netip.Addr, netip.Addr) {
	t.Helper()
	first := prefix.Masked().Addr()
	lastBytes := first.AsSlice()
	for bit := prefix.Bits(); bit < len(lastBytes)*8; bit++ {
		lastBytes[bit/8] |= byte(1 << (7 - bit%8))
	}
	last, ok := netip.AddrFromSlice(lastBytes)
	if !ok {
		t.Fatal("invalid independently calculated endpoint")
	}
	return first, last
}
