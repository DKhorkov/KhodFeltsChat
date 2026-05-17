package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers"
	mockupgrader "github.com/DKhorkov/kfc/mocks/upgrader"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/gorilla/mux"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	// Тесты запускаются из директории пакета, а шаблоны загружаются
	// относительно корня проекта. Переходим в корень.
	if err := os.Chdir("../../../.."); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func setupRouter(t *testing.T) *mux.Router {
	t.Helper()

	ctrl := gomock.NewController(t)

	rootMux := mux.NewRouter()
	handlers.SetupHandlers(
		rootMux,
		config.DocsConfig{
			Dir:      "api",
			Filepath: "api/swagger.json",
		},
		config.CookiesConfig{},
		config.NATSConfig{},
		mockusecases.NewMockUsersUseCases(ctrl),
		mockusecases.NewMockAuthUseCases(ctrl),
		mockusecases.NewMockChatsUseCases(ctrl),
		mockusecases.NewMockMessagesUseCases(ctrl),
		mockusecases.NewMockSettingsUseCases(ctrl),
		mockusecases.NewMockWebPushSubscriptionsUseCases(ctrl),
		mocks.NewMockLogger(ctrl),
		mockupgrader.NewMockUpgrader(ctrl),
		nil,
		"",
	)

	return rootMux
}

func TestSetupHandlers_APIRoutesRegistered(t *testing.T) {
	t.Parallel()

	rootMux := setupRouter(t)

	routes := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/users"},
		{method: http.MethodGet, path: "/api/users/me"},
		{method: http.MethodGet, path: "/api/chats"},
		{method: http.MethodPost, path: "/api/sessions"},
		{method: http.MethodPut, path: "/api/sessions"},
		{method: http.MethodDelete, path: "/api/sessions"},
		{method: http.MethodPost, path: "/api/users"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, http.NoBody)
		match := mux.RouteMatch{}

		if !rootMux.Match(req, &match) {
			t.Errorf("Expected %s %s to be registered", route.method, route.path)
		}
	}
}

func TestSetupHandlers_WebRoutesRegistered(t *testing.T) {
	t.Parallel()

	rootMux := setupRouter(t)

	routes := []string{
		"/web/login",
		"/web/forget-password",
		"/web/chat",
		"/web/verify-email/test-token",
		"/web/sw.js",
	}

	for _, path := range routes {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		match := mux.RouteMatch{}

		if !rootMux.Match(req, &match) {
			t.Errorf("Expected GET %s to be registered", path)
		}
	}
}

func TestSetupHandlers_WebPagesRender(t *testing.T) {
	t.Parallel()

	rootMux := setupRouter(t)

	pages := []string{
		"/web/login",
		"/web/forget-password",
		"/web/chat",
	}

	for _, path := range pages {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		rr := httptest.NewRecorder()

		rootMux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for %s, got %d", http.StatusOK, path, rr.Code)
		}

		body := rr.Body.String()
		if body == "" {
			t.Errorf("Expected non-empty body for %s", path)
		}
	}
}

func TestSetupHandlers_DocsRoute(t *testing.T) {
	t.Parallel()

	rootMux := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/docs", http.NoBody)
	match := mux.RouteMatch{}

	if !rootMux.Match(req, &match) {
		t.Error("Expected GET /docs to be registered")
	}
}

func TestSetupHandlers_NotFoundHandler(t *testing.T) {
	t.Parallel()

	rootMux := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-page", http.NoBody)
	rr := httptest.NewRecorder()

	rootMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf(
			"Expected status %d for not found (home page), got %d",
			http.StatusOK,
			rr.Code,
		)
	}
}

func TestSetupHandlers_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	rootMux := setupRouter(t)

	// POST /metrics — зарегистрирован только для GET, gorilla/mux корректно возвращает 405.
	req := httptest.NewRequest(http.MethodPost, "/metrics", http.NoBody)
	rr := httptest.NewRecorder()

	rootMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"Expected status %d for method not allowed, got %d",
			http.StatusMethodNotAllowed,
			rr.Code,
		)
	}
}
