package verify_email

import (
	"html/template"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

const (
	TokenRouteKey = "token"
)

var verifyEmailTemplate = template.Must(
	template.ParseFiles("internal/controllers/http/handlers/web/templates/verify_email.html"),
)

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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

		if err := verifyEmailTemplate.Execute(w, nil); err != nil {
			webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
		}
	})
}
