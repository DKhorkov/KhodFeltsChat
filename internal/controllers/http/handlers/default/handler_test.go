package default_handler_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	default_handler "github.com/DKhorkov/kfc/internal/controllers/http/handlers/default"
)

func TestMain(m *testing.M) {
	// Тесты запускаются из директории пакета, а шаблоны загружаются
	// относительно корня проекта. Переходим в корень.
	if err := os.Chdir("../../../../.."); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestHandler_Success(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	default_handler.Handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != common.TextHTMLContentType {
		t.Errorf("Expected Content-Type '%s', got '%s'", common.TextHTMLContentType, contentType)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Expected non-empty body")
	}
}

func TestHandler_ContainsNavbar(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	default_handler.Handler(rr, req)

	body := rr.Body.String()
	if !contains(body, "navbar") {
		t.Error("Expected body to contain navbar")
	}

	if !contains(body, "KFC Chat") {
		t.Error("Expected body to contain 'KFC Chat'")
	}
}

func TestHandler_Concurrent(t *testing.T) {
	t.Parallel()

	const concurrentCalls = 50

	done := make(chan bool, concurrentCalls)

	for i := range concurrentCalls {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			rr := httptest.NewRecorder()

			default_handler.Handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Goroutine %d: expected status %d, got %d",
					id, http.StatusOK, rr.Code)
			}

			done <- true
		}(i)
	}

	for range concurrentCalls {
		<-done
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
