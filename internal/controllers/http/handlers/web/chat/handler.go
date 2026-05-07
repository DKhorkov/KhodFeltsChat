package chat

import (
	"html/template"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var chatTemplate = template.Must(
	template.ParseFiles(
		"internal/controllers/http/handlers/web/templates/chat.html",
		"internal/controllers/http/handlers/web/templates/navbar.html",
	),
)

// swagger:route GET /web/chat web ChatPage
//
// ChatPage
//
// Serves the chat page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the chat page.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		if err := chatTemplate.Execute(w, nil); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
