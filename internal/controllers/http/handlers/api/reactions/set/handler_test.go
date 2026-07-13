package set_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/reactions/set"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
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
	messageID string,
	body string,
	userID uint64,
	withUser bool,
) *http.Request {
	t.Helper()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages/"+messageID+"/reactions",
		strings.NewReader(body),
	)
	req = mux.SetURLVars(req, map[string]string{common.IDRouteKey: messageID})

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

	return set.Handler(u, b), u, b
}

func TestSetHandler_NoContent_Success_Broadcasts(t *testing.T) {
	t.Parallel()

	h, u, b := newHandler(t)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(&domains.Reaction{ID: 1, Emoji: "👍"}, nil)
	b.EXPECT().
		BroadcastReactionAdded(gomock.Any(), uint64(10), uint64(7), uint64(1)).
		Times(1)

	req := buildReq(t, "10", `{"reactionId":1}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestSetHandler_Unauthorized_NoUserID(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "10", `{"reactionId":1}`, 0, false)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSetHandler_BadRequest_InvalidMessageID(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "not-a-number", `{"reactionId":1}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetHandler_BadRequest_InvalidBody(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "10", `not-json`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetHandler_BadRequest_ZeroReactionID(t *testing.T) {
	t.Parallel()

	h, _, _ := newHandler(t)

	req := buildReq(t, "10", `{"reactionId":0}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetHandler_Conflict_Duplicate_NoBroadcast(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(nil, customerrors.ErrReactionAlreadyExists)

	// b без EXPECT — Broadcast НЕ должен быть вызван.
	req := buildReq(t, "10", `{"reactionId":1}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSetHandler_NotFound_UnknownReaction(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(nil, customerrors.ErrReactionNotFound)

	req := buildReq(t, "10", `{"reactionId":999}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetHandler_NotFound_MessageNotFound(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(nil, customerrors.ErrMessageNotFound)

	req := buildReq(t, "10", `{"reactionId":1}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetHandler_Forbidden_NotChatMember(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(nil, customerrors.ErrUserIsNotChatMember)

	req := buildReq(t, "10", `{"reactionId":1}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSetHandler_InternalError(t *testing.T) {
	t.Parallel()

	h, u, _ := newHandler(t)

	u.EXPECT().
		AddReaction(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("boom"))

	req := buildReq(t, "10", `{"reactionId":1}`, 7, true)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
