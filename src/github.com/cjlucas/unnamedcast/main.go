package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

type Item struct {
	Title     string    `xml:"title"`
	Enclosure Enclosure `xml:"enclosure"`
	GUID      string    `xml:"guid"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int    `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type RSSDoc struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

func main() {
	fp, _ := os.Open("/Users/chris/Downloads/rss.xml")
	defer fp.Close()

	var rss RSSDoc
	data, err := ioutil.ReadAll(fp)
	if err != nil {
		panic(err)
	}

	if err := xml.Unmarshal(data, &rss); err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", rss)
}
