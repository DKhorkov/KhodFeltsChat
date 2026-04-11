package not_allowed_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/not_allowed"
)

// TestHandler_Basic проверяет базовый случай.
func TestHandler_Basic(t *testing.T) {
	t.Parallel()

	// Подготовка тестовых данных
	testCases := []struct {
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			method:     "POST",
			path:       "/api/users",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `Method "POST" not allowed for URL "/api/users"!`,
		},
		{
			method:     "PUT",
			path:       "/api/users/123",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `Method "PUT" not allowed for URL "/api/users/123"!`,
		},
		{
			method:     "DELETE",
			path:       "/api/resource",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `Method "DELETE" not allowed for URL "/api/resource"!`,
		},
		{
			method:     "PATCH",
			path:       "/",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `Method "PATCH" not allowed for URL "/"!`,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.method, tc.path), func(t *testing.T) {
			t.Parallel()

			// Создаем запрос
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			rr := httptest.NewRecorder()

			// Вызываем обработчик
			not_allowed.Handler(rr, req)

			// Проверяем статус код
			if rr.Code != tc.wantStatus {
				t.Errorf("Status code = %v, want %v", rr.Code, tc.wantStatus)
			}

			// Проверяем тело ответа
			body, err := io.ReadAll(rr.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			gotBody := strings.TrimSpace(string(body))
			if gotBody != tc.wantBody {
				t.Errorf("Body = %q, want %q", gotBody, tc.wantBody)
			}
		})
	}
}

// TestHandler_AllMethods проверяет все HTTP методы.
func TestHandler_AllMethods(t *testing.T) {
	t.Parallel()

	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/test", http.NoBody)
			rr := httptest.NewRecorder()

			not_allowed.Handler(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("For method %s: status = %v, want %v",
					method, rr.Code, http.StatusMethodNotAllowed)
			}

			body, _ := io.ReadAll(rr.Body)

			expected := fmt.Sprintf(`Method %q not allowed for URL "/test"!`, method)

			if strings.TrimSpace(string(body)) != expected {
				t.Errorf("For method %s: body = %q, want %q",
					method, strings.TrimSpace(string(body)), expected)
			}
		})
	}
}

// TestHandler_QueryParams проверяет обработку URL с query параметрами.
func TestHandler_QueryParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		path     string
		expected string
	}{
		{"/test?param=value", `/test`}, // Query параметры должны игнорироваться в пути
		{"/api?filter=name&sort=asc", `/api`},
		{"/search?q=golang&page=2", `/search`},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPut, tc.path, http.NoBody)
			rr := httptest.NewRecorder()

			not_allowed.Handler(rr, req)

			body, _ := io.ReadAll(rr.Body)

			expected := fmt.Sprintf(`Method "PUT" not allowed for URL %q!`, tc.expected)

			if strings.TrimSpace(string(body)) != expected {
				t.Errorf("For path %s: body = %q, want %q",
					tc.path, strings.TrimSpace(string(body)), expected)
			}
		})
	}
}

// TestHandler_Concurrent проверяет конкурентный вызов.
func TestHandler_Concurrent(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	done := make(chan bool, goroutines)

	for i := range goroutines {
		go func(id int) {
			path := fmt.Sprintf("/api/%d", id)

			method := "POST"
			if id%2 == 0 {
				method = "PUT"
			}

			req := httptest.NewRequest(method, path, http.NoBody)
			rr := httptest.NewRecorder()

			not_allowed.Handler(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Goroutine %d: status = %v, want %v",
					id, rr.Code, http.StatusMethodNotAllowed)
			}

			expected := fmt.Sprintf(`Method %q not allowed for URL %q!`, method, path)

			body, _ := io.ReadAll(rr.Body)

			if strings.TrimSpace(string(body)) != expected {
				t.Errorf("Goroutine %d: body mismatch", id)
			}

			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for range goroutines {
		<-done
	}
}

// TestHandler_Headers проверяет заголовки ответа.
func TestHandler_Headers(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api", http.NoBody)
	rr := httptest.NewRecorder()

	not_allowed.Handler(rr, req)

	// Проверяем, что статус установлен корректно
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status not set correctly: got %d, want %d",
			rr.Code, http.StatusMethodNotAllowed)
	}
}

// TestHandler_BodyWithData проверяет обработку запросов с телом.
func TestHandler_BodyWithData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
	}{
		{"Empty body", ""},
		{"JSON body", `{"key": "value"}`},
		{"Form data", "username=admin&password=secret"},
		{"Binary data", string([]byte{0, 1, 2, 3, 255})},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/api", bytes.NewBufferString(tc.body))
			rr := httptest.NewRecorder()

			not_allowed.Handler(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: status = %v, want %v",
					tc.name, rr.Code, http.StatusMethodNotAllowed)
			}

			// Тело запроса не должно влиять на ответ
			body, _ := io.ReadAll(rr.Body)
			if !strings.Contains(string(body), `Method "POST" not allowed for URL "/api"!`) {
				t.Errorf("%s: unexpected response body: %s", tc.name, string(body))
			}
		})
	}
}

// TestHandler_TableDriven табличный тест с различными сценариями.
func TestHandler_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		path           string
		wantStatus     int
		wantBodyPrefix string
	}{
		{
			name:           "Simple POST",
			method:         "POST",
			path:           "/api",
			wantStatus:     http.StatusMethodNotAllowed,
			wantBodyPrefix: `Method "POST" not allowed for URL "/api"!`,
		},
		{
			name:           "Complex path",
			method:         "DELETE",
			path:           "/api/v1/users/123/profile",
			wantStatus:     http.StatusMethodNotAllowed,
			wantBodyPrefix: `Method "DELETE" not allowed for URL "/api/v1/users/123/profile"!`,
		},
		{
			name:           "Root with POST",
			method:         "POST",
			path:           "/",
			wantStatus:     http.StatusMethodNotAllowed,
			wantBodyPrefix: `Method "POST" not allowed for URL "/"!`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			rr := httptest.NewRecorder()

			not_allowed.Handler(rr, req)

			// Проверяем статус
			if rr.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v",
					tt.name, rr.Code, tt.wantStatus)
			}

			// Проверяем тело ответа
			body, _ := io.ReadAll(rr.Body)

			bodyStr := strings.TrimSpace(string(body))
			if !strings.HasPrefix(bodyStr, tt.wantBodyPrefix) {
				t.Errorf("%s: body = %q, should start with %q",
					tt.name, bodyStr, tt.wantBodyPrefix)
			}
		})
	}
}
