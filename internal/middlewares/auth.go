package middlewares

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/libs/contextlib"
	"github.com/DKhorkov/libs/security"
)

type IgnoreURL struct {
	Methods []string       `json:"methods"`
	Path    *regexp.Regexp `json:"path"`
}

func AuthMiddleware(
	securityConfig security.Config,
	ignoreURLs ...IgnoreURL,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если URL с вызванным методом должен игнорироваться - просто вызываем следующий хэндлер:
			for _, ignoreURL := range ignoreURLs {
				if ignoreURL.Path.MatchString(r.URL.Path) &&
					slices.Contains(ignoreURL.Methods, r.Method) {
					next.ServeHTTP(w, r)

					return
				}
			}

			accessTokenCookie, err := r.Cookie(auth.AccessTokenCookieName)
			if err != nil {
				http.Error(
					w,
					auth.AccessTokenCookieName+" cookie not provided",
					http.StatusUnauthorized,
				)

				return
			}

			accessTokenPayload, err := security.ParseJWT(
				accessTokenCookie.Value,
				securityConfig.JWT.SecretKey,
			)
			if err != nil {
				http.Error(
					w,
					fmt.Errorf("%w: %w", customerrors.ErrInvalidJWT, err).Error(),
					http.StatusUnauthorized,
				)

				return
			}

			floatUserID, ok := accessTokenPayload.(float64)
			if !ok {
				http.Error(
					w,
					fmt.Errorf("%w: failed to parse userID", customerrors.ErrInvalidJWT).Error(),
					http.StatusUnauthorized,
				)

				return
			}

			userID := uint64(floatUserID)

			ctx := contextlib.WithValue(
				r.Context(),
				auth.UserIDContextKey,
				userID,
			) // Устанавливаем значения для испоьзования в хэндлерах

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
