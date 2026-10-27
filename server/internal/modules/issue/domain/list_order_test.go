package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"uuid"
)

func TestIssueOrderSamples(t *testing.T) {
	raw, err := os.ReadFile("../../../../../api/testdata/issue-order.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		NameKeys  []struct{ Name, Key string } `json:"name_keys"`
		Scenarios []struct {
			Order ListOrder `json:"order"`
			Rows  []struct {
				ID       string
				Names    []string
				NameKeys []string `json:"name_keys"`
				Key      json.RawMessage
			}
			AscendingIDs  []string `json:"ascending_ids"`
			DescendingIDs []string `json:"descending_ids"`
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.NameKeys) == 0 || len(fixture.Scenarios) != 11 {
		t.Fatal("missing shared cases")
	}
	for _, tc := range fixture.NameKeys {
		if got := NameOrderKey(tc.Name); got != tc.Key {
			t.Errorf("NameOrderKey(%q)=%q want %q", tc.Name, got, tc.Key)
		}
	}
	// This checks the actual Go text representation's codepoint ordering, not
	// a surrogate complete row comparator. SQL ordering is tested by T5/T12.
	if NameOrderKey("～") >= NameOrderKey("😀") {
		t.Fatal("U+FF5E must precede U+1F600")
	}
	for _, scenario := range fixture.Scenarios {
		t.Run(string(scenario.Order), func(t *testing.T) {
			if len(scenario.Rows) == 0 || len(scenario.AscendingIDs) != len(scenario.Rows) {
				t.Fatal("incomplete shared rows")
			}
			orders := []ListOrder{scenario.Order}
			if scenario.Order != "sort_order" {
				if len(scenario.DescendingIDs) != len(scenario.Rows) {
					t.Fatal("missing descending expectation")
				}
				orders = append(orders, "-"+scenario.Order)
			}
			for _, row := range scenario.Rows {
				if len(row.Names) != len(row.NameKeys) {
					t.Fatal("missing per-name expected keys")
				}
				for i, name := range row.Names {
					if key := NameOrderKey(name); key != row.NameKeys[i] {
						t.Fatalf("name %q key=%q want=%q", name, key, row.NameKeys[i])
					}
				}

				for _, order := range orders {
					id, err := uuid.Parse(row.ID)
					if err != nil {
						t.Fatal(err)
					}
					encoded, err := EncodeListCursor(ListCursor{Order: order, Key: row.Key, ID: id})
					if err != nil {
						t.Fatal(err)
					}
					got, err := DecodeListCursor(encoded, order)
					if err != nil || got.ID != id || string(got.Key) != string(row.Key) {
						t.Fatalf("cursor=%+v err=%v expected key=%s ID=%s", got, err, row.Key, row.ID)
					}
					body, err := base64.RawURLEncoding.DecodeString(encoded)
					if err != nil {
						t.Fatal(err)
					}
					want := fmt.Sprintf(`{"v":1,"p":[%q,%s,%q]}`, order, row.Key, row.ID)
					if string(body) != want {
						t.Fatalf("payload=%s want=%s", body, want)
					}
				}
			}
		})
	}
}
