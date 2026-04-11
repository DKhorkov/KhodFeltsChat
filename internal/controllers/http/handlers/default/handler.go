package default_handler

import (
	"net/http"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/docs"
)

// swagger:route GET / DefaultHandler OK
// Default Handler for everything that is not a match.
// Works with all HTTP methods
//
// responses:
//  303: SeeOther
//  500: InternalServerError

// Handler is for handling everything that is not a match.
func Handler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, docs.URL, http.StatusSeeOther)
}
