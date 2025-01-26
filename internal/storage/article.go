package storage

import (
	"bot/internal/model"
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"
)

type ArticlePostgresStorage struct {
	db *sqlx.DB
}

type dbArticle struct {
	ID          int64          `db:"id"`
	SourceID    int64          `db:"source_id"`
	Title       string         `db:"title"`
	Link        string         `db:"link"`
	Summary     sql.NullString `db:"summary"`
	PublishedAt time.Time      `db:"published_at"`
	PostedAt    sql.NullTime   `db:"posted_at"`
	CreatedAt   time.Time      `db:"created_at"`
}

func NewArticleStorage(db *sqlx.DB) *ArticlePostgresStorage {
	return &ArticlePostgresStorage{
		db: db,
	}
}

func (s *ArticlePostgresStorage) Store(ctx context.Context, article model.Article) error {
	conn, err := s.db.Connx(ctx)
	defer conn.Close()

	if err != nil {
		return err
	}

	_, execErr := conn.ExecContext(
		ctx,
		`INSERT INTO articles (source_id, title, link, summary, published_at)
	    				VALUES ($1, $2, $3, $4, $5)
	    				ON CONFLICT DO NOTHING;`,
		article.SourceId,
		article.Title,
		article.Link,
		article.Summury,
		article.PublishedAt,
	)

	if execErr != nil {
		return execErr
	}

	return nil
}

func (s *ArticlePostgresStorage) MarkAsPosted(ctx context.Context, id int64) error {
	conn, err := s.db.Connx(ctx)
	defer conn.Close()

	if err != nil {
		return err
	}

	_, execErr := conn.ExecContext(
		ctx,
		`UPDATE articles SET posted_at = $1::TIMESTAMP WHERE id = $2;`,
		time.Now().UTC().Format(time.RFC3339),
		id,
	)

	if execErr != nil {
		return err
	}

	return nil
}

func (s *ArticlePostgresStorage) AllNotPosted(ctx context.Context, since time.Time, limit uint64) ([]model.Article, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var articles []dbArticle

	if err := conn.SelectContext(
		ctx,
		&articles,
		`SELECT * FROM articles
			WHERE posted_at IS NULL AND published_at >= $1::TIMESTAMP ORDER BY published_at DESC LIMIT $2`,
		since.UTC().Format(time.RFC3339),
		limit,
	); err != nil {
		return nil, err
	}

	return lo.Map(articles, func(article dbArticle, _ int) model.Article {
		return model.Article{
			ID:          article.ID,
			SourceId:    article.SourceID,
			Title:       article.Title,
			Link:        article.Link,
			Summury:     article.Summary.String,
			PublishedAt: article.PublishedAt,
			CreatedAt:   article.CreatedAt,
		}
	}), nil
}
