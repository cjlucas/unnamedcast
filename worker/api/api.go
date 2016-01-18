package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var httpClient = http.Client{}

type Feed struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Author   string `json:"author"`
	Items    []Item `json:"items"`
	ImageURL string `json:"image_url"`
	ITunesID int    `json:"itunes_id"`

	Category struct {
		Name          string   `json:"name"`
		Subcategories []string `json:"subcategories"`
	} `json:"category"`
}

type Item struct {
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

func PostFeed(feed *Feed) error {
	payload, err := json.Marshal(&feed)
	if err != nil {
		return err
	}

	r := bytes.NewReader(payload)
	url := "http://localhost:8081/api/feeds"
	resp, err := httpClient.Post(url, "application/json", r)
	if err != nil {
		return err
	}
	// Read entire response to prevent broken pipe
	ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return nil
}

func FeedExistsWithiTunesID(id int) (bool, error) {
	url := fmt.Sprintf("http://localhost:8081/api/feeds?itunes_id=%d", id)
	resp, err := httpClient.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	return resp.StatusCode == 200, nil
}
