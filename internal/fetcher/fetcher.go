package fetcher

import (
	"bot/internal/model"
	"bot/internal/source"
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

type ArticleStorage interface {
	Store(ctc context.Context, article model.Article) error
}

type SourseList interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

type Source interface {
	ID() int64
	Name() string

	Fetch(ctx context.Context) ([]model.Item, error)
}

type Fetcher struct {
	articles ArticleStorage
	soursec  SourseList

	keywords      []string
	fetchInterval time.Duration
}

func New(articles ArticleStorage, sources SourseList, keywords []string, fetchInterval time.Duration) *Fetcher {
	return &Fetcher{
		articles:      articles,
		soursec:       sources,
		keywords:      keywords,
		fetchInterval: fetchInterval,
	}
}

func (f *Fetcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.soursec.Sources(ctx)

	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)

		rssSource := source.NewRSSSourceFromModel(src)

		go func(source Source) {
			items, err := source.Fetch(ctx)

			if err != nil {
				log.Println("ERROR: fetch failed")
			}

			f.processItem(ctx, source, items)

			wg.Done()
		}(rssSource)
	}

	wg.Wait()

	return nil
}

func (f *Fetcher) processItem(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		item.Date = item.Date.UTC()

		if f.IsSkipped(item) {
			continue
		}

		article := model.Article{
			SourceId:    source.ID(),
			Title:       item.Title,
			Link:        item.Link,
			Summury:     item.Summary,
			PublishedAt: item.Date,
		}

		err := f.articles.Store(ctx, article)

		if err != nil {
			return err
		}
	}

	return nil
}

func (f *Fetcher) IsSkipped(item model.Item) bool {
	categories := item.Catigories

	for _, keyword := range f.keywords {
		isTitleContains := strings.Contains(strings.ToLower(item.Title), keyword)
		isCatContains := false

		for _, categorie := range categories {
			if categorie == keyword {
				isCatContains = true
			}
		}

		if isTitleContains || isCatContains {
			return true
		}
	}

	return false
}
