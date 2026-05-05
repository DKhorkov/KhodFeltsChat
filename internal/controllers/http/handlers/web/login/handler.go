package login

import (
	"html/template"
	"net/http"
)

var loginTemplate = template.Must(
	template.ParseFiles("internal/controllers/http/handlers/web/templates/login.html"),
)

// Handler serves the login/register page.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := loginTemplate.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
