package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/models"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
)

const (
	authSessionKey          = "auth"
	authSessionHydrationTTL = 10 * time.Minute
)

type AuthSnapshot struct {
	Token          string          // t
	UserID         uint            // id
	UserUID        string          // uid
	UserEmail      string          // em
	UserRole       models.UserRole // rl
	UserName       string          // nm
	UserProfilePic string          // pp
	UserTheme      string          // th
	UserLocale     string          // lc
	CachedAtUnix   int64           // cat
}

type LocalStrategy struct {
	resource LocalStrategyResource
}

type LocalStrategyResource interface {
	GormDB() *gorm.DB
	AppEnv() *appenv.Env
	SessionRead(c fiber.Ctx, key string, defaultVal string) string
	SessionWrite(c fiber.Ctx, k string, v any) error
	SessionRotate(c fiber.Ctx, k string, v any) error
}

func NewLocalStrategy(resource LocalStrategyResource) *LocalStrategy {
	return &LocalStrategy{
		resource: resource,
	}
}

// SessionUserStore reads and writes request session values.
type SessionUserStore interface {
	SessionRead(c fiber.Ctx, key string, defaultVal string) string
	SessionWrite(c fiber.Ctx, k string, v any) error
}

// RefreshSessionUser replaces the session's cached user with user, keeping the
// sign-in it belongs to. It does nothing when the session caches another user
// or none.
func RefreshSessionUser(c fiber.Ctx, store SessionUserStore, user models.User) error {
	snapshot, ok := DecodeAuthSnapshotQuery(store.SessionRead(c, authSessionKey, ""))
	if !ok || snapshot.UserUID != user.UID {
		return nil
	}

	return store.SessionWrite(c, authSessionKey, newAuthSnapshot(user, snapshot.Token).EncodeQuery())
}

// Authenticate returns the user of a bearer token (API clients) or of the
// signed-in session.
func (s LocalStrategy) Authenticate(c fiber.Ctx) (models.User, error) {
	if token, ok := bearerToken(c); ok {
		return s.authenticateWithJWT(c, token)
	}

	return s.authenticateWithSession(c)
}

// SignIn starts a session for user under a new session ID. The session holds
// a JWT that expires after 24h, or 7 days with rememberMe; a request once less
// than half of its lifetime is left refreshes it.
func (s LocalStrategy) SignIn(c fiber.Ctx, user models.User, rememberMe bool) error {
	token, err := JWTCreateToken(s.resource.AppEnv(), NewAuthClaims(user, rememberMe))
	if err != nil {
		return fmt.Errorf("%w: %w", errAuthSessionCreate, err)
	}

	snapshot := newAuthSnapshot(user, token)
	if err := s.resource.SessionRotate(c, authSessionKey, snapshot.EncodeQuery()); err != nil {
		return fmt.Errorf("%w: %w", errAuthSessionCreate, err)
	}

	return nil
}

func (s LocalStrategy) Metadata() AuthenticatorMeta {
	return AuthenticatorMeta{}
}

func (s LocalStrategy) FindUserByID(ctx context.Context, uid string) (models.User, error) {
	return gorm.G[models.User](s.resource.GormDB()).Where("uid = ?", uid).Take(ctx)
}

func (s LocalStrategy) authenticateWithJWT(c fiber.Ctx, token string) (models.User, error) {
	claims, err := decodeJWTToken(s.resource.AppEnv(), token)
	if err != nil {
		return models.User{}, errors.New("validate JWT token")
	}

	user, err := s.FindUserByID(c.Context(), claims.UID)
	if err != nil {
		return user, errors.New("find user with UID in JWT token")
	}

	return user, nil
}

// authenticateWithSession returns the session's user while its JWT is valid.
// It reloads the user from the DB once the cached copy is older than
// authSessionHydrationTTL, and refreshes the JWT once less than half of its
// lifetime is left.
func (s LocalStrategy) authenticateWithSession(c fiber.Ctx) (models.User, error) {
	snapshot, ok := DecodeAuthSnapshotQuery(s.resource.SessionRead(c, authSessionKey, ""))
	if !ok {
		return models.User{}, errors.New("no signed-in session")
	}

	claims, err := decodeJWTToken(s.resource.AppEnv(), snapshot.Token)
	if err != nil || claims.UID != snapshot.UserUID {
		s.signOutSession(c)
		return models.User{}, errors.New("session token expired or invalid")
	}

	now := time.Now()
	user := snapshot.toUser()
	changed := false

	if !snapshot.isFresh(now) {
		user, err = s.FindUserByID(c.Context(), snapshot.UserUID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.signOutSession(c)
			return models.User{}, errors.New("signed-in user not found")
		}
		if err != nil {
			return models.User{}, fmt.Errorf("find signed-in user: %w", err)
		}

		snapshot = newAuthSnapshot(user, snapshot.Token)
		changed = true
	}

	if claims.needsRefresh(now) {
		token, err := JWTCreateToken(s.resource.AppEnv(), claims.renewed(now))
		if err != nil {
			return models.User{}, fmt.Errorf("refresh session token: %w", err)
		}

		snapshot.Token = token
		changed = true
	}

	if changed {
		if err := s.resource.SessionWrite(c, authSessionKey, snapshot.EncodeQuery()); err != nil {
			log.Warn("save refreshed auth session:", err)
		}
	}

	return user, nil
}

