package common

import (
	"html/template"
	"net/http"

	handlerscommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
)

var errorTemplate = template.Must(
	template.ParseFiles(
		"internal/controllers/http/handlers/web/templates/error.html",
		"internal/controllers/http/handlers/web/templates/navbar.html",
	),
)

type errorData struct {
	Code    int
	Message string
}

// RenderError renders the error page with the given HTTP status code and message.
func RenderError(w http.ResponseWriter, code int, message string) {
	w.Header().Set(handlerscommon.ContentTypeHeaderName, handlerscommon.TextHTMLContentType)
	w.WriteHeader(code)

	if err := errorTemplate.Execute(w, errorData{Code: code, Message: message}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
