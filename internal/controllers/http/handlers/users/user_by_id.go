package users

import (
	"encoding/json"
	"errors"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

const (
	IDRouteKey = "id"
)

func GetUserByIDHandler(u interfaces.UsersUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := mux.Vars(r)[IDRouteKey]
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		user, err := u.GetUserByID(r.Context(), userID)
		switch {
		case errors.Is(err, customerrors.ErrUserNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		if err = json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
