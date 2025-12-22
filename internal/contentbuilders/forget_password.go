package contentbuilders

import (
	"fmt"
	"strconv"

	"github.com/DKhorkov/libs/security"

	"github.com/DKhorkov/khodfeltschat/internal/domains"
)

type ForgetPasswordContentBuilder struct {
	forgetPasswordURLBase string
}

func NewForgetPasswordContentBuilder(forgetPasswordURLBase string) *ForgetPasswordContentBuilder {
	return &ForgetPasswordContentBuilder{
		forgetPasswordURLBase: forgetPasswordURLBase,
	}
}

func (b *ForgetPasswordContentBuilder) Subject() string {
	return "Восстановление пароля от аккаунта"
}

func (b *ForgetPasswordContentBuilder) Body(user domains.User) string {
	link := fmt.Sprintf(
		"%s/%s",
		b.forgetPasswordURLBase,
		security.RawEncode([]byte(strconv.FormatUint(user.ID, 10))),
	)

	template := `<p>Добрый день, %s!</p>
<p>На данный email было запрошено письмо для восстановления забытого пароля.</p>
<p>Пожалуйста, перейдите по <a href="%s">ссылке</a>, чтобы сменить пароль!</p>
<p>Если это были не Вы - проигнорируйте данное письмо!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`

	return fmt.Sprintf(
		template,
		user.Username,
		link,
	)
}
