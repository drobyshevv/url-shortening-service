package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (r *UrlRepository) TotalItems(ctx context.Context, filter model.UrlFilter) (int, error) {
	query := `SELECT COUNT(*) FROM urls`

	query, args := buildFilterQuery(query, filter)

	var totalItems int

	err := r.pool.QueryRow(ctx, query, args...).Scan(&totalItems)
	if err != nil {
		return 0, err
	}

	return totalItems, err
}

func (r *UrlRepository) GetPage(ctx context.Context, pageSize int, offset int, filter model.UrlFilter) ([]model.UrlStats, error) {
	query := `
		SELECT id, url, short_code, created_at, updated_at, access_count
		FROM urls`

	query, args := buildFilterQuery(query, filter)

	sortColumn := "created_at"
	if filter.Sort == "url" {
		sortColumn = "url"
	} else if filter.Sort == "short_code" {
		sortColumn = "short_code"
	} else if filter.Sort == "date" {
		sortColumn = "created_at"
	}

	sortOrder := "DESC"
	if filter.OrderBy == "asc" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d",
		sortColumn, sortOrder, len(args)+1, len(args)+2)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
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

func buildFilterQuery(query string, filter model.UrlFilter) (string, []any) {
	var args []any
	var conditions []string
	argIndex := 1
	if filter.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(url ILIKE $%d OR short_code ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}

	if filter.MinClicks != nil {
		conditions = append(conditions, fmt.Sprintf("access_count >= $%d", argIndex))
		args = append(args, *filter.MinClicks)
		argIndex++
	}

	if filter.MaxClicks != nil {
		conditions = append(conditions, fmt.Sprintf("access_count <= $%d", argIndex))
		args = append(args, *filter.MaxClicks)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}
