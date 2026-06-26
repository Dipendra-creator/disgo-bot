// Package database exposes the embedded SQL migration files. Keeping the
// embed in its own package at the repository root lets the migration runner in
// internal/database load files that live outside its own directory.
package database

import "embed"

// Migrations holds every *.sql file under migrations/, applied in lexical order.
//
//go:embed migrations/*.sql
var Migrations embed.FS
