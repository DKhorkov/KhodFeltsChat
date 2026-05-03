package forget_password

import (
	"fmt"
	"strconv"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/security"
	"github.com/google/uuid"
)

type ContentBuilder struct{}

func New() *ContentBuilder {
	return &ContentBuilder{}
}

func (b *ContentBuilder) Subject() string {
	return "Восстановление пароля от аккаунта"
}

func (b *ContentBuilder) Body(user domains.User) string {
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
		security.RawEncode(
			[]byte(uuid.New().String()+common.SaltSeparator+strconv.FormatUint(user.ID, 10)),
		),
	)
}
