package verify_email

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/cache"
	"github.com/DKhorkov/libs/security"
	"github.com/google/uuid"
)

type ContentBuilder struct {
	baseURL       string
	cacheProvider cache.Provider
}

func New(baseURL string, cacheProvider cache.Provider) *ContentBuilder {
	return &ContentBuilder{
		baseURL:       baseURL,
		cacheProvider: cacheProvider,
	}
}

func (b *ContentBuilder) Subject() string {
	return "Подтверждение адреса электронной почты"
}

func (b *ContentBuilder) Body(ctx context.Context, user domains.User) (string, error) {
	salt := uuid.New().String()

	token := security.RawEncode(
		[]byte(salt + common.SaltSeparator + strconv.FormatUint(user.ID, 10)),
	)

	cacheKey := fmt.Sprintf("%s:%d", common.VerifyEmailTokenPrefix, user.ID)
	if err := b.cacheProvider.Set(
		ctx,
		cacheKey,
		token,
		common.TokenTTL,
	); err != nil {
		return "", fmt.Errorf("failed to set cache for %s key: %w", cacheKey, err)
	}

	link := fmt.Sprintf("%s/%s", b.baseURL, token)

	template := `<p>Добрый день, %s!</p>
<p>Пожалуйста, перейдите по <a href="%s">ссылке</a>, чтобы подтвердить адрес электронной почты!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`

	return fmt.Sprintf(
		template,
		user.Username,
		link,
	), nil
}
