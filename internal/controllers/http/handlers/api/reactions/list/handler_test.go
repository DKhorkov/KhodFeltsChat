package list_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/reactions/list"
	"github.com/DKhorkov/kfc/internal/domains"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestListHandler_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)
	u.EXPECT().
		ListReactions(gomock.Any()).
		Return([]domains.Reaction{{ID: 1, Emoji: "👍"}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/reactions", http.NoBody)
	rec := httptest.NewRecorder()

	list.Handler(u).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[{"id":1,"emoji":"👍"}]`, rec.Body.String())
}

func TestListHandler_ServiceError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)
	u.EXPECT().ListReactions(gomock.Any()).Return(nil, errors.New("boom"))

	req := httptest.NewRequest(http.MethodGet, "/api/reactions", http.NoBody)
	rec := httptest.NewRecorder()

	list.Handler(u).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestListHandler_EmptyDictionary(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	u := mockusecases.NewMockReactionsUseCases(ctrl)
	u.EXPECT().ListReactions(gomock.Any()).Return([]domains.Reaction{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/reactions", http.NoBody)
	rec := httptest.NewRecorder()

	list.Handler(u).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[]`, rec.Body.String())
}
