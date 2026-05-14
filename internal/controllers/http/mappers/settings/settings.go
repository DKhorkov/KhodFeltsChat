package settings

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapSettings(settings domains.Settings) schemas.Settings {
	return schemas.Settings{
		Theme: int(settings.Theme),
	}
}
