package profile

import (
	"html/template"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var profileTemplate = template.Must(
	template.ParseFiles(
		"internal/controllers/http/handlers/web/templates/profile.html",
		"internal/controllers/http/handlers/web/templates/navbar.html",
	),
)

// swagger:route GET /web/profile web ProfilePage
//
// ProfilePage
//
// Serves the user profile page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the user profile page.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		if err := profileTemplate.Execute(w, nil); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
