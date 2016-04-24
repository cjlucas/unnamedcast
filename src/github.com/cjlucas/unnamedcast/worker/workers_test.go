package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/cjlucas/unnamedcast/worker/rss"
)

func TestItemsFromRSSWithDescription(t *testing.T) {
	buf, err := ioutil.ReadFile("testdata/nominal.xml")
	if err != nil {
		t.Fatal(err)
	}

	doc, err := rss.ParseFeed(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	items := itemsFromRSS(doc)

	if items[0].Description != doc.Channel.Items[0].ContentEncoded {
		t.Error("item.Description != item.ContentEncoded")
	}
}
