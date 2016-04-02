package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

var httpClient = http.Client{}

type User struct {
	ID               string    `json:"id"`
	Username         string    `json:"username"`
	FeedIDs          []string  `json:"feeds"`
	CreationTime     time.Time `json:"creation_time"`
	ModificationTime time.Time `json:"modification_time"`
}

type Feed struct {
	ID                string `json:"id,omitempty"`
	Title             string `json:"title"`
	URL               string `json:"url"`
	Author            string `json:"author"`
	Items             []Item `json:"items"`
	ImageURL          string `json:"image_url"`
	ITunesID          int    `json:"itunes_id"`
	ITunesReviewCount int    `json:"itunes_review_count"`
	ITunesRatingCount int    `json:"itunes_rating_count"`

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

func GetFeed(feedID string) (*Feed, error) {
	url := fmt.Sprintf("http://localhost:8081/api/feeds/%s", feedID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get feed (code: %d)", resp.StatusCode)
	}

	var feed Feed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func UpdateFeed(feed *Feed) error {
	payload, err := json.Marshal(&feed)
	if err != nil {
		return err
	}

	r := bytes.NewReader(payload)
	url := fmt.Sprintf("http://localhost:8081/api/feeds/%s", feed.ID)
	req, err := http.NewRequest("PUT", url, r)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		if err, ok := err.(*net.DNSError); ok {
			panic(fmt.Sprintf("TURNS OUT IT IS A DNS error: %s", err))
		}
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

func CreateFeed(feed *Feed) (*Feed, error) {
	payload, err := json.Marshal(&feed)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(payload)
	apiURL := "http://localhost:8081/api/feeds"
	resp, err := httpClient.Post(apiURL, "application/json", r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read entire response to prevent broken pipe
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Received unexpected status code with body: %s", data)
	}

	if err := json.Unmarshal(data, feed); err != nil {
		return nil, err
	}

	return feed, nil
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

func FeedForURL(feedURL string) (*Feed, error) {
	url := fmt.Sprintf("http://localhost:8081/api/feeds?url=%s", feedURL)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feeds []Feed
	if err := json.Unmarshal(data, &feeds); err != nil {
		return nil, err
	}

	if len(feeds) == 0 {
		return nil, nil
	}

	return &feeds[0], nil
}