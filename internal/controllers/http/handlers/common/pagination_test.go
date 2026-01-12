package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
)

func TestGetPaginationFromRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		url         string
		method      string
		expected    *domains.Pagination
		expectedNil bool
	}{
		// Базовые случаи
		{
			name:   "Оба параметра: limit и offset",
			url:    "/test?limit=10&offset=20",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(10)),
				Offset: pointers.New(uint64(20)),
			},
		},
		{
			name:   "Только limit",
			url:    "/test?limit=50",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(50)),
				Offset: pointers.New(uint64(0)),
			},
		},
		{
			name:   "Только offset",
			url:    "/test?offset=100",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(0)),
				Offset: pointers.New(uint64(100)),
			},
		},
		{
			name:        "Нет параметров",
			url:         "/test",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "Пустые параметры",
			url:         "/test?limit=&offset=",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "Другие параметры, но не limit/offset",
			url:         "/test?page=1&sort=name",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},

		// Граничные значения
		{
			name:     "Limit = 0, Offset = 0",
			url:      "/test?limit=0&offset=0",
			method:   "GET",
			expected: nil,
		},
		{
			name:   "Limit = 1 (минимальное положительное)",
			url:    "/test?limit=1&offset=0",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(1)),
				Offset: pointers.New(uint64(0)),
			},
		},
		{
			name:   "Большие значения",
			url:    "/test?limit=999999&offset=123456789",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(999999)),
				Offset: pointers.New(uint64(123456789)),
			},
		},
		{
			name:   "Максимальные значения uint64",
			url:    "/test?limit=18446744073709551615&offset=18446744073709551614",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(18446744073709551615)),
				Offset: pointers.New(uint64(18446744073709551614)),
			},
		},
		{
			name:        "Нечисловые значения",
			url:         "/test?limit=abc&offset=xyz",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:   "Частично невалидные: валидный limit, невалидный offset",
			url:    "/test?limit=10&offset=invalid",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(10)),
				Offset: pointers.New(uint64(0)),
			},
		},
		{
			name:   "Частично невалидные: невалидный limit, валидный offset",
			url:    "/test?limit=invalid&offset=20",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(0)),
				Offset: pointers.New(uint64(20)),
			},
		},
		{
			name:        "Дробные числа",
			url:         "/test?limit=10.5&offset=20.7",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "Пробелы в значениях",
			url:         "/test?limit=%2010%20&offset=%2020%20",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:   "Значения с ведущими нулями",
			url:    "/test?limit=0010&offset=0020",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(10)),
				Offset: pointers.New(uint64(20)),
			},
		},
		{
			name:        "Hex значения",
			url:         "/test?limit=0xA&offset=0x14",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "Научная нотация",
			url:         "/test?limit=1e2&offset=2e3",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},

		// Разные HTTP методы
		{
			name:   "POST запрос с параметрами",
			url:    "/test?limit=10&offset=20",
			method: "POST",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(10)),
				Offset: pointers.New(uint64(20)),
			},
		},
		{
			name:   "PUT запрос с параметрами",
			url:    "/test?limit=5&offset=15",
			method: "PUT",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(5)),
				Offset: pointers.New(uint64(15)),
			},
		},
		{
			name:   "DELETE запрос с параметрами",
			url:    "/test?limit=25&offset=0",
			method: "DELETE",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(25)),
				Offset: pointers.New(uint64(0)),
			},
		},

		// Кейсы с кодировкой URL
		{
			name:        "Параметры с специальными символами",
			url:         "/test?limit=10%00&offset=20%00",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},
		{
			name:        "Параметры с + вместо пробела",
			url:         "/test?limit=+10+&offset=+20+",
			method:      "GET",
			expected:    nil,
			expectedNil: true,
		},

		// Множественные параметры (последний должен использоваться)
		{
			name:   "Множественные limit и offset",
			url:    "/test?limit=5&limit=10&offset=15&offset=20",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(5)),  // Первый limit
				Offset: pointers.New(uint64(15)), // Первый offset
			},
		},

		// Реальные сценарии использования
		{
			name:   "Пагинация первой страницы",
			url:    "/api/v1/users?limit=20&offset=0",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(20)),
				Offset: pointers.New(uint64(0)),
			},
		},
		{
			name:   "Пагинация второй страницы",
			url:    "/api/v1/users?limit=20&offset=20",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(20)),
				Offset: pointers.New(uint64(20)),
			},
		},
		{
			name:   "Только ограничение количества без offset",
			url:    "/api/v1/users?limit=50",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(50)),
				Offset: pointers.New(uint64(0)),
			},
		},
		{
			name:   "Пропуск первых N записей без limit",
			url:    "/api/v1/users?offset=100",
			method: "GET",
			expected: &domains.Pagination{
				Limit:  pointers.New(uint64(0)),
				Offset: pointers.New(uint64(100)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Создаем запрос
			req := httptest.NewRequest(tt.method, tt.url, http.NoBody)

			// Вызываем тестируемую функцию
			result := GetPaginationFromRequest(req)

			assert.Equal(t, tt.expected, result)
		})
	}
}
