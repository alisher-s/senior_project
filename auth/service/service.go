package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"

	"github.com/nu/student-event-ticketing-platform/auth/model"
	"github.com/nu/student-event-ticketing-platform/auth/repository"
	"github.com/nu/student-event-ticketing-platform/internal/config"
)

type Service struct {
	cfg           config.Config
	users        repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	jwt           authx.JWT
}

func New(
	cfg config.Config,
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	jwt authx.JWT,
) *Service {
	return &Service{
		cfg:           cfg,
		users:        users,
		refreshTokens: refreshTokens,
		jwt:           jwt,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (model.User, string, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if err := validateEmail(email); err != nil {
		return model.User{}, "", "", ErrEmailNotAllowed
	}
	if !strings.HasSuffix(email, "@"+strings.ToLower(s.cfg.Auth.NuEmailDomain)) {
		return model.User{}, "", "", ErrEmailNotAllowed
	}
	if len(password) < 8 {
		return model.User{}, "", "", errors.New("password too short")
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.Auth.BcryptCost)
	if err != nil {
		return model.User{}, "", "", err
	}

	u, err := s.users.CreateUser(ctx, email, string(pwHash), model.RoleStudent)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			return model.User{}, "", "", ErrEmailAlreadyExists
		}
		return model.User{}, "", "", err
	}

	access, err := s.jwt.GenerateAccessToken(u.ID, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	refresh, jti, expiresAt, err := s.jwt.GenerateRefreshToken(u.ID, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}
	if err := s.refreshTokens.InsertRefreshToken(ctx, jti, u.ID, expiresAt); err != nil {
		return model.User{}, "", "", err
	}

	return u, access, refresh, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (model.User, string, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return model.User{}, "", "", ErrInvalidCredentials
		}
		return model.User{}, "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return model.User{}, "", "", ErrInvalidCredentials
	}

	access, err := s.jwt.GenerateAccessToken(u.ID, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	refresh, jti, expiresAt, err := s.jwt.GenerateRefreshToken(u.ID, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	// Security choice: rotate refresh token on login so only one refresh token is usable.
	// We do revoke+insert atomically and serialize concurrent logins for the same user.
	if err := s.refreshTokens.RotateRefreshToken(ctx, u.ID, jti, expiresAt); err != nil {
		return model.User{}, "", "", err
	}

	return u, access, refresh, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (model.User, string, string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	// Role check prevents using a refresh token from another user/role scenario.
	if authx.Role(u.Role) != claims.Role {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return model.User{}, "", "", ErrRefreshTokenInvalid
	}

	now := time.Now().UTC()
	ok, err := s.refreshTokens.ConsumeRefreshToken(ctx, jti, u.ID, now)
	if err != nil {
		return model.User{}, "", "", err
	}
	if !ok {
		return model.User{}, "", "", ErrRefreshTokenConsumed
	}

	access, err := s.jwt.GenerateAccessToken(u.ID, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}

	refresh, newJti, expiresAt, err := s.jwt.GenerateRefreshToken(u.ID, authx.Role(u.Role))
	if err != nil {
		return model.User{}, "", "", err
	}
	if err := s.refreshTokens.InsertRefreshToken(ctx, newJti, u.ID, expiresAt); err != nil {
		return model.User{}, "", "", err
	}

	return u, access, refresh, nil
}

func validateEmail(email string) error {
	// net/mail is good enough for domain-level validation.
	// For production, consider stricter MX checks.
	_, err := mail.ParseAddress(email)
	return err
}

