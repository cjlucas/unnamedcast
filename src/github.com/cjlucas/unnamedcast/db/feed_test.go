package db

import "testing"

func createFeed(t *testing.T, db *DB, feed *Feed) *Feed {
	if err := db.Feeds.Create(feed); err != nil {
		t.Fatal("Could not create feed:", err)
	}

	return feed
}

func createItem(t *testing.T, db *DB, item *Item) *Item {
	if err := db.Items.Create(item); err != nil {
		t.Fatal("Could not create item:", err)
	}

	return item
}

func TestCreateFeed(t *testing.T) {
	db := newDB()

	feed := createFeed(t, db, &Feed{
		URL: "http://google.com",
	})

	if feed.CreationTime.IsZero() {
		t.Error("Creation time was not set")
	}
}

func TestUpdateItem_NoModification(t *testing.T) {
	db := newDB()

	item := &Item{
		GUID:   "http://google.com/1",
		FeedID: createFeed(t, db, &Feed{URL: "http://google.com"}).ID,
	}

	if err := db.Items.Create(item); err != nil {
		t.Fatal("Could not create item:", err)
	}

	modTime := item.ModificationTime

	if err := db.Items.Update(item); err != nil {
		t.Fatal("Could not update item:", err)
	}

	if !modTime.Equal(item.ModificationTime) {
		t.Errorf("ModificationTime mismatch: %s != %s", item.ModificationTime, modTime)
	}
}

func TestItemsWithFeedID(t *testing.T) {
	db := newDB()

	feed := createFeed(t, db, &Feed{URL: "http://google.com"})

	createItem(t, db, &Item{
		GUID:   "http://google.com/1",
		FeedID: feed.ID,
	})

	n, err := db.Items.ItemsWithFeedID(feed.ID).Count()
	if err != nil {
		t.Fatalf("ItemsWithFeedID failed: %s", err)
	}

	if n != 1 {
		t.Errorf("num items mismatch: %d != 1", n)
	}
}
