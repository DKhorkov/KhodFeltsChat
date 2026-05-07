package forget_password

import (
	"html/template"
	"net/http"
	"sync"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var (
	forgetPasswordTemplate     *template.Template
	forgetPasswordTemplateOnce sync.Once
)

func getForgetPasswordTemplate() *template.Template {
	forgetPasswordTemplateOnce.Do(func() {
		forgetPasswordTemplate = template.Must(
			template.ParseFiles(
				"internal/controllers/http/handlers/web/templates/forget_password.html",
				"internal/controllers/http/handlers/web/templates/navbar.html",
			),
		)
	})

	return forgetPasswordTemplate
}

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

		if err := getForgetPasswordTemplate().Execute(w, data); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
