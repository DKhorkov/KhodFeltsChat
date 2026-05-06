package web

import (
	"fmt"
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/forget_password"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/profile"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/verify_email"
	"github.com/gorilla/mux"
)

const (
	WebURL            = "/web"
	LoginURL          = "/login"
	ForgetPasswordURL = "/forget-password"
	VerifyEmailURL    = "/verify-email/{%s}"
	ProfileURL        = "/profile"
	StaticURL         = "/static/"

	StaticFilePath = "internal/controllers/http/handlers" + WebURL + StaticURL
)

func SetupHandlers(webMux *mux.Router) {
	getMux := webMux.Methods(http.MethodGet).Subrouter()
	getMux.Handle(LoginURL, login.Handler())
	getMux.Handle(ForgetPasswordURL, forget_password.Handler())
	getMux.Handle(ProfileURL, profile.Handler())
	getMux.Handle(
		fmt.Sprintf(VerifyEmailURL, verify_email.TokenRouteKey),
		verify_email.Handler(),
	)

	// Статические файлы (CSS, JS):
	webMux.PathPrefix(StaticURL).Handler(
		http.StripPrefix(WebURL+StaticURL,
			http.FileServer(http.Dir(StaticFilePath)),
		),
	)
}
