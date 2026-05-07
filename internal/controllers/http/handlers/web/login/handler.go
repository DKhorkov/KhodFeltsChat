package login

import (
	"html/template"
	"net/http"
	"sync"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var (
	loginTemplate     *template.Template
	loginTemplateOnce sync.Once
)

func getLoginTemplate() *template.Template {
	loginTemplateOnce.Do(func() {
		loginTemplate = template.Must(
			template.ParseFiles(
				"internal/controllers/http/handlers/web/templates/login.html",
				"internal/controllers/http/handlers/web/templates/navbar.html",
			),
		)
	})

	return loginTemplate
}

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
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		if err := getLoginTemplate().Execute(w, nil); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
