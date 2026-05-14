package web_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Тесты запускаются из директории пакета, а шаблоны загружаются
	// относительно корня проекта. Переходим в корень.
	if err := os.Chdir("../../../../.."); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestSetupHandlers_RoutesRegistered(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"GET /login", http.MethodGet, "/login"},
		{"GET /forget-password", http.MethodGet, "/forget-password"},
		{"GET /chat", http.MethodGet, "/chat"},
		{"GET /verify-email/test-token", http.MethodGet, "/verify-email/test-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			match := mux.RouteMatch{}
			assert.True(t, webMux.Match(req, &match), "Expected %s %s to be registered", tt.method, tt.path)
		})
	}
}

func TestSetupHandlers_Pages(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"login page", "/login", http.StatusOK},
		{"chat page", "/chat", http.StatusOK},
		{"forget password page", "/forget-password", http.StatusOK},
		{"verify email page", "/verify-email/some-token", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			rr := httptest.NewRecorder()

			webMux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestSetupHandlers_PostNotAllowed(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	match := mux.RouteMatch{}

	assert.False(t, webMux.Match(req, &match), "Expected POST /login to not match any route")
}
