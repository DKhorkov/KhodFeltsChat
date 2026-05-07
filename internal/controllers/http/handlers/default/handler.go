package default_handler

import (
	"html/template"
	"net/http"
	"sync"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	webcommon "github.com/DKhorkov/kfc/internal/controllers/http/handlers/web/common"
)

var (
	homeTemplate     *template.Template
	homeTemplateOnce sync.Once
)

func getHomeTemplate() *template.Template {
	homeTemplateOnce.Do(func() {
		homeTemplate = template.Must(
			template.ParseFiles(
				"internal/controllers/http/handlers/web/templates/home.html",
				"internal/controllers/http/handlers/web/templates/navbar.html",
			),
		)
	})

	return homeTemplate
}

// swagger:route GET / web HomePage
//
// HomePage
//
// Serves the home page.
//
// Responses:
//	200: OK
//	500: InternalServerError

// Handler serves the home page for unmatched routes.
func Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(common.ContentTypeHeaderName, common.TextHTMLContentType)

	if err := getHomeTemplate().Execute(w, nil); err != nil {
		webcommon.RenderError(w, http.StatusInternalServerError, err.Error())
	}
}
