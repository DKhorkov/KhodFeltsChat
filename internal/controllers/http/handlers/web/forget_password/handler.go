package forget_password

import (
	"html/template"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var forgetPasswordTemplate = template.Must(
	template.ParseFiles("internal/controllers/http/handlers/web/templates/forget_password.html"),
)

type templateData struct {
	Email string
}

// swagger:route GET /web/forget-password web ForgetPasswordPage
//
// ForgetPasswordPage
//
// Serves the forget password page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the forget password page.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		data := templateData{
			Email: r.URL.Query().Get("email"),
		}

		if err := forgetPasswordTemplate.Execute(w, data); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
