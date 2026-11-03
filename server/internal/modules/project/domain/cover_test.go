package domain

import (
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// Catches acceptance of non-canonical IDs or two competing image sources,
// and a patch that erases an omitted field or fails to clear the other choice.
func TestProjectCoverChoice(t *testing.T) {
	asset := uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	for _, id := range []string{"cover-01", "cover-15", "cover-29"} {
		got, err := CheckNewProject(NewProject{Name: "Images", Identifier: "IMG", CoverBuiltinID: &id})
		if err != nil || got.CoverBuiltinID == nil || *got.CoverBuiltinID != id {
			t.Errorf("valid %q: %+v %v", id, got, err)
		}
	}
	for _, id := range []string{"", "cover-00", "cover-30", "cover-1", "cover-001", "cover-0a", "COVER-01", "cover-01\n"} {
		_, err := CheckNewProject(NewProject{Name: "Images", Identifier: "IMG", CoverBuiltinID: &id})
		assertCoverFields(t, err, []string{"cover_builtin_id"})
		_, err = CheckProjectPatch(ProjectPatch{SetCoverBuiltin: true, CoverBuiltinID: &id})
		assertCoverFields(t, err, []string{"cover_builtin_id"})
	}
	_, err := CheckNewProject(NewProject{Name: "Images", Identifier: "IMG", CoverImageAssetID: &asset, CoverBuiltinID: ptr("cover-01")})
	assertCoverFields(t, err, []string{"cover_image_asset_id", "cover_builtin_id"})
	_, err = CheckProjectPatch(ProjectPatch{SetCoverImage: true, CoverImageAssetID: &asset, SetCoverBuiltin: true, CoverBuiltinID: ptr("cover-29")})
	assertCoverFields(t, err, []string{"cover_image_asset_id", "cover_builtin_id"})
	cases := []struct {
		name     string
		in, want ProjectPatch
	}{
		{"omitted", ProjectPatch{}, ProjectPatch{}},
		{"clear asset", ProjectPatch{SetCoverImage: true}, ProjectPatch{SetCoverImage: true}},
		{"clear builtin", ProjectPatch{SetCoverBuiltin: true}, ProjectPatch{SetCoverBuiltin: true}},
		{"clear both", ProjectPatch{SetCoverImage: true, SetCoverBuiltin: true}, ProjectPatch{SetCoverImage: true, SetCoverBuiltin: true}},
		{"choose asset", ProjectPatch{SetCoverImage: true, CoverImageAssetID: &asset}, ProjectPatch{SetCoverImage: true, CoverImageAssetID: &asset, SetCoverBuiltin: true}},
		{"choose builtin", ProjectPatch{SetCoverBuiltin: true, CoverBuiltinID: ptr("cover-29")}, ProjectPatch{SetCoverImage: true, SetCoverBuiltin: true, CoverBuiltinID: ptr("cover-29")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckProjectPatch(tc.in)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("patch=%+v err=%v want=%+v", got, err, tc.want)
			}
		})
	}
}
func assertCoverFields(t *testing.T, err error, want []string) {
	t.Helper()
	var got []string
	for _, field := range fieldsOf(t, err) {
		if field.Code != "not_allowed" {
			t.Errorf("cover error code=%s", field.Code)
		}
		got = append(got, field.Field)
	}
	if !slices.Equal(got, want) {
		t.Errorf("fields=%v want=%v (err=%v)", got, want, err)
	}
}
