package auth

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/cache"
	"github.com/DKhorkov/libs/logging"
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

func (d *CacheDecorator) LogoutUser(ctx context.Context, refreshToken string) error {
	return d.base.LogoutUser(ctx, refreshToken)
}

func (d *CacheDecorator) LogoutUserFromAllSessions(ctx context.Context, userID uint64) error {
	return d.base.LogoutUserFromAllSessions(ctx, userID)
}

func (d *CacheDecorator) VerifyEmail(ctx context.Context, verifyEmailCode uint64) error {
	cacheKey, userID, err := d.resolveCode(ctx, common.VerifyEmailTokenPrefix, verifyEmailCode)
	if err != nil {
		return err
	}

	if err = d.base.VerifyEmail(ctx, userID); err != nil {
		return err
	}

	return d.cacheProvider.Del(ctx, cacheKey)
}

func (d *CacheDecorator) ForgetPassword(
	ctx context.Context,
	forgetPasswordCode uint64,
	newPassword string,
) error {
	cacheKey, userID, err := d.resolveCode(
		ctx,
		common.ForgetPasswordTokenPrefix,
		forgetPasswordCode,
	)
	if err != nil {
		return err
	}

	if err = d.base.ForgetPassword(ctx, userID, newPassword); err != nil {
		return err
	}

	return d.cacheProvider.Del(ctx, cacheKey)
}

func (d *CacheDecorator) ChangePassword(ctx context.Context, dto domains.ChangePasswordDTO) error {
	return d.base.ChangePassword(ctx, dto)
}

func (d *CacheDecorator) SendVerifyEmailMessage(ctx context.Context, email string) error {
	if _, err := d.cacheProvider.Ping(ctx); err != nil {
		return d.base.SendVerifyEmailMessage(ctx, email)
	}

	cacheKey := fmt.Sprintf("%s:%s", common.VerifyEmailRateLimitPrefix, email)

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

	if counter >= common.VerifyEmailRateLimitCount {
		return fmt.Errorf(
			"%w: Too many tries to send message. Limit per minute is %d",
			errors.ErrLimitExceeded,
			common.VerifyEmailRateLimitCount,
		)
	}

	if err = d.base.SendVerifyEmailMessage(ctx, email); err != nil {
		return err
	}

	if counter == 0 {
		if err = d.cacheProvider.Set(
			ctx,
			cacheKey,
			common.InitCacheValue,
			common.VerifyEmailRateLimitTTL,
		); err != nil {
			logging.LogErrorContext(
				ctx,
				d.logger,
				fmt.Sprintf("Failed to set cache for %s key", cacheKey),
				err,
			)
		}

		return nil
	}

	if _, err = d.cacheProvider.Incr(ctx, cacheKey); err != nil {
		logging.LogErrorContext(
			ctx,
			d.logger,
			fmt.Sprintf("Failed to increment cache for %s key", cacheKey),
			err,
		)
	}

	return nil
}

func (d *CacheDecorator) SendForgetPasswordMessage(ctx context.Context, email string) error {
	if _, err := d.cacheProvider.Ping(ctx); err != nil {
		return d.base.SendForgetPasswordMessage(ctx, email)
	}

	cacheKey := fmt.Sprintf("%s:%s", common.ForgetPasswordRateLimitPrefix, email)

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

	if counter >= common.ForgetPasswordRateLimitCount {
		return fmt.Errorf(
			"%w: Too many tries to send message. Limit per minute is %d",
			errors.ErrLimitExceeded,
			common.ForgetPasswordRateLimitCount,
		)
	}

	if err = d.base.SendForgetPasswordMessage(ctx, email); err != nil {
		return err
	}

	if counter == 0 {
		if err = d.cacheProvider.Set(
			ctx,
			cacheKey,
			common.InitCacheValue,
			common.ForgetPasswordRateLimitTTL,
		); err != nil {
			logging.LogErrorContext(
				ctx,
				d.logger,
				fmt.Sprintf("Failed to set cache for %s key", cacheKey),
				err,
			)
		}

		return nil
	}

	if _, err = d.cacheProvider.Incr(ctx, cacheKey); err != nil {
		logging.LogErrorContext(
			ctx,
			d.logger,
			fmt.Sprintf("Failed to increment cache for %s key", cacheKey),
			err,
		)
	}

	return nil
}

func (d *CacheDecorator) resolveCode(
	ctx context.Context,
	prefix string,
	code uint64,
) (string, uint64, error) {
	cacheKey := fmt.Sprintf("%s:%d", prefix, code)

	rawUserID, err := d.cacheProvider.Get(ctx, cacheKey)
	if err != nil || rawUserID == "" {
		return "", 0, errors.ErrTokenExpired
	}

	userID, err := strconv.ParseUint(rawUserID, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("%w: invalid %s", errors.ErrInvalidJWT, prefix)
	}

	return cacheKey, userID, nil
}
