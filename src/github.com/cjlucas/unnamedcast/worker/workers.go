package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/cjlucas/unnamedcast/api"
	"github.com/cjlucas/unnamedcast/worker/itunes"
	"github.com/cjlucas/unnamedcast/worker/rss"
)

var iTunesIDRegexp = regexp.MustCompile(`/id(\d+)`)
var iTunesFeedURLRegexp = regexp.MustCompile(`https?://itunes.apple.com`)

type ScrapeiTunesFeeds struct {
	API api.API
}

func (w *ScrapeiTunesFeeds) scrapeGenre(url string) ([]string, error) {
	var feedListURLs []string

	urls, err := itunes.AlphabetPageListForFeedListPage(url)
	if err != nil {
		return nil, fmt.Errorf("Error getting alphabet page list: %s", err)
	}

	for _, url := range urls {
		page, err := itunes.NewFeedListPage(url)
		if err != nil {
			return nil, fmt.Errorf("Error creating new page list: %s", err)
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
		return nil, fmt.Errorf("Error creating new page list: %s", err)
	}

	return page.FeedURLs(), nil
}

func (w *ScrapeiTunesFeeds) Work(j *Job) error {
	const numURLResolvers = 10

	j.Logf("Fetching genres pages...")
	page, err := itunes.NewGenreListPage()
	if err != nil {
		return err
	}

	var feedListURLs []string
	for _, url := range page.GenreURLs() {
		urls, err := w.scrapeGenre(url)
		if err != nil {
			return fmt.Errorf("Could not scrape genre URL: %s", err)
		}

		for _, url := range urls {
			feedListURLs = append(feedListURLs, url)
		}
	}

	j.Logf("Scraping %d feed list urls...", len(feedListURLs))

	// Scan through all feed list pages and add feed url to map
	// (Map is used to prune duplicate urls)
	itunesIDFeedURLMap := make(map[int]string)
	for _, url := range feedListURLs {
		j.Logf("Scraping feed list: %s", url)
		urls, err := w.scrapeFeedList(url)
		if err != nil {
			return fmt.Errorf("Could not scrape feed list: %s", err)
		}

		for _, url := range urls {
			matches := iTunesIDRegexp.FindStringSubmatch(url)
			if len(matches) < 2 {
				j.Logf("No ID match found for url: %s", url)
				continue
			}

			id, err := strconv.ParseInt(matches[1], 10, 0)
			if err != nil {
				j.Logf("Could not parse iTunes ID: %s", matches[1])
				continue
			}

			itunesIDFeedURLMap[int(id)] = url
		}
	}

	j.Logf("Found %d feeds", len(itunesIDFeedURLMap))

	// Remove feeds that are already in the database
	for id := range itunesIDFeedURLMap {
		exists, err := w.API.FeedExistsWithiTunesID(id)
		if err != nil {
			return fmt.Errorf("Failed to check if feed exists: %s", err)
		}

		if exists {
			delete(itunesIDFeedURLMap, id)
		}
	}

	// Spin up workers to resolve all itunes feed urls

	type urlResolverResponse struct {
		ITunesID int
		URL      string
		Err      error
	}

	urlResolverInChan := make(chan int, len(itunesIDFeedURLMap))
	urlResolverOutChan := make(chan urlResolverResponse, len(itunesIDFeedURLMap))

	work := func(in <-chan int, out chan<- urlResolverResponse) {
		for {
			itunesID, ok := <-in
			if !ok {
				break
			}

			url := itunesIDFeedURLMap[itunesID]

			url, err := itunes.ResolveiTunesFeedURL(url)
			out <- urlResolverResponse{
				ITunesID: itunesID,
				URL:      url,
				Err:      err,
			}
		}
	}

	for id := range itunesIDFeedURLMap {
		urlResolverInChan <- id
	}

	close(urlResolverInChan)

	for i := 0; i < numURLResolvers; i++ {
		go work(urlResolverInChan, urlResolverOutChan)
	}

	for i := 0; i < len(itunesIDFeedURLMap); i++ {
		resp, ok := <-urlResolverOutChan
		if !ok {
			panic("Out channel closed. This should never happen")
		}

		if resp.Err != nil {
			j.Logf("Failed to resolve feed url, will continue. (error: %s)", resp.Err)
			continue
		}

		feed := &api.Feed{URL: resp.URL, ITunesID: resp.ITunesID}

		feed, err = w.API.CreateFeed(feed)
		if err != nil {
			j.Logf("Could not create feed, will continue (error: %s)", err)
			continue
		}

		job := api.Job{
			Queue:   queueUpdateFeed,
			Payload: &UpdateFeedPayload{FeedID: feed.ID},
		}
		if err = w.API.CreateJob(&job); err != nil {
			j.Logf("Failed to add update feed job (ID: %s) (error: %s)", feed.ID, err)
			continue
		}
	}

	close(urlResolverOutChan)

	return nil
}

type UpdateFeedWorker struct {
	API api.API
}

type UpdateFeedPayload struct {
	FeedID string `json:"feed_id"`
}

func (w *UpdateFeedWorker) guidItemsMap(items []api.Item) map[string]api.Item {
	guidMap := make(map[string]api.Item)

	for _, item := range items {
		guidMap[item.GUID] = item
	}

	return guidMap
}

func (w *UpdateFeedWorker) Work(j *Job) error {
	var payload UpdateFeedPayload
	if err := j.KodaJob.UnmarshalPayload(&payload); err != nil {
		return err
	}

	origFeed, err := w.API.GetFeed(payload.FeedID)
	if err != nil {
		return err
	}

	resp, err := http.Get(origFeed.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := rss.ParseFeed(resp.Body)
	if err != nil {
		return err
	}

	origItems, err := w.API.GetFeedItems(payload.FeedID)
	if err != nil {
		return err
	}

	existingFeedItemMap := w.guidItemsMap(origItems)
	newFeedItemMap := w.guidItemsMap(itemsFromRSS(doc))

	var newItems []api.Item
	var existingItems []api.Item
	for guid, item := range newFeedItemMap {
		if origItem, ok := existingFeedItemMap[guid]; ok {
			item.ID = origItem.ID
			existingItems = append(existingItems, item)
		} else {
			newItems = append(newItems, item)
		}
	}

	for i := range newItems {
		if err := w.API.CreateFeedItem(payload.FeedID, &newItems[i]); err != nil {
			return err
		}
	}

	for _, item := range existingItems {
		if _, err = w.API.UpdateFeedItem(payload.FeedID, &item); err != nil {
			return err
		}
	}

	// NOTE: Disabled for now until http unauthorized request bug is resolved
	// if origFeed.ITunesID != 0 {
	// 	var stats *itunes.ReviewStats
	// 	stats, err = itunes.FetchReviewStats(feed.ITunesID)
	// 	if err != nil {
	// 		fmt.Println("Failed to fetch review stats for feed (will continue):", err)
	// 	} else {
	// 		feed.ITunesReviewCount = stats.ReviewCount
	// 		feed.ITunesRatingCount = stats.RatingCount
	// 	}
	// }

	feed := feedFromRSS(doc)
	feed.ID = payload.FeedID
	feed.URL = origFeed.URL
	feed.ITunesRatingCount = origFeed.ITunesRatingCount
	feed.ITunesReviewCount = origFeed.ITunesReviewCount
	feed.LastScrapedTime = time.Now()
	if err := w.API.UpdateFeed(feed); err != nil {
		return err
	}

	users, err := w.API.GetFeedsUsers(payload.FeedID)
	if err != nil {
		return fmt.Errorf("Failed to get users' feeds: %s", err)
	}

	// Don't update user feeds on initial scrape
	if origFeed.LastScrapedTime.IsZero() {
		return nil
	}

	for i := range users {
		user := &users[i]
		for k := range newItems {
			err := w.API.UpdateUserItemState(user.ID, api.ItemState{
				ItemID:           newItems[k].ID,
				State:            api.StateUnplayed,
				ModificationTime: time.Now().UTC(),
			})
			if err != nil {
				j.Logf("Could not update user's item state, will continue. (error: %s)", err)
				continue
			}
		}
	}

	return nil
}

type UpdateUserFeedsWorker struct {
	API api.API
}

func (w *UpdateUserFeedsWorker) Work(job *Job) error {
	users, err := w.API.GetUsers()
	if err != nil {
		return err
	}

	job.Logf("Fetched %d users", len(users))

	for i := range users {
		feedIDs := users[i].FeedIDs
		for _, id := range feedIDs {
			j := api.Job{
				Queue:   queueUpdateFeed,
				Payload: &UpdateFeedPayload{FeedID: id},
			}
			if err = w.API.CreateJob(&j); err != nil {
				job.Logf("Failed to add update feed job (error: %s)", err)
				continue
			}
		}
	}

	return nil
}

func feedFromRSS(doc *rss.Document) *api.Feed {
	channel := doc.Channel
	var feed api.Feed

	feed.Title = channel.Title
	feed.ImageURL = channel.Image.URL
	feed.Author = channel.Author

	feed.Category.Name = channel.Category.Name
	feed.Category.Subcategories = make(
		[]string,
		len(channel.Category.Subcategories))

	for _, c := range channel.Category.Subcategories {
		feed.Category.Subcategories = append(feed.Category.Subcategories, c.Name)
	}

	return &feed
}

func itemsFromRSS(doc *rss.Document) []api.Item {
	items := make([]api.Item, len(doc.Channel.Items))
	for i, item := range doc.Channel.Items {
		jsonItem := &items[i]

		jsonItem.GUID = item.GUID
		jsonItem.Title = item.Title
		jsonItem.Author = item.Author
		jsonItem.URL = item.Enclosure.URL
		jsonItem.Size = item.Enclosure.Length
		jsonItem.PublicationTime = rss.ParseDate(item.PublicationDate)
		jsonItem.Duration, _ = rss.ParseDuration(item.Duration)
		jsonItem.ImageURL = item.Image.URL
		jsonItem.Link = item.Link

		// Choose one description and one summary
		// break when first preferred description is found

		chooseOne := func(s *string, choices []string) {
			for _, c := range choices {
				if c != "" {
					*s = c
					break
				}
			}
		}

		chooseOne(&jsonItem.Summary, []string{
			item.ITunesSummary,
			item.ITunesSubtitle,
			item.Description,
		})

		chooseOne(&jsonItem.Description, []string{
			item.ContentEncoded,
			item.ITunesSummary,
			item.Description,
			jsonItem.Summary,
		})
	}

	return items
}
