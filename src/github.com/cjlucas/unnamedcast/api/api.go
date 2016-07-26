package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var httpClient = http.Client{}

type itemState int

const (
	StateUnplayed   itemState = 0
	StateInProgress           = 1
	StatePlayed               = 2
)

type User struct {
	ID               string      `json:"id"`
	Username         string      `json:"username"`
	FeedIDs          []string    `json:"feeds"`
	ItemStates       []ItemState `json:"states"`
	CreationTime     time.Time   `json:"creation_time"`
	ModificationTime time.Time   `json:"modification_time"`
}

type ItemState struct {
	ItemID           string    `json:"item_id"`
	State            itemState `json:"state"`
	Position         float64   `json:"position"` // 0 if item is unplayed
	ModificationTime time.Time `json:"modification_time"`
}

type Feed struct {
	ID                 string    `json:"id,omitempty"`
	Title              string    `json:"title"`
	URL                string    `json:"url"`
	Author             string    `json:"author"`
	ImageURL           string    `json:"image_url"`
	ITunesID           int       `json:"itunes_id"`
	ITunesReviewCount  int       `json:"itunes_review_count"`
	ITunesRatingCount  int       `json:"itunes_rating_count"`
	CreationTime       time.Time `json:"creation_time"`
	ModificationTime   time.Time `json:"modification_time"`
	LastScrapedTime    time.Time `json:"last_scraped_time"`
	Items              []string  `json:"items"`
	SourceETag         string    `json:"src_etag"`
	SourceLastModified time.Time `json:"src_last_modified"`

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
	Summary          string        `json:"summary"`
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

type Job struct {
	ID       string      `json:"id"`
	KodaID   int         `json:"koda_id"`
	Priority int         `json:"priority"`
	Queue    string      `json:"queue"`
	State    string      `json:"state"`
	Payload  interface{} `json:"payload"`
	// CreationTime is the time at which the job was created in koda
	CreationTime     time.Time `json:"creation_time"`
	ModificationTime time.Time `json:"modification_time"`
	CompletionTime   time.Time `json:"completion_time"`
	Log              []struct {
		Time time.Time `json:"time"`
		Line string    `json:"line"`
	} `json:"log"`
}

type API struct {
	Host string
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

	url := fmt.Sprintf("http://%s%s", api.Host, apiReq.Endpoint)
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

func (api *API) CreateUser(username, password string) (*User, error) {
	var user User
	endpoint := fmt.Sprintf("/api/users?username=%s&password=%s", username, password)
	err := api.makeRequest(&apiRoundTrip{
		Method:       "POST",
		Endpoint:     endpoint,
		ResponseBody: &user,
	})
	return &user, err
}

func (api *API) UpdateUserFeeds(userID string, feedIDs []string) error {
	return api.makeRequest(&apiRoundTrip{
		Method:      "PUT",
		Endpoint:    fmt.Sprintf("/api/users/%s/feeds", userID),
		RequestBody: feedIDs,
	})
}

func (api *API) UpdateUserItemState(userID string, state ItemState) error {
	return api.makeRequest(&apiRoundTrip{
		Method:      "PUT",
		Endpoint:    fmt.Sprintf("/api/users/%s/states/%s", userID, state.ItemID),
		RequestBody: &state,
	})
}

func (api *API) DeleteUserItemState(userID, itemID string) error {
	return api.makeRequest(&apiRoundTrip{
		Method:   "DELETE",
		Endpoint: fmt.Sprintf("/api/users/%s/states/%s", userID, itemID),
	})
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

func (api *API) CreateFeedItem(feedID string, item *Item) error {
	return api.makeRequest(&apiRoundTrip{
		Method:       "POST",
		Endpoint:     fmt.Sprintf("/api/feeds/%s/items", feedID),
		RequestBody:  item,
		ResponseBody: item,
	})
}

// TODO: have this modify the passed in item instead of creating a new item
func (api *API) UpdateFeedItem(feedID string, item *Item) (*Item, error) {
	var out Item
	err := api.makeRequest(&apiRoundTrip{
		Method:       "PUT",
		Endpoint:     fmt.Sprintf("/api/feeds/%s/items/%s", feedID, item.ID),
		RequestBody:  item,
		ResponseBody: &out,
	})
	return &out, err
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

func (api *API) GetUsers() ([]User, error) {
	var users []User
	err := api.makeRequest(&apiRoundTrip{
		Method:       "GET",
		Endpoint:     "/api/users",
		ResponseBody: &users,
	})
	return users, err
}

func (api *API) CreateJob(job *Job) error {
	return api.makeRequest(&apiRoundTrip{
		Method:       "POST",
		Endpoint:     "/api/jobs",
		RequestBody:  job,
		ResponseBody: job,
	})
}
