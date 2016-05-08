package db

import (
	"os"
	"testing"
)

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

func TestUpdateItem_NoModification(t *testing.T) {
	db := newDB()

	item := &Item{
		GUID: "http://google.com/1",
	}

	if err := db.CreateItem(item); err != nil {
		t.Fatal("Could not create item:", err)
	}

	modTime := item.ModificationTime

	if err := db.UpdateItem(item); err != nil {
		t.Fatal("Could not update item:", err)
	}

	if modTime != item.ModificationTime {
		t.Errorf("ModificationTime mismatch: %s != %s", item.ModificationTime, modTime)
	}
}
