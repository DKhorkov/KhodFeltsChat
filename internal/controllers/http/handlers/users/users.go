package users

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/DKhorkov/libs/pointers"

	"github.com/DKhorkov/kfc/internal/controllers/http/mappers"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

const (
	usernameQueryKey = "username"
	limitQueryKey    = "limit"
	offsetQueryKey   = "offset"
)

// swagger:route GET /users users GetUsers
//
// GetUsers
//
// Provides list of Users.
//
// Responses:
//	200: []User
//	500: InternalServerError

// GetUsersHandler provides Users with filtration and pagination.
func GetUsersHandler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var filters *domains.UsersFilters
		if username := r.URL.Query().Get(usernameQueryKey); username != "" {
			filters = &domains.UsersFilters{
				Username: pointers.New(username),
			}
		}

		limitStr := r.URL.Query().Get(limitQueryKey)
		limit, _ := strconv.Atoi(limitStr)

		offsetStr := r.URL.Query().Get(offsetQueryKey)
		offset, _ := strconv.Atoi(offsetStr)

		var pagination *domains.Pagination
		if offset != 0 && limit != 0 {
			pagination = &domains.Pagination{
				Offset: pointers.New(uint64(offset)),
				Limit:  pointers.New(uint64(limit)),
			}
		}

		users, err := u.GetUsers(r.Context(), filters, pagination)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		if err = json.NewEncoder(w).Encode(mappers.MapUsers(users)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
