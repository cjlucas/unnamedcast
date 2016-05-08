package db

import "testing"

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
