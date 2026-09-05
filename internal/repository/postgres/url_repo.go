package postgres

import (
	"context"

	"github.com/drobyshevv/url-shortening-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UrlRepository struct {
	pool *pgxpool.Pool
}

func NewUrlRepository(pool *pgxpool.Pool) *UrlRepository {
	return &UrlRepository{
		pool: pool,
	}
}

func (r *UrlRepository) Create(ctx context.Context, url string, code string) (*model.Url, error) {
	var result model.Url

	row := r.pool.QueryRow(ctx, `
	INSERT INTO urls (short_code, url)
	VALUES ($1, $2)
	RETURNING id, url, short_code, created_at, updated_at
	`, code, url)

	err := row.Scan(
		&result.ID,
		&result.Url,
		&result.ShortCode,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *UrlRepository) Get(ctx context.Context, code string) (*model.Url, error) {
	var result model.Url

	err := r.pool.QueryRow(ctx, `
		SELECT id, url, short_code, created_at, updated_at
		FROM urls
		WHERE short_code = $1
	`, code).Scan(
		&result.ID,
		&result.Url,
		&result.ShortCode,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *UrlRepository) Update(ctx context.Context, code string, url string) (*model.Url, error) {
	var result model.Url

	err := r.pool.QueryRow(ctx, `
		UPDATE urls
		SET url = $1, updated_at = NOW()
		WHERE short_code = $2
		RETURNING id, url, short_code, created_at, updated_at
	`, url, code).Scan(
		&result.ID,
		&result.Url,
		&result.ShortCode,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *UrlRepository) Delete(ctx context.Context, code string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM urls
		WHERE short_code = $1
	`, code)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	return nil
}

func (r *UrlRepository) GetStatistics(ctx context.Context, code string) (*model.UrlStats, error) {
	var result model.UrlStats

	err := r.pool.QueryRow(ctx, `
		SELECT id, url, short_code, created_at, updated_at, access_count
		FROM urls
		WHERE short_code = $1
	`, code).Scan(
		&result.ID,
		&result.Url,
		&result.ShortCode,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.AccessCount,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *UrlRepository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	var exists bool

	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM urls
			WHERE url = $1
		)
	`, url).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
