package download

import (
	"errors"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/gorilla/mux"
)

const FileRouteKey = "file"

// swagger:route GET /api/files/download/{file} files DownloadFile
//
// DownloadFile
//
// Downloads a file by name. Public endpoint (no auth required).
//
// Responses:
//	200: OK
//	400: BadRequest
//	404: NotFound
//	500: InternalServerError

// Handler serves files from storage.
func Handler(fileStorageUseCases interfaces.FileStorageUseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName := mux.Vars(r)[FileRouteKey]
		if fileName == "" {
			http.Error(w, "file name is required", http.StatusBadRequest)

			return
		}

		data, err := fileStorageUseCases.Download(r.Context(), fileName)

		switch {
		case errors.Is(err, customerrors.ErrFileNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)

			return
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set(common.ContentTypeHeaderName, common.ImageJPEGContentType)
		w.Header().Set(common.ContentTypeOptionsHeaderName, common.NoSniffContentTypeOption)
		w.WriteHeader(http.StatusOK)

		//nolint:gosec // data is served with image/jpeg Content-Type and nosniff header.
		if _, err = w.Write(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}
}
