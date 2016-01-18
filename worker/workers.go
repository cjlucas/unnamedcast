package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/cjlucas/unnamedcast/worker/api"
	"github.com/cjlucas/unnamedcast/worker/itunes"
	"github.com/cjlucas/unnamedcast/worker/rss"
	"github.com/cjlucas/unnamedcast/yajq"
)

var iTunesIDRegexp = regexp.MustCompile(`/id(\d+)`)

type Worker interface {
	Work(q *yajq.Queue, j *yajq.Job) error
}

type ScrapeiTunesGenreListWorker struct {
}

func (w *ScrapeiTunesGenreListWorker) Work(q *yajq.Queue, j *yajq.Job) error {
	var payload struct {
		URL string `json:url`
	}
	if err := j.UnmarshalPayload(&payload); err != nil {
		return err
	}

	p, err := itunes.NewGenreListPage()
	if err != nil {
		return err
	}

	for _, genreURL := range p.GenreURLs() {
		job, err := yajq.Submit(queueScrapeiTunesGenre, 0, map[string]string{
			"url": genreURL,
		})

		if err != nil {
			return err
		}

		q.Log(j, fmt.Sprintf("Added job to parse url: %s (id: %d)", genreURL, job.ID))
	}

	return nil
}

type ScrapeiTunesGenreWorker struct {
}

type ScrapeiTunesGenrePayload struct {
	URL string `json:url`
}

func (w *ScrapeiTunesGenreWorker) Work(q *yajq.Queue, j *yajq.Job) error {
	var payload ScrapeiTunesGenrePayload
	if err := j.UnmarshalPayload(&payload); err != nil {
		return err
	}

	urls, err := itunes.AlphabetPageListForFeedListPage(payload.URL)
	if err != nil {
		return err
	}

	for _, url := range urls {
		page, err := itunes.NewFeedListPage(url)
		if err != nil {
			return err
		}

		for _, url := range page.PaginationPageList() {
			_, err := yajq.Submit(queueScrapeiTunesFeedList, 0, ScrapeiTunesFeedListPayload{
				URL: url,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type ScrapeiTunesFeedListWorker struct {
}

type ScrapeiTunesFeedListPayload struct {
	URL string `json:url`
}

func (w *ScrapeiTunesFeedListWorker) Work(q *yajq.Queue, j *yajq.Job) error {
	var payload ScrapeiTunesFeedListPayload
	if err := j.UnmarshalPayload(&payload); err != nil {
		return err
	}

	page, err := itunes.NewFeedListPage(payload.URL)
	if err != nil {
		return err
	}

	for _, url := range page.FeedURLs() {
		matches := iTunesIDRegexp.FindStringSubmatch(url)
		if len(matches) < 2 {
			fmt.Println("No iTunes ID found in url:", url)
			continue
		}

		id, err := strconv.ParseInt(matches[1], 10, 0)
		if err != nil {
			fmt.Println("Could not parse iTunes ID:", id)
			continue
		}

		if exists, err := api.FeedExistsWithiTunesID(int(id)); !exists && err == nil {
			feedURL, err := itunes.ResolveiTunesFeedURL(url)
			if err != nil {
				fmt.Println("Failed to resolve rss feed for url:", url)
				continue
			}

			_, err = yajq.Submit(queueUpdateFeed, 0, &UpdateFeedPayload{
				URL:      feedURL,
				ITunesID: int(id),
			})
			if err != nil {
				q.Log(j, fmt.Sprintf("Failed to add update feed job"))
				continue
			}
			// fmt.Println("Added update feed job for url:", url)
		} else if err != nil {
			return err
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

func (w *UpdateFeedWorker) Work(q *yajq.Queue, j *yajq.Job) error {
	var payload UpdateFeedPayload
	if err := j.UnmarshalPayload(&payload); err != nil {
		return err
	}

	resp, err := http.Get(payload.URL)
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

	if feed.ITunesID != 0 {
		feed.ITunesID = payload.ITunesID
	}

	return api.PostFeed(feed)
}
