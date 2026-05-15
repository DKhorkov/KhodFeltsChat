package subscribe_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/push/subscribe"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	createRequest := func(t *testing.T, userID uint64, body any) *http.Request {
		t.Helper()

		data, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", bytes.NewReader(data))
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := subscribe.Handler(mockUseCase)

		userID := uint64(123)
		now := time.Now()
		expectedSubscription := &domains.PushSubscription{
			ID:            1,
			UserID:        userID,
			Endpoint:      "https://fcm.googleapis.com/fcm/send/test",
			EncryptionKey: "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8p8V953hA",
			Auth:          "tBHItJI5svbpC7htE1Gg3A",
			CreatedAt:     now,
		}

		requestBody := map[string]any{
			"endpoint": "https://fcm.googleapis.com/fcm/send/test",
			"keys": map[string]string{
				"encryptionKey": "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8p8V953hA",
				"auth":          "tBHItJI5svbpC7htE1Gg3A",
			},
		}

		mockUseCase.EXPECT().
			CreatePushSubscription(gomock.Any(), domains.PushSubscription{
				UserID:        userID,
				Endpoint:      "https://fcm.googleapis.com/fcm/send/test",
				EncryptionKey: "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8p8V953hA",
				Auth:          "tBHItJI5svbpC7htE1Gg3A",
			}).
			Return(expectedSubscription, nil)

		req := createRequest(t, userID, requestBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["id"])
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := subscribe.Handler(mockUseCase)

		req := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockPushSubscriptionsUseCases(ctrl)
		handler := subscribe.Handler(mockUseCase)

		userID := uint64(123)

		req := httptest.NewRequest(http.MethodPost, "/api/push/subscribe", bytes.NewReader([]byte("invalid json")))
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
		req = req.WithContext(ctx)
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
		handler := subscribe.Handler(mockUseCase)

		userID := uint64(123)
		requestBody := map[string]any{
			"endpoint": "https://fcm.googleapis.com/fcm/send/test",
			"keys": map[string]string{
				"encryptionKey": "test-key",
				"auth":          "test-auth",
			},
		}

		mockUseCase.EXPECT().
			CreatePushSubscription(gomock.Any(), domains.PushSubscription{
				UserID:        userID,
				Endpoint:      "https://fcm.googleapis.com/fcm/send/test",
				EncryptionKey: "test-key",
				Auth:          "test-auth",
			}).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, userID, requestBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
	})
}
