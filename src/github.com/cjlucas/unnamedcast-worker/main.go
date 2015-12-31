package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Channel struct {
	Title  string `xml:"title"`
	Author string `xml:"author"`

	Image struct {
		URL string `xml:"href,attr"`
	} `xml:"image"`

	Items []struct {
		GUID   string `xml:"guid"`
		Title  string `xml:"title"`
		Link   string `xml:"link"`
		Author string `xml:"author"`

		Enclosure struct {
			URL    string `xml:"url,attr"`
			Length int    `xml:"length,attr"`
			Type   string `xml:"type,attr"`
		} `xml:"enclosure"`

		// Format is RFC2822
		PublicationDate string `xml:"pubDate"`

		// Represented as an integer (in seconds) or HH:MM:SS, H:MM:SS, MM:SS, or M:SS
		Duration string `xml:"duration"`

		Image struct {
			URL string `xml:"href,attr"`
		} `xml:"image"`
	} `xml:"item"`

	Category struct {
		Name          string `xml:"text,attr"`
		Subcategories []struct {
			Name string `xml:"text,attr"`
		}
	} `xml:"category"`
}

type RSSDoc struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type JSONFeed struct {
	Title    string     `json:"title"`
	URL      string     `json:"url"`
	Author   string     `json:"author"`
	Items    []JSONItem `json:"items"`
	ImageURL string     `json:"image_url"`

	Category struct {
		Name          string   `json:"name"`
		Subcategories []string `json:"subcategories"`
	} `json:"category"`
}

type JSONItem struct {
	GUID            string        `json:"guid"`
	Link            string        `json:"link"`
	Title           string        `json:"title"`
	Description     string        `json:"description"`
	URL             string        `json:"url"`
	Author          string        `json:"author"`
	Duration        time.Duration `json:"duration"`
	Size            int           `json:"size"`
	PublicationTime time.Time     `json:"publication_time"`
	ImageURL        string        `json:"image_url"`
}

const RFC2822 = "Mon, 02 Jan 2006 15:04:05 MST"

func parseDuration(duration string) time.Duration {
	// Simple case, an integer in seconds
	if val, err := strconv.ParseInt(duration, 0, 0); err == nil {
		return time.Duration(val)
	}

	// parse HH:MM:SS format backwards with incrementing multiplier
	split := strings.Split(duration, ":")
	secs := 0
	curMultiplier := 1
	for i := range split {
		val, err := strconv.ParseInt(split[len(split)-i-1], 0, 0)
		if err != nil {
			panic(err)
		}

		secs += int(val) * curMultiplier
		curMultiplier *= 60
	}

	return time.Duration(secs)
}

func parseDate(date string) time.Time {
	t, err := time.Parse(RFC2822, date)
	if err != nil {
		panic(err)
	}
	return t
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

	channel := rss.Channel
	var feed JSONFeed
	feed.Items = make([]JSONItem, len(channel.Items))

	feed.Title = channel.Title
	feed.ImageURL = channel.Image.URL
	feed.Author = channel.Author
	for i, item := range channel.Items {
		jsonItem := &feed.Items[i]

		jsonItem.GUID = item.GUID
		jsonItem.Title = item.Title
		jsonItem.Author = item.Author
		jsonItem.URL = item.Enclosure.URL
		jsonItem.Size = item.Enclosure.Length
		jsonItem.PublicationTime = parseDate(item.PublicationDate)
		jsonItem.Duration = parseDuration(item.Duration)
		jsonItem.ImageURL = item.Image.URL
		jsonItem.Link = item.Link
	}

	feed.Category.Name = channel.Category.Name
	feed.Category.Subcategories = make(
		[]string,
		len(channel.Category.Subcategories))

	for _, c := range channel.Category.Subcategories {
		feed.Category.Subcategories = append(feed.Category.Subcategories, c.Name)
	}

	if payload, err := json.Marshal(&feed); err != nil {
		panic(err)
	} else {
		r := bytes.NewReader(payload)
		http.Post("http://localhost:8081/api/feed", "application/json", r)
	}

}
