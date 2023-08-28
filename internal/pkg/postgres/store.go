package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

func NewDatabase(connURL string) (*sql.DB, func(db *sql.DB), error) {
	db, err := sql.Open("postgres", connURL)
	if err != nil {
		return nil, nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, nil, err
	}

	return db, closePostgresDb, nil
}

func closePostgresDb(db *sql.DB) {
	_ = db.Close()
}

func NewPlaceHolders(start, nArgs int) string {
	placeHolders := make([]string, nArgs)
	for i := start; i < start+nArgs; i++ {
		placeHolders[i-start] = fmt.Sprint("$", i+1)
	}

	joinedPlaceHolders := strings.Join(placeHolders, ",")
	return fmt.Sprintf("(%s)", joinedPlaceHolders)
}
