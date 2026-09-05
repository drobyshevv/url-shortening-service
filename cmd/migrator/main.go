package main

import (
	"database/sql"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := sql.Open("pgx", "postgres://user:password@localhost:5432/url_shortening?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	goose.SetDialect("postgres")

	if err := goose.Up(db, "migrations"); err != nil {
		panic(err)
	}
}
