// Package migrations embeds the SQL schema migrations. Files live in sql/ and
// are named NNNNN_<module>_<description>.sql.
package migrations

import (
	"embed"
	"io/fs"
)

// "all:" keeps sql/.gitkeep, so the pattern still matches while no migration
// exists; a plain *.sql pattern would fail to compile then.
//
//go:embed all:sql
var files embed.FS

// FS returns the migrations, with the files at its root.
func FS() fs.FS {
	sub, err := fs.Sub(files, "sql")
	if err != nil {
		panic(err) // unreachable: "sql" is a valid path embedded above
	}
	return sub
}
