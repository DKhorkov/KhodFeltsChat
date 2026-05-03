package verify_email

import (
	"fmt"
	"strconv"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/security"
	"github.com/google/uuid"
)

type ContentBuilder struct {
	baseURL string
}

func New(baseURL string) *ContentBuilder {
	return &ContentBuilder{
		baseURL: baseURL,
	}
}

func (b *ContentBuilder) Subject() string {
	return "Подтверждение адреса электронной почты"
}

func (b *ContentBuilder) Body(user domains.User) string {
	link := fmt.Sprintf(
		"%s/%s",
		b.baseURL,
		security.RawEncode(
			[]byte(uuid.New().String()+common.SaltSeparator+strconv.FormatUint(user.ID, 10)),
		),
	)

	template := `<p>Добрый день, %s!</p>
<p>Пожалуйста, перейдите по <a href="%s">ссылке</a>, чтобы подтвердить адрес электронной почты!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`

	return fmt.Sprintf(
		template,
		user.Username,
		link,
	)
}
