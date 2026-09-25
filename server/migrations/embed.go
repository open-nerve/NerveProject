// Package migrations embeds the SQL schema migrations. Files live in sql/ and
// are named NNNNN_<module>_<description>.sql; a migration belongs to the
// module that owns the table it changes (M2 design 3.14).
package migrations

import (
	"embed"
	"io/fs"
)

//go:embed sql/*.sql
var files embed.FS

// FS returns the migrations, with the files at its root.
func FS() fs.FS {
	sub, err := fs.Sub(files, "sql")
	if err != nil {
		panic(err) // unreachable: "sql" is a valid path embedded above
	}
	return sub
}
