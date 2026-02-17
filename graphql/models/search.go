package models

type ProductSearchResult struct {
	Items      []*Product `json:"items"`
	TotalCount int32      `json:"total_count"`
	PageInfo   *PageInfo  `json:"page_info"`
}

type PageInfo struct {
	PageSize    int32 `json:"page_size"`
	CurrentPage int32 `json:"current_page"`
	TotalPages  int32 `json:"total_pages"`
}
