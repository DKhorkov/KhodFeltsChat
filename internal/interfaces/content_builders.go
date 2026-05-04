package interfaces

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
)

type ContentBuilders struct {
	VerifyEmail    VerifyEmailContentBuilder
	ForgetPassword ForgetPasswordContentBuilder
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/verify_email_content_builder.go -package=mockcontentbuilders -exclude_interfaces=ForgetPasswordContentBuilder
type VerifyEmailContentBuilder interface {
	Subject() string
	Body(ctx context.Context, user domains.User) (string, error)
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/forget_password_content_builder.go -package=mockcontentbuilders -exclude_interfaces=VerifyEmailContentBuilder
type ForgetPasswordContentBuilder interface {
	Subject() string
	Body(ctx context.Context, user domains.User) (string, error)
}