// signOutSession removes the signed-in user from the session.
func (s LocalStrategy) signOutSession(c fiber.Ctx) {
	if err := s.resource.SessionWrite(c, authSessionKey, ""); err != nil {
		log.Warn("remove signed-in user from session:", err)
	}
}

// bearerToken returns the token of an "Authorization: Bearer" header.
func bearerToken(c fiber.Ctx) (string, bool) {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok || token == "" {
		return "", false
	}

	return token, true
}

func newAuthSnapshot(user models.User, token string) AuthSnapshot {
	return AuthSnapshot{
		Token:          token,
		UserID:         user.ID,
		UserUID:        user.UID,
		UserEmail:      user.Email,
		UserRole:       user.Role,
		UserName:       user.Name,
		UserProfilePic: user.ProfilePic,
		UserTheme:      user.Theme,
		UserLocale:     user.Locale,
		CachedAtUnix:   time.Now().Unix(),
	}
}

// EncodeQuery encodes the snapshot as URL query values.
func (sa AuthSnapshot) EncodeQuery() string {
	v := url.Values{}
	v.Set("t", sa.Token)
	v.Set("id", strconv.FormatUint(uint64(sa.UserID), 10))
	v.Set("uid", sa.UserUID)
	v.Set("em", sa.UserEmail)
	v.Set("rl", string(sa.UserRole))
	v.Set("nm", sa.UserName)
	v.Set("pp", sa.UserProfilePic)
	v.Set("th", sa.UserTheme)
	v.Set("lc", sa.UserLocale)
	v.Set("cat", strconv.FormatInt(sa.CachedAtUnix, 10))

	return v.Encode()
}

// DecodeAuthSnapshotQuery decodes a snapshot encoded by EncodeQuery.
func DecodeAuthSnapshotQuery(raw string) (AuthSnapshot, bool) {
	if raw == "" {
		return AuthSnapshot{}, false
	}

	v, err := url.ParseQuery(raw)
	if err != nil {
		return AuthSnapshot{}, false
	}

	cachedAtUnix, err := strconv.ParseInt(v.Get("cat"), 10, 64)
	if err != nil {
		return AuthSnapshot{}, false
	}

	sa := AuthSnapshot{
		Token:          v.Get("t"),
		UserID:         0,
		UserUID:        v.Get("uid"),
		UserEmail:      v.Get("em"),
		UserRole:       models.UserRole(v.Get("rl")),
		UserName:       v.Get("nm"),
		UserProfilePic: v.Get("pp"),
		UserTheme:      v.Get("th"),
		UserLocale:     v.Get("lc"),
		CachedAtUnix:   cachedAtUnix,
	}

	userID, err := strconv.ParseUint(v.Get("id"), 10, 64)
	if err != nil {
		return AuthSnapshot{}, false
	}
	sa.UserID = uint(userID)

	if sa.Token == "" || sa.UserUID == "" || sa.UserID == 0 {
		return AuthSnapshot{}, false
	}

	return sa, true
}

// isFresh reports whether the cached user is recent enough to skip the DB.
func (sa AuthSnapshot) isFresh(now time.Time) bool {
	if sa.CachedAtUnix <= 0 {
		return false
	}

	return now.Sub(time.Unix(sa.CachedAtUnix, 0)) <= authSessionHydrationTTL
}

func (sa AuthSnapshot) toUser() models.User {
	return models.User{
		ID:         sa.UserID,
		UID:        sa.UserUID,
		Email:      sa.UserEmail,
		Role:       sa.UserRole,
		Name:       sa.UserName,
		ProfilePic: sa.UserProfilePic,
		Theme:      sa.UserTheme,
		Locale:     sa.UserLocale,
	}
}
