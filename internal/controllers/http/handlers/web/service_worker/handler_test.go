package service_worker_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/service_worker"
)

func TestMain(m *testing.M) {
	// Тесты запускаются из директории пакета, а файл sw.js загружается
	// относительно корня проекта. Переходим в корень.
	if err := os.Chdir("../../../../../../"); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestHandler_Success(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/sw.js", http.NoBody)
	rr := httptest.NewRecorder()

	service_worker.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandler_ContentType(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/sw.js", http.NoBody)
	rr := httptest.NewRecorder()

	service_worker.Handler().ServeHTTP(rr, req)

	contentType := rr.Header().Get(common.ContentTypeHeaderName)
	if contentType != common.ApplicationJavaScriptContentType {
		t.Errorf(
			"Expected Content-Type '%s', got '%s'",
			common.ApplicationJavaScriptContentType,
			contentType,
		)
	}
}

func TestHandler_ServiceWorkerAllowedHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/sw.js", http.NoBody)
	rr := httptest.NewRecorder()

	service_worker.Handler().ServeHTTP(rr, req)

	swAllowed := rr.Header().Get(common.ServiceWorkerAllowedHeaderName)
	if swAllowed != "/" {
		t.Errorf("Expected Service-Worker-Allowed '/', got '%s'", swAllowed)
	}
}

func TestHandler_NonEmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/sw.js", http.NoBody)
	rr := httptest.NewRecorder()

	service_worker.Handler().ServeHTTP(rr, req)

	body := rr.Body.String()
	if body == "" {
		t.Error("Expected non-empty body")
	}
}

func TestHandler_BodyContainsServiceWorkerCode(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/sw.js", http.NoBody)
	rr := httptest.NewRecorder()

	service_worker.Handler().ServeHTTP(rr, req)

	body := rr.Body.String()

	if !contains(body, "self.addEventListener") {
		t.Error("Expected body to contain service worker event listener")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
