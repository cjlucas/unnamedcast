package rss

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
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

type Document struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

var pubDateFmts = []string{
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 -0700",
}

func ParseFeed(r io.Reader) (*Document, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var doc Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func ParseDuration(duration string) (time.Duration, error) {
	if duration == "" {
		return time.Duration(0), nil
	}

	// Simple case, an integer in seconds
	if val, err := strconv.ParseInt(duration, 10, 0); err == nil {
		return time.Duration(val), nil
	}

	// parse HH:MM:SS format backwards with incrementing multiplier
	split := strings.Split(duration, ":")
	secs := 0
	curMultiplier := 1
	for i := range split {
		val, err := strconv.ParseInt(split[len(split)-i-1], 10, 0)
		if err != nil {
			return time.Duration(0), fmt.Errorf("Could not parse duration: %s", duration)
		}

		secs += int(val) * curMultiplier
		curMultiplier *= 60
	}

	return time.Duration(secs), nil
}

func ParseDate(date string) time.Time {
	for _, fmt := range pubDateFmts {
		if t, err := time.Parse(fmt, date); err == nil {
			return t
		}
	}

	return time.Time{}.UTC()
}
