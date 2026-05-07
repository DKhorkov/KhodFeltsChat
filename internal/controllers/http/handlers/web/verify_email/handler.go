package verify_email

import (
	"html/template"
	"net/http"
	"sync"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

const (
	TokenRouteKey = "token"
)

var (
	verifyEmailTemplate     *template.Template
	verifyEmailTemplateOnce sync.Once
)

func getVerifyEmailTemplate() *template.Template {
	verifyEmailTemplateOnce.Do(func() {
		verifyEmailTemplate = template.Must(
			template.ParseFiles(
				"internal/controllers/http/handlers/web/templates/verify_email.html",
			),
		)
	})

	return verifyEmailTemplate
}

// swagger:route GET /web/verify-email/{token} web VerifyEmailPage
//
// VerifyEmailPage
//
// Serves the email verification page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the email verification page.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		if err := getVerifyEmailTemplate().Execute(w, nil); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
