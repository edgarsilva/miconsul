package auth

import (
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func TestRefreshSessionUser(t *testing.T) {
	user := models.User{ID: 1, UID: "user_1", Email: "u@example.com", Name: "Old Name"}
	renamed := user
	renamed.Name = "New Name"

	stale := newAuthSnapshot(user, "token_1")
	stale.CachedAtUnix = time.Now().Add(-time.Minute).Unix()

	tests := []struct {
		name     string
		cached   string
		user     models.User
		wantName string
		wantSame bool
	}{
		{name: "refreshes the cached user and keeps its sign-in", cached: stale.EncodeQuery(), user: renamed, wantName: "New Name"},
		{name: "leaves another user's cache alone", cached: stale.EncodeQuery(), user: models.User{ID: 2, UID: "user_2", Name: "Other"}, wantSame: true},
		{name: "does nothing without a cached user", cached: "", user: renamed, wantSame: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mapSessionStore{values: map[string]string{authSessionKey: tt.cached}}

			if err := RefreshSessionUser(nil, store, tt.user); err != nil {
				t.Fatalf("RefreshSessionUser: %v", err)
			}

			if tt.wantSame {
				if store.values[authSessionKey] != tt.cached {
					t.Fatalf("expected the session unchanged")
				}
				return
			}

			snapshot, ok := DecodeAuthSnapshotQuery(store.values[authSessionKey])
			if !ok {
				t.Fatalf("expected a cached user")
			}
			if snapshot.UserName != tt.wantName || snapshot.Token != stale.Token {
				t.Fatalf("expected name %q with token %q, got %q with %q", tt.wantName, stale.Token, snapshot.UserName, snapshot.Token)
			}
			if !snapshot.isFresh(time.Now()) || snapshot.CachedAtUnix <= stale.CachedAtUnix {
				t.Fatalf("expected a freshly cached user")
			}
		})
	}
}

type mapSessionStore struct {
	values map[string]string
}

func (m *mapSessionStore) SessionRead(c fiber.Ctx, key string, defaultVal string) string {
	if v, ok := m.values[key]; ok {
		return v
	}
	return defaultVal
}

func (m *mapSessionStore) SessionWrite(c fiber.Ctx, k string, v any) error {
	m.values[k] = v.(string)
	return nil
}
