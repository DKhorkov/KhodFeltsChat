package usecases

import (
	"context"
	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/security"
	"github.com/DKhorkov/libs/validation"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
)

func NewAuthUseCases(
	authService interfaces.AuthService,
	usersService interfaces.UsersService,
	securityConfig security.Config,
	validationConfig config.ValidationConfig,
) *AuthUseCases {
	return &AuthUseCases{
		authService:      authService,
		usersService:     usersService,
		securityConfig:   securityConfig,
		validationConfig: validationConfig,
	}
}

type AuthUseCases struct {
	authService      interfaces.AuthService
	usersService     interfaces.UsersService
	securityConfig   security.Config
	validationConfig config.ValidationConfig
}

func (u *AuthUseCases) RegisterUser(
	ctx context.Context,
	userData domains.RegisterDTO,
) (*domains.User, error) {
	if !validation.ValidateValueByRule(userData.Email, u.validationConfig.EmailRegExp) {
		return nil, &validation.Error{Message: "invalid email address"}
	}

	if !validation.ValidateValueByRules(userData.Password, u.validationConfig.PasswordRegExps) {
		return nil, &validation.Error{Message: "invalid password"}
	}

	if !validation.ValidateValueByRules(userData.Username, u.validationConfig.UsernameRegExps) {
		return nil, &validation.Error{Message: "invalid username"}
	}

	hashedPassword, err := security.Hash(userData.Password, u.securityConfig.HashCost)
	if err != nil {
		return nil, err
	}

	userData.Password = hashedPassword

	return u.authService.RegisterUser(ctx, userData)
}

