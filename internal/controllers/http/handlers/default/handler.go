package default_handler

import (
	"html/template"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var homeTemplate = template.Must(
	template.ParseFiles("internal/controllers/http/handlers/web/templates/home.html"),
)

// swagger:route GET / web HomePage
//
// HomePage
//
// Serves the home page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the home page for unmatched routes.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

	if err := homeTemplate.Execute(w, nil); err != nil {
		webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
	}
}
