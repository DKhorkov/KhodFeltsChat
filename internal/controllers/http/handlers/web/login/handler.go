package login

import (
	"html/template"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var loginTemplate = template.Must(
	template.ParseFiles("internal/controllers/http/handlers/web/templates/login.html"),
)

// swagger:route GET /web/login web LoginPage
//
// LoginPage
//
// Serves the login and registration page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the login/register page.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		if err := loginTemplate.Execute(w, nil); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
