// Package schema holds the chat store's SQL and turns it into the two things
// that must never disagree: the DDL the store creates its tables from, and the
// input sqlc compiles the queries in ../db against. Both read the same files,
// so a column can only be added in one place.
package schema

//go:generate sqlc generate

import (
	"embed"
	"path"
	"strings"
	"sync"
)

//go:embed ddl/*.sql
var ddlFS embed.FS

var ddl = sync.OnceValue(func() string {
	entries, err := ddlFS.ReadDir("ddl")
	if err != nil {
		// The files are compiled into the binary, so a failure here is a build
		// defect rather than a runtime condition.
		panic("chat schema: embedded DDL is unreadable: " + err.Error())
	}
	// ReadDir sorts by filename, which is the order sqlc applies the files in
	// too, so a later file may depend on an earlier one.
	var b strings.Builder
	for _, e := range entries {
		content, err := ddlFS.ReadFile(path.Join("ddl", e.Name()))
		if err != nil {
			panic("chat schema: embedded DDL is unreadable: " + err.Error())
		}
		b.Write(content)
		b.WriteByte('\n')
	}
	return b.String()
})

// DDL returns every ddl/*.sql file concatenated in filename order, ready to
// execute against a fresh database.
func DDL() string {
	return ddl()
}
