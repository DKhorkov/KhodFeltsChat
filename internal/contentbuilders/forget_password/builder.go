package forget_password

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
	cacheProvider cache.Provider
}

func New(cacheProvider cache.Provider) *ContentBuilder {
	return &ContentBuilder{
		cacheProvider: cacheProvider,
	}
}

func (b *ContentBuilder) Subject() string {
	return "Восстановление пароля от аккаунта"
}

func (b *ContentBuilder) Body(ctx context.Context, user domains.User) (string, error) {
	salt := uuid.New().String()

	token := security.RawEncode(
		[]byte(salt + common.SaltSeparator + strconv.FormatUint(user.ID, 10)),
	)

	cacheKey := fmt.Sprintf("%s:%d", common.ForgetPasswordTokenPrefix, user.ID)
	if err := b.cacheProvider.Set(
		ctx,
		cacheKey,
		token,
		common.TokenTTL,
	); err != nil {
		return "", fmt.Errorf("failed to set cache for %s key: %w", cacheKey, err)
	}

	template := `<p>Добрый день, %s!</p>
<p>На данный email было запрошено письмо для восстановления забытого пароля.</p>
<p>Пожалуйста, используйте токен <b>%s</b>, чтобы сменить пароль!</p>
<p>Если это были не Вы - проигнорируйте данное письмо!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`

	return fmt.Sprintf(
		template,
		user.Username,
		token,
	), nil
}
