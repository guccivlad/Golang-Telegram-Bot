package storage

import (
	"bot/internal/model"
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"
)

type SourcePostgresStorage struct {
	db *sqlx.DB
}

type dbSource struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	FeedUrl   string    `db:"feed_url"`
	CreatedAt time.Time `db:"created_at"`
}

func NewSourceStorage(db *sqlx.DB) *SourcePostgresStorage {
	return &SourcePostgresStorage{
		db: db,
	}
}

func (s *SourcePostgresStorage) Sources(ctx context.Context) ([]model.Source, error) {
	conn, err := s.db.Connx(ctx)
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	var sources []dbSource
	err = conn.SelectContext(ctx, &sources, "SELECT * FROM sources")

	if err != nil {
		return nil, err
	}

	return lo.Map(sources, func(source dbSource, _ int) model.Source { return model.Source(source) }), nil
}

func (s *SourcePostgresStorage) sourceByID(ctx context.Context, id int64) (*model.Source, error) {
	conn, err := s.db.Connx(ctx)
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	var source dbSource

	err = conn.GetContext(ctx, &source, "SELECT * FROM sources WHERE id = $1", id)

	if err != nil {
		return nil, err
	}

	result := model.Source(source)

	return &result, nil
}

func (s *SourcePostgresStorage) add(ctx context.Context, source model.Source) (int64, error) {
	conn, err := s.db.Connx(ctx)
	defer conn.Close()

	if err != nil {
		return -1, err
	}

	var id int64

	row := conn.QueryRowxContext(ctx, `INSERT INTO sources (name, feed_url, created_at) VALUES($1, $2, $3) RETURNING id;`,
		source.Name, source.FeedUrl, source.CreatedAt)

	err = row.Scan(&id)

	if err != nil {
		return -1, err
	}

	return id, nil
}

func (s *SourcePostgresStorage) delete(ctx context.Context, id int64) error {
	conn, err := s.db.Connx(ctx)
	defer conn.Close()

	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, "DELETE FROM sources WHERE id = $1", id)

	if err != nil {
		return err
	}

	return nil
}
