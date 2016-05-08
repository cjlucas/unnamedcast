package db

import "os"

func newDB() *DB {
	db, err := New(os.Getenv("DB_URL"))
	if err != nil {
		panic(err)
	}

	if err := db.Drop(); err != nil {
		panic(err)
	}

	return db
}
