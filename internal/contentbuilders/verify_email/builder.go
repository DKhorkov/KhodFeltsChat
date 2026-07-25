package verify_email

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/cache"
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
	userIDStr := strconv.FormatUint(user.ID, 10)

	var code uint64

	for range common.OTPGenerateAttempts {
		generated, err := common.GenerateOTP()
		if err != nil {
			return "", fmt.Errorf("failed to generate OTP: %w", err)
		}

		cacheKey := fmt.Sprintf("%s:%d", common.VerifyEmailTokenPrefix, generated)

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

	link := fmt.Sprintf("%s/%d", b.baseURL, code)

	template := `<p>Добрый день, %s!</p>
<p>Пожалуйста, перейдите по <a href="%s">ссылке</a>, чтобы подтвердить адрес электронной почты!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`

	return fmt.Sprintf(
		template,
		user.Username,
		link,
		code,
	), nil
}
