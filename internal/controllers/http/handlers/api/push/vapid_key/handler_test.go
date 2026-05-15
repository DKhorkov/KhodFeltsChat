package vapid_key_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	vapidkey "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/push/vapid_key"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	t.Run("returns VAPID public key", func(t *testing.T) {
		t.Parallel()

		// Arrange
		publicKey := "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8p8V953hA"
		handler := vapidkey.Handler(publicKey)

		req := httptest.NewRequest(http.MethodGet, "/api/push/vapid-key", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, publicKey, response["publicKey"])
	})
}
