package set

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
)

// swagger:route POST /api/messages/{id}/reactions reactions SetMessageReaction
//
// SetMessageReaction
//
// Sets a reaction on a message for the current user. Duplicate → 409.
//
// Security:
// - cookieAuth: []
//
// Responses:
//	204: NoContent
//	400: BadRequest
//	401: Unauthorized
//	403: Forbidden
//	404: NotFound
//	409: Conflict
//	500: InternalServerError

// Handler sets a reaction on a message.
func Handler(
	u interfaces.ReactionsUseCases,
	broadcaster interfaces.WSBroadcaster,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := contextlib.ValueFromContext[uint64](
			r.Context(),
			authmiddleware.UserIDContextKey,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)

			return
		}

		messageID, err := strconv.ParseUint(mux.Vars(r)[common.IDRouteKey], 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		var dto domains.MessageReactionDTO
		if err = json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto.MessageID = messageID
		dto.UserID = userID

		if dto.ReactionID == 0 {
			http.Error(w, "reactionId is required", http.StatusBadRequest)

			return
		}

		chatID, emoji, err := u.AddReaction(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrReactionAlreadyExists):
			http.Error(w, err.Error(), http.StatusConflict)

			return
		case errors.Is(err, customerrors.ErrReactionNotFound),
			errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrUserIsNotChatMember):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		broadcaster.BroadcastReactionAdded(
			r.Context(),
			chatID, dto.MessageID, dto.UserID, dto.ReactionID,
			emoji,
		)

		w.WriteHeader(http.StatusNoContent)
	}
}
