package interfaces

import (
	"github.com/DKhorkov/kfc/internal/domains"
)

type ContentBuilders struct {
	VerifyEmail    VerifyEmailContentBuilder
	ForgetPassword ForgetPasswordContentBuilder
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/verify_email_content_builder.go -package=mockcontentbuilders -exclude_interfaces=ForgetPasswordContentBuilder
type VerifyEmailContentBuilder interface {
	Subject() string
	Body(user domains.User) string
}

//go:generate mockgen -source=content_builders.go -destination=../../mocks/contentbuilders/forget_password_content_builder.go -package=mockcontentbuilders -exclude_interfaces=VerifyEmailContentBuilder
type ForgetPasswordContentBuilder interface {
	Subject() string
	Body(user domains.User) string
}
