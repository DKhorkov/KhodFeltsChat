package common_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	handlerscommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

func TestMain(m *testing.M) {
	// Тесты запускаются из директории пакета, а шаблоны загружаются
	// относительно корня проекта. Переходим в корень.
	if err := os.Chdir("../../../../../.."); err != nil {
		panic("failed to chdir to project root: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestRenderError_StatusCode(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()

	webcommon.RenderError(rr, http.StatusNotFound, "page not found")

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestRenderError_ContentType(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()

	webcommon.RenderError(rr, http.StatusInternalServerError, "something went wrong")

	contentType := rr.Header().Get("Content-Type")
	if contentType != handlerscommon.TextHTMLContentType {
		t.Errorf(
			"Expected Content-Type '%s', got '%s'",
			handlerscommon.TextHTMLContentType,
			contentType,
		)
	}
}

func TestRenderError_ContainsMessage(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()

	webcommon.RenderError(rr, http.StatusBadRequest, "invalid request")

	body := rr.Body.String()
	if !strings.Contains(body, "invalid request") {
		t.Error("Expected body to contain error message")
	}
}

func TestRenderError_ContainsStatusCode(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()

	webcommon.RenderError(rr, http.StatusNotFound, "not found")

	body := rr.Body.String()
	if !strings.Contains(body, "404") {
		t.Error("Expected body to contain status code '404'")
	}
}

func TestRenderError_ContainsNavbar(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()

	webcommon.RenderError(rr, http.StatusInternalServerError, "error")

	body := rr.Body.String()
	if !strings.Contains(body, "navbar") {
		t.Error("Expected body to contain navbar")
	}
}

func TestRenderError_Concurrent(t *testing.T) {
	t.Parallel()

	const concurrentCalls = 50

	done := make(chan bool, concurrentCalls)

	for i := range concurrentCalls {
		go func(id int) {
			rr := httptest.NewRecorder()

			webcommon.RenderError(rr, http.StatusInternalServerError, "error")

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("Goroutine %d: expected status %d, got %d",
					id, http.StatusInternalServerError, rr.Code)
			}

			done <- true
		}(i)
	}

	for range concurrentCalls {
		<-done
	}
}
