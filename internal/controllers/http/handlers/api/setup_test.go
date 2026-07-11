package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api"
	mockupgrader "github.com/DKhorkov/kfc/mocks/upgrader"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupAPIRouter(t *testing.T) *mux.Router {
	t.Helper()

	ctrl := gomock.NewController(t)

	apiMux := mux.NewRouter()
	api.SetupHandlers(
		apiMux,
		config.CookiesConfig{},
		config.NATSConfig{},
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockAuthUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		mockusecases.NewMockReactionsUseCases(ctrl),
		mockusecases.NewMockSettingsUseCases(ctrl),
		mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl),
		mockusecases.NewMockFileStorageUseCases(ctrl),
		mocks.NewMockLogger(ctrl),
		mockupgrader.NewMockUpgrader(ctrl),
		nil,
		"",
	)

	return apiMux
}

func TestSetupHandlers_RoutesRegistered(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"GET /users", http.MethodGet, "/users"},
		{"GET /users/me", http.MethodGet, "/users/me"},
		{"GET /users/1", http.MethodGet, "/users/1"},
		{"GET /ws", http.MethodGet, "/ws"},
		{"GET /chats", http.MethodGet, "/chats"},
		{"GET /chats/1/messages", http.MethodGet, "/chats/1/messages"},
		{"GET /users/email/verify/test-token", http.MethodGet, "/users/email/verify/test-token"},
		{"POST /users", http.MethodPost, "/users"},
		{"POST /sessions", http.MethodPost, "/sessions"},
		{"POST /users/password/change", http.MethodPost, "/users/password/change"},
		{"POST /users/email/verify", http.MethodPost, "/users/email/verify"},
		{
			"POST /users/password/forget/test-token",
			http.MethodPost,
			"/users/password/forget/test-token",
		},
		{"POST /users/password/forget", http.MethodPost, "/users/password/forget"},
		{"POST /chats", http.MethodPost, "/chats"},
		{"PUT /users/me", http.MethodPut, "/users/me"},
		{"PUT /sessions", http.MethodPut, "/sessions"},
		{"DELETE /sessions", http.MethodDelete, "/sessions"},
		{"GET /users/me/settings", http.MethodGet, "/users/me/settings"},
		{"PUT /users/me/settings", http.MethodPut, "/users/me/settings"},
		{"GET /web-push/vapid-key", http.MethodGet, "/web-push/vapid-key"},
		{"POST /web-push/subscribe", http.MethodPost, "/web-push/subscribe"},
		{"DELETE /web-push/subscribe/1", http.MethodDelete, "/web-push/subscribe/1"},
		{"GET /reactions", http.MethodGet, "/reactions"},
		{"POST /messages/1/reactions", http.MethodPost, "/messages/1/reactions"},
		{"DELETE /messages/1/reactions/2", http.MethodDelete, "/messages/1/reactions/2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			match := mux.RouteMatch{}
			assert.True(
				t,
				apiMux.Match(req, &match),
				"Expected %s %s to be registered",
				tt.method,
				tt.path,
			)
		})
	}
}

func TestSetupHandlers_UnregisteredRoute(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	rr := httptest.NewRecorder()

	apiMux.ServeHTTP(rr, req)

	assert.Equal(
		t,
		http.StatusNotFound,
		rr.Code,
		"Expected status %d for unregistered route, got %d",
		http.StatusNotFound,
		rr.Code,
	)
}
