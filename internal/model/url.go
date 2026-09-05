package model

import "time"

type Url struct {
	ID        int64     `json:"id" validate:"required,gt=0"`
	Url       string    `json:"url" validate:"required,url"`
	ShortCode string    `json:"short_code" validate:"required,len=8,alphanum"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UrlStats struct {
	Url
	AccessCount int64 `json:"access_count"`
}
