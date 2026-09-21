package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

var migrationName = regexp.MustCompile(`^\d{5}_[a-z][a-z0-9]*_[a-z0-9_]+\.sql$`)

func TestMigrationFilesFollowNamingConvention(t *testing.T) {
	entries, err := fs.ReadDir(FS(), ".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") { // .gitkeep, .DS_Store: not migrations
			continue
		}
		if e.IsDir() || !migrationName.MatchString(e.Name()) {
			t.Errorf("%s: want NNNNN_<module>_<description>.sql, e.g. 00001_identity_create_users.sql", e.Name())
		}
	}
}
