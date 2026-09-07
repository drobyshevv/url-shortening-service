package model

import "time"

type UrlFilter struct {
	Search *string `json:"search" validate:"omitempty,min=1"`

	DateFrom *time.Time `json:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo   *time.Time `json:"date_to" validate:"omitempty,datetime=2006-01-02"`

	MinClicks *int `json:"min_clicks" validate:"omitempty,min=0"`
	MaxClicks *int `json:"max_clicks" validate:"omitempty,min=1"`

	Sort    string `json:"sort" validate:"omitempty,oneof=url short_code date"`
	OrderBy string `json:"order_by" validate:"omitempty,oneof=asc desc"`
}
