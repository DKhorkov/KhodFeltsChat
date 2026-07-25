package forget_password

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/cache"
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
	userIDStr := strconv.FormatUint(user.ID, 10)

	var code uint64

	for range common.OTPGenerateAttempts {
		generated, err := common.GenerateOTP()
		if err != nil {
			return "", fmt.Errorf("failed to generate OTP: %w", err)
		}

		cacheKey := fmt.Sprintf("%s:%d", common.ForgetPasswordTokenPrefix, generated)

		if err = b.cacheProvider.SetNX(ctx, cacheKey, userIDStr, common.TokenTTL); err != nil {
			return "", fmt.Errorf("failed to set cache for %s key: %w", cacheKey, err)
		}

		stored, err := b.cacheProvider.Get(ctx, cacheKey)
		if err != nil {
			return "", fmt.Errorf("failed to verify cache for %s key: %w", cacheKey, err)
		}

		if stored == userIDStr {
			code = generated

			break
		}
	}

	if code == 0 {
		return "", common.ErrOTPCollision
	}

	template := `<p>Добрый день, %s!</p>
<p>На данный email было запрошено письмо для восстановления забытого пароля.</p>
<p>Пожалуйста, используйте код <b>%d</b>, чтобы сменить пароль!</p>
<p>Если это были не Вы - проигнорируйте данное письмо!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`

	return fmt.Sprintf(
		template,
		user.Username,
		code,
	), nil
}
