package web_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web"
	"github.com/gorilla/mux"
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

	routes := []struct {
		path   string
		method string
	}{
		{path: "/login", method: http.MethodGet},
		{path: "/forget-password", method: http.MethodGet},
		{path: "/chat", method: http.MethodGet},
		{path: "/verify-email/test-token", method: http.MethodGet},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, http.NoBody)
		match := mux.RouteMatch{}

		if !webMux.Match(req, &match) {
			t.Errorf("Expected route %s %s to be registered", route.method, route.path)
		}
	}
}

func TestSetupHandlers_LoginPage(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	req := httptest.NewRequest(http.MethodGet, "/login", http.NoBody)
	rr := httptest.NewRecorder()

	webMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for /login, got %d", http.StatusOK, rr.Code)
	}
}

func TestSetupHandlers_ChatPage(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	req := httptest.NewRequest(http.MethodGet, "/chat", http.NoBody)
	rr := httptest.NewRecorder()

	webMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for /chat, got %d", http.StatusOK, rr.Code)
	}
}

func TestSetupHandlers_ForgetPasswordPage(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	req := httptest.NewRequest(http.MethodGet, "/forget-password", http.NoBody)
	rr := httptest.NewRecorder()

	webMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for /forget-password, got %d", http.StatusOK, rr.Code)
	}
}

func TestSetupHandlers_VerifyEmailPage(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	req := httptest.NewRequest(http.MethodGet, "/verify-email/some-token", http.NoBody)
	rr := httptest.NewRecorder()

	webMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for /verify-email/{token}, got %d", http.StatusOK, rr.Code)
	}
}

func TestSetupHandlers_PostNotAllowed(t *testing.T) {
	t.Parallel()

	webMux := mux.NewRouter()
	web.SetupHandlers(webMux)

	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	match := mux.RouteMatch{}

	if webMux.Match(req, &match) {
		t.Error("Expected POST /login to not match any route")
	}
}
