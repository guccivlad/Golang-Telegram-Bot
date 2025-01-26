package source

import (
	"bot/internal/model"
	"context"

	"github.com/SlyMarbo/rss"
	"github.com/samber/lo"
)

type RSSSource struct {
	URL        string
	sourceID   int64
	sourceName string
}

func (s RSSSource) ID() int64 {
	return s.sourceID
}

func (s RSSSource) Name() string {
	return s.sourceName
}

func NewRSSSourceFromModel(m model.Source) RSSSource {
	return RSSSource{
		URL:        m.FeedUrl,
		sourceID:   m.ID,
		sourceName: m.Name,
	}
}

func (s RSSSource) Fetch(ctx context.Context) ([]model.Item, error) {
	feed, err := s.loadFeed(ctx, s.URL)

	if err != nil {
		return nil, err
	}

	return lo.Map(feed.Items, func(item *rss.Item, _ int) model.Item {
		return model.Item{
			Title:      item.Title,
			Catigories: item.Categories,
			Link:       item.Link,
			Date:       item.Date,
			Summary:    item.Summary,
			SourceName: s.sourceName,
		}
	}), nil
}

func (s RSSSource) loadFeed(ctx context.Context, url string) (*rss.Feed, error) {
	feedChan := make(chan *rss.Feed)
	errorChan := make(chan error)

	go func() {
		feed, err := rss.Fetch(url)

		if err != nil {
			errorChan <- err
			return
		}

		feedChan <- feed

	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errorChan:
		return nil, err
	case feed := <-feedChan:
		return feed, nil
	}
}
