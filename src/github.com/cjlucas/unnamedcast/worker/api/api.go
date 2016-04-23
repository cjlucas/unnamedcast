package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var httpClient = http.Client{}

type User struct {
	ID               string      `json:"id"`
	Username         string      `json:"username"`
	FeedIDs          []string    `json:"feeds"`
	ItemStates       []ItemState `json:"states"`
	CreationTime     time.Time   `json:"creation_time"`
	ModificationTime time.Time   `json:"modification_time"`
}

type ItemState struct {
	FeedID   string  `json:"feed_id"`
	ItemGUID string  `json:"item_guid"`
	Position float64 `json:"position"` // 0 if item is unplayed
}

type Feed struct {
	ID                string    `json:"id,omitempty"`
	Title             string    `json:"title"`
	URL               string    `json:"url"`
	Author            string    `json:"author"`
	ImageURL          string    `json:"image_url"`
	ITunesID          int       `json:"itunes_id"`
	ITunesReviewCount int       `json:"itunes_review_count"`
	ITunesRatingCount int       `json:"itunes_rating_count"`
	CreationTime      time.Time `json:"creation_time"`
	ModificationTime  time.Time `json:"modification_time"`

	Category struct {
		Name          string   `json:"name"`
		Subcategories []string `json:"subcategories"`
	} `json:"category"`
}

type Item struct {
	ID               string        `json:"id,omitempty"`
	GUID             string        `json:"guid"`
	Link             string        `json:"link"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	URL              string        `json:"url"`
	Author           string        `json:"author"`
	Duration         time.Duration `json:"duration"`
	Size             int           `json:"size"`
	PublicationTime  time.Time     `json:"publication_time"`
	ImageURL         string        `json:"image_url"`
	CreationTime     time.Time     `json:"creation_time"`
	ModificationTime time.Time     `json:"modification_time"`
}

type API struct {
	BaseURL *url.URL
}

type apiRoundTrip struct {
	Method       string
	Endpoint     string
	RequestBody  interface{}
	Response     *http.Response
	ResponseBody interface{}
}

func (api *API) makeRequest(apiReq *apiRoundTrip) error {
	var reqBody io.Reader
	if apiReq.RequestBody != nil {
		data, err := json.Marshal(apiReq.RequestBody)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s%s", api.BaseURL, apiReq.Endpoint)
	req, err := http.NewRequest(apiReq.Method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	data, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	apiReq.Response = resp

	if apiReq.ResponseBody != nil {
		if err := json.Unmarshal(data, apiReq.ResponseBody); err != nil {
			return err
		}
	}

	return nil
}

func (api *API) GetFeed(feedID string) (*Feed, error) {
	var feed Feed
	err := api.makeRequest(&apiRoundTrip{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("/api/feeds/%s", feedID),
		ResponseBody: &feed,
	})
	return &feed, err
}

func (api *API) UpdateFeed(feed *Feed) error {
	return api.makeRequest(&apiRoundTrip{
		Method:      "PUT",
		Endpoint:    fmt.Sprintf("/api/feeds/%s", feed.ID),
		RequestBody: feed,
	})
}

func (api *API) CreateFeed(feed *Feed) (*Feed, error) {
	var respFeed Feed
	err := api.makeRequest(&apiRoundTrip{
		Method:       "POST",
		Endpoint:     "/api/feeds",
		RequestBody:  feed,
		ResponseBody: &respFeed,
	})
	return &respFeed, err
}

func (api *API) feedExistsWithKey(key, value string) (bool, error) {
	req := apiRoundTrip{
		Method:   "GET",
		Endpoint: fmt.Sprintf("/api/feeds?%s=%s", key, value),
	}
	err := api.makeRequest(&req)
	return req.Response.StatusCode == http.StatusOK, err
}

func (api *API) FeedExistsWithURL(url string) (bool, error) {
	return api.feedExistsWithKey("url", url)
}

func (api *API) FeedExistsWithiTunesID(id int) (bool, error) {
	return api.feedExistsWithKey("itunes_id", strconv.Itoa(id))
}

func (api *API) FeedForURL(feedURL string) (*Feed, error) {
	var feeds []Feed
	err := api.makeRequest(&apiRoundTrip{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("/api/feeds?url=%s", feedURL),
		ResponseBody: &feeds,
	})
	if len(feeds) == 0 {
		return nil, err
	}
	return &feeds[0], err
}

func (api *API) GetFeedItems(feedID string) ([]Item, error) {
	var items []Item
	err := api.makeRequest(&apiRoundTrip{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("/api/feeds/%s/items", feedID),
		ResponseBody: &items,
	})
	return items, err
}

func (api *API) GetFeedsUsers(feedID string) ([]User, error) {
	var users []User
	err := api.makeRequest(&apiRoundTrip{
		Method:       "GET",
		Endpoint:     fmt.Sprintf("/api/feeds/%s/users", feedID),
		ResponseBody: &users,
	})
	return users, err
}

func (api *API) PutItemStates(userID string, states []ItemState) error {
	return api.makeRequest(&apiRoundTrip{
		Method:      "PUT",
		Endpoint:    fmt.Sprintf("/api/users/%s/states", userID),
		RequestBody: &states,
	})
}

func (api *API) GetUsers() ([]User, error) {
	var users []User
	err := api.makeRequest(&apiRoundTrip{
		Method:       "GET",
		Endpoint:     "/api/users",
		ResponseBody: &users,
	})
	return users, err
}
