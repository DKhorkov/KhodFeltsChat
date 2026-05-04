package common

import "time"

const (
	VerifyEmailRateLimitPrefix = "email_verification"
	VerifyEmailRateLimitCount  = 3
	VerifyEmailRateLimitTTL    = 3 * time.Minute

	ForgetPasswordRateLimitPrefix = "forget_password"
	ForgetPasswordRateLimitCount  = 3
	ForgetPasswordRateLimitTTL    = 3 * time.Minute

	VerifyEmailTokenPrefix    = "verify_email_token"
	ForgetPasswordTokenPrefix = "forget_password_token"
	TokenTTL                  = 15 * time.Minute

	InitCacheValue = 1
)
