package model

import (
	"time"
)

type Item struct {
	Title      string
	Catigories []string
	Link       string
	Date       time.Time // дата публикации в источнике
	Summary    string
	SourceName string
}

type Source struct {
	ID        int64
	Name      string
	FeedUrl   string
	CreatedAt time.Time
}

type Article struct {
	ID          int64
	SourceId    int64
	Title       string
	Link        string
	Summury     string
	PublishedAt time.Time // время публикации в источнике
	CreatedAt   time.Time
	PostedAt    time.Time
}
