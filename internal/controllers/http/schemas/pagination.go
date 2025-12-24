package schemas

// Pagination
// swagger:parameters Pagination
type Pagination struct {
	// Maximum amount of entries
	// required: false
	// nullable: false
	// in: query
	Limit int `json:"limit"`

	// Number of entries to skip from the beginning of the result set before starting to return the desired entries
	// required: false
	// nullable: false
	// in: query
	Offset int `json:"offset"`
}
