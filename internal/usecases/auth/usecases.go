package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/validation"
	"github.com/golang-jwt/jwt/v5"
)

type UseCases struct {
	authService      interfaces.AuthService
	usersService     interfaces.UsersService
	securityConfig   security.Config
	validationConfig config.ValidationConfig
}

func New(
	authService interfaces.AuthService,
	usersService interfaces.UsersService,
	securityConfig security.Config,
	validationConfig config.ValidationConfig,
) *UseCases {
	return &UseCases{
		authService:      authService,
		usersService:     usersService,
		securityConfig:   securityConfig,
		validationConfig: validationConfig,
	}
}

func (u *UseCases) RegisterUser(
	ctx context.Context,
	dto domains.RegisterDTO,
) (*domains.User, error) {
	if !validation.ValidateValueByRule(dto.Email, u.validationConfig.EmailRegExp) {
		return nil, fmt.Errorf("%w: invalid email address", customerrors.ErrValidationFailed)
	}

	if !validation.ValidateValueByRules(dto.Password, u.validationConfig.PasswordRegExps) {
		return nil, fmt.Errorf("%w: invalid password", customerrors.ErrValidationFailed)
	}

	if !validation.ValidateValueByRules(dto.Username, u.validationConfig.UsernameRegExps) {
		return nil, fmt.Errorf("%w: invalid username", customerrors.ErrValidationFailed)
	}

	hashedPassword, err := security.Hash(dto.Password, u.securityConfig.HashCost)
	if err != nil {
		return nil, err
	}

	dto.Password = hashedPassword

	return u.authService.RegisterUser(ctx, dto)
}

