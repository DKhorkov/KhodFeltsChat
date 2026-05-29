package download_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/files/download"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fileName       string
		setupMock      func(m *mockusecases.MockFileStorageUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:     "successful download",
			fileName: "uuid.jpg",
			setupMock: func(m *mockusecases.MockFileStorageUseCases) {
				m.EXPECT().
					Download(gomock.Any(), "uuid.jpg").
					Return([]byte("image data"), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "image/jpeg", rr.Header().Get("Content-Type"))
				assert.Equal(t, "image data", rr.Body.String())
			},
		},
		{
			name:     "file not found",
			fileName: "nonexistent.jpg",
			setupMock: func(m *mockusecases.MockFileStorageUseCases) {
				m.EXPECT().
					Download(gomock.Any(), "nonexistent.jpg").
					Return(nil, customerrors.ErrFileNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "internal server error",
			fileName: "uuid.jpg",
			setupMock: func(m *mockusecases.MockFileStorageUseCases) {
				m.EXPECT().
					Download(gomock.Any(), "uuid.jpg").
					Return(nil, errors.New("disk error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockUC := mockusecases.NewMockFileStorageUseCases(ctrl)

			tt.setupMock(mockUC)

			handler := download.Handler(mockUC)

			req := httptest.NewRequest(http.MethodGet, "/api/files/download/"+tt.fileName, nil)
			req = mux.SetURLVars(req, map[string]string{
				download.FileRouteKey: tt.fileName,
			})

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}
