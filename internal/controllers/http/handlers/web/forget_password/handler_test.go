package forget_password_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/forget_password"
)

func TestMain(m *testing.M) {
	// Тесты запускаются из директории пакета, а шаблоны загружаются
	// относительно корня проекта. Переходим в корень.
	if err := os.Chdir("../../../../../.."); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestHandler_Success(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/web/forget-password", http.NoBody)
	rr := httptest.NewRecorder()

	forget_password.Handler().ServeHTTP(rr, req)

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

func TestHandler_WithEmailParam(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(
		http.MethodGet,
		"/web/forget-password?email=test@example.com",
		http.NoBody,
	)
	rr := httptest.NewRecorder()

	forget_password.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "test@example.com") {
		t.Error("Expected body to contain the email address")
	}
}

func TestHandler_WithoutEmailParam(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/web/forget-password", http.NoBody)
	rr := httptest.NewRecorder()

	forget_password.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Expected non-empty body")
	}
}

func TestHandler_ContainsNavbar(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/web/forget-password", http.NoBody)
	rr := httptest.NewRecorder()

	forget_password.Handler().ServeHTTP(rr, req)

	body := rr.Body.String()

	if !strings.Contains(body, "navbar") {
		t.Error("Expected body to contain navbar")
	}

	if !strings.Contains(body, "KFC Chat") {
		t.Error("Expected body to contain 'KFC Chat'")
	}
}

func TestHandler_Concurrent(t *testing.T) {
	t.Parallel()

	const concurrentCalls = 50

	done := make(chan bool, concurrentCalls)

	for i := range concurrentCalls {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/web/forget-password", http.NoBody)
			rr := httptest.NewRecorder()

			forget_password.Handler().ServeHTTP(rr, req)

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
