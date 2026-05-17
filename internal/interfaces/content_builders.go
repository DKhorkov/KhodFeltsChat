package interfaces

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
)

type ContentBuilders struct {
	VerifyEmail    VerifyEmailContentBuilder
	ForgetPassword ForgetPasswordContentBuilder
	NewMessage     NewMessageContentBuilder
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/verify_email_content_builder.go -package=mockcontentbuilders -exclude_interfaces=ForgetPasswordContentBuilder,NewMessageContentBuilder
type VerifyEmailContentBuilder interface {
	Subject() string
	Body(ctx context.Context, user domains.User) (string, error)
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/forget_password_content_builder.go -package=mockcontentbuilders -exclude_interfaces=VerifyEmailContentBuilder,NewMessageContentBuilder
type ForgetPasswordContentBuilder interface {
	Subject() string
	Body(ctx context.Context, user domains.User) (string, error)
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/new_message_content_builder.go -package=mockcontentbuilders -exclude_interfaces=VerifyEmailContentBuilder,ForgetPasswordContentBuilder
type NewMessageContentBuilder interface {
	Subject() string
	Body(ctx context.Context, message domains.Message, chat domains.Chat) (string, error)
}
