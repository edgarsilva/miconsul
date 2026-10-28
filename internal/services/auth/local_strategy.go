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

// Authenticate returns the user of a bearer token (API clients) or of the
// signed-in session.
func (ls LocalStrategy) Authenticate(c fiber.Ctx) (models.User, error) {
	if token, ok := bearerToken(c); ok {
		return ls.authenticateWithJWT(c, token)
	}

	return ls.authenticateWithSession(c)
}

// SignIn starts a session for user under a new session ID. The session holds
// a JWT that expires after 24h, or 7 days with rememberMe; a request in the
// token's last hour refreshes it.
func (ls LocalStrategy) SignIn(c fiber.Ctx, user models.User, rememberMe bool) error {
	token, err := JWTCreateTokenWithTTL(ls.resource.AppEnv(), user.Email, user.UID, authTokenTTL(rememberMe), rememberMe)
	if err != nil {
		return fmt.Errorf("%w: %w", errAuthSessionCreate, err)
	}

	snapshot := newAuthSnapshot(user, token)
	if err := ls.resource.SessionRotate(c, authSessionKey, snapshot.EncodeQuery()); err != nil {
		return fmt.Errorf("%w: %w", errAuthSessionCreate, err)
	}

	return nil
}

func (ls LocalStrategy) Metadata() AuthenticatorMeta {
	return AuthenticatorMeta{}
}

func (ls LocalStrategy) FindUserByID(ctx context.Context, uid string) (models.User, error) {
	return gorm.G[models.User](ls.resource.GormDB()).Where("uid = ?", uid).Take(ctx)
}

func (ls LocalStrategy) authenticateWithJWT(c fiber.Ctx, token string) (models.User, error) {
	claims, err := decodeJWTToken(ls.resource.AppEnv(), token)
	if err != nil {
		return models.User{}, errors.New("validate JWT token")
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		uid = ""
	}

	user, err := ls.FindUserByID(c.Context(), uid)
	if err != nil {
		return user, errors.New("find user with UID in JWT token")
	}

	return user, nil
}

// authenticateWithSession returns the session's user while its JWT is valid.
// It reloads the user from the DB once the cached copy is older than
// authSessionHydrationTTL, and refreshes the JWT when it is about to expire.
func (ls LocalStrategy) authenticateWithSession(c fiber.Ctx) (models.User, error) {
	snapshot, ok := DecodeAuthSnapshotQuery(ls.resource.SessionRead(c, authSessionKey, ""))
	if !ok {
		return models.User{}, errors.New("no signed-in session")
	}

	claims, err := decodeJWTToken(ls.resource.AppEnv(), snapshot.Token)
	if err != nil || claims["uid"] != snapshot.UserUID {
		ls.signOutSession(c)
		return models.User{}, errors.New("session token expired or invalid")
	}

	user := snapshot.toUser()
	changed := false

	if !snapshot.isFresh(time.Now()) {
		user, err = ls.FindUserByID(c.Context(), snapshot.UserUID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ls.signOutSession(c)
			return models.User{}, errors.New("signed-in user not found")
		}
		if err != nil {
			return models.User{}, fmt.Errorf("find signed-in user: %w", err)
		}

		snapshot = newAuthSnapshot(user, snapshot.Token)
		changed = true
	}

	token, _, err := RefreshJWTToken(ls.resource.AppEnv(), snapshot.Token, claims)
	if err != nil {
		return models.User{}, fmt.Errorf("refresh session token: %w", err)
	}
	if token != snapshot.Token {
		snapshot.Token = token
		changed = true
	}

	if changed {
		if err := ls.resource.SessionWrite(c, authSessionKey, snapshot.EncodeQuery()); err != nil {
			log.Warn("save refreshed auth session:", err)
		}
	}

	return user, nil
}

// signOutSession removes the signed-in user from the session.
func (ls LocalStrategy) signOutSession(c fiber.Ctx) {
	if err := ls.resource.SessionWrite(c, authSessionKey, ""); err != nil {
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