func (u *UseCases) LoginUser(
	ctx context.Context,
	dto domains.LoginDTO,
) (*domains.TokensDTO, error) {
	if dto.Login == "" {
		return nil, fmt.Errorf("%w: login is required", customerrors.ErrValidationFailed)
	}

	if dto.Password == "" {
		return nil, fmt.Errorf("%w: password is required", customerrors.ErrValidationFailed)
	}

	user, _ := u.usersService.GetUserByEmail(ctx, dto.Login)

	// Fallback логина по имени пользователя
	if user == nil {
		user, _ = u.usersService.GetUserByUsername(ctx, dto.Login)
	}

	// Если пользователь все еще не найден - значит такого пользователя не существует
	if user == nil {
		return nil, customerrors.ErrUserNotFound
	}

	if !user.EmailConfirmed {
		return nil, customerrors.ErrEmailNotConfirmed
	}

	if !security.ValidateHash(dto.Password, user.Password) {
		return nil, customerrors.ErrWrongPassword
	}

	dbRefreshToken, err := u.authService.GetRefreshTokenByUserID(ctx, user.ID)
	if err == nil {
		if err = u.authService.ExpireRefreshToken(ctx, dbRefreshToken.ID); err != nil {
			return nil, err
		}
	}

	// Create tokens:
	accessToken, err := security.GenerateJWT(
		user.ID,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.AccessTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := security.GenerateJWT(
		accessToken,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.RefreshTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	// Save token to Database:
	if _, err = u.authService.CreateRefreshToken(
		ctx,
		user.ID,
		refreshToken,
		u.securityConfig.JWT.RefreshTokenTTL,
	); err != nil {
		return nil, err
	}

	// Encoding refresh token for secure usage via internet:
	encodedRefreshToken := security.RawEncode([]byte(refreshToken))

	return &domains.TokensDTO{
		AccessToken:  accessToken,
		RefreshToken: encodedRefreshToken,
	}, nil
}

func (u *UseCases) RefreshTokens(
	ctx context.Context,
	refreshToken string,
) (*domains.TokensDTO, error) {
	// Decoding refresh token to get original JWT and compare its value with value in Database:
	oldRefreshTokenBytes, err := security.RawDecode(refreshToken)
	if err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	// Retrieving refresh token payload to get access token from refresh token:
	oldRefreshToken := string(oldRefreshTokenBytes)

	refreshTokenPayload, err := security.ParseJWT(
		oldRefreshToken,
		u.securityConfig.JWT.SecretKey,
	)
	if err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	oldAccessToken, ok := refreshTokenPayload.(string)
	if !ok {
		return nil, customerrors.ErrInvalidJWT
	}

	// Retrieving access token payload to get user ID:
	accessTokenPayload, err := security.ParseJWT(
		oldAccessToken,
		u.securityConfig.JWT.SecretKey,
		jwt.WithoutClaimsValidation(), // not validating claims due to expiration of JWT TTL
	)
	if err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	// Selecting refresh token model from Database, if refresh token has not expired yet:
	floatUserID, ok := accessTokenPayload.(float64)
	if !ok {
		return nil, customerrors.ErrInvalidJWT
	}

	userID := uint64(floatUserID)

	dbRefreshToken, err := u.authService.GetRefreshTokenByUserID(ctx, userID)
	if err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	// Checking if access token belongs to refresh token:
	if oldRefreshToken != dbRefreshToken.Value {
		return nil, customerrors.ErrAccessTokenDoesNotBelongToRefreshToken
	}

	// Expiring old refresh token in Database to have only one valid refresh token instance:
	if err = u.authService.ExpireRefreshToken(ctx, dbRefreshToken.ID); err != nil {
		return nil, customerrors.ErrInvalidJWT
	}

	// Create tokens:
	newAccessToken, err := security.GenerateJWT(
		userID,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.AccessTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := security.GenerateJWT(
		newAccessToken,
		u.securityConfig.JWT.SecretKey,
		u.securityConfig.JWT.RefreshTokenTTL,
		u.securityConfig.JWT.Algorithm,
	)
	if err != nil {
		return nil, err
	}

	// Save token to Database:
	if _, err = u.authService.CreateRefreshToken(
		ctx,
		userID,
		newRefreshToken,
		u.securityConfig.JWT.RefreshTokenTTL,
	); err != nil {
		return nil, err
	}

	// Encoding refresh token for secure usage via internet:
	encodedRefreshToken := security.RawEncode([]byte(newRefreshToken))

	return &domains.TokensDTO{
		AccessToken:  newAccessToken,
		RefreshToken: encodedRefreshToken,
	}, nil
}

func (u *UseCases) LogoutUser(ctx context.Context, userID uint64) error {
	refreshToken, _ := u.authService.GetRefreshTokenByUserID(ctx, userID)
	if refreshToken == nil {
		return nil
	}

	return u.authService.ExpireRefreshToken(ctx, refreshToken.ID)
}

func (u *UseCases) VerifyEmail(ctx context.Context, verifyEmailToken string) error {
	decodedToken, err := security.RawDecode(verifyEmailToken)
	if err != nil {
		return customerrors.ErrInvalidJWT
	}

	_, rawUserID, found := strings.Cut(string(decodedToken), common.SaltSeparator)
	if !found {
		return customerrors.ErrInvalidJWT
	}

	intUserID, err := strconv.ParseUint(rawUserID, 10, 64)
	if err != nil {
		return err
	}

	user, err := u.usersService.GetUserByID(ctx, intUserID)
	if err != nil {
		return err
	}

	if user.EmailConfirmed {
		return customerrors.ErrEmailAlreadyConfirmed
	}

	return u.authService.VerifyEmail(ctx, user.ID)
}

func (u *UseCases) ForgetPassword(
	ctx context.Context,
	forgetPasswordToken, newPassword string,
) error {
	if !validation.ValidateValueByRules(newPassword, u.validationConfig.PasswordRegExps) {
		return fmt.Errorf("%w: invalid password", customerrors.ErrValidationFailed)
	}

	decodedToken, err := security.RawDecode(forgetPasswordToken)
	if err != nil {
		return customerrors.ErrInvalidJWT
	}

	_, rawUserID, found := strings.Cut(string(decodedToken), common.SaltSeparator)
	if !found {
		return customerrors.ErrInvalidJWT
	}

	intUserID, err := strconv.ParseUint(rawUserID, 10, 64)
	if err != nil {
		return err
	}

	user, err := u.usersService.GetUserByID(ctx, intUserID)
	if err != nil {
		return err
	}

	if security.ValidateHash(newPassword, user.Password) {
		return fmt.Errorf(
			"%w: new password can not be equal to old password",
			customerrors.ErrNewPasswordEqualToOldPassword,
		)
	}

	hashedPassword, err := security.Hash(newPassword, u.securityConfig.HashCost)
	if err != nil {
		return err
	}

	return u.authService.ForgetPassword(ctx, user.ID, hashedPassword)
}

func (u *UseCases) ChangePassword(ctx context.Context, dto domains.ChangePasswordDTO) error {
	if dto.OldPassword == dto.NewPassword {
		return fmt.Errorf(
			"%w: new password can not be equal to old password",
			customerrors.ErrValidationFailed,
		)
	}

	if !validation.ValidateValueByRules(dto.NewPassword, u.validationConfig.PasswordRegExps) {
		return fmt.Errorf("%w: invalid password", customerrors.ErrValidationFailed)
	}

	user, err := u.usersService.GetUserByID(ctx, dto.UserID)
	if err != nil {
		return err
	}

	if !security.ValidateHash(dto.OldPassword, user.Password) {
		return customerrors.ErrWrongPassword
	}

	hashedPassword, err := security.Hash(dto.NewPassword, u.securityConfig.HashCost)
	if err != nil {
		return err
	}

	return u.authService.ChangePassword(ctx, dto.UserID, hashedPassword)
}

func (u *UseCases) SendVerifyEmailMessage(ctx context.Context, email string) error {
	user, err := u.usersService.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}

	if user.EmailConfirmed {
		return customerrors.ErrEmailAlreadyConfirmed
	}

	return u.authService.SendVerifyEmailMessage(ctx, email)
}

func (u *UseCases) SendForgetPasswordMessage(ctx context.Context, email string) error {
	user, err := u.usersService.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}

	if !user.EmailConfirmed {
		return customerrors.ErrEmailNotConfirmed
	}

	return u.authService.SendForgetPasswordMessage(ctx, email)
}
