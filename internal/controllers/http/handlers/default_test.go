package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers"
)

// TestDefaultHandler_Redirect проверяет, что хендлер выполняет редирект.
func TestDefaultHandler_Redirect(t *testing.T) {
	t.Parallel()

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем хендлер
	handlers.DefaultHandler(rr, req)

	// Проверяем статус код
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status code %d, got %d", http.StatusSeeOther, rr.Code)
	}

	// Проверяем заголовок Location
	location := rr.Header().Get("Location")
	if location != handlers.DocsURL {
		t.Errorf("Expected Location header '%s', got '%s'", handlers.DocsURL, location)
	}
}

// TestDefaultHandler_Method проверяет, что хендлер работает с разными методами HTTP.
func TestDefaultHandler_Method(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		method     string
		wantStatus int
	}{
		{"GET", http.StatusSeeOther},
		{"POST", http.StatusSeeOther},
		{"PUT", http.StatusSeeOther},
		{"DELETE", http.StatusSeeOther},
		{"HEAD", http.StatusSeeOther},
		{"OPTIONS", http.StatusSeeOther},
		{"PATCH", http.StatusSeeOther},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, "/", http.NoBody)
			rr := httptest.NewRecorder()

			handlers.DefaultHandler(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("For method %s: expected status %d, got %d",
					tc.method, tc.wantStatus, rr.Code)
			}

			location := rr.Header().Get("Location")
			if location != handlers.DocsURL {
				t.Errorf("For method %s: expected Location '%s', got '%s'",
					tc.method, handlers.DocsURL, location)
			}
		})
	}
}

// TestDefaultHandler_EmptyRequest проверяет обработку пустого запроса.
func TestDefaultHandler_EmptyRequest(t *testing.T) {
	t.Parallel()

	// Создаем nil запрос (крайний случай)
	req, _ := http.NewRequest(http.MethodGet, "", http.NoBody)
	rr := httptest.NewRecorder()

	handlers.DefaultHandler(rr, req)

	// Проверяем, что редирект все равно происходит
	if rr.Code != http.StatusSeeOther {
		t.Errorf("Expected status code %d for empty request, got %d",
			http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != handlers.DocsURL {
		t.Errorf("Expected Location '%s' for empty request, got '%s'",
			handlers.DocsURL, location)
	}
}

// TestDefaultHandler_Concurrent проверяет конкурентный вызов хендлера.
func TestDefaultHandler_Concurrent(t *testing.T) {
	t.Parallel()

	const concurrentCalls = 100

	done := make(chan bool, concurrentCalls)

	for i := range concurrentCalls {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			rr := httptest.NewRecorder()

			handlers.DefaultHandler(rr, req)

			if rr.Code != http.StatusSeeOther {
				t.Errorf("Goroutine %d: expected status %d, got %d",
					id, http.StatusSeeOther, rr.Code)
			}

			location := rr.Header().Get("Location")
			if location != handlers.DocsURL {
				t.Errorf("Goroutine %d: expected Location '%s', got '%s'",
					id, handlers.DocsURL, location)
			}

			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for range concurrentCalls {
		<-done
	}
}

// TestDefaultHandler_Body проверяет, что тело запроса не влияет на редирект.
func TestDefaultHandler_Body(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
	}{
		{"Empty body", ""},
		{"Simple body", "test data"},
		{"JSON body", `{"key": "value"}`},
		{"Large body", string(make([]byte, 1024))}, // 1KB данных
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tc.body))
			rr := httptest.NewRecorder()

			handlers.DefaultHandler(rr, req)

			if rr.Code != http.StatusSeeOther {
				t.Errorf("%s: expected status %d, got %d",
					tc.name, http.StatusSeeOther, rr.Code)
			}

			location := rr.Header().Get("Location")
			if location != handlers.DocsURL {
				t.Errorf("%s: expected Location '%s', got '%s'",
					tc.name, handlers.DocsURL, location)
			}
		})
	}
}

// TestDefaultHandler_Headers проверяет сохранение заголовков в ответе.
func TestDefaultHandler_Headers(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	handlers.DefaultHandler(rr, req)

	// Проверяем обязательные заголовки
	if location := rr.Header().Get("Location"); location != handlers.DocsURL {
		t.Errorf("Location header incorrect: got %s, want %s", location, handlers.DocsURL)
	}

	// Проверяем, что Content-Length не установлен (для редиректа это нормально)
	contentLength := rr.Header().Get("Content-Length")
	if contentLength != "" && contentLength != "0" {
		t.Errorf("Content-Length should be empty or 0 for redirect, got %s", contentLength)
	}
}
