package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/cjlucas/unnamedcast/koda"
	"github.com/cjlucas/unnamedcast/worker/api"
	"github.com/cjlucas/unnamedcast/worker/itunes"
	"github.com/cjlucas/unnamedcast/worker/rss"
)

var iTunesIDRegexp = regexp.MustCompile(`/id(\d+)`)
var iTunesFeedURLRegexp = regexp.MustCompile(`https?://itunes.apple.com`)

type Worker interface {
	Work(q *koda.Queue, j *koda.Job) error
}

type ScrapeiTunesFeeds struct {
}

func (w *ScrapeiTunesFeeds) scrapeGenre(url string) ([]string, error) {
	var feedListURLs []string

	urls, err := itunes.AlphabetPageListForFeedListPage(url)
	if err != nil {
		return nil, err
	}

	for _, url := range urls {
		page, err := itunes.NewFeedListPage(url)
		if err != nil {
			return nil, err
		}

		for _, url := range page.PaginationPageList() {
			feedListURLs = append(feedListURLs, url)
		}
	}

	return feedListURLs, nil
}

func (w *ScrapeiTunesFeeds) scrapeFeedList(url string) ([]string, error) {
	page, err := itunes.NewFeedListPage(url)
	if err != nil {
		return nil, err
	}

	return page.FeedURLs(), nil
}

func (w *ScrapeiTunesFeeds) Work(q *koda.Queue, j *koda.Job) error {
	fmt.Println("Fetching the genres...")
	page, err := itunes.NewGenreListPage()
	if err != nil {
		return err
	}

	var feedListURLs []string
	for _, url := range page.GenreURLs() {
		fmt.Println("Scraping genre URL:", url)
		urls, err := w.scrapeGenre(url)
		if err != nil {
			return err
		}

		for _, url := range urls {
			feedListURLs = append(feedListURLs, url)
		}
	}

	fmt.Printf("Now scraping %d feed list urls\n", len(feedListURLs))

	// Scan through all feed list pages and add feed url to map
	// (Map is used to prune duplicate urls)
	itunesIDFeedURLMap := make(map[int]string)
	for _, url := range feedListURLs {
		fmt.Println("Scraping feed list:", url)
		urls, err := w.scrapeFeedList(url)
		if err != nil {
			return err
		}

		for _, url := range urls {
			matches := iTunesIDRegexp.FindStringSubmatch(url)
			if len(matches) < 2 {
				fmt.Println("No ID match found for url", url)
				continue
			}

			id, err := strconv.ParseInt(matches[1], 10, 0)
			if err != nil {
				fmt.Println("Could not parse id:", matches[1])
				continue
			}

			itunesIDFeedURLMap[int(id)] = url
		}
	}

	fmt.Printf("Found %d feeds\n", len(itunesIDFeedURLMap))

	for id, url := range itunesIDFeedURLMap {
		exists, err := api.FeedExistsWithiTunesID(id)
		if exists {
			continue
		} else if err != nil {
			return err
		}

		_, err = koda.Submit(queueUpdateFeed, 0, &UpdateFeedPayload{
			URL:      url,
			ITunesID: id,
		})

		if err != nil {
			j.Logf("Failed to add update feed job")
			continue
		}
	}

	return nil
}

type UpdateFeedWorker struct {
}

type UpdateFeedPayload struct {
	URL      string `json:url`
	ITunesID int    `json:itunes_id`
}

func rssDocumentToAPIPayload(doc *rss.Document) (*api.Feed, error) {
	channel := doc.Channel
	var feed api.Feed

	feed.Title = channel.Title
	feed.ImageURL = channel.Image.URL
	feed.Author = channel.Author
	feed.Items = make([]api.Item, len(channel.Items))

	for i, item := range channel.Items {
		jsonItem := &feed.Items[i]

		jsonItem.GUID = item.GUID
		jsonItem.Title = item.Title
		jsonItem.Author = item.Author
		jsonItem.URL = item.Enclosure.URL
		jsonItem.Size = item.Enclosure.Length
		jsonItem.PublicationTime = rss.ParseDate(item.PublicationDate)
		jsonItem.Duration, _ = rss.ParseDuration(item.Duration)
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

	return &feed, nil
}

func (w *UpdateFeedWorker) Work(q *koda.Queue, j *koda.Job) error {
	var payload UpdateFeedPayload
	if err := j.UnmarshalPayload(&payload); err != nil {
		return err
	}

	url := payload.URL
	if iTunesFeedURLRegexp.MatchString(url) {
		feedURL, err := itunes.ResolveiTunesFeedURL(url)
		if err != nil {
			return fmt.Errorf("Error occurred while resolving iTunes URL: %s", err)
		}
		url = feedURL
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := rss.ParseFeed(resp.Body)
	if err != nil {
		return err
	}

	feed, err := rssDocumentToAPIPayload(doc)
	if err != nil {
		return err
	}

	feed.URL = url
	if payload.ITunesID != 0 {
		feed.ITunesID = payload.ITunesID
	}

	return api.PostFeed(feed)
}
