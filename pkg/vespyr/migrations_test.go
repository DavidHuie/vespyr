package vespyr

import (
	_ "github.com/mattes/migrate/database/postgres"
)

// var (
// 	o sync.Once
// )

// func runMigrations() {
// 	m, err := migrate.New(
// 		"code://",
// 		"postgres://vespyr:password@localhost:5432/vespyr?sslmode=disable")
// 	if err != nil {
// 		panic(err)
// 	}
// 	m.Steps(100)
// }

// func init() {
// 	runMigrations()
// }

// func TestMigrations(t *testing.T) {
// }
