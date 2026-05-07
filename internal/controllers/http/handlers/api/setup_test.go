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
	"go.uber.org/mock/gomock"
)

func setupAPIRouter(t *testing.T) *mux.Router {
	t.Helper()

	ctrl := gomock.NewController(t)

	apiMux := mux.NewRouter()
	api.SetupHandlers(
		apiMux,
		config.CookiesConfig{},
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockAuthUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		mocks.NewMockLogger(ctrl),
		mockupgrader.NewMockUpgrader(ctrl),
	)

	return apiMux
}

func TestSetupHandlers_GetRoutesRegistered(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	routes := []string{
		"/users",
		"/users/me",
		"/users/1",
		"/ws",
		"/chats",
		"/chats/1/messages",
		"/users/email/verify/test-token",
	}

	for _, path := range routes {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		match := mux.RouteMatch{}

		if !apiMux.Match(req, &match) {
			t.Errorf("Expected GET %s to be registered", path)
		}
	}
}

func TestSetupHandlers_PostRoutesRegistered(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	routes := []string{
		"/users",
		"/sessions",
		"/users/password/change",
		"/users/email/verify",
		"/users/password/forget/test-token",
		"/users/password/forget",
		"/chats",
	}

	for _, path := range routes {
		req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
		match := mux.RouteMatch{}

		if !apiMux.Match(req, &match) {
			t.Errorf("Expected POST %s to be registered", path)
		}
	}
}

func TestSetupHandlers_PutRoutesRegistered(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	routes := []string{
		"/users/me",
		"/sessions",
	}

	for _, path := range routes {
		req := httptest.NewRequest(http.MethodPut, path, http.NoBody)
		match := mux.RouteMatch{}

		if !apiMux.Match(req, &match) {
			t.Errorf("Expected PUT %s to be registered", path)
		}
	}
}

func TestSetupHandlers_DeleteRoutesRegistered(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	routes := []string{
		"/sessions",
	}

	for _, path := range routes {
		req := httptest.NewRequest(http.MethodDelete, path, http.NoBody)
		match := mux.RouteMatch{}

		if !apiMux.Match(req, &match) {
			t.Errorf("Expected DELETE %s to be registered", path)
		}
	}
}

func TestSetupHandlers_UnregisteredRoute(t *testing.T) {
	t.Parallel()

	apiMux := setupAPIRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	rr := httptest.NewRecorder()

	apiMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for unregistered route, got %d", http.StatusNotFound, rr.Code)
	}
}
