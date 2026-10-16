package migrations

import "embed"

// Files contains all versioned SQL migrations compiled into the server binary.
//
//go:embed *.sql
var Files embed.FS
