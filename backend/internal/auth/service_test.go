package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	user     User
	findErr  error
	sessions []Session
}

func (f *fakeStore) FindUserByUsername(context.Context, string) (User, error) {
	return f.user, f.findErr
}
func (f *fakeStore) CreateSession(_ context.Context, session Session) error {
	f.sessions = append(f.sessions, session)
	return nil
}
func (f *fakeStore) FindSessionByTokenHash(_ context.Context, tokenHash string) (SessionUser, error) {
	for _, session := range f.sessions {
		if session.TokenHash == tokenHash {
			return SessionUser{Session: session, User: f.user}, nil
		}
	}
	return SessionUser{}, errors.New("not found")
}
func (f *fakeStore) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	for index, session := range f.sessions {
		if session.TokenHash == tokenHash {
			f.sessions = append(f.sessions[:index], f.sessions[index+1:]...)
			return nil
		}
	}
	return nil
}

func TestLoginCreatesHashedOpaqueSession(t *testing.T) {
	passwordHash, _ := HashPassword("warehouse-secret")
	store := &fakeStore{user: User{ID: uuid.New(), TenantID: uuid.New(), Username: "warehouse", PasswordHash: passwordHash}}
	service := NewService(store, time.Hour)

	result, err := service.Login(context.Background(), "warehouse", "warehouse-secret")

	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected opaque session token")
	}
	if len(store.sessions) != 1 {
		t.Fatalf("expected one session, got %d", len(store.sessions))
	}
	if store.sessions[0].TokenHash == result.Token {
		t.Fatal("raw token persisted")
	}
	if store.sessions[0].UserID != store.user.ID {
		t.Fatal("session user mismatch")
	}
}

func TestLoginUsesSamePublicErrorForUnknownWrongAndLockedUsers(t *testing.T) {
	passwordHash, _ := HashPassword("correct-password")
	cases := []fakeStore{
		{findErr: errors.New("not found")},
		{user: User{ID: uuid.New(), TenantID: uuid.New(), PasswordHash: passwordHash}},
		{user: User{ID: uuid.New(), TenantID: uuid.New(), PasswordHash: passwordHash, Locked: true}},
	}
	passwords := []string{"anything", "wrong-password", "correct-password"}
	for i := range cases {
		_, err := NewService(&cases[i], time.Hour).Login(context.Background(), "user", passwords[i])
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("case %d returned %v", i, err)
		}
	}
}

func TestAuthenticateAcceptsActiveTokenAndLogoutRevokesIt(t *testing.T) {
	passwordHash, _ := HashPassword("warehouse-secret")
	store := &fakeStore{user: User{ID: uuid.New(), TenantID: uuid.New(), Username: "warehouse", PasswordHash: passwordHash}}
	service := NewService(store, time.Hour)
	login, err := service.Login(context.Background(), "warehouse", "warehouse-secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	user, err := service.Authenticate(context.Background(), login.Token)
	if err != nil || user.ID != store.user.ID {
		t.Fatalf("Authenticate returned user=%v err=%v", user.ID, err)
	}
	if err := service.Logout(context.Background(), login.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), login.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected revoked token rejection, got %v", err)
	}
}
