package types

type PaginationResult[T any] struct {
	Items           []T  `json:"items"`
	TotalItems      int  `json:"total_items"`
	Page            int  `json:"page"`
	PageSize        int  `json:"page_size"`
	TotalPages      int  `json:"total_pages"`
	HasNextPage     bool `json:"has_next_page"`
	HasPreviousPage bool `json:"has_previous_page"`
}
