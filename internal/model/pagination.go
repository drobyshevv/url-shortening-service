package model

type Pagination struct {
	CurrentPage int  `json:"current_page"`
	PageSize    int  `json:"page_size"`
	TotalItems  int  `json:"total_items"`
	TotalPages  int  `json:"total_pages"`
	HasNextPage bool `json:"has_next"`
	HasPrevPage bool `json:"has_prev"`
}

func (p *Pagination) Offset() int {
	return (p.CurrentPage - 1) * p.PageSize
}
