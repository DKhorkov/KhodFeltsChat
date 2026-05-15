package unsubscribe_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/push/unsubscribe"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	createRequest := func(t *testing.T, userID uint64, subscriptionID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodDelete, "/api/push/subscriptions/"+strconv.FormatUint(subscriptionID, 10), http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		vars := map[string]string{
			common.IDRouteKey: strconv.FormatUint(subscriptionID, 10),
		}

		return mux.SetURLVars(req.WithContext(ctx), vars)
	}

	t.Run("successful deletion", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := unsubscribe.Handler(mockUseCase)

		userID := uint64(123)
		subscriptionID := uint64(1)

		mockUseCase.EXPECT().
			DeletePushSubscription(gomock.Any(), subscriptionID).
			Return(nil)

		req := createRequest(t, userID, subscriptionID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := unsubscribe.Handler(mockUseCase)

		req := httptest.NewRequest(http.MethodDelete, "/api/push/subscriptions/1", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := unsubscribe.Handler(mockUseCase)

		userID := uint64(123)

		req := httptest.NewRequest(http.MethodDelete, "/api/push/subscriptions/invalid", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		vars := map[string]string{
			common.IDRouteKey: "invalid",
		}

		req = mux.SetURLVars(req.WithContext(ctx), vars)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("internal server error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := unsubscribe.Handler(mockUseCase)

		userID := uint64(123)
		subscriptionID := uint64(1)

		mockUseCase.EXPECT().
			DeletePushSubscription(gomock.Any(), subscriptionID).
			Return(errors.New("database connection failed"))

		req := createRequest(t, userID, subscriptionID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
	})
}
