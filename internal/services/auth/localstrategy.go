package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/model"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	authSessionKey          = "auth"
	authSessionHydrationTTL = 10 * time.Minute
)

type AuthSnapshot struct {
	Token          string         // t
	UserID         string         // uid
	UserEmail      string         // em
	UserRole       model.UserRole // rl
	UserName       string         // nm
	UserProfilePic string         // pp
	CachedAtUnix   int64          // cat
}

type LocalStrategy struct {
	resource LocalStrategyResource
}

type LocalStrategyResource interface {
	GormDB() *gorm.DB
	NewCookie(name, value string, validFor time.Duration) *fiber.Cookie
	AppEnv() *appenv.Env
	SessionRead(c fiber.Ctx, key string, defaultVal string) string
	SessionWrite(c fiber.Ctx, k string, v any) error
}

func NewLocalStrategy(resource LocalStrategyResource) *LocalStrategy {
	return &LocalStrategy{
		resource: resource,
	}
}

func (ls LocalStrategy) Authenticate(c fiber.Ctx) (model.User, error) {
	token := getToken(c)
	if token == "" {
		return model.User{}, errors.New("failed to retrieve JWT token, it is blank")
	}

	userFromSession, ok := ls.currentUserFromSession(c, token)
	if ok {
		return userFromSession, nil
	}

	user, err := ls.authenticateWithJWT(c, token)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	ls.saveCurrentUserToSession(c, token, user)

	return user, nil
}

func (ls LocalStrategy) Metadata() AuthenticatorMeta {
	return AuthenticatorMeta{}
}

func (ls LocalStrategy) FindUserById(ctx context.Context, uid string) (model.User, error) {
	return gorm.G[model.User](ls.resource.GormDB()).Where("id = ?", uid).Take(ctx)
}

func (ls LocalStrategy) authenticateWithJWT(c fiber.Ctx, token string) (model.User, error) {
	claims, err := decodeJWTToken(ls.resource.AppEnv(), token)
	if err != nil {
		return model.User{}, errors.New("failed to validate JWT token")
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		uid = ""
	}

	user, err := ls.FindUserById(c.Context(), uid)
	if err != nil {
		return user, errors.New("failed to find user with UID in JWT token")
	}

	refreshedJWT, validFor, err := RefreshJWTToken(ls.resource.AppEnv(), token, claims)
	if err != nil {
		return user, errors.New("failed to refresh JWT token")
	}
	if refreshedJWT != token {
		ls.refreshAuthCookie(c, refreshedJWT, validFor)
	}

	return user, nil
}

// getToken returns the JWT token from the request
func getToken(c fiber.Ctx) string {
	token := c.Cookies("Auth", "")
	if token == "" {
		token = strings.TrimPrefix(c.Get("Authorization", ""), "Bearer ")
	}

	return token
}

func (ls LocalStrategy) refreshAuthCookie(c fiber.Ctx, jwt string, validFor time.Duration) {
	if validFor <= 0 {
		validFor = defaultAuthTokenTTL
	}

	c.Cookie(ls.resource.NewCookie("Auth", jwt, validFor))
}

func (ls LocalStrategy) currentUserFromSession(c fiber.Ctx, token string) (model.User, bool) {
	snapshot, ok := ls.readAuthSnapshot(c)
	if !ok {
		return model.User{}, false
	}

	tokenHash := tokenDigest(token)
	isInvalidToken := !snapshot.isValidForToken(tokenHash, time.Now())
	if isInvalidToken {
		return model.User{}, false
	}

	return snapshot.toUser(), true
}

func (ls LocalStrategy) saveCurrentUserToSession(c fiber.Ctx, token string, user model.User) {
	if token == "" || user.ID == "" {
		return
	}

	ls.writeAuthSnapshot(c, AuthSnapshot{
		Token:          tokenDigest(token),
		UserID:         user.ID,
		UserEmail:      user.Email,
		UserRole:       user.Role,
		UserName:       user.Name,
		UserProfilePic: user.ProfilePic,
		CachedAtUnix:   time.Now().Unix(),
	})
}

func tokenDigest(token string) string {
	if token == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (ls LocalStrategy) readAuthSnapshot(c fiber.Ctx) (AuthSnapshot, bool) {
	raw := ls.resource.SessionRead(c, authSessionKey, "")
	return DecodeAuthSnapshot(raw)
}

func (ls LocalStrategy) writeAuthSnapshot(c fiber.Ctx, snapshot AuthSnapshot) {
	_ = ls.resource.SessionWrite(c, authSessionKey, EncodeAuthSnapshot(snapshot))
}

func EncodeAuthSnapshot(sa AuthSnapshot) string {
	v := url.Values{}
	v.Set("t", sa.Token)
	v.Set("uid", sa.UserID)
	v.Set("em", sa.UserEmail)
	v.Set("rl", string(sa.UserRole))
	v.Set("nm", sa.UserName)
	v.Set("pp", sa.UserProfilePic)
	v.Set("cat", strconv.FormatInt(sa.CachedAtUnix, 10))

	return v.Encode()
}

func DecodeAuthSnapshot(raw string) (AuthSnapshot, bool) {
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
		UserID:         v.Get("uid"),
		UserEmail:      v.Get("em"),
		UserRole:       model.UserRole(v.Get("rl")),
		UserName:       v.Get("nm"),
		UserProfilePic: v.Get("pp"),
		CachedAtUnix:   cachedAtUnix,
	}
	if sa.Token == "" || sa.UserID == "" {
		return AuthSnapshot{}, false
	}

	return sa, true
}

func (sa AuthSnapshot) isValidForToken(token string, now time.Time) bool {
	if token == "" || sa.Token == "" || sa.UserID == "" {
		return false
	}
	if sa.Token != token {
		return false
	}
	if sa.CachedAtUnix <= 0 {
		return false
	}

	return now.Sub(time.Unix(sa.CachedAtUnix, 0)) <= authSessionHydrationTTL
}

func (sa AuthSnapshot) toUser() model.User {
	return model.User{
		ID:         sa.UserID,
		Email:      sa.UserEmail,
		Role:       sa.UserRole,
		Name:       sa.UserName,
		ProfilePic: sa.UserProfilePic,
	}
}
