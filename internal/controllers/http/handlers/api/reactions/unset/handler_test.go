package unset_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/reactions/unset"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockcontrollers "github.com/DKhorkov/kfc/mocks/controllers"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func buildReq(
	t *testing.T,
	messageID, reactionID string,
	userID uint64,
	withUser bool,
) *http.Request {
	t.Helper()

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/messages/"+messageID+"/reactions/"+reactionID,
		http.NoBody,
	)
	req = mux.SetURLVars(req, map[string]string{
		common.IDRouteKey:         messageID,
		common.ReactionIDRouteKey: reactionID,
	})

	if withUser {
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
		req = req.WithContext(ctx)
	}

	return req
}

func newHandler(t *testing.T) (
	http.HandlerFunc,
	*mockusecases.MockReactionsUseCases,
	*mockcontrollers.MockWSBroadcaster,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)
	b := mockcontrollers.NewMockWSBroadcaster(ctrl)

	return unset.Handler(u, b), u, b
}

func TestUnsetHandler_NoContent_Success_Broadcasts(t *testing.T) {
	t.Parallel()

	h, u, b := newHandler(t)

	u.EXPECT().
		RemoveReaction(gomock.Any(), gomock.Any()).
		Return(nil)
	b.EXPECT().
		BroadcastReactionRemoved(gomock.Any(), uint64(10), uint64(7), uint64(1)).
		Times(1)

	req := buildReq(t, "10", "1", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUnsetHandler_NoContent_Idempotent_WhenNothingDeleted_NoBroadcast(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		RemoveReaction(gomock.Any(), gomock.Any()).
		Return(customerrors.ErrReactionNotSet)
	// b без EXPECT — Broadcast НЕ должен вызваться.

	req := buildReq(t, "10", "1", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUnsetHandler_Unauthorized_NoUserID(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "10", "1", 0, false)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUnsetHandler_BadRequest_InvalidMessageID(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "not-a-number", "1", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUnsetHandler_BadRequest_InvalidReactionID(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "10", "not-a-number", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUnsetHandler_NotFound_MessageNotFound(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		RemoveReaction(gomock.Any(), gomock.Any()).
		Return(customerrors.ErrMessageNotFound)

	req := buildReq(t, "10", "1", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUnsetHandler_Forbidden_NotChatMember(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		RemoveReaction(gomock.Any(), gomock.Any()).
		Return(customerrors.ErrUserIsNotChatMember)

	req := buildReq(t, "10", "1", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUnsetHandler_InternalError(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		RemoveReaction(gomock.Any(), gomock.Any()).
		Return(errors.New("boom"))

	req := buildReq(t, "10", "1", 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
