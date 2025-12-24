package handlers

import (
	"net/http"
)

// swagger:route GET / DefaultHandler OK
// Default Handler for everything that is not a match.
// Works with all HTTP methods
//
// responses:
//  303: SeeOther
//  500: InternalServerError

// DefaultHandler is for handling everything that is not a match.
func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, docsURL, http.StatusSeeOther)
}
