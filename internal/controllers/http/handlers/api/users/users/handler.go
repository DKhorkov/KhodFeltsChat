package users

import (
	"encoding/json"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/users"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/pointers"
)

const (
	usernameQueryKey = "username"
)

// swagger:route GET /api/users users GetUsers
//
// GetUsers
//
// Provides list of Users.
//
// Responses:
//	200: []User
//	500: InternalServerError

// Handler provides Users with filtration and pagination.
func Handler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var filters *domains.UsersFilters
		if username := r.URL.Query().Get(usernameQueryKey); username != "" {
			filters = &domains.UsersFilters{
				Username: pointers.New(username),
			}
		}

		pagination := common.GetPaginationFromRequest(r)

		users, err := u.GetUsers(r.Context(), filters, pagination)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Устанавливаем статус код и заголовок Content-Type
		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJSONContentType)

		// HTTP статус код должен устанавливаться перед записью тела ответа
		w.WriteHeader(http.StatusOK)

		if err = json.NewEncoder(w).Encode(mappers.MapUsers(users)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
