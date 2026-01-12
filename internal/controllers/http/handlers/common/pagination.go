package common

import (
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/pointers"
)

const (
	limitQueryKey  = "limit"
	offsetQueryKey = "offset"
)

func GetPaginationFromRequest(r *http.Request) *domains.Pagination {
	var pagination *domains.Pagination

	limitStr := r.URL.Query().Get(limitQueryKey)
	limit, _ := strconv.ParseUint(limitStr, 10, 64)

	offsetStr := r.URL.Query().Get(offsetQueryKey)
	offset, _ := strconv.ParseUint(offsetStr, 10, 64)

	if offset != 0 || limit != 0 {
		pagination = &domains.Pagination{
			Offset: pointers.New(offset),
			Limit:  pointers.New(limit),
		}
	}

	return pagination
}
