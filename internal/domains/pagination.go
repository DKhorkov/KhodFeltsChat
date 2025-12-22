package domains

type Pagination struct {
	Limit  *uint64 `json:"limit,omitempty"`
	Offset *uint64 `json:"offset,omitempty"`
}