func (u *AuthUseCases) LoginUser(
	ctx context.Context,
	userData domains.LoginDTO,
) (*domains.TokensDTO, error) {
	// Check if user with provided email exists and password is valid:
	user, err := u.usersService.GetUserByEmail(ctx, userData.Email)
	if err != nil {
		return nil, err
	}

	if !user.EmailConfirmed {
		return nil, &customerrors.EmailIsNotConfirmedError{}
	}

	if !security.ValidateHash(userData.Password, user.Password) {
		return nil, &customerrors.WrongPasswordError{}
	}

	if dbRefreshToken, err := u.authService.GetRefreshTokenByUserID(ctx, user.ID); err == nil {
		if err = u.authService.ExpireRefreshToken(ctx, dbRefreshToken.Value); err != nil {
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

func (u *AuthUseCases) RefreshTokens(
	ctx context.Context,
	refreshToken string,
) (*domains.TokensDTO, error) {
	// Decoding refresh token to get original JWT and compare its value with value in Database:
	oldRefreshTokenBytes, err := security.RawDecode(refreshToken)
	if err != nil {
		return nil, &security.InvalidJWTError{}
	}

	// Retrieving refresh token payload to get access token from refresh token:
	oldRefreshToken := string(oldRefreshTokenBytes)

	refreshTokenPayload, err := security.ParseJWT(
		oldRefreshToken,
		u.securityConfig.JWT.SecretKey,
	)
	if err != nil {
		return nil, &security.InvalidJWTError{}
	}

	oldAccessToken, ok := refreshTokenPayload.(string)
	if !ok {
		return nil, &security.InvalidJWTError{}
	}

	// Retrieving access token payload to get user ID:
	accessTokenPayload, err := security.ParseJWT(
		oldAccessToken,
		u.securityConfig.JWT.SecretKey,
		jwt.WithoutClaimsValidation(), // not validating claims due to expiration of JWT TTL
	)
	if err != nil {
		return nil, &security.InvalidJWTError{}
	}

	// Selecting refresh token model from Database, if refresh token has not expired yet:
	floatUserID, ok := accessTokenPayload.(float64)
	if !ok {
		return nil, &security.InvalidJWTError{}
	}

	userID := uint64(floatUserID)

	dbRefreshToken, err := u.authService.GetRefreshTokenByUserID(ctx, userID)
	if err != nil {
		return nil, &security.InvalidJWTError{}
	}

	// Checking if access token belongs to refresh token:
	if oldRefreshToken != dbRefreshToken.Value {
		return nil, &customerrors.AccessTokenDoesNotBelongToRefreshTokenError{}
	}

	// Expiring old refresh token in Database to have only one valid refresh token instance:
	if err = u.authService.ExpireRefreshToken(ctx, dbRefreshToken.Value); err != nil {
		return nil, &security.InvalidJWTError{}
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

func (u *AuthUseCases) LogoutUser(ctx context.Context, accessToken string) error {
	accessTokenPayload, err := security.ParseJWT(accessToken, u.securityConfig.JWT.SecretKey)
	if err != nil {
		return &security.InvalidJWTError{}
	}

	floatUserID, ok := accessTokenPayload.(float64)
	if !ok {
		return &security.InvalidJWTError{}
	}

	userID := uint64(floatUserID)

	refreshToken, _ := u.authService.GetRefreshTokenByUserID(ctx, userID)
	if refreshToken == nil {
		return nil
	}

	return u.authService.ExpireRefreshToken(ctx, refreshToken.Value)
}

func (u *AuthUseCases) VerifyUserEmail(ctx context.Context, verifyEmailToken string) error {
	strUserID, err := security.RawDecode(verifyEmailToken)
	if err != nil {
		return err
	}

	intUserID, err := strconv.Atoi(string(strUserID))
	if err != nil {
		return err
	}

	user, err := u.usersService.GetUserByID(ctx, uint64(intUserID))
	if err != nil {
		return err
	}

	if user.EmailConfirmed {
		return &customerrors.EmailAlreadyConfirmedError{}
	}

	return u.authService.VerifyEmail(ctx, user.ID)
}

func (u *AuthUseCases) ForgetPassword(ctx context.Context, forgetPasswordToken, newPassword string) error {
	if !validation.ValidateValueByRules(newPassword, u.validationConfig.PasswordRegExps) {
		return &validation.Error{Message: "invalid password"}
	}

	strUserID, err := security.RawDecode(forgetPasswordToken)
	if err != nil {
		return err
	}

	intUserID, err := strconv.Atoi(string(strUserID))
	if err != nil {
		return err
	}

	user, err := u.usersService.GetUserByID(ctx, uint64(intUserID))
	if err != nil {
		return err
	}

	if security.ValidateHash(newPassword, user.Password) {
		return &validation.Error{Message: "new password can not be equal to old password"}
	}

	hashedPassword, err := security.Hash(newPassword, u.securityConfig.HashCost)
	if err != nil {
		return err
	}

	return u.authService.ForgetPassword(ctx, user.ID, hashedPassword)
}

func (u *AuthUseCases) ChangePassword(
	ctx context.Context,
	accessToken string,
	oldPassword string,
	newPassword string,
) error {
	if oldPassword == newPassword {
		return &validation.Error{Message: "new password can not be equal to old password"}
	}

	if !validation.ValidateValueByRules(newPassword, u.validationConfig.PasswordRegExps) {
		return &validation.Error{Message: "invalid password"}
	}

	accessTokenPayload, err := security.ParseJWT(accessToken, u.securityConfig.JWT.SecretKey)
	if err != nil {
		return &security.InvalidJWTError{}
	}

	floatUserID, ok := accessTokenPayload.(float64)
	if !ok {
		return &security.InvalidJWTError{}
	}

	userID := uint64(floatUserID)

	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if !security.ValidateHash(oldPassword, user.Password) {
		return &customerrors.WrongPasswordError{}
	}

	hashedPassword, err := security.Hash(newPassword, u.securityConfig.HashCost)
	if err != nil {
		return err
	}

	return u.authService.ChangePassword(ctx, userID, hashedPassword)
}

func (u *AuthUseCases) SendVerifyEmailMessage(ctx context.Context, email string) error {
	return u.authService.SendVerifyEmailMessage(ctx, email)
}

func (u *AuthUseCases) SendForgetPasswordMessage(ctx context.Context, email string) error {
	return u.authService.SendForgetPasswordMessage(ctx, email)
}
