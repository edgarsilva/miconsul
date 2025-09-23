package appenv

import (
	"log"

	"github.com/edgarsilva/simpleenv"
)

type Env struct {
	AppName     string `env:"APP_NAME"`
	AppEnv      string `env:"APP_ENV"`
	AppProtocol string `env:"APP_PROTOCOL"`
	AppDomain   string `env:"APP_DOMAIN"`
	AppVersion  string `env:"APP_VERSION"`
	AppPort     string `env:"APP_PORT"`

	CookieSecret string `env:"COOKIE_SECRET"`
	JWTSecret    string `env:"JWT_SECRET"`

	DBPath        string `env:"DB_PATH"`
	SessionDBPath string `env:"SESSION_DB_PATH"`
	CacheDBPath   string `env:"CACHE_DB_PATH;optional"`

	EmailSender      string `env:"EMAIL_SENDER"`
	EmailSecret      string `env:"EMAIL_SECRET"`
	EmailFromAddress string `env:"EMAIL_FROM_ADDRESS"`
	EmailSMTPURL     string `env:"EMAIL_SMTP_URL"`

	GooseDriver       string `env:"GOOSE_DRIVER"`
	GooseDBString     string `env:"GOOSE_DBSTRING"`
	GooseMigrationDir string `env:"GOOSE_MIGRATION_DIR"`

	LogtoURL       string `env:"LOGTO_URL;optional"`
	LogtoAppID     string `env:"LOGTO_APP_ID;optional"`
	LogtoAppSecret string `env:"LOGTO_APP_SECRET;optional"`

	UptraceDSN      string `env:"UPTRACE_DSN;optional"`
	UptraceEndpoint string `env:"UPTRACE_ENDPOINT;optional"`

	AssetsDir string `env:"ASSETS_DIR"`
}

func New() *Env {
	e := Env{}
	err := simpleenv.Load(&e)
	if err != nil {
		log.Fatal("failed to load loading ENV variables:", err)
	}

	return &e
}
