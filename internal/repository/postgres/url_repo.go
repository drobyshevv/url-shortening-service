package postgres

import (
	"context"
	"errors"

	"github.com/drobyshevv/url-shortening-service/internal/errs"
	"github.com/drobyshevv/url-shortening-service/internal/model"
	"github.com/jackc/pgx/v5"
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

func (r *UrlRepository) Create(ctx context.Context, url string, code string) (*model.ShortLink, error) {
	var result model.ShortLink

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

func (r *UrlRepository) Get(ctx context.Context, code string) (*model.ShortLink, error) {
	var result model.ShortLink

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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}

	return &result, nil
}

func (r *UrlRepository) Update(ctx context.Context, code string, url string) (*model.ShortLink, error) {
	var result model.ShortLink

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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
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
		return errs.ErrNotFound
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotFound
		}
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

func (r *UrlRepository) TotalItems(ctx context.Context) (int, error) {
	var totalItems int

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM urls
	`).Scan(&totalItems)
	if err != nil {
		return 0, err
	}

	return totalItems, err
}

func (r *UrlRepository) GetPage(ctx context.Context, pageSize int, offset int) ([]model.UrlStats, error) {
	query := `
		SELECT id, url, short_code, created_at, updated_at, access_count
		FROM urls
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.UrlStats, 0)
	for rows.Next() {
		var item model.UrlStats
		err := rows.Scan(
			&item.ID,
			&item.Url,
			&item.ShortCode,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.AccessCount,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
