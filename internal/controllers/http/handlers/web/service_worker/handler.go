package service_worker

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
)

const staticFilePath = "internal/controllers/http/handlers/web/static/sw.js"

// Handler serves the Service Worker script with proper headers.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.ApplicationJavaScriptContentType)
		w.Header().Set(common.ServiceWorkerAllowedHeaderName, "/")
		http.ServeFile(w, r, staticFilePath)
	}
}
