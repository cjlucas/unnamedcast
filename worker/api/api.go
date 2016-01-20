package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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
	defer resp.Body.Close()

	// Read entire response to prevent broken pipe
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Received unexpected status code with body: %s", data)
	}

	return nil
}

func feedExistsWithKey(key, value string) (bool, error) {
	url := fmt.Sprintf("http://localhost:8081/api/feeds?%s=%s", key, value)
	resp, err := httpClient.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	ioutil.ReadAll(resp.Body)

	return resp.StatusCode == 200, nil
}

func FeedExistsWithURL(url string) (bool, error) {
	return feedExistsWithKey("url", url)
}

func FeedExistsWithiTunesID(id int) (bool, error) {
	return feedExistsWithKey("itunes_id", strconv.Itoa(id))
}
