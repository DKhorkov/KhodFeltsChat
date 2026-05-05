package web

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/login"
	"github.com/gorilla/mux"
)

const (
	LoginURL  = "/login"
	StaticURL = "/static/"
)

func SetupHandlers(webMux *mux.Router) {
	webMux.Handle(LoginURL, login.Handler()).Methods(http.MethodGet)

	// Статические файлы (CSS, JS):
	webMux.PathPrefix(StaticURL).Handler(
		http.StripPrefix("/web/static/",
			http.FileServer(http.Dir("internal/controllers/http/handlers/web/static")),
		),
	)
}
