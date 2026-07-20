package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrUnauthenticated = errors.New("authentication required")

type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Username     string
	DisplayName  string
	PasswordHash string
	Locked       bool
	Permissions  []string
}

type Session struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

type Store interface {
	FindUserByUsername(ctx context.Context, username string) (User, error)
	CreateSession(ctx context.Context, session Session) error
	FindSessionByTokenHash(ctx context.Context, tokenHash string) (SessionUser, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
}

type SessionUser struct {
	Session Session
	User    User
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

type Service struct {
	store Store
	ttl   time.Duration
	now   func() time.Time
}

func NewService(store Store, ttl time.Duration) *Service {
	return &Service{store: store, ttl: ttl, now: time.Now}
}

func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	user, err := s.store.FindUserByUsername(ctx, strings.TrimSpace(username))
	if err != nil || user.Locked || !VerifyPassword(user.PasswordHash, password) {
		return LoginResult{}, ErrInvalidCredentials
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return LoginResult{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := tokenHash(token)
	expiresAt := s.now().Add(s.ttl)
	session := Session{ID: uuid.New(), TenantID: user.TenantID, UserID: user.ID, TokenHash: hash, ExpiresAt: expiresAt}
	if err := s.store.CreateSession(ctx, session); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	result, err := s.store.FindSessionByTokenHash(ctx, tokenHash(token))
	if err != nil || result.User.Locked || !result.Session.ExpiresAt.After(s.now()) {
		return User{}, ErrUnauthenticated
	}
	return result.User, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSessionByTokenHash(ctx, tokenHash(token))
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
