package new_message

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
)

type ContentBuilder struct{}

func New() *ContentBuilder {
	return &ContentBuilder{}
}

func (*ContentBuilder) Subject() string {
	return "Новое сообщение в чате"
}

func (*ContentBuilder) Body(
	_ context.Context,
	message domains.Message,
	chat domains.Chat,
) (string, error) {
	chatName := "Личный чат"
	if chat.Title != nil {
		chatName = *chat.Title
	}

	template := `<p>Добрый день!</p>
<p>Вы получили новое сообщение от пользователя <b>%s</b> в чате <b>%s</b>:</p>
<blockquote>%s</blockquote>
<p>Перейдите в приложение, чтобы ответить.</p>
<p>С уважением,<br>
команда KFC Chat.</p>
`

	return fmt.Sprintf(template, message.Sender.Username, chatName, message.Text), nil
}
