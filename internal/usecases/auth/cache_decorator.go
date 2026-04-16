package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cache"
	"github.com/DKhorkov/libs/logging"
)

const (
	verifyEmailPrefix    = "email_verification"
	verifyEmailLimit     = 3
	verifyEmailTTL       = time.Minute
	forgetPasswordPrefix = "forget_password"
	forgetPasswordLimit  = 3
	forgetPasswordTTL    = time.Minute

	initCacheValue = 1
)

type CacheDecorator struct {
	cacheProvider cache.Provider
	logger        logging.Logger
	base          interfaces.AuthUseCases
}

func NewCacheDecorator(
	cacheProvider cache.Provider,
	logger logging.Logger,
	base interfaces.AuthUseCases,
) *CacheDecorator {
	return &CacheDecorator{
		cacheProvider: cacheProvider,
		base:          base,
		logger:        logger,
	}
}

func (d *CacheDecorator) RegisterUser(
	ctx context.Context,
	dto domains.RegisterDTO,
) (*domains.User, error) {
	return d.base.RegisterUser(ctx, dto)
}

func (d *CacheDecorator) LoginUser(
	ctx context.Context,
	dto domains.LoginDTO,
) (*domains.TokensDTO, error) {
	return d.base.LoginUser(ctx, dto)
}

func (d *CacheDecorator) RefreshTokens(
	ctx context.Context,
	refreshToken string,
) (*domains.TokensDTO, error) {
	return d.base.RefreshTokens(ctx, refreshToken)
}

func (d *CacheDecorator) LogoutUser(ctx context.Context, userID uint64) error {
	return d.base.LogoutUser(ctx, userID)
}

func (d *CacheDecorator) VerifyEmail(ctx context.Context, verifyEmailToken string) error {
	return d.base.VerifyEmail(ctx, verifyEmailToken)
}

func (d *CacheDecorator) ForgetPassword(
	ctx context.Context,
	forgetPasswordToken, newPassword string,
) error {
	return d.base.ForgetPassword(ctx, forgetPasswordToken, newPassword)
}

func (d *CacheDecorator) ChangePassword(ctx context.Context, dto domains.ChangePasswordDTO) error {
	return d.base.ChangePassword(ctx, dto)
}

func (d *CacheDecorator) SendVerifyEmailMessage(ctx context.Context, email string) error {
	if _, err := d.cacheProvider.Ping(ctx); err != nil {
		return d.base.SendVerifyEmailMessage(ctx, email)
	}

	cacheKey := fmt.Sprintf("%s:%s", verifyEmailPrefix, email)

	strCounter, err := d.cacheProvider.Get(ctx, cacheKey)
	if err != nil {
		logging.LogErrorContext(
			ctx,
			d.logger,
			fmt.Sprintf("Failed to get cache for %s key", cacheKey),
			err,
		)
	}

	counter, err := strconv.ParseInt(strCounter, 10, 64)
	if err != nil && strCounter != "" {
		logging.LogErrorContext(
			ctx,
			d.logger,
			fmt.Sprintf("Invalid value=%s for %s cache key", strCounter, cacheKey),
			err,
		)
	}

	if counter >= verifyEmailLimit {
		return fmt.Errorf(
			"%w: Too many tries to send message. Limit per minute is %d",
			errors.ErrLimitExceeded,
			verifyEmailLimit,
		)
	}

	if err = d.base.SendVerifyEmailMessage(ctx, email); err != nil {
		return err
	}

	if counter == 0 {
		if err = d.cacheProvider.Set(ctx, cacheKey, initCacheValue, verifyEmailTTL); err != nil {
			logging.LogErrorContext(
				ctx,
				d.logger,
				fmt.Sprintf("Failed to set cache for %s key", cacheKey),
				err,
			)
		}
	} else {
		if _, err = d.cacheProvider.Incr(ctx, cacheKey); err != nil {
			logging.LogErrorContext(
				ctx,
				d.logger,
				fmt.Sprintf("Failed to increment cache for %s key", cacheKey),
				err,
			)
		}
	}

	return nil
}

func (d *CacheDecorator) SendForgetPasswordMessage(ctx context.Context, email string) error {
	if _, err := d.cacheProvider.Ping(ctx); err != nil {
		return d.base.SendForgetPasswordMessage(ctx, email)
	}

	cacheKey := fmt.Sprintf("%s:%s", forgetPasswordPrefix, email)

	strCounter, err := d.cacheProvider.Get(ctx, cacheKey)
	if err != nil {
		logging.LogErrorContext(
			ctx,
			d.logger,
			fmt.Sprintf("Failed to get cache for %s key", cacheKey),
			err,
		)
	}

	counter, err := strconv.ParseInt(strCounter, 10, 64)
	if err != nil && strCounter != "" {
		logging.LogErrorContext(
			ctx,
			d.logger,
			fmt.Sprintf("Invalid value=%s for %s cache key", strCounter, cacheKey),
			err,
		)
	}

	if counter >= forgetPasswordLimit {
		return fmt.Errorf(
			"%w: Too many tries to send message. Limit per minute is %d",
			errors.ErrLimitExceeded,
			forgetPasswordLimit,
		)
	}

	if err = d.base.SendForgetPasswordMessage(ctx, email); err != nil {
		return err
	}

	if counter == 0 {
		if err = d.cacheProvider.Set(ctx, cacheKey, initCacheValue, forgetPasswordTTL); err != nil {
			logging.LogErrorContext(
				ctx,
				d.logger,
				fmt.Sprintf("Failed to set cache for %s key", cacheKey),
				err,
			)
		}
	} else {
		if _, err = d.cacheProvider.Incr(ctx, cacheKey); err != nil {
			logging.LogErrorContext(
				ctx,
				d.logger,
				fmt.Sprintf("Failed to increment cache for %s key", cacheKey),
				err,
			)
		}
	}

	return nil
}
