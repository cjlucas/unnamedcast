package db

import "os"

func newDB() *DB {
	db, err := New(Config{
		URL:                os.Getenv("DB_URL"),
		Clean:              true,
		ForceIndexCreation: true,
	})

	if err != nil {
		panic(err)
	}

	return db
}
