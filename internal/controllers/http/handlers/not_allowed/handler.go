package not_allowed

import (
	"fmt"
	"net/http"
)

// swagger:route GET /*
// Default Handler for endpoints used with incorrect HTTP request method
//
// responses:
//	404: ErrorMessage
//	500: InternalServerError

// Handler is executed when the HTTP method is incorrect.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)

	_, err := fmt.Fprintf(w, "Method \"%s\" not allowed for URL \"%s\"!\n", r.Method, r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}
