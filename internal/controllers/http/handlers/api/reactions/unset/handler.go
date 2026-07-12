package unset

import (
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

const ReactionIDRouteKey = "reactionId"

// swagger:route DELETE /api/messages/{id}/reactions/{reactionId} reactions UnsetMessageReaction
//
// UnsetMessageReaction
//
// Removes a reaction from a message for the current user. Idempotent — 204 even if no such reaction.
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
//	500: InternalServerError

// Handler removes reaction from a message.
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

		reactionID, err := strconv.ParseUint(mux.Vars(r)[ReactionIDRouteKey], 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		dto := domains.MessageReactionDTO{
			MessageID:  messageID,
			ReactionID: reactionID,
			UserID:     userID,
		}

		err = u.RemoveReaction(r.Context(), dto)

		switch {
		case errors.Is(err, customerrors.ErrReactionNotSet):
			// Идемпотентная семантика: реакции не было — отвечаем успехом,
			// WS-событие не публикуем (иначе спамили бы всех клиентов).
			w.WriteHeader(http.StatusNoContent)

			return
		case errors.Is(err, customerrors.ErrMessageNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case errors.Is(err, customerrors.ErrUserIsNotChatMember):
			http.Error(w, err.Error(), http.StatusForbidden)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		broadcaster.BroadcastReactionRemoved(
			r.Context(),
			dto.MessageID, dto.UserID, dto.ReactionID,
		)

		w.WriteHeader(http.StatusNoContent)
	}
}
